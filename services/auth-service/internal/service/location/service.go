package location

import (
	"net/http"
	"shared/pkg/logger"
	"time"
)

type LocationService struct {
	Endpoint string
	client   *http.Client
	log      logger.Logger
}

func NewLocationService(endpoint string, log logger.Logger) *LocationService {
	return &LocationService{
		Endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		log: log,
	}
}

var _ LocationServiceInterface = (*LocationService)(nil)
