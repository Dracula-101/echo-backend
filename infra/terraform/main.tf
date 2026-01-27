terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  backend "gcs" {
    bucket = "echo-terraform-state"
    prefix = "terraform/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "cloudflare" {
  api_token = var.enable_cloudflare ? var.cloudflare_api_token : null
}

# GCP Infrastructure
module "gcp" {
  source = "./modules/gcp"
  
  project_id          = var.project_id
  region              = var.region
  environment         = var.environment
  gke_cluster_name    = var.gke_cluster_name
  gke_node_count      = var.gke_node_count
  gke_machine_type    = var.gke_machine_type
  network_name        = var.network_name
}

# Database Infrastructure
module "database" {
  source = "./modules/database"
  
  project_id              = var.project_id
  region                  = var.region
  environment             = var.environment
  postgres_instance_name  = var.postgres_instance_name
  postgres_tier           = var.postgres_tier
  postgres_disk_size      = var.postgres_disk_size
  postgres_version        = var.postgres_version
  redis_tier              = var.redis_tier
  redis_memory_size       = var.redis_memory_size
  network_id              = module.gcp.network_id
  private_vpc_connection  = module.gcp.private_vpc_connection
}

# Cloudflare Configuration (Optional)
module "cloudflare" {
  count  = var.enable_cloudflare ? 1 : 0
  source = "./modules/cloudflare"

  zone_id          = var.cloudflare_zone_id
  domain_name      = var.domain_name
  api_subdomain    = var.api_subdomain
  ws_subdomain     = var.ws_subdomain
  load_balancer_ip = module.gcp.load_balancer_ip
}
