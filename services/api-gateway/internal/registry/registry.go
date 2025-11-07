package registry

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"
)

var (
	// ErrServiceNotFound is returned when a service is not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrNoHealthyInstances is returned when no healthy instances are available
	ErrNoHealthyInstances = errors.New("no healthy instances available")
)

// ServiceInstance represents a single instance of a service
type ServiceInstance struct {
	ID       string
	Name     string
	BaseURL  string
	Healthy  bool
	Metadata map[string]string
	LastSeen time.Time
}

// Service represents a registered service with multiple instances
type Service struct {
	Name      string
	Instances map[string]*ServiceInstance
	Strategy  LoadBalancingStrategy
	mu        sync.RWMutex
}

// LoadBalancingStrategy defines how to select an instance
type LoadBalancingStrategy string

const (
	// StrategyRoundRobin cycles through instances
	StrategyRoundRobin LoadBalancingStrategy = "round_robin"

	// StrategyRandom selects a random instance
	StrategyRandom LoadBalancingStrategy = "random"

	// StrategyLeastConnections selects instance with least connections
	StrategyLeastConnections LoadBalancingStrategy = "least_connections"
)

// Registry manages service registration and discovery
type Registry struct {
	services map[string]*Service
	mu       sync.RWMutex
	counters map[string]int
}

// New creates a new service registry
func New() *Registry {
	return &Registry{
		services: make(map[string]*Service),
		counters: make(map[string]int),
	}
}

// Register registers a service instance
func (r *Registry) Register(instance ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exists := r.services[instance.Name]
	if !exists {
		service = &Service{
			Name:      instance.Name,
			Instances: make(map[string]*ServiceInstance),
			Strategy:  StrategyRoundRobin,
		}
		r.services[instance.Name] = service
	}

	instance.Healthy = true
	instance.LastSeen = time.Now()
	service.Instances[instance.ID] = &instance

	return nil
}

// Deregister removes a service instance
func (r *Registry) Deregister(serviceName, instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exists := r.services[serviceName]
	if !exists {
		return ErrServiceNotFound
	}

	delete(service.Instances, instanceID)

	// Clean up empty services
	if len(service.Instances) == 0 {
		delete(r.services, serviceName)
	}

	return nil
}

// GetInstance returns a healthy instance using the configured strategy
func (r *Registry) GetInstance(serviceName string) (*ServiceInstance, error) {
	r.mu.RLock()
	service, exists := r.services[serviceName]
	r.mu.RUnlock()

	if !exists {
		return nil, ErrServiceNotFound
	}

	return r.selectInstance(service)
}

// GetAllInstances returns all instances of a service
func (r *Registry) GetAllInstances(serviceName string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[serviceName]
	if !exists {
		return nil, ErrServiceNotFound
	}

	service.mu.RLock()
	defer service.mu.RUnlock()

	instances := make([]*ServiceInstance, 0, len(service.Instances))
	for _, instance := range service.Instances {
		instances = append(instances, instance)
	}

	return instances, nil
}

// GetHealthyInstances returns only healthy instances
func (r *Registry) GetHealthyInstances(serviceName string) ([]*ServiceInstance, error) {
	instances, err := r.GetAllInstances(serviceName)
	if err != nil {
		return nil, err
	}

	healthy := make([]*ServiceInstance, 0)
	for _, instance := range instances {
		if instance.Healthy {
			healthy = append(healthy, instance)
		}
	}

	if len(healthy) == 0 {
		return nil, ErrNoHealthyInstances
	}

	return healthy, nil
}

// SetStrategy sets the load balancing strategy for a service
func (r *Registry) SetStrategy(serviceName string, strategy LoadBalancingStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exists := r.services[serviceName]
	if !exists {
		return ErrServiceNotFound
	}

	service.Strategy = strategy
	return nil
}

// MarkUnhealthy marks an instance as unhealthy
func (r *Registry) MarkUnhealthy(serviceName, instanceID string) error {
	r.mu.RLock()
	service, exists := r.services[serviceName]
	r.mu.RUnlock()

	if !exists {
		return ErrServiceNotFound
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	instance, exists := service.Instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	instance.Healthy = false
	return nil
}

// MarkHealthy marks an instance as healthy
func (r *Registry) MarkHealthy(serviceName, instanceID string) error {
	r.mu.RLock()
	service, exists := r.services[serviceName]
	r.mu.RUnlock()

	if !exists {
		return ErrServiceNotFound
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	instance, exists := service.Instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	instance.Healthy = true
	instance.LastSeen = time.Now()
	return nil
}

// GetServices returns all registered service names
func (r *Registry) GetServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}

	return services
}

func (r *Registry) selectInstance(service *Service) (*ServiceInstance, error) {
	service.mu.RLock()
	instances := make([]*ServiceInstance, 0)
	for _, instance := range service.Instances {
		if instance.Healthy {
			instances = append(instances, instance)
		}
	}
	service.mu.RUnlock()

	if len(instances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	switch service.Strategy {
	case StrategyRandom:
		return instances[rand.Intn(len(instances))], nil

	case StrategyRoundRobin:
		r.mu.Lock()
		counter := r.counters[service.Name]
		r.counters[service.Name] = (counter + 1) % len(instances)
		r.mu.Unlock()
		return instances[counter%len(instances)], nil

	default:
		return instances[0], nil
	}
}

// HealthChecker performs periodic health checks on service instances
type HealthChecker struct {
	registry    *Registry
	interval    time.Duration
	timeout     time.Duration
	httpClient  HTTPClient
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// HTTPClient interface for health checks
type HTTPClient interface {
	Get(url string) (Response, error)
}

// Response interface for HTTP responses
type Response interface {
	StatusCode() int
	Close() error
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(registry *Registry, interval time.Duration, httpClient HTTPClient) *HealthChecker {
	return &HealthChecker{
		registry:    registry,
		interval:    interval,
		timeout:     5 * time.Second,
		httpClient:  httpClient,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Start begins periodic health checking
func (hc *HealthChecker) Start() {
	go hc.run()
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
	<-hc.stoppedChan
}

func (hc *HealthChecker) run() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()
	defer close(hc.stoppedChan)

	for {
		select {
		case <-ticker.C:
			hc.checkAll()
		case <-hc.stopChan:
			return
		}
	}
}

func (hc *HealthChecker) checkAll() {
	services := hc.registry.GetServices()

	for _, serviceName := range services {
		instances, err := hc.registry.GetAllInstances(serviceName)
		if err != nil {
			continue
		}

		for _, instance := range instances {
			hc.checkInstance(serviceName, instance)
		}
	}
}

func (hc *HealthChecker) checkInstance(serviceName string, instance *ServiceInstance) {
	healthURL, err := url.JoinPath(instance.BaseURL, "/health")
	if err != nil {
		hc.registry.MarkUnhealthy(serviceName, instance.ID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		resp, err := hc.httpClient.Get(healthURL)
		if err != nil {
			done <- false
			return
		}
		defer resp.Close()

		done <- resp.StatusCode() >= 200 && resp.StatusCode() < 300
	}()

	select {
	case healthy := <-done:
		if healthy {
			hc.registry.MarkHealthy(serviceName, instance.ID)
		} else {
			hc.registry.MarkUnhealthy(serviceName, instance.ID)
		}
	case <-ctx.Done():
		hc.registry.MarkUnhealthy(serviceName, instance.ID)
	}
}

// ServiceDiscovery provides a higher-level interface for service discovery
type ServiceDiscovery struct {
	registry *Registry
}

// NewServiceDiscovery creates a new service discovery client
func NewServiceDiscovery(registry *Registry) *ServiceDiscovery {
	return &ServiceDiscovery{
		registry: registry,
	}
}

// Discover finds a service instance and returns its base URL
func (sd *ServiceDiscovery) Discover(serviceName string) (string, error) {
	instance, err := sd.registry.GetInstance(serviceName)
	if err != nil {
		return "", err
	}

	return instance.BaseURL, nil
}

// DiscoverWithPath finds a service and constructs the full URL
func (sd *ServiceDiscovery) DiscoverWithPath(serviceName, path string) (string, error) {
	baseURL, err := sd.Discover(serviceName)
	if err != nil {
		return "", err
	}

	fullURL, err := url.JoinPath(baseURL, path)
	if err != nil {
		return "", fmt.Errorf("failed to construct URL: %w", err)
	}

	return fullURL, nil
}

// GetServiceHealth returns health status for all instances of a service
func (sd *ServiceDiscovery) GetServiceHealth(serviceName string) (map[string]bool, error) {
	instances, err := sd.registry.GetAllInstances(serviceName)
	if err != nil {
		return nil, err
	}

	health := make(map[string]bool)
	for _, instance := range instances {
		health[instance.ID] = instance.Healthy
	}

	return health, nil
}
