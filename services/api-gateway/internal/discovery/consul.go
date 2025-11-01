package discovery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ConsulClient struct {
	address string
	client  *http.Client
}

func NewConsulClient(address string) *ConsulClient {
	return &ConsulClient{
		address: address,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *ConsulClient) GetServices() (map[string]string, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/v1/catalog/services", c.address))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var services map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for name := range services {
		if strings.HasSuffix(name, "-service") {
			prefix := fmt.Sprintf("/api/v1/%s", strings.TrimSuffix(name, "-service"))
			result[prefix] = name
		}
	}
	
	return result, nil
}

func (c *ConsulClient) GetServiceAddress(serviceName string) (string, error) {
	url := fmt.Sprintf("%s/v1/health/service/%s?passing=true", c.address, serviceName)
	resp, err := c.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var health []struct {
		Service struct {
			Address string `json:"Address"`
			Port    int    `json:"Port"`
		} `json:"Service"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return "", err
	}

	if len(health) == 0 {
		return "", fmt.Errorf("no healthy instances of %s", serviceName)
	}

	service := health[0].Service
	return fmt.Sprintf("http://%s:%d", service.Address, service.Port), nil
}
