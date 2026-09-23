// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

package delivery

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"k8s.io/apimachinery/pkg/util/validation"
)

// ParseRequest accepts the router's decoded nodeName and URL.RawQuery. Parsing
// RawQuery avoids URL.Query silently dropping malformed query parameters.
func ParseRequest(nodeName, rawQuery string) (Request, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Request{}, fmt.Errorf("parse configuration query: %w", err)
	}
	if len(query["nodeUID"]) != 1 {
		return Request{}, errors.New("exactly one nodeUID is required")
	}
	request := Request{Node: NodeIdentity{NodeName: nodeName, NodeUID: query.Get("nodeUID")}}
	revisions, hasRevision := query["lastAppliedAssignmentRevision"]
	generations, hasGeneration := query["lastAppliedRackDeliveryGeneration"]
	if hasRevision || hasGeneration {
		if len(revisions) != 1 || len(generations) != 1 {
			return Request{}, errors.New("lastAppliedAssignmentRevision and lastAppliedRackDeliveryGeneration must appear exactly once together")
		}
		revision, err := parseCounter("lastAppliedAssignmentRevision", revisions[0])
		if err != nil {
			return Request{}, err
		}
		generation, err := parseCounter("lastAppliedRackDeliveryGeneration", generations[0])
		if err != nil {
			return Request{}, err
		}
		request.LastApplied = &Cursor{AssignmentRevision: revision, RackDeliveryGeneration: generation}
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func parseCounter(name, value string) (int64, error) {
	if value == "" || strings.ContainsFunc(value, func(r rune) bool { return r < '0' || r > '9' }) {
		return 0, fmt.Errorf("%s must be a nonnegative decimal integer", name)
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return n, nil
}

// Validate also supports requests constructed locally rather than parsed from HTTP.
func (r Request) Validate() error {
	if reasons := validation.IsDNS1123Subdomain(r.Node.NodeName); len(reasons) != 0 {
		return fmt.Errorf("invalid nodeName: %s", strings.Join(reasons, "; "))
	}
	if r.Node.NodeUID == "" || strings.ContainsFunc(r.Node.NodeUID, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return errors.New("nodeUID must be a nonempty opaque identifier without whitespace or control characters")
	}
	if r.LastApplied != nil {
		return r.LastApplied.validate()
	}
	return nil
}
