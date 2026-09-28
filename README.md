# KubeArmor Benchmark Calculator (kbc)

Automates [KubeArmor performance benchmarking](https://github.com/kubearmor/KubeArmor/wiki/Kubearmor-Performance-Benchmarking-Guide) against Google's Online Boutique using Locust and Prometheus, producing reports shaped like the [Benchmarking Data](https://github.com/kubearmor/KubeArmor/wiki/KubeArmor-Performance-Benchmarking-Data) wiki page.

## Prerequisites

- Go 1.22+
- `kubectl` configured for a Guide-shaped cluster (4 nodes; one tainted `color=blue` / labeled `nodetype=node1`)
- Helm 3 (used only to install the KubeArmor Operator when missing)
- Cluster access with permission to create Deployments, HPAs, CRDs, and namespaces

## Build

```bash
go build -o kbc ./cmd/kbc
```

## Quick start

```bash
# Optional: provision GKE (see deploy/terraform)
cd deploy/terraform
terraform init
terraform apply -var="project=YOUR_GCP_PROJECT"

# Apply boutique + Prometheus/KSM + HPA + headless Locust
./kbc setup --users=2000

# Or run the full release profile (setup + baseline + ensure KA + full matrix)
./kbc run --profile=release --out=./out
```

Commands:

| Command | Purpose |
|---------|---------|
| `kbc setup` | Apply local manifests, HPAs, headless Locust |
| `kbc ensure` | Detect/install KubeArmor via Helm Operator + `KubeArmorConfig`, pin relay |
| `kbc run` | Full matrix from `configs/profiles/release.yaml` |
| `kbc destroy` | Tear down workload; `--uninstall-ka` removes Helm release |

Resume a long run:

```bash
./kbc run --profile=release --from-scenario=bpf-process --checkpoint=./out/checkpoint.json --out=./out
```

## Cluster shape

Matches the wiki Guide:

- 3× `e2-custom-2-4096` worker nodes
- 1× `e2-standard-4` tainted node for Locust + KubeArmor relay
- Prometheus NodePort `30000` (or in-cluster `prometheus-service.monitoring.svc:8080`)

Terraform lives in [`deploy/terraform`](deploy/terraform).

## Design

See [DESIGN.md](DESIGN.md).

## License

Apache-2.0
