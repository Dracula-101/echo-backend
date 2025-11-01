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

variable "postgres_instance_name" {
  description = "Cloud SQL instance name"
  type        = string
}

variable "postgres_tier" {
  description = "Cloud SQL tier"
  type        = string
}

variable "postgres_disk_size" {
  description = "Disk size in GB"
  type        = number
}

variable "postgres_version" {
  description = "PostgreSQL version"
  type        = string
}

variable "redis_tier" {
  description = "Redis tier"
  type        = string
}

variable "redis_memory_size" {
  description = "Redis memory size in GB"
  type        = number
}

variable "network_id" {
  description = "VPC network ID"
  type        = string
}
