# Delivery fixtures

These are hand-authored conformance snapshots for the
[node-agent delivery contract](../../../../enhancements/meps/0001-mokka-control-plane/node-agent-delivery.md),
not output from an implemented control-plane adapter.

- `assigned-a100.json` uses the hardware, software, PCI words and byte quantities
  from the [A100 rack example](../../../../examples/controlplane-crds/sgpu-rack-profile.yaml).
  It adds representative assigned identities and an explicit count. GPU index 0
  deliberately uses minor 3: consumers must preserve identity rather than assume
  index and minor are interchangeable.
- `assigned-gb200.json` exercises the consumer representation using a selected
  subset of the shipped [GB200 profile](../../../../deployments/nvml-mock/helm/nvml-mock/profiles/gb200.yaml).
  It retains four GPUs, shared board serials, module order 2/1/4/3, slot 21, tray
  11, NVLink/C2C and switch topology. It replaces the placeholder fabric identity,
  canonicalizes PCI addresses and resolves the HCA count. It is not a claim that
  the current rack CRDs can produce all these fields.
- `unassigned.json` is an ordered withdrawal, not an empty engine profile.
- `not-ready.json` carries no replacement configuration.

The tests load the payload through existing engine types and the YAML consumer,
check the per-device PCI-address getter used during device creation, and verify
the fabric UUID reported by a configurable device. They check selected source
values, not a live agent or the application of local overrides.

Mapping work still needed before serving these shapes from rack declarations:

- The current engine supports only eight GPUs, although rack profiles allow 64.
  The contract rejects unsupported counts rather than accepting a clipped view.
- GPU and fabric UUIDs use canonical lowercase dashed form. In particular, the
  engine's fabric parser does not preserve URN-form UUIDs.
- GB200 chassis, slot, tray and module placement are not fully represented by the
  current CRDs. Rack/node indices are not physical placement values.
- Total host cores do not determine CPU affinity; these fixtures do not invent it.
- The network adapter must explicitly map supported models to HCA metadata;
  the A100 example's ConnectX-7 is represented as MT4129.
- The legacy `bandwidth_per_link_mbps` name has a units mismatch with its byte-rate
  examples. The GB200 fixture preserves the current consumer's value of 53125;
  the future adapter must test the observable conversion rather than infer units
  from the field name.

No CRD fields, consumer reload behavior or runtime-policy precedence are changed
by these fixtures.
