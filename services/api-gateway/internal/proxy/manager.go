// proxy/manager.go
package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"echo-backend/services/api-gateway/internal/config"
	"shared/pkg/logger"
	"shared/server/response"

	"github.com/gorilla/mux"
)

var hopByHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
	return dst
}

func removeHopByHop(h http.Header) {
	for _, hn := range hopByHopHeaders {
		h.Del(hn)
	}
	if conns, ok := h["Connection"]; ok {
		for _, c := range conns {
			for _, token := range strings.Split(c, ",") {
				token = strings.TrimSpace(token)
				if token != "" {
					h.Del(token)
				}
			}
		}
		h.Del("Connection")
	}
}

type Manager struct {
	config   *config.Config
	logger   logger.Logger
	services map[string]config.ServiceConfig
	proxies  map[string]*httputil.ReverseProxy
}

func NewManager(cfg *config.Config, log logger.Logger) (*Manager, error) {
	m := &Manager{
		config:   cfg,
		logger:   log,
		proxies:  make(map[string]*httputil.ReverseProxy),
		services: cfg.Services,
	}

	for name, svc := range cfg.Services {
		if len(svc.Addresses) == 0 {
			continue
		}
		addr := svc.Addresses[0]
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			if svc.Protocol == "grpc" || svc.Protocol == "grpcs" {
				continue
			}
			addr = "http://" + addr
		}
		target, err := url.Parse(addr)
		if err != nil {
			log.Error("Failed to parse service address",
				logger.String("service", name),
				logger.String("address", addr),
				logger.Error(err),
			)
			continue
		}
		proxy := newSingleHostReverseProxy(target, name, mlog{log})
		m.proxies[name] = proxy
		log.Info("Proxy initialized",
			logger.String("service", name),
			logger.String("target", target.String()),
		)
	}

	return m, nil
}

type mlog struct {
	logger.Logger
}

func newSingleHostReverseProxy(target *url.URL, serviceName string, l mlog) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if req.URL.Scheme == "" {
			req.URL.Scheme = target.Scheme
		}
		if req.URL.Host == "" {
			req.URL.Host = target.Host
		}
		if req.Host == "" {
			req.Host = target.Host
		}
		removeHopByHop(req.Header)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		l.Debug("Received response from service",
			logger.String("service", serviceName),
			logger.Int("status", resp.StatusCode),
			logger.Any("headers", resp.Header),
		)
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		l.Error("Proxy error",
			logger.String("service", serviceName),
			logger.String("path", r.URL.Path),
			logger.String("method", r.Method),
			logger.Error(err),
		)
		response.ServiceUnavailableError(r.Context(), r, w, serviceName, 30)
	}
	return proxy
}

func (m *Manager) ProxyHandler(serviceName string, transform bool) http.HandlerFunc {
	proxy, exists := m.proxies[serviceName]
	if !exists {
		return func(w http.ResponseWriter, r *http.Request) {
			m.logger.Error("Service proxy not found",
				logger.String("service", serviceName),
			)
			response.ServiceUnavailableError(r.Context(), r, w, serviceName, 30)
		}
	}

	serviceConfig := m.services[serviceName]

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		remainingPath := vars["rest"]
		if remainingPath == "" {
			remainingPath = vars["path"]
		}

		originalPath := r.URL.Path

		if transform {
			if remainingPath != "" {
				r.URL.Path = "/" + remainingPath
			} else {
				r.URL.Path = "/"
			}
		}

		if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}

		m.logger.Debug("Forwarding request to service",
			logger.String("service", serviceName),
			logger.String("method", r.Method),
			logger.String("original_path", originalPath),
			logger.String("forwarded_path", r.URL.Path),
			logger.String("remaining_path", remainingPath),
			logger.Bool("transform", transform),
			logger.String("query", r.URL.RawQuery),
			logger.String("authorization", r.Header.Get("Authorization")),
			logger.String("content-type", r.Header.Get("Content-Type")),
		)

		if serviceConfig.Timeout > 0 {
			ctx, cancel := context.WithTimeout(r.Context(), serviceConfig.Timeout)
			defer cancel()
			r = r.WithContext(ctx)
		}

		req := r
		req.Header = cloneHeaders(r.Header)
		removeHopByHop(req.Header)

		clientIP := r.Header.Get("X-Real-IP")
		if clientIP == "" {
			clientIP = r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
					clientIP = host
				} else {
					clientIP = r.RemoteAddr
				}
			}
		}
		if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
			req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			req.Header.Set("X-Forwarded-For", clientIP)
		}

		req.URL.Path = r.URL.Path
		req.URL.RawQuery = r.URL.RawQuery

		proxy.ServeHTTP(w, req)
	}
}
