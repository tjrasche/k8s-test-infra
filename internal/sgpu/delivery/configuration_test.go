// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

package delivery_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/NVIDIA/go-nvml/pkg/nvml/mock/dgxa100"
	mockserver "github.com/NVIDIA/go-nvml/pkg/nvml/mock/server"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	mokkav1alpha1 "github.com/NVIDIA/k8s-test-infra/api/v1alpha1"
	"github.com/NVIDIA/k8s-test-infra/internal/sgpu/delivery"
	"github.com/NVIDIA/k8s-test-infra/pkg/gpu/mocknvml/engine"
)

func TestConfigurationRequiresResolvedIdentity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*delivery.Configuration)
	}{
		{"missing NVML", func(c *delivery.Configuration) { c.NVML = nil }},
		{"missing network", func(c *delivery.Configuration) { c.Infiniband = nil }},
		{"unknown engine schema", func(c *delivery.Configuration) { c.NVML.Version = "future" }},
		{"missing driver", func(c *delivery.Configuration) { c.NVML.System.DriverVersion = "" }},
		{"missing NVML version", func(c *delivery.Configuration) { c.NVML.System.NVMLVersion = "" }},
		{"missing CUDA version", func(c *delivery.Configuration) { c.NVML.System.CUDAVersion = "" }},
		{"implicit count", func(c *delivery.Configuration) { c.NVML.System.NumDevices = 0 }},
		{"count mismatch", func(c *delivery.Configuration) { c.NVML.System.NumDevices++ }},
		{"implicit UUID", func(c *delivery.Configuration) { c.NVML.Devices[0].UUID = "" }},
		{"malformed GPU UUID", func(c *delivery.Configuration) { c.NVML.Devices[0].UUID = "not-a-GPU-UUID" }},
		{"duplicate UUID", func(c *delivery.Configuration) { c.NVML.Devices[1].UUID = c.NVML.Devices[0].UUID }},
		{"implicit minor", func(c *delivery.Configuration) { c.NVML.Devices[0].MinorNumber = nil }},
		{"reserved minor", func(c *delivery.Configuration) { c.NVML.Devices[0].MinorNumber = new(255) }},
		{"duplicate minor", func(c *delivery.Configuration) { c.NVML.Devices[1].MinorNumber = c.NVML.Devices[0].MinorNumber }},
		{"duplicate index", func(c *delivery.Configuration) { c.NVML.Devices[1].Index = 0 }},
		{"index outside count", func(c *delivery.Configuration) { c.NVML.Devices[0].Index = 999999999 }},
		{"implicit serial", func(c *delivery.Configuration) { c.NVML.Devices[0].Serial = "" }},
		{"missing hardware", func(c *delivery.Configuration) { c.NVML.DeviceDefaults.Name = "" }},
		{"unknown architecture", func(c *delivery.Configuration) { c.NVML.DeviceDefaults.Architecture = "unknown" }},
		{"missing memory", func(c *delivery.Configuration) { c.NVML.DeviceDefaults.Memory = nil }},
		{"implicit PCI address", func(c *delivery.Configuration) { c.NVML.Devices[0].PCI.BusID = "" }},
		{"defaults-only PCI address", func(c *delivery.Configuration) {
			c.NVML.DeviceDefaults.PCI.BusID = c.NVML.Devices[0].PCI.BusID
			c.NVML.Devices[0].PCI = nil
		}},
		{"invalid PCI address", func(c *delivery.Configuration) { c.NVML.Devices[0].PCI.BusID = "../../host" }},
		{"duplicate PCI address", func(c *delivery.Configuration) { c.NVML.Devices[1].PCI.BusID = c.NVML.Devices[0].PCI.BusID }},
		{"missing PCI words", func(c *delivery.Configuration) { c.NVML.DeviceDefaults.PCI = nil }},
		{"implicit fabric identity", func(c *delivery.Configuration) { c.NVML.DeviceDefaults.Fabric.ClusterUUID = "" }},
		{"URN fabric identity", func(c *delivery.Configuration) {
			c.NVML.DeviceDefaults.Fabric.ClusterUUID = "urn:uuid:" + c.NVML.DeviceDefaults.Fabric.ClusterUUID
		}},
		{"noncanonical GPU UUID", func(c *delivery.Configuration) {
			c.NVML.Devices[0].UUID = "GPU-urn:uuid:00000000-0000-0000-0000-000000000001"
		}},
		{"implicit link speed", func(c *delivery.Configuration) { c.NVML.NVLink.BandwidthPerLinkMbps = 0 }},
		{"implicit HCA count", func(c *delivery.Configuration) { c.Infiniband.HCACountOverride = 0 }},
		{"implicit network defaults", func(c *delivery.Configuration) { c.Infiniband.HCAType = "" }},
		{"negative link speed", func(c *delivery.Configuration) { c.Infiniband.RateGbps = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := response(t, "assigned-gb200")
			tc.mutate(candidate.Configuration)
			require.Error(t, candidate.Validate(request()))
		})
	}
}

func TestConfigurationRejectsCountsBeyondConsumerCapacity(t *testing.T) {
	t.Parallel()
	for _, count := range []int{engine.MaxDevices, engine.MaxDevices + 1} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			t.Parallel()
			candidate := response(t, "assigned-a100")
			config := candidate.Configuration.NVML
			config.System.NumDevices = count
			config.Devices = make([]engine.DeviceOverride, count)
			for i := range count {
				config.Devices[i] = engine.DeviceOverride{
					Index: i, UUID: fmt.Sprintf("GPU-00000000-0000-0000-0000-%012d", i), MinorNumber: new(i),
					DeviceConfig: engine.DeviceConfig{Serial: strconv.Itoa(i), PCI: &engine.PCIConfig{BusID: fmt.Sprintf("0000:%02x:00.0", i+1)}},
				}
			}
			err := candidate.Validate(request())
			if count > engine.MaxDevices {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigurationSupportsDisabledNetworkAndIndexedDevices(t *testing.T) {
	t.Parallel()
	candidate := response(t, "assigned-gb200")
	candidate.Configuration.Infiniband.Enabled = false
	devices := candidate.Configuration.NVML.Devices
	devices[0], devices[3] = devices[3], devices[0]
	require.NoError(t, candidate.Validate(request()))
	consumer := &engine.Config{YAMLConfig: candidate.Configuration.NVML}
	require.Equal(t, devices[0].UUID, consumer.GetDeviceUUID(3))
}

func TestPayloadLoadsThroughExistingNVMLConsumer(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"assigned-a100", "assigned-gb200"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			payload := response(t, name).Configuration.NVML
			data, err := yaml.Marshal(payload)
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, data, 0o600))
			loaded, err := engine.LoadYAMLConfig(path)
			require.NoError(t, err)
			require.Equal(t, payload, loaded)
			consumer := &engine.Config{YAMLConfig: loaded}
			for _, device := range payload.Devices {
				require.Equal(t, device.UUID, consumer.GetDeviceUUID(device.Index))
				require.Equal(t, *device.MinorNumber, consumer.GetDeviceMinorNumber(device.Index))
				require.Equal(t, device.PCI.BusID, consumer.GetDevicePCIBusID(device.Index))
			}
		})
	}
}

func TestA100FixtureMatchesRackExampleHardware(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "controlplane-crds", "sgpu-rack-profile.yaml"))
	require.NoError(t, err)
	var profile mokkav1alpha1.SGPURackProfile
	require.NoError(t, yaml.UnmarshalStrict(data, &profile))
	payload := response(t, "assigned-a100").Configuration
	gpu := (&engine.Config{YAMLConfig: payload.NVML}).GetDeviceConfig(0)
	declared := profile.Spec.Node.GPUs
	require.Equal(t, declared.Model.ProductName, gpu.Name)
	require.Equal(t, strings.ToLower(declared.Model.Architecture), gpu.Architecture)
	require.Equal(t, uint64(declared.Memory.Capacity.Value()), gpu.Memory.TotalBytes)
	require.Equal(t, uint64(declared.Memory.Reserved.Value()), gpu.Memory.ReservedBytes)
	require.Equal(t, declared.PCI.VendorID, fmt.Sprintf("%04x", gpu.PCI.DeviceID&0xffff))
	require.Equal(t, declared.PCI.DeviceID, fmt.Sprintf("%04x", gpu.PCI.DeviceID>>16))
	require.Equal(t, declared.PCI.SubsystemVendorID, fmt.Sprintf("%04x", gpu.PCI.SubsystemID&0xffff))
	require.Equal(t, declared.PCI.SubsystemDeviceID, fmt.Sprintf("%04x", gpu.PCI.SubsystemID>>16))
	require.Equal(t, profile.Spec.Node.Topology.GPUSlots[0].PCIAddress, gpu.PCI.BusID)
	require.Equal(t, profile.Spec.Software.DriverVersion, payload.NVML.System.DriverVersion)
	require.Equal(t, "MT4129", payload.Infiniband.HCAType)
	require.Zero(t, payload.NVML.PCIeTopology.CoresPerNUMA, "do not infer CPU affinity from host core count")
}

func TestGB200FixturePreservesConsumerFieldsMissingFromRackSchema(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "deployments", "nvml-mock", "helm", "nvml-mock", "profiles", "gb200.yaml"))
	require.NoError(t, err)
	var legacy engine.YAMLConfig
	require.NoError(t, yaml.Unmarshal(data, &legacy))
	payload := response(t, "assigned-gb200").Configuration.NVML
	consumer := &engine.Config{YAMLConfig: payload}
	original := &engine.Config{YAMLConfig: &legacy}
	for i := range payload.Devices {
		actual, expected := consumer.GetDeviceConfig(i), original.GetDeviceConfig(i)
		require.Equal(t, expected.Platform, actual.Platform)
		require.Equal(t, expected.Memory.TotalBytes, actual.Memory.TotalBytes)
		require.Equal(t, expected.PCI.DeviceID, actual.PCI.DeviceID)
	}
	require.Equal(t, legacy.NVLink.BandwidthPerLinkMbps, payload.NVLink.BandwidthPerLinkMbps)
	require.Equal(t, legacy.NVLink.C2CEnabled, payload.NVLink.C2CEnabled)
	require.Len(t, payload.NVLink.Switches, len(legacy.NVLink.Switches))
	require.NotEqual(t, legacy.DeviceDefaults.Fabric.ClusterUUID, payload.DeviceDefaults.Fabric.ClusterUUID, "delivered rack identity replaces the legacy placeholder")
	require.Equal(t, uint32(1), payload.DeviceDefaults.Fabric.CliqueID)

	base, ok := dgxa100.New().Devices[0].(*mockserver.Device)
	require.True(t, ok)
	device := engine.NewConfigurableDevice(0, base, consumer.GetDeviceConfig(0), consumer.GetDeviceUUID(0), consumer.GetDevicePCIBusID(0), consumer.GetDeviceMinorNumber(0), nil)
	fabric, result := device.GetMockFabricInfo()
	require.Equal(t, nvml.SUCCESS, result)
	require.Equal(t, payload.DeviceDefaults.Fabric.ClusterUUID, uuid.UUID(fabric.ClusterUUID).String())
	require.Equal(t, payload.DeviceDefaults.Fabric.CliqueID, fabric.CliqueID)
}
