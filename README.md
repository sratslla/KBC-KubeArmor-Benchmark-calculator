# GKE Cluster Automation with Terraform and GitHub Actions

This project automates the creation of a Google Kubernetes Engine (GKE) cluster using Terraform and GitHub Actions. It runs a CLI tool on the GKE cluster, captures the output, and updates a GitHub Wiki page and a Slack channel with the results.

## Prerequisites

1. **Google Cloud Platform (GCP) Project**: Ensure you have a GCP project.
2. **Service Account Key**: Create a service account key with the necessary permissions.
3. **GitHub Repository**: Ensure you have a GitHub repository with GitHub Actions enabled.
4. **Docker Image for CLI Tool**: Create and push a Docker image of the CLI tool to a container registry.
5. **Slack App**: Create a Slack app with a bot user and obtain the bot token.