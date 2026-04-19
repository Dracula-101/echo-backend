package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shared/pkg/cache"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/middleware"
	"shared/server/request"
	"time"
	"user-service/internal/service/location"
)

var (
	LocationTTL = 24 * time.Hour
)

func LocationFromIP(locationService location.LocationService, memoryCache cache.Cache, log logger.Logger) middleware.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ip := utils.RealIP(r)

			if utils.IsPrivateIP(ip) {
				next.ServeHTTP(w, r)
				return
			}

			cacheKey := "geo:" + ip

			if cached, err := memoryCache.Get(ctx, cacheKey); err == nil {
				var location request.IpAddressInfo
				if json.Unmarshal(cached, &location) == nil {
					log.Info(fmt.Sprintf("Location info for IP %s found in cache", ip),
						logger.String("ip", ip),
						logger.Any("location", location),
					)
					next.ServeHTTP(w, r.WithContext(request.WithIPAddressInfo(ctx, location)))
					return
				}
			}
			location, _ := locationService.Lookup(ip)
			log.Info(fmt.Sprintf("Location info for IP %s retrieved from service", ip),
				logger.String("ip", ip),
				logger.Any("location", location),
			)
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
