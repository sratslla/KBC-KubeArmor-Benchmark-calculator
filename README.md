# GKE Cluster Automation with Terraform and GitHub Actions

This project automates the creation of a Google Kubernetes Engine (GKE) cluster using Terraform and GitHub Actions. It runs a CLI tool on the GKE cluster, captures the output, and updates a GitHub Wiki page and a Slack channel with the results.

## Running
1. Use the Terraform file to set up the cluster.
2. clone this repo on the machine
3. Build the tool
   ``` go build -o KBC main ```
4. Start the  tool
   ```./KBC start```

## NOTE
It takes about 10 mins to apply the Terraform Configuration and around 60 mins to calculate Benchmark.
