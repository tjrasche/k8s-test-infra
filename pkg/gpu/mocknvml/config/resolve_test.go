// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	nvmlconfig "github.com/NVIDIA/k8s-test-infra/pkg/gpu/mocknvml/config"
)

func TestDeviceResolution(t *testing.T) {
	t.Parallel()

	config := &nvmlconfig.YAMLConfig{
		System: nvmlconfig.SystemConfig{NumDevices: 2},
		DeviceDefaults: nvmlconfig.DeviceConfig{
			Name:      "default GPU",
			PCI:       &nvmlconfig.PCIConfig{DeviceID: 1, SubsystemID: 2},
			Platform:  &nvmlconfig.PlatformConfig{SlotNumber: 3, ModuleID: 1},
			Processes: []nvmlconfig.ProcessConfig{{PID: 100}},
		},
		Devices: []nvmlconfig.DeviceOverride{{
			Index: 1, UUID: "GPU-1", MinorNumber: new(2),
			DeviceConfig: nvmlconfig.DeviceConfig{
				PCI:       &nvmlconfig.PCIConfig{BusID: "0000:02:00.0"},
				Platform:  &nvmlconfig.PlatformConfig{ModuleID: 2},
				Processes: []nvmlconfig.ProcessConfig{},
			},
		}},
	}

	resolved := config.GetDeviceConfig(1)
	require.Equal(t, "default GPU", resolved.Name)
	require.Equal(t, &nvmlconfig.PCIConfig{DeviceID: 1, SubsystemID: 2, BusID: "0000:02:00.0"}, resolved.PCI)
	require.Equal(t, &nvmlconfig.PlatformConfig{SlotNumber: 3, ModuleID: 2}, resolved.Platform)
	require.Empty(t, resolved.Processes)
	require.Equal(t, "GPU-1", config.GetDeviceUUID(1))
	require.Equal(t, "0000:02:00.0", config.GetDevicePCIBusID(1))
	require.Equal(t, 2, config.GetDeviceMinorNumber(1))
	require.NoError(t, nvmlconfig.ValidateMinorNumbers(config))

	inherited := config.GetDeviceConfig(0)
	require.Empty(t, inherited.PCI.BusID, "resolution must not change shared PCI defaults")
	require.Equal(t, uint8(1), inherited.Platform.ModuleID, "resolution must not change shared platform defaults")
	require.Equal(t, []nvmlconfig.ProcessConfig{{PID: 100}}, inherited.Processes)
	require.Equal(t, 0, config.GetDeviceMinorNumber(0))
}

func TestDeviceResolutionWithoutYAML(t *testing.T) {
	t.Parallel()

	var config *nvmlconfig.YAMLConfig
	require.Nil(t, config.GetDeviceConfig(1))
	require.Empty(t, config.GetDeviceUUID(1))
	require.Empty(t, config.GetDevicePCIBusID(1))
	require.Equal(t, 1, config.GetDeviceMinorNumber(1))
	require.NoError(t, nvmlconfig.ValidateMinorNumbers(config))
}

func TestMinorValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		minor   int
		wantErr string
	}{
		{"explicit zero", 0, ""},
		{"maximum GPU minor", 254, ""},
		{"negative", -1, "device minor number out of range"},
		{"reserved for nvidiactl", 255, "device minor number out of range"},
		{"collides with implicit device", 1, "duplicate device minor number: 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			config := &nvmlconfig.YAMLConfig{
				System:  nvmlconfig.SystemConfig{NumDevices: 2},
				Devices: []nvmlconfig.DeviceOverride{{Index: 0, MinorNumber: new(tc.minor)}},
			}
			err := nvmlconfig.ValidateMinorNumbers(config)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
