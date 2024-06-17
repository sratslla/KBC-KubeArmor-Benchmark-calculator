provider "google" {
  project = "kbc-1-426708"
  region  = "us-central1"
}

resource "google_container_cluster" "primary" {
  name               = "example-cluster"
  location           = "us-central1-c"
  initial_node_count        = 1
  remove_default_node_pool  = true
}

resource "google_container_node_pool" "primary_nodes" {
  cluster    = google_container_cluster.primary.name
  location   = google_container_cluster.primary.location
  node_count = 3

  node_config {
    machine_type = "e2-custom-2-4096"
  }
}

resource "google_container_node_pool" "tainted_node" {
  cluster    = google_container_cluster.primary.name
  location   = google_container_cluster.primary.location
  node_count = 1

  node_config {
    machine_type = "e2-standard-4"
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

resource "google_compute_firewall" "allow_node_port"{
  name    = "test-node-port"
  network = google_container_cluster.primary.network

  allow {
    protocol = "tcp"
    ports    = ["30000", "30001"]
  }

  source_ranges = ["0.0.0.0/0"]
}

output "kubernetes_cluster_name" {
  value = google_container_cluster.primary.name
}
