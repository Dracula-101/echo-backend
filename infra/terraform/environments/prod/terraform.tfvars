# Production Environment Configuration

project_id       = "echo-backend-prod"
region           = "us-central1"
environment      = "prod"

# GKE Configuration
gke_cluster_name = "echo-backend-prod-cluster"
gke_node_count   = 5
gke_machine_type = "e2-standard-4"
network_name     = "echo-prod-network"

# Database Configuration
postgres_instance_name = "echo-postgres-prod"
postgres_tier          = "db-custom-8-32768" # 8 vCPU, 32GB RAM
postgres_disk_size     = 500
postgres_version       = "POSTGRES_15"

redis_tier        = "STANDARD_HA"
redis_memory_size = 10

# Cloudflare Configuration
domain_name    = "echo.com"
api_subdomain  = "api"
ws_subdomain   = "ws"
