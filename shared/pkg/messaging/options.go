package messaging

import "time"

type ProducerOption func(*ProducerConfig)

func WithProducerBrokers(brokers []string) ProducerOption {
	return func(c *ProducerConfig) { c.Brokers = brokers }
}

func WithProducerClientID(clientID string) ProducerOption {
	return func(c *ProducerConfig) { c.ClientID = clientID }
}

func WithProducerMaxRetries(n int) ProducerOption {
	return func(c *ProducerConfig) { c.MaxRetries = n }
}

func WithProducerRetryBackoff(d time.Duration) ProducerOption {
	return func(c *ProducerConfig) { c.RetryBackoff = d }
}

func WithCompression(codec string) ProducerOption {
	return func(c *ProducerConfig) { c.Compression = codec }
}

func WithAcks(acks string) ProducerOption {
	return func(c *ProducerConfig) { c.Acks = acks }
}

func WithIdempotence(enabled bool) ProducerOption {
	return func(c *ProducerConfig) { c.EnableIdempotence = enabled }
}

type ConsumerOption func(*ConsumerConfig)

func WithConsumerBrokers(brokers []string) ConsumerOption {
	return func(c *ConsumerConfig) { c.Brokers = brokers }
}

func WithConsumerClientID(clientID string) ConsumerOption {
	return func(c *ConsumerConfig) { c.ClientID = clientID }
}

func WithGroupID(groupID string) ConsumerOption {
	return func(c *ConsumerConfig) { c.GroupID = groupID }
}

func WithSessionTimeout(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) { c.SessionTimeout = d }
}

func WithHeartbeatInterval(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) { c.HeartbeatInterval = d }
}
