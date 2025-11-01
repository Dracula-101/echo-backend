# Database Module - Cloud SQL and Redis

resource "google_sql_database_instance" "postgres" {
  name             = var.postgres_instance_name
  database_version = var.postgres_version
  region           = var.region
  project          = var.project_id

  settings {
    tier              = var.postgres_tier
    disk_size         = var.postgres_disk_size
    disk_type         = "PD_SSD"
    disk_autoresize   = true
    availability_type = "REGIONAL"

    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
      start_time                     = "02:00"
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 30
      }
    }

    ip_configuration {
      ipv4_enabled    = false
      private_network = var.network_id
      require_ssl     = true
    }

    database_flags {
      name  = "max_connections"
      value = "200"
    }

    database_flags {
      name  = "shared_buffers"
      value = "4194304" # 4GB in KB
    }

    insights_config {
      query_insights_enabled  = true
      query_string_length     = 1024
      record_application_tags = true
      record_client_address   = true
    }

    maintenance_window {
      day  = 7 # Sunday
      hour = 3
    }
  }

  deletion_protection = true
}

resource "google_sql_database" "database" {
  name     = "echo_db"
  instance = google_sql_database_instance.postgres.name
  project  = var.project_id
}

resource "google_sql_user" "user" {
  name     = "echo_user"
  instance = google_sql_database_instance.postgres.name
  password = random_password.db_password.result
  project  = var.project_id
}

resource "random_password" "db_password" {
  length  = 32
  special = true
}

# Redis Instance
resource "google_redis_instance" "redis" {
  name               = "${var.environment}-redis"
  tier               = var.redis_tier
  memory_size_gb     = var.redis_memory_size
  region             = var.region
  redis_version      = "REDIS_7_0"
  display_name       = "Echo Backend Redis"
  authorized_network = var.network_id
  project            = var.project_id

  redis_configs = {
    maxmemory-policy = "allkeys-lru"
  }

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 3
        minutes = 0
      }
    }
  }
}

# Outputs
output "postgres_connection_name" {
  value       = google_sql_database_instance.postgres.connection_name
  description = "Connection name for Cloud SQL"
}

output "postgres_private_ip" {
  value       = google_sql_database_instance.postgres.private_ip_address
  description = "Private IP for Cloud SQL"
}

output "postgres_password" {
  value       = random_password.db_password.result
  description = "Database password"
  sensitive   = true
}

output "redis_host" {
  value       = google_redis_instance.redis.host
  description = "Redis host"
}

output "redis_port" {
  value       = google_redis_instance.redis.port
  description = "Redis port"
}
