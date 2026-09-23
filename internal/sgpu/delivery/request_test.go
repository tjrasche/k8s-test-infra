// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

package delivery_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/k8s-test-infra/internal/sgpu/delivery"
)

func TestParseRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		query string
		want  *delivery.Cursor
	}{
		{name: "bootstrap", query: "nodeUID=node-instance-a"},
		{name: "assigned", query: "nodeUID=node-instance-a&lastAppliedAssignmentRevision=7&lastAppliedRackDeliveryGeneration=12", want: &delivery.Cursor{AssignmentRevision: 7, RackDeliveryGeneration: 12}},
		{name: "unassigned", query: "nodeUID=node-instance-a&lastAppliedAssignmentRevision=8&lastAppliedRackDeliveryGeneration=0", want: &delivery.Cursor{AssignmentRevision: 8}},
		{name: "maximum revision", query: "nodeUID=node-instance-a&lastAppliedAssignmentRevision=9223372036854775807&lastAppliedRackDeliveryGeneration=1", want: &delivery.Cursor{AssignmentRevision: math.MaxInt64, RackDeliveryGeneration: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request, err := delivery.ParseRequest("worker-01", tc.query)
			require.NoError(t, err)
			require.Equal(t, delivery.NodeIdentity{NodeName: "worker-01", NodeUID: "node-instance-a"}, request.Node)
			require.Equal(t, tc.want, request.LastApplied)
		})
	}
}

func TestParseRequestRejectsAmbiguousIdentityAndCursor(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		nodeName string
		query    string
	}{
		{name: "missing name", query: "nodeUID=a"},
		{name: "path in name", nodeName: "../worker", query: "nodeUID=a"},
		{name: "missing nodeUID", nodeName: "worker-01"},
		{name: "wrong UID name", nodeName: "worker-01", query: "uuid=a"},
		{name: "empty nodeUID", nodeName: "worker-01", query: "nodeUID="},
		{name: "whitespace nodeUID", nodeName: "worker-01", query: "nodeUID=+a"},
		{name: "duplicate nodeUID", nodeName: "worker-01", query: "nodeUID=a&nodeUID=b"},
		{name: "bad escaping", nodeName: "worker-01", query: "nodeUID=%ZZ"},
		{name: "semicolon", nodeName: "worker-01", query: "nodeUID=a;uuid=b"},
		{name: "assignment only", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=7"},
		{name: "generation only", nodeName: "worker-01", query: "nodeUID=a&lastAppliedRackDeliveryGeneration=12"},
		{name: "duplicate revision", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=7&lastAppliedAssignmentRevision=8&lastAppliedRackDeliveryGeneration=12"},
		{name: "duplicate generation", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=7&lastAppliedRackDeliveryGeneration=12&lastAppliedRackDeliveryGeneration=13"},
		{name: "empty cursor", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=&lastAppliedRackDeliveryGeneration="},
		{name: "zero assignment", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=0&lastAppliedRackDeliveryGeneration=0"},
		{name: "negative generation", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=7&lastAppliedRackDeliveryGeneration=-1"},
		{name: "floating point", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=7.0&lastAppliedRackDeliveryGeneration=12"},
		{name: "overflow", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=9223372036854775808&lastAppliedRackDeliveryGeneration=12"},
		{name: "signed decimal", nodeName: "worker-01", query: "nodeUID=a&lastAppliedAssignmentRevision=%2B7&lastAppliedRackDeliveryGeneration=12"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := delivery.ParseRequest(tc.nodeName, tc.query)
			require.Error(t, err)
		})
	}
}
