package discovery

import "fmt"

type StaticDiscovery struct {
	services map[string]string
}

func NewStaticDiscovery() *StaticDiscovery {
	return &StaticDiscovery{
		services: map[string]string{
			"/api/v1/auth":          "http://localhost:8081",
			"/api/v1/users":         "http://localhost:8082",
			"/api/v1/messages":      "http://localhost:9000",
			"/api/v1/media":         "http://localhost:8083",
			"/api/v1/notifications": "http://localhost:9001",
			"/api/v1/presence":      "http://localhost:9002",
		},
	}
}

func (s *StaticDiscovery) GetServices() (map[string]string, error) {
	result := make(map[string]string)
	for prefix, address := range s.services {
		result[prefix] = address
	}
	return result, nil
}

func (s *StaticDiscovery) GetServiceAddress(prefix string) (string, error) {
	if address, exists := s.services[prefix]; exists {
		return address, nil
	}
	return "", fmt.Errorf("service not found for prefix: %s", prefix)
}
