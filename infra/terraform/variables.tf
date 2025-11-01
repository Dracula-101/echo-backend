variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment (dev, staging, prod)"
  type        = string
  default     = "prod"
}

# GKE Variables
variable "gke_cluster_name" {
  description = "GKE cluster name"
  type        = string
  default     = "echo-backend-cluster"
}

variable "gke_node_count" {
  description = "Number of GKE nodes"
  type        = number
  default     = 3
}

variable "gke_machine_type" {
  description = "GKE node machine type"
  type        = string
  default     = "e2-standard-4"
}

variable "network_name" {
  description = "VPC network name"
  type        = string
  default     = "echo-backend-network"
}

# Database Variables
variable "postgres_instance_name" {
  description = "Cloud SQL instance name"
  type        = string
  default     = "echo-postgres"
}

variable "postgres_tier" {
  description = "Cloud SQL tier"
  type        = string
  default     = "db-custom-4-16384"
}

variable "postgres_disk_size" {
  description = "Cloud SQL disk size in GB"
  type        = number
  default     = 100
}

variable "postgres_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "POSTGRES_15"
}

variable "redis_tier" {
  description = "Redis tier"
  type        = string
  default     = "STANDARD_HA"
}

variable "redis_memory_size" {
  description = "Redis memory size in GB"
  type        = number
  default     = 5
}

# Cloudflare Variables
variable "cloudflare_api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "domain_name" {
  description = "Domain name"
  type        = string
  default     = "echo.com"
}

variable "api_subdomain" {
  description = "API subdomain"
  type        = string
  default     = "api"
}

variable "ws_subdomain" {
  description = "WebSocket subdomain"
  type        = string
  default     = "ws"
}
