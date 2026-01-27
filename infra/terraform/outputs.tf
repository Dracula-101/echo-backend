# =====================================================
# TERRAFORM OUTPUTS
# =====================================================

# GKE Cluster Outputs
output "gke_cluster_endpoint" {
  value       = module.gcp.cluster_endpoint
  description = "GKE cluster endpoint"
  sensitive   = true
}

output "gke_cluster_ca_certificate" {
  value       = module.gcp.cluster_ca_certificate
  description = "GKE cluster CA certificate"
  sensitive   = true
}

output "gke_cluster_name" {
  value       = var.gke_cluster_name
  description = "GKE cluster name"
}

# Network Outputs
output "vpc_network_name" {
  value       = module.gcp.network_name
  description = "VPC network name"
}

output "vpc_network_id" {
  value       = module.gcp.network_id
  description = "VPC network ID"
}

output "load_balancer_ip" {
  value       = module.gcp.load_balancer_ip
  description = "Load balancer IP address"
}

# Database Outputs
output "postgres_connection_name" {
  value       = module.database.postgres_connection_name
  description = "Cloud SQL instance connection name"
}

output "postgres_private_ip" {
  value       = module.database.postgres_private_ip
  description = "Cloud SQL private IP address"
}

output "postgres_password" {
  value       = module.database.postgres_password
  description = "PostgreSQL database password"
  sensitive   = true
}

output "redis_host" {
  value       = module.database.redis_host
  description = "Redis instance host"
}

output "redis_port" {
  value       = module.database.redis_port
  description = "Redis instance port"
}

# Cloudflare Outputs (only if enabled)
output "cloudflare_dns_records" {
  value       = var.enable_cloudflare ? module.cloudflare[0].dns_records : null
  description = "Cloudflare DNS records (null if Cloudflare is disabled)"
}

output "api_hostname" {
  value       = var.enable_cloudflare ? "${var.api_subdomain}.${var.domain_name}" : "Configure DNS manually to point to: ${module.gcp.load_balancer_ip}"
  description = "API hostname or manual DNS instruction"
}

output "ws_hostname" {
  value       = var.enable_cloudflare ? "${var.ws_subdomain}.${var.domain_name}" : "Configure DNS manually to point to: ${module.gcp.load_balancer_ip}"
  description = "WebSocket hostname or manual DNS instruction"
}

# Useful Commands
output "gcloud_get_credentials_command" {
  value       = "gcloud container clusters get-credentials ${var.gke_cluster_name} --region ${var.region} --project ${var.project_id}"
  description = "Command to get GKE credentials"
}

output "kubectl_config_command" {
  value       = "kubectl config use-context gke_${var.project_id}_${var.region}_${var.gke_cluster_name}"
  description = "Command to set kubectl context"
}
