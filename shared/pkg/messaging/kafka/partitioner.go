package kafka

import (
	"hash/fnv"

	"github.com/IBM/sarama"
)

type customPartitioner struct{}

func NewCustomPartitioner(topic string) sarama.Partitioner {
	return &customPartitioner{}
}

func (p *customPartitioner) Partition(message *sarama.ProducerMessage, numPartitions int32) (int32, error) {
	if message.Key == nil {
		return 0, nil
	}

	key, err := message.Key.Encode()
	if err != nil {
		return 0, err
	}

	h := fnv.New32a()
	h.Write(key)
	return int32(h.Sum32()) % numPartitions, nil
}

func (p *customPartitioner) RequiresConsistency() bool {
	return true
}
