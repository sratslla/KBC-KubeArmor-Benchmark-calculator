# SPDX-License-Identifier: Apache-2.0

variable "project" {
  type        = string
  description = "GCP project id"
}

variable "region" {
  type        = string
  description = "GCP region"
  default     = "us-central1"
}

variable "zone" {
  type        = string
  description = "GCP zone"
  default     = "us-central1-c"
}

variable "cluster_name" {
  type        = string
  description = "GKE cluster name"
  default     = "kbc-benchmark"
}

provider "google" {
  project = var.project
  region  = var.region
}

resource "random_id" "suffix" {
  byte_length = 4
}

resource "google_container_cluster" "primary" {
  name                     = var.cluster_name
  location                 = var.zone
  initial_node_count       = 1
  remove_default_node_pool = true
}

resource "google_container_node_pool" "primary_nodes" {
  cluster    = google_container_cluster.primary.name
  location   = google_container_cluster.primary.location
  node_count = 3

  node_config {
    machine_type = "e2-custom-2-4096"
    disk_size_gb = 40
  }
}

resource "google_container_node_pool" "tainted_node" {
  cluster    = google_container_cluster.primary.name
  location   = google_container_cluster.primary.location
  node_count = 1

  node_config {
    machine_type = "e2-standard-4"
    disk_size_gb = 40
    taint {
      key    = "color"
      value  = "blue"
      effect = "NO_SCHEDULE"
    }
    labels = {
      nodetype = "node1"
    }
  }
}

resource "google_compute_firewall" "allow_node_port" {
  name    = "kbc-node-port-${random_id.suffix.hex}"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["30000", "30001"]
  }

  source_ranges = ["0.0.0.0/0"]

  depends_on = [google_container_cluster.primary]
}

output "kubernetes_cluster_name" {
  value = google_container_cluster.primary.name
}

output "kubernetes_cluster_zone" {
  value = google_container_cluster.primary.location
}
