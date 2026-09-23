// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

package delivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// DecodeResponse decodes one complete snapshot, permitting additive informational
// fields. Transport status and response size must be checked by the HTTP caller.
func DecodeResponse(data []byte, request Request) (Response, error) {
	// Zero is a valid tombstone generation; absence or null must not invent it.
	response := Response{Cursor: Cursor{RackDeliveryGeneration: -1}}
	if err := json.Unmarshal(data, &response); err != nil {
		return Response{}, fmt.Errorf("decode node configuration: %w", err)
	}
	if err := response.Validate(request); err != nil {
		return Response{}, err
	}
	return response, nil
}

// Validate checks the wire invariants and the request's identity and lower bound.
// It does not acknowledge application or prove server-side cache consistency.
func (r Response) Validate(request Request) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("invalid configuration request: %w", err)
	}
	if r.APIVersion != APIVersion {
		return fmt.Errorf("unsupported apiVersion %q", r.APIVersion)
	}
	if r.Node != request.Node {
		return errors.New("response nodeName and nodeUID must match the request")
	}
	if err := r.Cursor.validate(); err != nil {
		return err
	}
	if request.LastApplied != nil && r.Cursor.Compare(*request.LastApplied) < 0 {
		return errors.New("configuration regresses the last-applied cursor")
	}
	return r.validateState()
}

func (r Response) validateState() error {
	switch r.State {
	case Assigned:
		if r.RackDeliveryGeneration == 0 {
			return errors.New("assigned configuration requires a positive rackDeliveryGeneration")
		}
		if err := r.Rack.validate(); err != nil {
			return err
		}
		if r.Configuration == nil {
			return errors.New("assigned configuration requires a payload")
		}
		return r.Configuration.validate()
	case Unassigned:
		if r.RackDeliveryGeneration != 0 || r.Rack != nil || r.Configuration != nil {
			return errors.New("unassigned configuration requires generation zero and no rack or payload")
		}
		return nil
	default:
		return fmt.Errorf("unsupported configuration state %q", r.State)
	}
}

func (r *RackReference) validate() error {
	if r == nil || r.Name == "" || r.UID == "" || r.NodeIndex == nil || *r.NodeIndex < 0 {
		return errors.New("assigned configuration requires an exact rack and nonnegative nodeIndex")
	}
	return nil
}

// ValidateSuccessor additionally checks publication invariants against an already
// validated, successfully applied snapshot from the same configured authority.
// Equality is safe to deduplicate only after application in this agent lifecycle.
func (r Response) ValidateSuccessor(applied Response) error {
	if err := r.Validate(Request{Node: applied.Node, LastApplied: &applied.Cursor}); err != nil {
		return err
	}
	if r.AssignmentRevision != applied.AssignmentRevision {
		return nil
	}
	if r.State != applied.State || !reflect.DeepEqual(r.Rack, applied.Rack) {
		return errors.New("assignment changed without advancing assignmentRevision")
	}
	if r.RackDeliveryGeneration == applied.RackDeliveryGeneration && !reflect.DeepEqual(r.Configuration, applied.Configuration) {
		return errors.New("configuration changed at an equal delivery cursor")
	}
	return nil
}
