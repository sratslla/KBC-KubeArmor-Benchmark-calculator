# Benchmarking data
You can setup the benchmarking environment by following [this](https://github.com/kubearmor/KubeArmor/wiki/Kubearmor-Performance-Benchmarking-Guide) guide.
### Config 
- Node: 3 e2-custom-2-4096 (2vCPU. 4GB RAM), 1 e2-standard-4  = 4 Node Cluster
- Platform - GKE
- Workload -> [Microservices-Demo](https://github.com/GoogleCloudPlatform/microservices-demo)
- Tool -> Locust Loadgenerator (request at front-end service)

{{- range .Reports }}

## Report for {{.Case}}

Users | Kubearmor CPU | Kubearmor Relay CPU (m) | Kubearmor Memory | Kubearmor Relay Memory | Throughput (req/s) | Percentage Drop | {{- range .ResourceUsages }}CPU ({{ .Name }}), Memory ({{ .Name }}) | {{- end }} |
--  |  --  |  --  |  --  |  --  |  --  |  --  |{{- range .ResourceUsages }}--  |  --{{- end }}|
 {{USER}} | - | - | - | - | {{.Throughput}} | {{ .PercentageDrop }} | {{- range .ResourceUsages }}CPU={{ .CPU }}, Memory={{ .Memory }} | {{- end }} |

{{- end }}