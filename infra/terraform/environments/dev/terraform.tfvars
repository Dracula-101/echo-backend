# Development Environment Configuration

project_id       = "echo-backend-dev"
region           = "us-central1"
environment      = "dev"

# GKE Configuration
gke_cluster_name = "echo-backend-dev-cluster"
gke_node_count   = 2
gke_machine_type = "e2-standard-2"
network_name     = "echo-dev-network"

# Database Configuration
postgres_instance_name = "echo-postgres-dev"
postgres_tier          = "db-custom-2-8192" # 2 vCPU, 8GB RAM
postgres_disk_size     = 50
postgres_version       = "POSTGRES_15"

redis_tier        = "BASIC"
redis_memory_size = 2

# Cloudflare Configuration
domain_name    = "echo-dev.com"
api_subdomain  = "api"
ws_subdomain   = "ws"
