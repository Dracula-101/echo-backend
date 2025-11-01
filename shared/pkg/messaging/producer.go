package messaging

type ProducerConfig struct {
	Acks              int
	CompressionType   string
	MaxMessageBytes   int
	Retries           int
	RetryBackoff      int
	BatchSize         int
	LingerMs          int
	BufferMemory      int64
	EnableIdempotence bool
}

type SendResult struct {
	Topic     string
	Partition int32
	Offset    int64
	Timestamp int64
	Error     error
}
