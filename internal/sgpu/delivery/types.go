// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

// Package delivery defines the node-agent read protocol, independently of
// Kubernetes resource schemas and the agent's local execution settings.
package delivery

import (
	"cmp"
	"errors"

	ibconfig "github.com/NVIDIA/k8s-test-infra/internal/ib/config"
	nvmlconfig "github.com/NVIDIA/k8s-test-infra/pkg/gpu/mocknvml/config"
)

// The API version is independent of the embedded NVML consumer schema version.
const (
	APIVersion   = "v1alpha1"
	RoutePattern = "/v1alpha1/nodes/{nodeName}/configuration"
)

// NodeIdentity distinguishes replacement Nodes that reuse a name. NodeUID is
// opaque Kubernetes identity, not a GPU UUID or a caller credential.
type NodeIdentity struct {
	NodeName string `json:"nodeName"`
	NodeUID  string `json:"nodeUID"`
}

// Cursor orders configurations within one Node UID and control-plane authority.
// A new assignment may move to a rack with a lower delivery generation.
type Cursor struct {
	AssignmentRevision     int64 `json:"assignmentRevision"`
	RackDeliveryGeneration int64 `json:"rackDeliveryGeneration"`
}

// Compare requires cursors from the same Node UID and authority.
func (c Cursor) Compare(other Cursor) int {
	if order := cmp.Compare(c.AssignmentRevision, other.AssignmentRevision); order != 0 {
		return order
	}
	return cmp.Compare(c.RackDeliveryGeneration, other.RackDeliveryGeneration)
}

func (c Cursor) validate() error {
	if c.AssignmentRevision <= 0 || c.RackDeliveryGeneration < 0 {
		return errors.New("assignmentRevision must be positive and rackDeliveryGeneration must be nonnegative")
	}
	return nil
}

// Request binds an optional applied cursor to its exact Node identity.
type Request struct {
	Node        NodeIdentity
	LastApplied *Cursor
}

// State separates authoritative withdrawal from configuration unavailability.
type State string

// Assigned and Unassigned are the only successful delivery states.
const (
	Assigned   State = "assigned"
	Unassigned State = "unassigned"
)

// RackReference identifies a logical slot, not a physical chassis location.
type RackReference struct {
	Name      string `json:"name"`
	UID       string `json:"uid"`
	NodeIndex *int32 `json:"nodeIndex"`
}

// Configuration reuses the current consumer formats as a transitional payload.
// The protocol's required fields are validated separately from file-mode defaults.
// Changes to these consumer types must be reviewed for wire compatibility.
type Configuration struct {
	NVML       *nvmlconfig.YAMLConfig `json:"nvml"`
	Infiniband *ibconfig.Infiniband   `json:"infiniband"`
}

// Response is a full snapshot; receiving it does not acknowledge application.
type Response struct {
	APIVersion string       `json:"apiVersion"`
	Node       NodeIdentity `json:"node"`
	Cursor
	State         State          `json:"state"`
	Rack          *RackReference `json:"rack,omitempty"`
	Configuration *Configuration `json:"configuration,omitempty"`
}

// ErrorCode identifies a failure without relying on diagnostic message text.
type ErrorCode string

// Error codes correspond to HTTP 400, 409 and 503 respectively.
const (
	InvalidRequest        ErrorCode = "InvalidRequest"
	NodeUIDMismatch       ErrorCode = "NodeUIDMismatch"
	ConfigurationNotReady ErrorCode = "ConfigurationNotReady"
)

// ErrorResponse never carries replacement configuration. HTTP status and Code,
// rather than Message, determine how callers handle the failure.
type ErrorResponse struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}
