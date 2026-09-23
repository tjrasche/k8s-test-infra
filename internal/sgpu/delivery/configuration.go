// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright 2026 NVIDIA CORPORATION

package delivery

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/NVIDIA/k8s-test-infra/internal/gpuarch"
	nvmlconfig "github.com/NVIDIA/k8s-test-infra/pkg/gpu/mocknvml/config"
)

var canonicalPCIAddress = regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]$`)

func (c Configuration) validate() error {
	if c.NVML == nil || c.Infiniband == nil {
		return errors.New("configuration requires nvml and infiniband objects")
	}
	if err := validateNVML(c.NVML); err != nil {
		return err
	}
	ib := *c.Infiniband
	if ib.Enabled && (ib.HCACountOverride <= 0 || ib.RateGbps <= 0 || ib != ib.Defaults()) {
		return errors.New("enabled infiniband requires an explicit positive hca_count, link speed and resolved defaults")
	}
	return nil
}

func validateNVML(config *nvmlconfig.YAMLConfig) error {
	if config.Version != "1.0" {
		return fmt.Errorf("unsupported nvml version %q", config.Version)
	}
	if err := validateSystem(config); err != nil {
		return err
	}
	if err := validateDevices(config); err != nil {
		return err
	}
	if link := config.NVLink; link != nil && (link.Version <= 0 || link.LinksPerGPU <= 0 || link.BandwidthPerLinkMbps <= 0) {
		return errors.New("nvml.nvlink requires resolved version, link count and bandwidth")
	}
	return nil
}

func validateSystem(config *nvmlconfig.YAMLConfig) error {
	system := config.System
	if system.DriverVersion == "" || system.NVMLVersion == "" || system.CUDAVersion == "" || system.CUDAVersionMajor <= 0 || system.CUDAVersionMinor < 0 {
		return errors.New("nvml.system requires resolved driver, NVML and CUDA versions")
	}
	if system.NumDevices <= 0 || system.NumDevices > nvmlconfig.MaxDevices || system.NumDevices != len(config.Devices) {
		return fmt.Errorf("nvml.system.num_devices must be between 1 and %d and match the explicit device list", nvmlconfig.MaxDevices)
	}
	return nil
}

func validateDevices(config *nvmlconfig.YAMLConfig) error {
	indices := make(map[int]bool, len(config.Devices))
	uuids := make(map[string]bool, len(config.Devices))
	addresses := make(map[string]bool, len(config.Devices))
	for _, device := range config.Devices {
		if device.Index < 0 || device.Index >= len(config.Devices) || indices[device.Index] {
			return fmt.Errorf("invalid or duplicate GPU index %d", device.Index)
		}
		indices[device.Index] = true
		if err := validateDeviceIdentity(device); err != nil {
			return fmt.Errorf("GPU %d: %w", device.Index, err)
		}
		if uuids[device.UUID] {
			return fmt.Errorf("duplicate GPU UUID %q", device.UUID)
		}
		uuids[device.UUID] = true
		resolved := config.GetDeviceConfig(device.Index)
		if err := validateDeviceHardware(resolved); err != nil {
			return fmt.Errorf("GPU %d: %w", device.Index, err)
		}
		if addresses[resolved.PCI.BusID] {
			return fmt.Errorf("duplicate GPU PCI address %q", resolved.PCI.BusID)
		}
		addresses[resolved.PCI.BusID] = true
	}
	return nvmlconfig.ValidateMinorNumbers(config)
}

func validateDeviceIdentity(device nvmlconfig.DeviceOverride) error {
	id, prefixed := strings.CutPrefix(device.UUID, "GPU-")
	if !prefixed || !isCanonicalUUID(id) {
		return errors.New("explicit canonical GPU UUID is required")
	}
	if device.Serial == "" || device.MinorNumber == nil {
		return errors.New("explicit serial and minor_number are required")
	}
	// The engine's actual PCI address getter does not read device_defaults.
	if device.PCI == nil || !canonicalPCIAddress.MatchString(device.PCI.BusID) {
		return errors.New("explicit canonical per-device pci.bus_id is required")
	}
	return nil
}

func validateDeviceHardware(device *nvmlconfig.DeviceConfig) error {
	if _, known := gpuarch.Parse(device.Architecture); device.Name == "" || !known {
		return errors.New("resolved GPU name and supported architecture are required")
	}
	if device.Memory == nil || device.Memory.TotalBytes == 0 || device.Memory.ReservedBytes > device.Memory.TotalBytes {
		return errors.New("resolved GPU memory capacity is required")
	}
	if err := validatePCI(device.PCI); err != nil {
		return err
	}
	if device.Fabric != nil && !isCanonicalUUID(device.Fabric.ClusterUUID) {
		return errors.New("fabric cluster_uuid must be a canonical dashed UUID")
	}
	return nil
}

func validatePCI(pci *nvmlconfig.PCIConfig) error {
	if pci == nil || pci.DeviceID == 0 || pci.SubsystemID == 0 {
		return errors.New("resolved PCI identity words are required")
	}
	return nil
}

func isCanonicalUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id.String() == value
}
