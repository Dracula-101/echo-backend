package discovery

type ServiceInfo struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Port    int               `json:"port"`
	Health  string            `json:"health"`
	Tags    map[string]string `json:"tags,omitempty"`
}

type RouteInfo struct {
	Prefix      string `json:"prefix"`
	ServiceName string `json:"service_name"`
	StripPrefix bool   `json:"strip_prefix"`
}
