variable "zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "domain_name" {
  description = "Domain name"
  type        = string
}

variable "api_subdomain" {
  description = "API subdomain"
  type        = string
}

variable "ws_subdomain" {
  description = "WebSocket subdomain"
  type        = string
}

variable "load_balancer_ip" {
  description = "Load balancer IP address"
  type        = string
}
