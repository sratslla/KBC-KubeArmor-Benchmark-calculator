# Design: KubeArmor Benchmark Calculator (kbc)

## Purpose

Reproduce the methodology in:

- [Kubearmor Performance Benchmarking Guide](https://github.com/kubearmor/KubeArmor/wiki/Kubearmor-Performance-Benchmarking-Guide)
- [KubeArmor Performance Benchmarking Data](https://github.com/kubearmor/KubeArmor/wiki/KubeArmor-Performance-Benchmarking-Data)

as an automated, repeatable CLI suitable for upstreaming to [`kubearmor/kbc`](https://github.com/kubearmor/kbc).

## Principles

1. **Methodology as config** — `configs/profiles/release.yaml` encodes users, soak, sample count, and the full scenario matrix (baseline, BPF policies, AppArmor, system-monitor/visibility, visibility modes).
2. **client-go control plane** — Deployments, HPAs, namespace annotations, KSP/Config CRDs via typed/dynamic clients. No `kubectl`/`karmor`/`curl|sh` on the happy path.
3. **Documented install** — If KubeArmor is missing: Helm Operator + `KubeArmorConfig` (official deployment guide). Relay pinned to the Locust tainted node.
4. **Condition waits + intentional soak** — Readiness uses `PollUntilContextTimeout`. The Guide’s 10–15 minute stable region is a deliberate soak + N Prometheus samples (default 12m + 5×1m).
5. **Freeze replicas** — After Locust reaches target users, wait for HPA stability, delete HPAs, and pin Deployment replicas for fair comparison across scenarios.
6. **Locust headless** — Same Boutique locustfile/image family as the Guide, run with `--headless` and fixed `--run-time` for unattended execution; optional `--debug-ui` NodePort.
7. **One binary** — Works with kubeconfig or in-cluster config; Job manifest under `deploy/job/`.

## Scenario matrix

| Section | What changes |
|---------|----------------|
| Without KubeArmor | Measure before `ensure` |
| BPF LSM | Policy ladder process → +network → +file |
| AppArmor Pure | Prefer `apparmor` LSM; visibility none; same policies (SKIPPED if unavailable) |
| System Monitor Visibility None | BPF + visibility none + policies |
| Visibility Modes | none / process / process+file / process+network / process+network+file |

## Metrics

Prometheus instant queries for Locust RPS/failures, KubeArmor/relay CPU+memory, frontend/cart/currency CPU and replica counts. Aggregate = mean of N samples; percentage drop vs baseline throughput.

## Package map

```
cmd/kbc                 entrypoint
internal/cli            Cobra commands
internal/cluster        kube clients + GVRs
internal/ensure         Helm + KubeArmorConfig + relay pin
internal/workload       manifests, HPA, freeze, Locust, policies, visibility
internal/metrics        Prometheus sampler
internal/scenario       profile runner + checkpoint
internal/report         markdown/JSON
internal/waitutil       polling helpers
```

## Non-goals

- Building a kubebuilder controller for the benchmarker itself
- Replacing Locust with k6/fortio in this rewrite (continuity with published wiki numbers)
- Auto-applying Terraform on every PR
