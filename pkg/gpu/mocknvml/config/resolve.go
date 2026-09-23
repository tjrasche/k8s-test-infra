// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"sort"
)

// GetDeviceConfig returns the device configuration for a specific index,
// merging defaults with per-device overrides
func (c *YAMLConfig) GetDeviceConfig(index int) *DeviceConfig {
	if c == nil {
		return nil
	}

	// Start with a copy of defaults. The copy is shallow, so every pointer
	// field still aliases the shared defaults. mergeDeviceOverride writes
	// THROUGH the PCI pointer instead of replacing it, so clone PCIConfig here
	// or a per-device PCI override lands on the defaults and leaks into every
	// device merged afterwards (issue #589). mergePlatformOverride clones
	// Platform itself; the remaining branches replace the pointer wholesale.
	merged := c.DeviceDefaults
	if merged.PCI != nil {
		pci := *merged.PCI
		merged.PCI = &pci
	}

	// Find and apply per-device overrides
	for _, override := range c.Devices {
		if override.Index == index {
			mergeDeviceOverride(&merged, &override)
			break
		}
	}

	return &merged
}

// GetDeviceUUID returns the UUID for a specific device index
func (c *YAMLConfig) GetDeviceUUID(index int) string {
	if c == nil {
		return ""
	}

	for _, dev := range c.Devices {
		if dev.Index == index {
			return dev.UUID
		}
	}
	return ""
}

// GetDeviceMinorNumber returns the minor number for a specific device index.
func (c *YAMLConfig) GetDeviceMinorNumber(index int) int {
	return DeviceMinorNumber(c, index)
}

// maxDeviceMinor is the highest minor major 195 leaves for a GPU: the driver
// keeps 255 for nvidiactl.
const maxDeviceMinor = 254

// DeviceMinorNumber returns the /dev/nvidia<N> a device is staged under,
// defaulting to the index for devices that do not declare one — the numbering
// a driver produces when it probes in PCI enumeration order.
//
// The agent and the engine both resolve minors through here so the nodes that
// get staged and the nodes the visibility filter looks for cannot drift apart.
func DeviceMinorNumber(config *YAMLConfig, index int) int {
	if config == nil {
		return index
	}

	for _, dev := range config.Devices {
		if dev.Index == index && dev.MinorNumber != nil {
			return *dev.MinorNumber
		}
	}
	return index
}

// ValidateMinorNumbers rejects minors that no device node can carry and minors
// two devices would end up sharing. Defaulted devices take part: one that never
// declares a minor still occupies its index, so an explicit value elsewhere can
// collide with it.
func ValidateMinorNumbers(config *YAMLConfig) error {
	if config == nil {
		return nil
	}

	for _, dev := range config.Devices {
		if dev.MinorNumber == nil {
			continue
		}
		if *dev.MinorNumber < 0 || *dev.MinorNumber > maxDeviceMinor {
			return fmt.Errorf("device %d: device minor number out of range (0-%d): %d",
				dev.Index, maxDeviceMinor, *dev.MinorNumber)
		}
	}

	seen := make(map[int]int, deviceSpace(config))
	for _, index := range deviceIndices(config) {
		minor := DeviceMinorNumber(config, index)
		if other, dup := seen[minor]; dup {
			return fmt.Errorf("duplicate device minor number: %d (devices %d and %d)", minor, other, index)
		}
		seen[minor] = index
	}

	return nil
}

// deviceIndices lists every device the config brings into being: those the
// count covers, plus any the overrides declare beyond it.
func deviceIndices(config *YAMLConfig) []int {
	n := deviceSpace(config)
	indices := make([]int, 0, n)
	seen := make(map[int]bool, n)
	for i := 0; i < n; i++ {
		indices = append(indices, i)
		seen[i] = true
	}
	for _, dev := range config.Devices {
		if !seen[dev.Index] {
			indices = append(indices, dev.Index)
			seen[dev.Index] = true
		}
	}
	sort.Ints(indices)
	return indices
}

// deviceSpace is how many devices the config describes before any runtime cap.
func deviceSpace(config *YAMLConfig) int {
	n := len(config.Devices)
	if config.System.NumDevices > n {
		n = config.System.NumDevices
	}
	return n
}

// GetDevicePCIBusID returns the PCI bus ID for a specific device index
func (c *YAMLConfig) GetDevicePCIBusID(index int) string {
	if c == nil {
		return ""
	}

	for _, dev := range c.Devices {
		if dev.Index == index && dev.PCI != nil {
			return dev.PCI.BusID
		}
	}
	return ""
}

// mergeDeviceOverride merges non-zero override values into the base config
//
//nolint:cyclop // existing complexity; refactor deferred
func mergeDeviceOverride(base *DeviceConfig, override *DeviceOverride) {
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.Serial != "" {
		base.Serial = override.Serial
	}
	if override.Brand != "" {
		base.Brand = override.Brand
	}
	if override.BoardPartNumber != "" {
		base.BoardPartNumber = override.BoardPartNumber
	}
	if override.VBIOSVersion != "" {
		base.VBIOSVersion = override.VBIOSVersion
	}
	if override.Architecture != "" {
		base.Architecture = override.Architecture
	}
	if override.PCI != nil {
		if base.PCI == nil {
			base.PCI = &PCIConfig{}
		}
		if override.PCI.BusID != "" {
			base.PCI.BusID = override.PCI.BusID
		}
		if override.PCI.DeviceID != 0 {
			base.PCI.DeviceID = override.PCI.DeviceID
		}
		if override.PCI.SubsystemID != 0 {
			base.PCI.SubsystemID = override.PCI.SubsystemID
		}
	}
	if override.Memory != nil {
		base.Memory = override.Memory
	}
	if override.BAR1Memory != nil {
		base.BAR1Memory = override.BAR1Memory
	}
	if override.Power != nil {
		base.Power = override.Power
	}
	if override.Thermal != nil {
		base.Thermal = override.Thermal
	}
	if override.Clocks != nil {
		base.Clocks = override.Clocks
	}
	if override.ClocksThrottleReasons != nil {
		base.ClocksThrottleReasons = override.ClocksThrottleReasons
	}
	if override.Utilization != nil {
		base.Utilization = override.Utilization
	}
	if override.ECC != nil {
		base.ECC = override.ECC
	}
	if override.DynamicMetrics != nil {
		base.DynamicMetrics = override.DynamicMetrics
	}
	if override.Failure != nil {
		base.Failure = override.Failure
	}
	if override.Fabric != nil {
		base.Fabric = override.Fabric
	}
	if override.NVLinkError != nil {
		base.NVLinkError = override.NVLinkError
	}
	if override.Processes != nil {
		base.Processes = override.Processes // nil = not overridden; [] = explicit clear
	}
	if override.MIG != nil {
		mergeMIGOverride(base, override.MIG)
	}
	if override.Platform != nil {
		mergePlatformOverride(base, override.Platform)
	}
	// Add more fields as needed
}

// mergeMIGOverride merges per-field, for the same reason as
// mergePlatformOverride: what varies between the GPUs of a node is the mode and
// the layout, while the partition table and the instance ceiling describe the
// board they all are.
//
// Replacing the block would make a devices[] entry restate the table to keep
// it, and since the table is a document of its own it cannot — so that GPU
// would answer ERROR_NOT_SUPPORTED while its siblings partitioned normally. A
// device that does declare a table still wins, because a node is free to mix
// boards and the table is then the only thing saying which one this GPU is.
//
// The block is copied before it is written to, as the platform merge is: every
// device's merge starts out pointing at the same MIGConfig.
func mergeMIGOverride(base *DeviceConfig, override *MIGConfig) {
	if base.MIG == nil {
		base.MIG = &MIGConfig{}
	} else {
		clone := *base.MIG
		base.MIG = &clone
	}
	if override.ModeCurrent != "" {
		base.MIG.ModeCurrent = override.ModeCurrent
	}
	if override.ModePending != "" {
		base.MIG.ModePending = override.ModePending
	}
	if override.MaxGPUInstances != 0 {
		base.MIG.MaxGPUInstances = override.MaxGPUInstances
	}
	if len(override.SupportedProfiles) > 0 {
		base.MIG.SupportedProfiles = override.SupportedProfiles
	}
	if len(override.GPUInstances) > 0 {
		base.MIG.GPUInstances = override.GPUInstances
	}
	// Absent and present-but-empty differ here: an empty list is a partitioned
	// board with every instance deleted, so the pointer is carried as given.
	if override.Instances != nil {
		base.MIG.Instances = override.Instances
	}
}

// mergePlatformOverride merges per-field rather than replacing the block, so a
// device can set its own module_id — the one field that varies between the GPUs
// of a node — without restating the chassis, slot, tray, and host id it shares
// with them.
//
// The block is copied before it is written to: GetDeviceConfig copies
// DeviceDefaults shallowly, so every device's merge starts out pointing at the
// same PlatformConfig. Editing that in place would give all of them whichever
// module id was merged last.
func mergePlatformOverride(base *DeviceConfig, override *PlatformConfig) {
	if base.Platform == nil {
		base.Platform = &PlatformConfig{}
	} else {
		clone := *base.Platform
		base.Platform = &clone
	}
	if override.ChassisSerialNumber != "" {
		base.Platform.ChassisSerialNumber = override.ChassisSerialNumber
	}
	if override.SlotNumber != 0 {
		base.Platform.SlotNumber = override.SlotNumber
	}
	if override.TrayIndex != 0 {
		base.Platform.TrayIndex = override.TrayIndex
	}
	if override.HostID != 0 {
		base.Platform.HostID = override.HostID
	}
	if override.PeerType != "" {
		base.Platform.PeerType = override.PeerType
	}
	if override.ModuleID != 0 {
		base.Platform.ModuleID = override.ModuleID
	}
}
