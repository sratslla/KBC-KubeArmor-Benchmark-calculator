# Benchmarking data template (legacy html/template style retained for optional use)

You can setup the benchmarking environment by following [this](https://github.com/kubearmor/KubeArmor/wiki/Kubearmor-Performance-Benchmarking-Guide) guide.

### Config
- Node: 3 e2-custom-2-4096 (2vCPU. 4GB RAM), 1 e2-standard-4 = 4 Node Cluster
- Platform - GKE
- Workload -> [Microservices-Demo](https://github.com/GoogleCloudPlatform/microservices-demo)
- Tool -> Locust Loadgenerator (request at front-end service)

Prefer `kbc run` which writes wiki-shaped `final_report.md` via `internal/report`.
