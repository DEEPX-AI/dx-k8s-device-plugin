<div align="center">

# DX K8s Device Plugin

**Kubernetes device plugin for DEEPX NPUs.**

[![Release](https://img.shields.io/github/v/release/DEEPX-AI/dx-k8s-device-plugin)](https://github.com/DEEPX-AI/dx-k8s-device-plugin/releases)

</div>

DX K8s Device Plugin advertises DEEPX NPUs to Kubernetes as the `deepx.ai/dx-m1` extended resource and injects
the NPU device nodes into pods via CDI. It is part of [DX-AllSuite](https://github.com/DEEPX-AI/dx-all-suite),
which deploys it with the `dx-npu` Helm chart. The plugin only discovers, health-checks and schedules the cards;
the NPU driver, firmware and DX-RT must already be installed on each node by
[DX-Runtime](https://github.com/DEEPX-AI/dx-runtime).

## What You Need

| | |
|---|---|
| **NPU** | DX-M1 (verified on hardware). Every card under `/sys/class/dxrt/` is advertised as `deepx.ai/dx-m1` (whole device); DX-M1M and DX-H1 Quattro are not yet verified |
| **Arch** | amd64, arm64 (multi-arch image `ghcr.io/deepx-ai/dx-k8s-device-plugin`) |
| **Kubernetes** | k3s (tested), kubelet Device Plugin API v1beta1 |
| **Container runtime** | containerd with CDI enabled — default in 2.x; 1.7 needs `enable_cdi=true` and `/etc/cdi` in `cdi_spec_dirs` |
| **Node (host)** | NPU driver, firmware and DX-RT from `dx-runtime/install.sh --runtime-only` |
| **Build** | Go 1.22+ |

The runtime base image needs a glibc at least as new as the host's DX-RT build, or every card reports Unhealthy.

On each NPU node, stop the host runtime service — pods run their own `dxrtd`, and two on one card hang inference:
`sudo systemctl disable --now dxrt.service`. The node's DX-RT must be the same build as the one in the pod image
(`dxcli --version` on both), or pods fail with undefined symbol errors.

## Quick Start

On a cluster whose NPU nodes already have the driver, firmware and DX-RT, install the `dx-npu` chart and check
that the nodes advertise the NPU:

```bash
helm install dx-npu oci://ghcr.io/deepx-ai/charts/dx-npu -n dx-system --create-namespace
kubectl describe nodes | grep deepx.ai/dx-m1
```

A pod then requests a card like CPU or memory: `resources.limits: {deepx.ai/dx-m1: 1}`.
NFD, chart values and the containerd CDI setup → [Kubernetes Quickstart](https://developer.deepx.ai/tech-docs/DX-AllSuite/Kubernetes_Quickstart/) · [DX-AllSuite `deploy/k8s`](https://github.com/DEEPX-AI/dx-all-suite/tree/main/deploy/k8s)

<details>
<summary><b>Install Without Helm (development)</b></summary>

```bash
# On each NPU node: driver + firmware + DX-RT
cd dx-runtime && ./install.sh --runtime-only

# Then, with containerd CDI enabled:
kubectl apply -f deploy/dx-device-plugin.yaml
kubectl apply -f deploy/test-pod.yaml && kubectl logs dx-m1-test
```

</details>

<details>
<summary><b>Configuration, Components, Device Facts and Tests</b></summary>

**Configuration**

| Environment variable | Description |
|---|---|
| `METRICS_ADDR` | Address for the `deepx_npu_*` Prometheus metrics endpoint. Metrics are off when unset. |
| `NFD_FEATURE_DIR` | NFD `features.d` directory for the node feature file. |

**Components**

- **`internal/dxdevice`** — enumeration and health. sysfs (`/sys/class/dxrt/dxrtN`) is authoritative for the
  allocatable card list; `dxcli -s` supplies metadata (product, RT/PCIe driver, firmware, PCIe BDF) and health.
- **`internal/cdi`** — CDI 0.6.0 spec generation (`/etc/cdi/deepx.json`): one CDI device per card, plus host
  runtime-lib and `dxcli` mounts so app images stay thin.
- **`internal/plugin`** — kubelet Device Plugin API: `ListAndWatch` (health-aware) and `Allocate` (CDI dual path:
  typed `CDIDevices` + legacy annotation).
- **`internal/nfd`** — node-feature-discovery local feature file, turned by the NFD worker into node labels for
  card count, product, and firmware/driver versions.
- **`internal/metrics`** — `deepx_npu_*` Prometheus gauges (device health, per-core temperature/voltage/clock).
- **`internal/monitor` + `cmd/dx-device-plugin`** — regeneration loop, gRPC server, kubelet registration,
  re-registration on kubelet restart (fsnotify).
- **`Dockerfile`, `.github/`, `deploy/`** — multi-arch image build and publish to ghcr, raw DaemonSet and
  smoke-test pod.

**Device facts** (DX-M1, verified on hardware)

| | |
|---|---|
| PCI vendor:device | `1ff4:0100` (M1), `1ff4:0110` (M1M), `1ff4:0000` (legacy ID on older firmware) |
| Device node | `/dev/dxrtN` (one per card, char major 507) |
| Enumeration | `ls /sys/class/dxrt/` |
| Metadata / health | `dxcli -s [-d N]` |
| Resource | `deepx.ai/dx-m1` (whole device) |

**Tests**

```bash
go test ./...
```

`TestList_RealHardware` runs the full sysfs + `dxcli` path on a node with an NPU and skips automatically where
none is present.

</details>

## Documentation & Support

- Setup guides: **[Kubernetes Quickstart](https://developer.deepx.ai/tech-docs/DX-AllSuite/Kubernetes_Quickstart/)** · [Beginner Guide](https://developer.deepx.ai/tech-docs/DX-AllSuite/Kubernetes_Beginner_Guide/) · [DX-AllSuite `deploy/k8s`](https://github.com/DEEPX-AI/dx-all-suite/tree/main/deploy/k8s)
- For technical questions, please ask in the DEEPX Developer Community: https://community.deepx.ai/c/support/5

## License

See [LICENSE](./LICENSE) for terms of use.

Copyright © DEEPX Co., Ltd.
