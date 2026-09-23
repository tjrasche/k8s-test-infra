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

package engine

import nvmlconfig "github.com/NVIDIA/k8s-test-infra/pkg/gpu/mocknvml/config"

// Schema aliases preserve engine callers while allowing configuration producers
// to use the schema without importing the native runtime.
type (
	// YAMLConfig is documented in [nvmlconfig.YAMLConfig].
	YAMLConfig = nvmlconfig.YAMLConfig
	// SystemConfig is documented in [nvmlconfig.SystemConfig].
	SystemConfig = nvmlconfig.SystemConfig
	// DeviceConfig is documented in [nvmlconfig.DeviceConfig].
	DeviceConfig = nvmlconfig.DeviceConfig
	// PlatformConfig is documented in [nvmlconfig.PlatformConfig].
	PlatformConfig = nvmlconfig.PlatformConfig
	// NVLinkErrorInjectionConfig is documented in [nvmlconfig.NVLinkErrorInjectionConfig].
	NVLinkErrorInjectionConfig = nvmlconfig.NVLinkErrorInjectionConfig
	// DeviceOverride is documented in [nvmlconfig.DeviceOverride].
	DeviceOverride = nvmlconfig.DeviceOverride
	// ComputeCapabilityConfig is documented in [nvmlconfig.ComputeCapabilityConfig].
	ComputeCapabilityConfig = nvmlconfig.ComputeCapabilityConfig
	// InfoROMConfig is documented in [nvmlconfig.InfoROMConfig].
	InfoROMConfig = nvmlconfig.InfoROMConfig
	// MemoryConfig is documented in [nvmlconfig.MemoryConfig].
	MemoryConfig = nvmlconfig.MemoryConfig
	// BAR1MemoryConfig is documented in [nvmlconfig.BAR1MemoryConfig].
	BAR1MemoryConfig = nvmlconfig.BAR1MemoryConfig
	// PCIConfig is documented in [nvmlconfig.PCIConfig].
	PCIConfig = nvmlconfig.PCIConfig
	// PCIeConfig is documented in [nvmlconfig.PCIeConfig].
	PCIeConfig = nvmlconfig.PCIeConfig
	// PowerConfig is documented in [nvmlconfig.PowerConfig].
	PowerConfig = nvmlconfig.PowerConfig
	// WorkloadPowerProfilesConfig is documented in [nvmlconfig.WorkloadPowerProfilesConfig].
	WorkloadPowerProfilesConfig = nvmlconfig.WorkloadPowerProfilesConfig
	// WorkloadPowerProfileConfig is documented in [nvmlconfig.WorkloadPowerProfileConfig].
	WorkloadPowerProfileConfig = nvmlconfig.WorkloadPowerProfileConfig
	// ThermalConfig is documented in [nvmlconfig.ThermalConfig].
	ThermalConfig = nvmlconfig.ThermalConfig
	// FanConfig is documented in [nvmlconfig.FanConfig].
	FanConfig = nvmlconfig.FanConfig
	// ClocksConfig is documented in [nvmlconfig.ClocksConfig].
	ClocksConfig = nvmlconfig.ClocksConfig
	// ClocksThrottleReasonsConfig is documented in [nvmlconfig.ClocksThrottleReasonsConfig].
	ClocksThrottleReasonsConfig = nvmlconfig.ClocksThrottleReasonsConfig
	// ThrottleCountersConfig is documented in [nvmlconfig.ThrottleCountersConfig].
	ThrottleCountersConfig = nvmlconfig.ThrottleCountersConfig
	// SupportedClocksConfig is documented in [nvmlconfig.SupportedClocksConfig].
	SupportedClocksConfig = nvmlconfig.SupportedClocksConfig
	// MemoryClockConfig is documented in [nvmlconfig.MemoryClockConfig].
	MemoryClockConfig = nvmlconfig.MemoryClockConfig
	// UtilizationConfig is documented in [nvmlconfig.UtilizationConfig].
	UtilizationConfig = nvmlconfig.UtilizationConfig
	// EncoderStatsConfig is documented in [nvmlconfig.EncoderStatsConfig].
	EncoderStatsConfig = nvmlconfig.EncoderStatsConfig
	// FBCStatsConfig is documented in [nvmlconfig.FBCStatsConfig].
	FBCStatsConfig = nvmlconfig.FBCStatsConfig
	// ECCConfig is documented in [nvmlconfig.ECCConfig].
	ECCConfig = nvmlconfig.ECCConfig
	// ECCSramConfig is documented in [nvmlconfig.ECCSramConfig].
	ECCSramConfig = nvmlconfig.ECCSramConfig
	// ECCSramCountsConfig is documented in [nvmlconfig.ECCSramCountsConfig].
	ECCSramCountsConfig = nvmlconfig.ECCSramCountsConfig
	// ECCSramSourcesConfig is documented in [nvmlconfig.ECCSramSourcesConfig].
	ECCSramSourcesConfig = nvmlconfig.ECCSramSourcesConfig
	// ECCErrorsConfig is documented in [nvmlconfig.ECCErrorsConfig].
	ECCErrorsConfig = nvmlconfig.ECCErrorsConfig
	// ECCErrorCountsConfig is documented in [nvmlconfig.ECCErrorCountsConfig].
	ECCErrorCountsConfig = nvmlconfig.ECCErrorCountsConfig
	// ECCMemoryErrorsConfig is documented in [nvmlconfig.ECCMemoryErrorsConfig].
	ECCMemoryErrorsConfig = nvmlconfig.ECCMemoryErrorsConfig
	// RetiredPagesConfig is documented in [nvmlconfig.RetiredPagesConfig].
	RetiredPagesConfig = nvmlconfig.RetiredPagesConfig
	// RetirementInfoConfig is documented in [nvmlconfig.RetirementInfoConfig].
	RetirementInfoConfig = nvmlconfig.RetirementInfoConfig
	// RemappedRowsConfig is documented in [nvmlconfig.RemappedRowsConfig].
	RemappedRowsConfig = nvmlconfig.RemappedRowsConfig
	// RowRemapHistogramConfig is documented in [nvmlconfig.RowRemapHistogramConfig].
	RowRemapHistogramConfig = nvmlconfig.RowRemapHistogramConfig
	// DisplayConfig is documented in [nvmlconfig.DisplayConfig].
	DisplayConfig = nvmlconfig.DisplayConfig
	// MIGConfig is documented in [nvmlconfig.MIGConfig].
	MIGConfig = nvmlconfig.MIGConfig
	// GPUOperationModeConfig is documented in [nvmlconfig.GPUOperationModeConfig].
	GPUOperationModeConfig = nvmlconfig.GPUOperationModeConfig
	// DriverModelConfig is documented in [nvmlconfig.DriverModelConfig].
	DriverModelConfig = nvmlconfig.DriverModelConfig
	// AccountingConfig is documented in [nvmlconfig.AccountingConfig].
	AccountingConfig = nvmlconfig.AccountingConfig
	// VirtualizationConfig is documented in [nvmlconfig.VirtualizationConfig].
	VirtualizationConfig = nvmlconfig.VirtualizationConfig
	// GSPFirmwareConfig is documented in [nvmlconfig.GSPFirmwareConfig].
	GSPFirmwareConfig = nvmlconfig.GSPFirmwareConfig
	// FeaturesConfig is documented in [nvmlconfig.FeaturesConfig].
	FeaturesConfig = nvmlconfig.FeaturesConfig
	// CPUConfig is documented in [nvmlconfig.CPUConfig].
	CPUConfig = nvmlconfig.CPUConfig
	// ProcessConfig is documented in [nvmlconfig.ProcessConfig].
	ProcessConfig = nvmlconfig.ProcessConfig
	// TopologyConfig is documented in [nvmlconfig.TopologyConfig].
	TopologyConfig = nvmlconfig.TopologyConfig
	// DynamicMetricsConfig is documented in [nvmlconfig.DynamicMetricsConfig].
	DynamicMetricsConfig = nvmlconfig.DynamicMetricsConfig
	// DynamicTemperatureConfig is documented in [nvmlconfig.DynamicTemperatureConfig].
	DynamicTemperatureConfig = nvmlconfig.DynamicTemperatureConfig
	// DynamicPowerConfig is documented in [nvmlconfig.DynamicPowerConfig].
	DynamicPowerConfig = nvmlconfig.DynamicPowerConfig
	// DynamicUtilizationConfig is documented in [nvmlconfig.DynamicUtilizationConfig].
	DynamicUtilizationConfig = nvmlconfig.DynamicUtilizationConfig
	// GPMConfig is documented in [nvmlconfig.GPMConfig].
	GPMConfig = nvmlconfig.GPMConfig
	// FailureInjectionConfig is documented in [nvmlconfig.FailureInjectionConfig].
	FailureInjectionConfig = nvmlconfig.FailureInjectionConfig
	// XidErrorConfig is documented in [nvmlconfig.XidErrorConfig].
	XidErrorConfig = nvmlconfig.XidErrorConfig
	// FabricConfig is documented in [nvmlconfig.FabricConfig].
	FabricConfig = nvmlconfig.FabricConfig
	// FabricHealthConfig is documented in [nvmlconfig.FabricHealthConfig].
	FabricHealthConfig = nvmlconfig.FabricHealthConfig
	// NVLinkConfig is documented in [nvmlconfig.NVLinkConfig].
	NVLinkConfig = nvmlconfig.NVLinkConfig
	// NVSwitchConfig is documented in [nvmlconfig.NVSwitchConfig].
	NVSwitchConfig = nvmlconfig.NVSwitchConfig
	// NVLinkDefaults is documented in [nvmlconfig.NVLinkDefaults].
	NVLinkDefaults = nvmlconfig.NVLinkDefaults
	// DeviceLinksConfig is documented in [nvmlconfig.DeviceLinksConfig].
	DeviceLinksConfig = nvmlconfig.DeviceLinksConfig
	// NVLinkLinkConfig is documented in [nvmlconfig.NVLinkLinkConfig].
	NVLinkLinkConfig = nvmlconfig.NVLinkLinkConfig
	// PCIeTopologyConfig is documented in [nvmlconfig.PCIeTopologyConfig].
	PCIeTopologyConfig = nvmlconfig.PCIeTopologyConfig
	// RootComplexConfig is documented in [nvmlconfig.RootComplexConfig].
	RootComplexConfig = nvmlconfig.RootComplexConfig
)

// Failure modes are shared with the configuration schema.
const (
	FailureModeHealthy          = nvmlconfig.FailureModeHealthy
	FailureModeLost             = nvmlconfig.FailureModeLost
	FailureModeFallenOffBus     = nvmlconfig.FailureModeFallenOffBus
	FailureModeECCUncorrectable = nvmlconfig.FailureModeECCUncorrectable
)

// MaxDevices is the maximum number of devices supported by the mock server.
const MaxDevices = nvmlconfig.MaxDevices

// TopologyDocument is the cluster-level ConfigMap that maps individual
// nodes (by Kubernetes node name) to fabric clusters and cliques. It is
// the single source of truth for ComputeDomain topology in the mock.
//
// At LoadConfig() time the engine looks up the current node (via
// NODE_NAME) and, if found in the topology, overrides the per-device
// FabricConfig.ClusterUUID / CliqueID. Nodes absent from the topology
// keep their default fabric config (or report NOT_SUPPORTED when no
// FabricConfig is present).
type TopologyDocument struct {
	Version int              `json:"version"`
	Domains []TopologyDomain `json:"domains"`
}

// TopologyDomain represents one NVLink fabric domain (cluster UUID).
type TopologyDomain struct {
	Name    string           `json:"name,omitempty"`
	UUID    string           `json:"uuid"`
	Cliques []TopologyClique `json:"cliques"`
}

// TopologyClique groups the Kubernetes node names that share a clique
// inside a fabric domain.
type TopologyClique struct {
	ID    uint32   `json:"id"`
	Nodes []string `json:"nodes"`
}
