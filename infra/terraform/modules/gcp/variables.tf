variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "environment" {
  description = "Environment"
  type        = string
}

variable "gke_cluster_name" {
  description = "GKE cluster name"
  type        = string
}

variable "gke_node_count" {
  description = "Number of GKE nodes"
  type        = number
}

variable "gke_machine_type" {
  description = "GKE machine type"
  type        = string
}

variable "network_name" {
  description = "VPC network name"
  type        = string
}
