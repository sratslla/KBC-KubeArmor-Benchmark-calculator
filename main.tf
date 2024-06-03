provider "google" {
  project = "kbc1-424602"
  region  = "us-central1"
}

resource "google_container_cluster" "primary" {
  name               = "example-cluster"
  location           = "us-central1-c"
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
  }
}

output "kubernetes_cluster_name" {
  value = google_container_cluster.primary.name
}
