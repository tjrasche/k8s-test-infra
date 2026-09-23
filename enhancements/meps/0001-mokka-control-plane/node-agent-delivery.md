# Node-agent delivery contract

This document specifies the `v1alpha1` read protocol for the
[MEP0001 node agent](README.md#node-agent).

**Implementation status:** shared wire types, validation and payload fixtures live
in [`internal/sgpu/delivery`](../../../internal/sgpu/delivery). The HTTP endpoint,
controller publication machinery, polling client and persistent cache are not
implemented by this contract. The current server still exposes health endpoints
only; these request examples describe the target API, not a runnable service.

## Ownership

The control plane delivers a complete configuration for one exact Kubernetes
Node. It resolves assignment, assigned GPU/fabric identities, supported defaults,
and eventually runtime policies. The agent translates that configuration into
simulator inputs; it does not allocate capacity or choose policies.

Rack profiles remain rack-oriented declarations. This node-oriented delivery
representation is a consumer format, not a replacement declaration schema.

## Request

```http
GET /v1alpha1/nodes/worker-01/configuration?nodeUID=node-instance-a&lastAppliedAssignmentRevision=7&lastAppliedRackDeliveryGeneration=12
Accept: application/json
```

| Input | Meaning |
| --- | --- |
| `nodeName`, in the path | Required Kubernetes Node name. Selects the resource. |
| `nodeUID`, in the query | Required opaque Kubernetes Node UID. Requires that the name still identifies the expected Node object, rather than a replacement. |
| `lastAppliedAssignmentRevision` | Optional positive signed-64-bit decimal integer. |
| `lastAppliedRackDeliveryGeneration` | Optional nonnegative signed-64-bit decimal integer. |

The cursor parameters must appear exactly once together or both be absent. Omit
them when there is no successfully applied configuration for the requested
`nodeUID`; `(0, 0)` is not a successful bootstrap state. Duplicate identity/cursor
parameters, malformed escaping, signed values, fractions and overflow are invalid.

Use the name **`nodeUID`**, not `uid`, `UUID` or `nodeUUID`, in the request and
Node response object. Go uses `NodeUID`. Kubernetes Node UIDs are opaque strings;
unlike GPU UUIDs, the examples need not have UUID syntax. Node UID is not Pod UID.

This is a read-only operation. The cursor supplies a lower bound for the response,
not a persisted acknowledgement, heartbeat, or proof of authorization.

## Successful response

Return `200 OK`, `Content-Type: application/json`, and
`Cache-Control: no-store`. Return a full snapshot even when its cursor is
unchanged. This version has no patches, ETags/304, long-polling or acknowledgement
endpoint.

The envelope contains:

| Field | Assigned state | Unassigned state |
| --- | --- | --- |
| `apiVersion` | `"v1alpha1"` | Same |
| `node.nodeName`, `node.nodeUID` | Exactly the requested identity | Same |
| `assignmentRevision` | Positive, scoped to this Node UID | Same |
| `rackDeliveryGeneration` | Positive, scoped to the rack UID | Explicit zero |
| `state` | `"assigned"` | `"unassigned"` |
| `rack` | Exact `name`, `uid`, and nonnegative `nodeIndex` | Omitted |
| `configuration` | Required typed `nvml` and `infiniband` objects | Omitted |

`rack.nodeIndex` is the logical slot, not a chassis slot or physical tray index.
The body carries `apiVersion` so a persisted envelope can be validated without
its original HTTP URL. It is not a Kubernetes resource and has no `kind`, `spec`
or `status` wrapper.

Complete assigned examples:

- [A100 snapshot](../../../internal/sgpu/delivery/testdata/assigned-a100.json)
- [GB200 rack-fabric snapshot](../../../internal/sgpu/delivery/testdata/assigned-gb200.json)

An explicit unassigned snapshot is:

```json
{
  "apiVersion": "v1alpha1",
  "node": {"nodeName": "worker-01", "nodeUID": "node-instance-a"},
  "assignmentRevision": 9,
  "rackDeliveryGeneration": 0,
  "state": "unassigned"
}
```

Only a published unassigned snapshot requests withdrawal. Empty profiles,
missing resources, zero telemetry and failed requests never imply unassignment.
A never-assigned Node without a published ordering record receives not-ready;
the server must not invent a revision-zero tombstone.

## Consumer payload

`configuration.nvml` reuses the current engine configuration representation,
with engine schema `version: "1.0"`. `configuration.infiniband` reuses the current
network configuration representation. They are typed JSON objects, not YAML
strings or arbitrary extension maps. Local conversion to the engine's YAML
input must preserve the delivered identities. NVML and the agent's host-surface
inputs derive from the same representation, not independently authored device
lists.

The initial static assigned payload requires:

- explicit driver, NVML and CUDA versions;
- `system.num_devices` between 1 and the current engine limit of 8, matching the
  complete device list;
- exactly one device for each index from zero to count minus one;
- explicit unique GPU UUIDs in `GPU-` plus lowercase dashed UUID form, explicit
  valid/unique minors and per-device serials (serials need not be unique);
- resolved GPU name, supported architecture, memory capacity, packed PCI identity
  words and unique canonical lowercase PCI addresses in each `devices[].pci.bus_id`;
  placing an address only in `device_defaults` is not supported by the consumer;
- a lowercase dashed fabric UUID when a fabric block is supplied, and explicit NVLink
  version/count/bandwidth when NVLink is supplied;
- an InfiniBand object, with resolved defaults and a positive explicit HCA count
  and link speed when enabled.

Common hardware may use the engine's `device_defaults` block and per-device
identity overrides. This is a self-contained consumer document: no unresolved
profile references or dependence on a chart-selected profile or local
`GPU_COUNT`/`DRIVER_VERSION` to author its identity. Although rack declarations
allow up to 64 GPUs per node, the current engine clips to eight; delivery rejects
larger snapshots instead of silently losing devices. The existing engine cannot
reliably represent an assigned zero-visible-GPU state; this initial contract
rejects it rather than invoking its default-device behavior. Withdrawal uses
the separate unassigned state.

The validators check envelope and core consumer invariants, not every possible
simulator setting or the controller's dependency manifest. The future server
adapter must validate supported mappings before offering a snapshot. Fixtures
prove representation and current-consumer loading, **not** complete CRD-to-NVML
conversion or live reconfiguration.

This is a transitional dependency on existing consumer types. Changes to those
types require wire-compatibility review; they are not automatically new protocol
features. The [fixture notes](../../../internal/sgpu/delivery/testdata/README.md)
record known mapping gaps. In particular, current rack declarations cannot
supply all GB200 platform fields, and host core count does not establish CPU
affinity. Do not manufacture these values to make a conversion succeed.

Local filesystem roots, credential/cache paths, sockets, simulation capability
settings and runtime override files are outside the payload. Policy objects are
not delivered. The control-plane baseline does not confer ownership of
node-local fault-injection or allocation-watcher overrides.

## Ordering and application

The two-counter ordering and replica consistency requirements are defined in
[the node-agent architecture](README.md#node-agent) and
[runtime view computation and delivery](README.md#runtime-view-computation-and-delivery).
The wire implementation applies those rules:

| Applied | Offered | Result |
| --- | --- | --- |
| `(7, 12)` | `(7, 13)` | Accept candidate |
| `(7, 12)` | `(7, 11)` or `(6, 100)` | Reject regression |
| `(7, 12)` | `(8, 2)` | Accept assignment change, even with a lower rack generation |
| `(7, 12)` | `(8, 0)`, unassigned | Accept withdrawal candidate |
| `(8, 0)` | `(7, 99)`, assigned | Reject restoration of an old assignment |

Changing rack identity, logical slot or assigned/unassigned state requires a
higher assignment revision. For the same Node UID and pair, the typed
configuration must be identical, including array ordering. JSON object key
ordering and ignored informational fields do not matter. Different pairs may
carry the same configuration: rack-wide generation can advance because another
node changed.

Acceptance is not acknowledgement. Advance the applied cursor only after the
required application succeeds, with durable cache promotion participating when
implemented. A repeated failed candidate remains retryable. Even an equal
cursor cannot be deduplicated after restart until the agent restores/applies its
completed snapshot. The contract helpers do not store or advance cursors.

A different `nodeUID` starts a fresh domain only after independently establishing
that Node identity. A mismatched response cannot switch the agent to another
Node. Cursors and cached snapshots are also scoped to the operator-configured
control-plane authority; changing authority cannot reuse an unrelated cursor.

This provides internally consistent, non-regressing delivery, not a linearizable
latest-state read. A lagging replica may offer an older consistent snapshot
until the agent reports a newer applied pair. During outages the last completed
configuration can remain stale indefinitely; no TTL turns silence into capacity
reclamation or GPU withdrawal.

## Failure responses

Every failure has no replacement configuration. Return JSON and
`Cache-Control: no-store`:

| HTTP status | `code` | Action |
| --- | --- | --- |
| `400` | `InvalidRequest` | Retain applied state; surface the request/configuration error. |
| `409` | `NodeUIDMismatch` | Retain state; revalidate identity using the trusted identity source. Do not adopt a UID from the response. |
| `503` | `ConfigurationNotReady` | Retain state and retry. A bounded `Retry-After` hint may be included. |

Example body:

```json
{
  "code": "ConfigurationNotReady",
  "message": "A complete published configuration is not available."
}
```

Messages are diagnostic, not machine-readable conditions. Use not-ready for an
unsynchronized cache, unknown/missing exact Node, ambiguous/deleting binding,
partial publication, missing dependencies or a response behind the applied
cursor. A known Node with a different UID yields the identity-mismatch response.

Network failures, malformed/unsupported responses, other non-success statuses
and authentication failures also retain applied state. A 404 is never withdrawal.
Unknown API versions/state values are rejected. Additive informational fields
may be ignored; new behavior that an older agent must apply needs a new API
version rather than relying on silently ignored fields.

## Identity and transport prerequisite

The operator chooses the endpoint. Name and Node UID are consistency checks,
not authentication. Production authentication and trusted Node UID bootstrap
remain deployment prerequisites; this contract does not authorize an
unauthenticated rollout. HTTPS must verify the server's identity and trust chain.
