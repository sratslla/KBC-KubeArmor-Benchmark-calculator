# GKE Cluster Automation with Terraform and GitHub Actions

This project automates the creation of a Google Kubernetes Engine (GKE) cluster using Terraform and GitHub Actions. It runs a CLI tool on the GKE cluster, captures the output, and updates a GitHub Wiki page and a Slack channel with the results.

## Workflow

1. **ACTIONS**: Github Workflow will be triggered on each release.
2. **GKE Cluster Spin-up**: ACTIONS will spinup a GKE cluster using terraform main.tf file.
3. **CLI-Tool Run**: Will use the docker image of the cli tool to run a job on the cluster and retrieve the output.
4. **Update Wiki and Slack**: Update the new benchmark on Wiki and SLack.