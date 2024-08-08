# Benchmarking data
You can setup the benchmarking environment by following [this](https://github.com/kubearmor/KubeArmor/wiki/Kubearmor-Performance-Benchmarking-Guide) guide.
### Config 
- Node: 3 e2-custom-2-4096 (2vCPU. 4GB RAM), 1 e2-standard-4  = 4 Node Cluster
- Platform - GKE
- Workload -> [Microservices-Demo](https://github.com/GoogleCloudPlatform/microservices-demo)
- Tool -> Locust Loadgenerator (request at front-end service)

{{- range .Reports }}

## Report for {{.Case}} - {{.MetricName}}

Users | {{- range .KubearmorResourceUsages }} {{ .Name }}  | {{- end }} Throughput (req/s) | Percentage Drop | {{- range .ResourceUsages }} {{ .Name }}  | {{- end }} |
--  |{{- range .KubearmorResourceUsages }}--  |  --{{- end }}|  --  |  --  |{{- range .ResourceUsages }}--  |  --{{- end }}|
 {{.Users}} | {{- range .KubearmorResourceUsages }}CPU={{ .CPU }}, Memory={{ .Memory }} | {{- end }} | | {{.Throughput}} | {{ .PercentageDrop }} | {{- range .ResourceUsages }}CPU={{ .CPU }}, Memory={{ .Memory }} | {{- end }} |

{{- end }}