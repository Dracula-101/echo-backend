package shutdown

import "shared/pkg/logger"

type shutdownLogger struct {
	log logger.Logger
}

func (s *shutdownLogger) Info(msg string, keysAndValues ...interface{}) {
	fields := make([]logger.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, logger.Any(key, keysAndValues[i+1]))
	}
	s.log.Info(msg, fields...)
}

func (s *shutdownLogger) Error(msg string, err error, keysAndValues ...interface{}) {
	fields := make([]logger.Field, 0, len(keysAndValues)/2+1)
	fields = append(fields, logger.Error(err))
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, logger.Any(key, keysAndValues[i+1]))
	}
	s.log.Error(msg, fields...)
}
