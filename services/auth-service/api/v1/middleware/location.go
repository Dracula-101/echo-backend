package middleware

import (
	"auth-service/internal/service/location"
	"encoding/json"
	"net"
	"net/http"
	"shared/pkg/cache"
	"shared/server/headers"
	"shared/server/middleware"
	"shared/server/request"
	"strings"
	"time"
)

var (
	LocationTTL = 24 * time.Hour
)

func LocationFromIP(locationService location.LocationService, memoryCache cache.Cache) middleware.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ip := realIP(r)

			if isPrivateIP(ip) {
				next.ServeHTTP(w, r)
				return
			}

			cacheKey := "geo:" + ip

			if cached, err := memoryCache.Get(ctx, cacheKey); err == nil {
				var location request.IpAddressInfo
				if json.Unmarshal(cached, &location) == nil {
					next.ServeHTTP(w, r.WithContext(request.WithIPAddressInfo(ctx, location)))
					return
				}
			}
			location, _ := locationService.Lookup(ip)
			if location != nil {
				if data, err := json.Marshal(location); err == nil {
					memoryCache.Set(ctx, cacheKey, data, LocationTTL)
				}
				next.ServeHTTP(w, r.WithContext(request.WithIPAddressInfo(ctx, *location)))
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func realIP(r *http.Request) string {
	if xff := r.Header.Get(headers.XForwardedFor); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xri := r.Header.Get(headers.XRealIP); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() || parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() {
		return true
	}
	privateRanges := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7", "fe80::/10"}
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}
