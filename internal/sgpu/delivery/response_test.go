// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

package delivery_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/k8s-test-infra/internal/sgpu/delivery"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	require.NoError(t, err)
	return data
}

func request() delivery.Request {
	return delivery.Request{Node: delivery.NodeIdentity{NodeName: "worker-01", NodeUID: "node-instance-a"}}
}

func response(t *testing.T, name string) delivery.Response {
	t.Helper()
	result, err := delivery.DecodeResponse(fixture(t, name), request())
	require.NoError(t, err)
	return result
}

func TestResponseFixtures(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"assigned-a100", "assigned-gb200", "unassigned"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := response(t, name)
			encoded, err := json.Marshal(result)
			require.NoError(t, err)
			roundTrip, err := delivery.DecodeResponse(encoded, request())
			require.NoError(t, err)
			require.Equal(t, result, roundTrip)
			require.Contains(t, string(encoded), `"nodeUID":"node-instance-a"`)
			if result.State == delivery.Unassigned {
				require.Contains(t, string(encoded), `"rackDeliveryGeneration":0`)
				require.NotContains(t, string(encoded), `"rack":`)
				require.NotContains(t, string(encoded), `"configuration":`)
			}
		})
	}
}

func TestDecodeResponseValidatesWireFields(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, old, replacement string }{
		{"nodeUID cannot be named uid", `"nodeUID":`, `"uid":`},
		{"nodeUID cannot be named UUID", `"nodeUID":`, `"UUID":`},
		{"wrong nodeUID", `"node-instance-a"`, `"node-instance-b"`},
		{"wrong nodeName", `"worker-01"`, `"worker-02"`},
		{"unknown API version", `"v1alpha1"`, `"v2"`},
		{"unknown state", `"unassigned"`, `"missing"`},
		{"missing generation", `"rackDeliveryGeneration": 0,`, ``},
		{"null generation", `"rackDeliveryGeneration": 0`, `"rackDeliveryGeneration": null`},
		{"fractional revision", `"assignmentRevision": 9`, `"assignmentRevision": 9.5`},
		{"overflow revision", `"assignmentRevision": 9`, `"assignmentRevision": 9223372036854775808`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := strings.Replace(string(fixture(t, "unassigned")), tc.old, tc.replacement, 1)
			_, err := delivery.DecodeResponse([]byte(data), request())
			require.Error(t, err)
		})
	}
	for _, data := range []string{"null", "{}", "[]", "{", string(fixture(t, "unassigned")) + "{}"} {
		_, err := delivery.DecodeResponse([]byte(data), request())
		require.Error(t, err)
	}
}

func TestResponseAllowsInformationalFields(t *testing.T) {
	t.Parallel()
	data := strings.Replace(string(fixture(t, "assigned-a100")), `"apiVersion":`, `"diagnostic": "extra", "apiVersion":`, 1)
	decoded, err := delivery.DecodeResponse([]byte(data), request())
	require.NoError(t, err)
	require.Equal(t, response(t, "assigned-a100"), decoded)
}

func TestResponseInvariants(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*delivery.Response)
	}{
		{"zero assignment revision", func(r *delivery.Response) { r.AssignmentRevision = 0 }},
		{"negative generation", func(r *delivery.Response) { r.RackDeliveryGeneration = -1 }},
		{"zero assigned generation", func(r *delivery.Response) { r.RackDeliveryGeneration = 0 }},
		{"missing rack", func(r *delivery.Response) { r.Rack = nil }},
		{"missing rack UID", func(r *delivery.Response) { r.Rack.UID = "" }},
		{"missing node index", func(r *delivery.Response) { r.Rack.NodeIndex = nil }},
		{"negative node index", func(r *delivery.Response) { r.Rack.NodeIndex = new(int32(-1)) }},
		{"missing configuration", func(r *delivery.Response) { r.Configuration = nil }},
		{"empty configuration", func(r *delivery.Response) { r.Configuration = &delivery.Configuration{} }},
		{"unassigned carrying payload", func(r *delivery.Response) { r.State = delivery.Unassigned; r.RackDeliveryGeneration = 0 }},
		{"unassigned carrying rack", func(r *delivery.Response) {
			r.State = delivery.Unassigned
			r.RackDeliveryGeneration = 0
			r.Configuration = nil
		}},
		{"unassigned positive generation", func(r *delivery.Response) { r.State = delivery.Unassigned; r.Rack = nil; r.Configuration = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := response(t, "assigned-a100")
			tc.mutate(&candidate)
			require.Error(t, candidate.Validate(request()))
		})
	}
}

func TestResponseOrdering(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		applied delivery.Cursor
		offered delivery.Cursor
		wantErr bool
	}{
		{"new generation", delivery.Cursor{7, 12}, delivery.Cursor{7, 13}, false},
		{"equal pair", delivery.Cursor{7, 12}, delivery.Cursor{7, 12}, false},
		{"older generation", delivery.Cursor{7, 12}, delivery.Cursor{7, 11}, true},
		{"older assignment higher generation", delivery.Cursor{7, 12}, delivery.Cursor{6, 100}, true},
		{"new assignment lower generation", delivery.Cursor{7, 12}, delivery.Cursor{8, 2}, false},
		{"release", delivery.Cursor{7, 12}, delivery.Cursor{8, 0}, false},
		{"stale response after release", delivery.Cursor{8, 0}, delivery.Cursor{7, 99}, true},
		{"rebind", delivery.Cursor{8, 0}, delivery.Cursor{9, 1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := request()
			req.LastApplied = &tc.applied
			candidate := response(t, "assigned-a100")
			candidate.Cursor = tc.offered
			if tc.offered.RackDeliveryGeneration == 0 {
				candidate.State, candidate.Rack, candidate.Configuration = delivery.Unassigned, nil, nil
			}
			err := candidate.Validate(req)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Validation never acknowledges reception, so an unchanged failed
				// candidate remains valid against the same last-applied cursor.
				require.NoError(t, candidate.Validate(req))
			}
			require.Equal(t, tc.applied, *req.LastApplied)
		})
	}
}

func TestResponseRejectsConflictingPublication(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*delivery.Response)
	}{
		{"changed rack UID", func(r *delivery.Response) { r.Rack.UID = "rack-instance-b"; r.RackDeliveryGeneration++ }},
		{"changed rack name", func(r *delivery.Response) { r.Rack.Name = "training-1"; r.RackDeliveryGeneration++ }},
		{"changed node slot", func(r *delivery.Response) { r.Rack.NodeIndex = new(int32(1)); r.RackDeliveryGeneration++ }},
		{"changed payload at equal pair", func(r *delivery.Response) { r.Configuration.NVML.System.DriverVersion = "999" }},
		{"replacement Node", func(r *delivery.Response) { r.Node.NodeUID = "node-instance-b"; r.AssignmentRevision++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			applied := response(t, "assigned-a100")
			candidate := response(t, "assigned-a100")
			tc.mutate(&candidate)
			require.Error(t, candidate.ValidateSuccessor(applied))
		})
	}
	applied := response(t, "assigned-a100")
	require.NoError(t, applied.ValidateSuccessor(applied))
	require.NoError(t, response(t, "assigned-gb200").ValidateSuccessor(applied))
	require.NoError(t, response(t, "unassigned").ValidateSuccessor(applied))
}

func TestReplacementNodeStartsFreshOrderingDomain(t *testing.T) {
	t.Parallel()
	candidate := response(t, "assigned-a100")
	candidate.Node.NodeUID = "node-instance-b"
	candidate.Cursor = delivery.Cursor{AssignmentRevision: 1, RackDeliveryGeneration: 1}
	req := delivery.Request{Node: candidate.Node}
	require.NoError(t, candidate.Validate(req))
	require.Error(t, candidate.Validate(request()))
	req.LastApplied = &delivery.Cursor{}
	require.Error(t, candidate.Validate(req), "absence of an applied cursor is not revision zero")
}

func TestUnassignedCannotRebindAtTheSameAssignmentRevision(t *testing.T) {
	t.Parallel()
	applied := response(t, "unassigned")
	candidate := response(t, "assigned-a100")
	candidate.AssignmentRevision = applied.AssignmentRevision
	require.Error(t, candidate.ValidateSuccessor(applied))
}

func TestErrorResponseCannotBecomeReplacementState(t *testing.T) {
	t.Parallel()
	data := fixture(t, "not-ready")
	var failure delivery.ErrorResponse
	require.NoError(t, json.Unmarshal(data, &failure))
	require.Equal(t, delivery.ConfigurationNotReady, failure.Code)
	encoded, err := json.Marshal(failure)
	require.NoError(t, err)
	require.JSONEq(t, string(data), string(encoded))
	_, err = delivery.DecodeResponse(data, request())
	require.Error(t, err)
}
