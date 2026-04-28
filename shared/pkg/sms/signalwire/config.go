package signalwire

type Config struct {
	ProjectID  string // SignalWire Project ID
	APIToken   string // SignalWire API Token
	SpaceURL   string // e.g. "yourspace.signalwire.com"
	FromNumber string // E.164 format e.g. "+12345678900"
}
