# Cloudflare Module - DNS and CDN Configuration

resource "cloudflare_record" "api" {
  zone_id = var.zone_id
  name    = var.api_subdomain
  value   = var.load_balancer_ip
  type    = "A"
  ttl     = 1
  proxied = true
}

resource "cloudflare_record" "ws" {
  zone_id = var.zone_id
  name    = var.ws_subdomain
  value   = var.load_balancer_ip
  type    = "A"
  ttl     = 1
  proxied = true
}

resource "cloudflare_page_rule" "api_cache" {
  zone_id  = var.zone_id
  target   = "${var.api_subdomain}.${var.domain_name}/*"
  priority = 1

  actions {
    cache_level = "bypass"
    ssl         = "strict"
  }
}

resource "cloudflare_page_rule" "ws_cache" {
  zone_id  = var.zone_id
  target   = "${var.ws_subdomain}.${var.domain_name}/*"
  priority = 2

  actions {
    cache_level = "bypass"
    ssl         = "strict"
  }
}

resource "cloudflare_rate_limit" "api_rate_limit" {
  zone_id   = var.zone_id
  threshold = 100
  period    = 60
  match {
    request {
      url_pattern = "${var.api_subdomain}.${var.domain_name}/*"
    }
  }
  action {
    mode    = "simulate"
    timeout = 60
  }
}

resource "cloudflare_firewall_rule" "block_bad_bots" {
  zone_id     = var.zone_id
  description = "Block bad bots"
  filter_id   = cloudflare_filter.bad_bots.id
  action      = "block"
}

resource "cloudflare_filter" "bad_bots" {
  zone_id     = var.zone_id
  description = "Bad bots filter"
  expression  = "(cf.client.bot) and not (cf.verified_bot_category in {\"Search Engine Crawler\" \"Page Preview\" \"Monitoring & Analytics\"})"
}

# Outputs
output "dns_records" {
  value = {
    api_hostname = cloudflare_record.api.hostname
    ws_hostname  = cloudflare_record.ws.hostname
  }
  description = "DNS records"
}
