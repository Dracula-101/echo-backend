package proxy

import (
	"bytes"
	"context"
	"echo-backend/services/api-gateway/internal/config"
	gwErrors "echo-backend/services/api-gateway/internal/errors"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"shared/pkg/logger"
	contextx "shared/server/context"
	"shared/server/response"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ============================================================================
// HTTP Headers Configuration
// ============================================================================

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

// ============================================================================
// Helper Functions
// ============================================================================

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

// ============================================================================
// Proxy Manager
// ============================================================================

type Manager struct {
	config   *config.Config
	logger   logger.Logger
	services map[string]config.ServiceConfig
	proxies  map[string]*httputil.ReverseProxy
}

func NewManager(cfg *config.Config, log logger.Logger) (*Manager, error) {
	if cfg == nil {
		panic("Config is required for ProxyManager")
	}
	if log == nil {
		panic("Logger is required for ProxyManager")
	}

	log.Info("Initializing Proxy Manager",
		logger.String("service", gwErrors.ServiceName),
		logger.Int("service_count", len(cfg.Services)),
	)

	m := &Manager{
		config:   cfg,
		logger:   log,
		proxies:  make(map[string]*httputil.ReverseProxy),
		services: cfg.Services,
	}

	for name, svc := range cfg.Services {
		log.Debug("Processing service configuration",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", name),
			logger.Int("address_count", len(svc.Addresses)),
			logger.String("protocol", svc.Protocol),
		)

		if len(svc.Addresses) == 0 {
			log.Warn("Service has no addresses configured, skipping",
				logger.String("service", gwErrors.ServiceName),
				logger.String("target_service", name),
			)
			continue
		}

		addr := svc.Addresses[0]
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			if svc.Protocol == "grpc" || svc.Protocol == "grpcs" {
				log.Debug("Skipping gRPC service",
					logger.String("service", gwErrors.ServiceName),
					logger.String("target_service", name),
					logger.String("protocol", svc.Protocol),
				)
				continue
			}
			addr = "http://" + addr
		}

		target, err := url.Parse(addr)
		if err != nil {
			log.Error("Failed to parse service address",
				logger.String("service", gwErrors.ServiceName),
				logger.String("target_service", name),
				logger.String("address", addr),
				logger.String("error_code", gwErrors.CodeConfigurationError),
				logger.Error(err),
			)
			continue
		}

		proxy := newSingleHostReverseProxy(target, name, mlog{log})
		m.proxies[name] = proxy
		log.Info("Proxy initialized successfully",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", name),
			logger.String("target_url", target.String()),
		)
	}

	log.Info("Proxy Manager initialized",
		logger.String("service", gwErrors.ServiceName),
		logger.Int("active_proxies", len(m.proxies)),
	)

	return m, nil
}

// ============================================================================
// Reverse Proxy Configuration
// ============================================================================

type mlog struct {
	logger.Logger
}

func newSingleHostReverseProxy(target *url.URL, serviceName string, l mlog) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		l.Debug("Directing request to upstream service",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.String("path", req.URL.Path),
			logger.String("method", req.Method),
		)

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
		l.Debug("Received response from upstream service",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.Int("status", resp.StatusCode),
			logger.String("content_type", resp.Header.Get("Content-Type")),
		)

		// Only modify JSON responses that might contain our response structure
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return nil
		}

		// Get the start time from the request context
		startTime, ok := resp.Request.Context().Value(contextx.StartTimeKey).(time.Time)
		if !ok {
			// No start time in context, can't recalculate duration
			return nil
		}

		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			l.Error("Failed to read response body for duration correction",
				logger.String("service", gwErrors.ServiceName),
				logger.String("target_service", serviceName),
				logger.Error(err),
			)
			return err
		}
		resp.Body.Close()

		// Try to parse as our standard response structure
		var apiResponse struct {
			Success  bool                   `json:"success"`
			Message  string                 `json:"message,omitempty"`
			Data     interface{}            `json:"data,omitempty"`
			Error    interface{}            `json:"error,omitempty"`
			Metadata map[string]interface{} `json:"metadata,omitempty"`
		}

		if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
			// Not our response format or invalid JSON, return as-is
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			return nil
		}

		// Check if metadata exists
		if apiResponse.Metadata == nil {
			// No metadata to update
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			return nil
		}

		// Calculate actual duration from gateway perspective
		actualDuration := time.Since(startTime)

		// Update duration fields in metadata
		apiResponse.Metadata["duration"] = actualDuration.String()
		apiResponse.Metadata["duration_ms"] = float64(actualDuration.Microseconds()) / 1000.0

		// Re-marshal the response
		modifiedBody, err := json.Marshal(apiResponse)
		if err != nil {
			l.Error("Failed to marshal modified response",
				logger.String("service", gwErrors.ServiceName),
				logger.String("target_service", serviceName),
				logger.Error(err),
			)
			// Return original body on error
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			return nil
		}

		// Update response body and content length
		resp.Body = io.NopCloser(bytes.NewReader(modifiedBody))
		resp.ContentLength = int64(len(modifiedBody))
		resp.Header.Set("Content-Length", strconv.Itoa(len(modifiedBody)))

		l.Debug("Updated response duration",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.String("duration", actualDuration.String()),
			logger.Float64("duration_ms", float64(actualDuration.Microseconds())/1000.0),
		)

		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		l.Error("Proxy error occurred",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.String("path", r.URL.Path),
			logger.String("method", r.Method),
			logger.String("error_code", gwErrors.CodeProxyError),
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
				logger.String("service", gwErrors.ServiceName),
				logger.String("target_service", serviceName),
				logger.String("error_code", gwErrors.CodeServiceNotFound),
			)
			response.ServiceUnavailableError(r.Context(), r, w, serviceName, 30)
		}
	}

	serviceConfig := m.services[serviceName]

	return func(w http.ResponseWriter, r *http.Request) {
		m.logger.Info("Proxy request received",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.String("method", r.Method),
			logger.String("path", r.URL.Path),
		)

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

		m.logger.Debug("Forwarding request to upstream service",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.String("method", r.Method),
			logger.String("original_path", originalPath),
			logger.String("forwarded_path", r.URL.Path),
			logger.String("remaining_path", remainingPath),
			logger.Bool("transform", transform),
			logger.String("query", r.URL.RawQuery),
			logger.Bool("has_auth", r.Header.Get("Authorization") != ""),
			logger.String("content_type", r.Header.Get("Content-Type")),
		)

		if serviceConfig.Timeout > 0 {
			m.logger.Debug("Setting request timeout",
				logger.String("service", gwErrors.ServiceName),
				logger.String("target_service", serviceName),
				logger.Duration("timeout", serviceConfig.Timeout),
			)
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

		m.logger.Debug("Client IP extracted",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.String("client_ip", clientIP),
		)

		req.URL.Path = r.URL.Path
		req.URL.RawQuery = r.URL.RawQuery

		proxy.ServeHTTP(w, req)

		m.logger.Info("Proxy request completed",
			logger.String("service", gwErrors.ServiceName),
			logger.String("target_service", serviceName),
			logger.String("method", r.Method),
			logger.String("path", originalPath),
		)
	}
}
