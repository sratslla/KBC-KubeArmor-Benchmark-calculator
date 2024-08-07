# Final Report

| Case                    | Metric Name | Users | Throughput | Percentage Drop | Resource Usages |
|-------------------------|-------------|-------|------------|-----------------|-----------------|
{{- range .Reports }}
| {{ .Case }}             | {{ .MetricName }} | {{ .Users }} | {{ .Throughput }} | {{ .PercentageDrop }} | {{ range .ResourceUsages }}{{ .Name }}: CPU={{ .CPU }}, Memory={{ .Memory }}{{ end }} |
{{- end }}