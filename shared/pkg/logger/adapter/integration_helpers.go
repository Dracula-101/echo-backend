package adapter

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPLogSink sends logs to HTTP endpoints
type HTTPLogSink struct {
	mu            sync.RWMutex
	endpoint      string
	client        *http.Client
	batchSize     int
	buffer        []HTTPLogEntry
	headers       map[string]string
	retryPolicy   *RetryPolicy
	rateLimiter   *HTTPRateLimiter
	transformer   LogTransformer
	authenticator Authenticator
	compression   bool
	timeout       time.Duration
}

// HTTPLogEntry represents a log entry for HTTP transmission
type HTTPLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	Source    string                 `json:"source"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// NewHTTPLogSink creates a new HTTP log sink
func NewHTTPLogSink(endpoint string, batchSize int) *HTTPLogSink {
	return &HTTPLogSink{
		endpoint:  endpoint,
		client:    &http.Client{Timeout: 30 * time.Second},
		batchSize: batchSize,
		buffer:    make([]HTTPLogEntry, 0, batchSize),
		headers:   make(map[string]string),
		retryPolicy: &RetryPolicy{
			MaxRetries:   3,
			InitialDelay: time.Second,
			MaxDelay:     time.Minute,
		},
		rateLimiter: NewHTTPRateLimiter(100, time.Second),
		compression: true,
		timeout:     30 * time.Second,
	}
}

// Send sends a log entry
func (hls *HTTPLogSink) Send(entry HTTPLogEntry) error {
	hls.mu.Lock()
	defer hls.mu.Unlock()

	if hls.transformer != nil {
		transformed, err := hls.transformer.Transform(entry)
		if err != nil {
			return fmt.Errorf("transform error: %w", err)
		}
		entry = transformed
	}

	hls.buffer = append(hls.buffer, entry)

	if len(hls.buffer) >= hls.batchSize {
		return hls.flushLocked()
	}

	return nil
}

// Flush flushes the buffer
func (hls *HTTPLogSink) Flush() error {
	hls.mu.Lock()
	defer hls.mu.Unlock()
	return hls.flushLocked()
}

// flushLocked flushes the buffer (must be called with lock held)
func (hls *HTTPLogSink) flushLocked() error {
	if len(hls.buffer) == 0 {
		return nil
	}

	if !hls.rateLimiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	data, err := hls.preparePayload(hls.buffer)
	if err != nil {
		return fmt.Errorf("prepare payload error: %w", err)
	}

	req, err := http.NewRequest("POST", hls.endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	for key, value := range hls.headers {
		req.Header.Set(key, value)
	}

	if hls.compression {
		req.Header.Set("Content-Encoding", "gzip")
	}
	req.Header.Set("Content-Type", "application/json")

	if hls.authenticator != nil {
		if err := hls.authenticator.Authenticate(req); err != nil {
			return fmt.Errorf("authentication error: %w", err)
		}
	}

	err = hls.sendWithRetry(req)
	if err != nil {
		return err
	}

	hls.buffer = hls.buffer[:0]
	return nil
}

// preparePayload prepares the payload for sending
func (hls *HTTPLogSink) preparePayload(entries []HTTPLogEntry) ([]byte, error) {
	var buf bytes.Buffer

	if hls.compression {
		gw := gzip.NewWriter(&buf)
		defer gw.Close()

		// Write entries (simplified - in production would use proper JSON encoding)
		fmt.Fprintf(gw, "[")
		for i, entry := range entries {
			if i > 0 {
				fmt.Fprintf(gw, ",")
			}
			fmt.Fprintf(gw, "{\"timestamp\":\"%s\",\"level\":\"%s\",\"message\":\"%s\"}",
				entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message)
		}
		fmt.Fprintf(gw, "]")

		gw.Close()
		return buf.Bytes(), nil
	}

	// Uncompressed payload
	fmt.Fprintf(&buf, "[")
	for i, entry := range entries {
		if i > 0 {
			fmt.Fprintf(&buf, ",")
		}
		fmt.Fprintf(&buf, "{\"timestamp\":\"%s\",\"level\":\"%s\",\"message\":\"%s\"}",
			entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message)
	}
	fmt.Fprintf(&buf, "]")

	return buf.Bytes(), nil
}

// sendWithRetry sends with retry logic
func (hls *HTTPLogSink) sendWithRetry(req *http.Request) error {
	var lastErr error
	delay := hls.retryPolicy.InitialDelay

	for attempt := 0; attempt <= hls.retryPolicy.MaxRetries; attempt++ {
		resp, err := hls.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(delay)
			delay *= 2
			if delay > hls.retryPolicy.MaxDelay {
				delay = hls.retryPolicy.MaxDelay
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)

		if resp.StatusCode >= 500 {
			time.Sleep(delay)
			delay *= 2
			if delay > hls.retryPolicy.MaxDelay {
				delay = hls.retryPolicy.MaxDelay
			}
			continue
		}

		return lastErr
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// SetHeader sets a custom header
func (hls *HTTPLogSink) SetHeader(key, value string) {
	hls.mu.Lock()
	defer hls.mu.Unlock()
	hls.headers[key] = value
}

// SetAuthenticator sets the authenticator
func (hls *HTTPLogSink) SetAuthenticator(auth Authenticator) {
	hls.mu.Lock()
	defer hls.mu.Unlock()
	hls.authenticator = auth
}

// SetTransformer sets the log transformer
func (hls *HTTPLogSink) SetTransformer(transformer LogTransformer) {
	hls.mu.Lock()
	defer hls.mu.Unlock()
	hls.transformer = transformer
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	BackoffFunc  func(attempt int, delay time.Duration) time.Duration
}

// HTTPRateLimiter limits HTTP request rate
type HTTPRateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// NewHTTPRateLimiter creates a new HTTP rate limiter
func NewHTTPRateLimiter(maxTokens int, refillRate time.Duration) *HTTPRateLimiter {
	return &HTTPRateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (hrl *HTTPRateLimiter) Allow() bool {
	hrl.mu.Lock()
	defer hrl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(hrl.lastRefill)

	if elapsed >= hrl.refillRate {
		hrl.tokens = hrl.maxTokens
		hrl.lastRefill = now
	}

	if hrl.tokens > 0 {
		hrl.tokens--
		return true
	}

	return false
}

// LogTransformer transforms log entries
type LogTransformer interface {
	Transform(entry HTTPLogEntry) (HTTPLogEntry, error)
}

// Authenticator authenticates requests
type Authenticator interface {
	Authenticate(req *http.Request) error
}

// BasicAuthenticator provides basic authentication
type BasicAuthenticator struct {
	username string
	password string
}

// NewBasicAuthenticator creates a new basic authenticator
func NewBasicAuthenticator(username, password string) *BasicAuthenticator {
	return &BasicAuthenticator{
		username: username,
		password: password,
	}
}

// Authenticate authenticates a request
func (ba *BasicAuthenticator) Authenticate(req *http.Request) error {
	auth := ba.username + ":" + ba.password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encoded)
	return nil
}

// BearerAuthenticator provides bearer token authentication
type BearerAuthenticator struct {
	token string
}

// NewBearerAuthenticator creates a new bearer authenticator
func NewBearerAuthenticator(token string) *BearerAuthenticator {
	return &BearerAuthenticator{token: token}
}

// Authenticate authenticates a request
func (ba *BearerAuthenticator) Authenticate(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+ba.token)
	return nil
}

// FileLogSink writes logs to files
type FileLogSink struct {
	mu            sync.RWMutex
	file          *os.File
	filename      string
	maxSize       int64
	maxBackups    int
	currentSize   int64
	rotationTime  time.Time
	format        FileFormat
	buffer        *bytes.Buffer
	flushInterval time.Duration
	ticker        *time.Ticker
	done          chan bool
}

// FileFormat represents file format
type FileFormat string

const (
	FormatJSONL FileFormat = "jsonl"
	FormatCSV   FileFormat = "csv"
	FormatXML   FileFormat = "xml"
	FormatText  FileFormat = "text"
)

// NewFileLogSink creates a new file log sink
func NewFileLogSink(filename string, maxSize int64, maxBackups int, format FileFormat) (*FileLogSink, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	fls := &FileLogSink{
		file:          file,
		filename:      filename,
		maxSize:       maxSize,
		maxBackups:    maxBackups,
		currentSize:   info.Size(),
		rotationTime:  time.Now(),
		format:        format,
		buffer:        new(bytes.Buffer),
		flushInterval: time.Second * 5,
		ticker:        time.NewTicker(time.Second * 5),
		done:          make(chan bool),
	}

	go fls.autoFlush()
	return fls, nil
}

// Write writes a log entry
func (fls *FileLogSink) Write(entry HTTPLogEntry) error {
	fls.mu.Lock()
	defer fls.mu.Unlock()

	var data []byte
	var err error

	switch fls.format {
	case FormatJSONL:
		data, err = fls.formatJSON(entry)
	case FormatCSV:
		data, err = fls.formatCSV(entry)
	case FormatXML:
		data, err = fls.formatXML(entry)
	default:
		data, err = fls.formatText(entry)
	}

	if err != nil {
		return err
	}

	fls.buffer.Write(data)
	fls.buffer.WriteByte('\n')

	if fls.buffer.Len() > 1024*1024 { // 1MB buffer
		return fls.flushLocked()
	}

	return nil
}

// flushLocked flushes the buffer (must be called with lock held)
func (fls *FileLogSink) flushLocked() error {
	if fls.buffer.Len() == 0 {
		return nil
	}

	n, err := fls.file.Write(fls.buffer.Bytes())
	if err != nil {
		return err
	}

	fls.currentSize += int64(n)
	fls.buffer.Reset()

	if fls.currentSize >= fls.maxSize {
		return fls.rotateLocked()
	}

	return nil
}

// rotateLocked rotates the log file (must be called with lock held)
func (fls *FileLogSink) rotateLocked() error {
	if err := fls.file.Close(); err != nil {
		return err
	}

	// Rotate backups
	for i := fls.maxBackups - 1; i >= 0; i-- {
		oldName := fmt.Sprintf("%s.%d", fls.filename, i)
		newName := fmt.Sprintf("%s.%d", fls.filename, i+1)

		if i == fls.maxBackups-1 {
			os.Remove(newName)
		}

		if _, err := os.Stat(oldName); err == nil {
			os.Rename(oldName, newName)
		}
	}

	os.Rename(fls.filename, fls.filename+".0")

	file, err := os.OpenFile(fls.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	fls.file = file
	fls.currentSize = 0
	fls.rotationTime = time.Now()

	return nil
}

// autoFlush periodically flushes the buffer
func (fls *FileLogSink) autoFlush() {
	for {
		select {
		case <-fls.ticker.C:
			fls.mu.Lock()
			fls.flushLocked()
			fls.mu.Unlock()
		case <-fls.done:
			fls.ticker.Stop()
			return
		}
	}
}

// Close closes the file sink
func (fls *FileLogSink) Close() error {
	fls.done <- true
	fls.mu.Lock()
	defer fls.mu.Unlock()
	fls.flushLocked()
	return fls.file.Close()
}

// formatJSON formats entry as JSON
func (fls *FileLogSink) formatJSON(entry HTTPLogEntry) ([]byte, error) {
	return []byte(fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"%s"}`,
		entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message)), nil
}

// formatCSV formats entry as CSV
func (fls *FileLogSink) formatCSV(entry HTTPLogEntry) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{
		entry.Timestamp.Format(time.RFC3339),
		entry.Level,
		entry.Message,
	})
	w.Flush()
	return buf.Bytes(), nil
}

// formatXML formats entry as XML
func (fls *FileLogSink) formatXML(entry HTTPLogEntry) ([]byte, error) {
	type XMLEntry struct {
		XMLName   xml.Name  `xml:"entry"`
		Timestamp time.Time `xml:"timestamp"`
		Level     string    `xml:"level"`
		Message   string    `xml:"message"`
	}

	xmlEntry := XMLEntry{
		Timestamp: entry.Timestamp,
		Level:     entry.Level,
		Message:   entry.Message,
	}

	return xml.Marshal(xmlEntry)
}

// formatText formats entry as text
func (fls *FileLogSink) formatText(entry HTTPLogEntry) ([]byte, error) {
	return []byte(fmt.Sprintf("[%s] %s: %s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Level,
		entry.Message)), nil
}

// CloudLogSink sends logs to cloud services
type CloudLogSink struct {
	mu        sync.RWMutex
	provider  CloudProvider
	projectID string
	buffer    []HTTPLogEntry
	batchSize int
	labels    map[string]string
	resource  *CloudResource
}

// CloudProvider represents cloud provider types
type CloudProvider string

const (
	CloudProviderGCP   CloudProvider = "gcp"
	CloudProviderAWS   CloudProvider = "aws"
	CloudProviderAzure CloudProvider = "azure"
)

// CloudResource represents a cloud resource
type CloudResource struct {
	Type   string
	Labels map[string]string
}

// NewCloudLogSink creates a new cloud log sink
func NewCloudLogSink(provider CloudProvider, projectID string, batchSize int) *CloudLogSink {
	return &CloudLogSink{
		provider:  provider,
		projectID: projectID,
		buffer:    make([]HTTPLogEntry, 0, batchSize),
		batchSize: batchSize,
		labels:    make(map[string]string),
	}
}

// Send sends a log entry
func (cls *CloudLogSink) Send(entry HTTPLogEntry) error {
	cls.mu.Lock()
	defer cls.mu.Unlock()

	cls.buffer = append(cls.buffer, entry)

	if len(cls.buffer) >= cls.batchSize {
		return cls.flushLocked()
	}

	return nil
}

// flushLocked flushes the buffer
func (cls *CloudLogSink) flushLocked() error {
	if len(cls.buffer) == 0 {
		return nil
	}

	// Implementation would depend on cloud provider SDK
	// This is a placeholder

	cls.buffer = cls.buffer[:0]
	return nil
}

// StreamProcessor processes log streams
type StreamProcessor struct {
	mu         sync.RWMutex
	processors []StreamHandler
	filters    []StreamFilter
	metrics    *StreamMetrics
}

// StreamHandler handles log stream events
type StreamHandler interface {
	Handle(entry HTTPLogEntry) error
	Name() string
}

// StreamFilter filters log stream events
type StreamFilter interface {
	Filter(entry HTTPLogEntry) bool
	Name() string
}

// StreamMetrics tracks stream metrics
type StreamMetrics struct {
	processed int64
	filtered  int64
	errors    int64
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		processors: make([]StreamHandler, 0),
		filters:    make([]StreamFilter, 0),
		metrics:    &StreamMetrics{},
	}
}

// AddHandler adds a stream handler
func (sp *StreamProcessor) AddHandler(handler StreamHandler) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.processors = append(sp.processors, handler)
}

// AddFilter adds a stream filter
func (sp *StreamProcessor) AddFilter(filter StreamFilter) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.filters = append(sp.filters, filter)
}

// Process processes a log entry
func (sp *StreamProcessor) Process(entry HTTPLogEntry) error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	atomic.AddInt64(&sp.metrics.processed, 1)

	// Apply filters
	for _, filter := range sp.filters {
		if !filter.Filter(entry) {
			atomic.AddInt64(&sp.metrics.filtered, 1)
			return nil
		}
	}

	// Process with handlers
	for _, handler := range sp.processors {
		if err := handler.Handle(entry); err != nil {
			atomic.AddInt64(&sp.metrics.errors, 1)
			return err
		}
	}

	return nil
}

// GetMetrics returns stream metrics
func (sp *StreamProcessor) GetMetrics() StreamMetrics {
	return StreamMetrics{
		processed: atomic.LoadInt64(&sp.metrics.processed),
		filtered:  atomic.LoadInt64(&sp.metrics.filtered),
		errors:    atomic.LoadInt64(&sp.metrics.errors),
	}
}

// WebSocketLogSink sends logs via WebSocket
type WebSocketLogSink struct {
	mu         sync.RWMutex
	url        string
	conn       interface{} // Placeholder for websocket.Conn
	buffer     []HTTPLogEntry
	batchSize  int
	connected  bool
	reconnect  bool
	maxRetries int
}

// NewWebSocketLogSink creates a new WebSocket log sink
func NewWebSocketLogSink(url string, batchSize int) *WebSocketLogSink {
	return &WebSocketLogSink{
		url:        url,
		buffer:     make([]HTTPLogEntry, 0, batchSize),
		batchSize:  batchSize,
		reconnect:  true,
		maxRetries: 5,
	}
}

// Connect connects to WebSocket
func (wls *WebSocketLogSink) Connect() error {
	wls.mu.Lock()
	defer wls.mu.Unlock()

	// Placeholder for WebSocket connection logic
	wls.connected = true
	return nil
}

// Send sends a log entry
func (wls *WebSocketLogSink) Send(entry HTTPLogEntry) error {
	wls.mu.Lock()
	defer wls.mu.Unlock()

	if !wls.connected && wls.reconnect {
		if err := wls.connectLocked(); err != nil {
			return err
		}
	}

	wls.buffer = append(wls.buffer, entry)

	if len(wls.buffer) >= wls.batchSize {
		return wls.flushLocked()
	}

	return nil
}

// connectLocked connects (must be called with lock held)
func (wls *WebSocketLogSink) connectLocked() error {
	wls.connected = true
	return nil
}

// flushLocked flushes the buffer
func (wls *WebSocketLogSink) flushLocked() error {
	if len(wls.buffer) == 0 || !wls.connected {
		return nil
	}

	// Placeholder for WebSocket send logic

	wls.buffer = wls.buffer[:0]
	return nil
}

// Close closes the WebSocket connection
func (wls *WebSocketLogSink) Close() error {
	wls.mu.Lock()
	defer wls.mu.Unlock()
	wls.connected = false
	return nil
}

// KafkaLogSink sends logs to Kafka
type KafkaLogSink struct {
	mu        sync.RWMutex
	brokers   []string
	topic     string
	producer  interface{} // Placeholder for kafka producer
	buffer    []HTTPLogEntry
	batchSize int
	partition int32
}

// NewKafkaLogSink creates a new Kafka log sink
func NewKafkaLogSink(brokers []string, topic string, batchSize int) *KafkaLogSink {
	return &KafkaLogSink{
		brokers:   brokers,
		topic:     topic,
		buffer:    make([]HTTPLogEntry, 0, batchSize),
		batchSize: batchSize,
		partition: -1,
	}
}

// Send sends a log entry
func (kls *KafkaLogSink) Send(entry HTTPLogEntry) error {
	kls.mu.Lock()
	defer kls.mu.Unlock()

	kls.buffer = append(kls.buffer, entry)

	if len(kls.buffer) >= kls.batchSize {
		return kls.flushLocked()
	}

	return nil
}

// flushLocked flushes the buffer
func (kls *KafkaLogSink) flushLocked() error {
	if len(kls.buffer) == 0 {
		return nil
	}

	// Placeholder for Kafka send logic

	kls.buffer = kls.buffer[:0]
	return nil
}

// Close closes the Kafka producer
func (kls *KafkaLogSink) Close() error {
	kls.mu.Lock()
	defer kls.mu.Unlock()
	return kls.flushLocked()
}

// ElasticsearchLogSink sends logs to Elasticsearch
type ElasticsearchLogSink struct {
	mu        sync.RWMutex
	endpoint  string
	index     string
	client    *http.Client
	buffer    []HTTPLogEntry
	batchSize int
}

// NewElasticsearchLogSink creates a new Elasticsearch log sink
func NewElasticsearchLogSink(endpoint, index string, batchSize int) *ElasticsearchLogSink {
	return &ElasticsearchLogSink{
		endpoint:  endpoint,
		index:     index,
		client:    &http.Client{Timeout: 30 * time.Second},
		buffer:    make([]HTTPLogEntry, 0, batchSize),
		batchSize: batchSize,
	}
}

// Send sends a log entry
func (els *ElasticsearchLogSink) Send(entry HTTPLogEntry) error {
	els.mu.Lock()
	defer els.mu.Unlock()

	els.buffer = append(els.buffer, entry)

	if len(els.buffer) >= els.batchSize {
		return els.flushLocked()
	}

	return nil
}

// flushLocked flushes the buffer
func (els *ElasticsearchLogSink) flushLocked() error {
	if len(els.buffer) == 0 {
		return nil
	}

	// Build bulk request
	var buf bytes.Buffer
	for _, entry := range els.buffer {
		// Index action
		fmt.Fprintf(&buf, `{"index":{"_index":"%s"}}`, els.index)
		buf.WriteByte('\n')

		// Document
		fmt.Fprintf(&buf, `{"timestamp":"%s","level":"%s","message":"%s"}`,
			entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message)
		buf.WriteByte('\n')
	}

	url := fmt.Sprintf("%s/_bulk", els.endpoint)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := els.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch error: %s", resp.Status)
	}

	els.buffer = els.buffer[:0]
	return nil
}

// MultiSink sends logs to multiple sinks
type MultiSink struct {
	mu    sync.RWMutex
	sinks []LogSink
}

// LogSink interface for log sinks
type LogSink interface {
	Send(entry HTTPLogEntry) error
	Close() error
}

// NewMultiSink creates a new multi sink
func NewMultiSink() *MultiSink {
	return &MultiSink{
		sinks: make([]LogSink, 0),
	}
}

// AddSink adds a sink
func (ms *MultiSink) AddSink(sink LogSink) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.sinks = append(ms.sinks, sink)
}

// Send sends to all sinks
func (ms *MultiSink) Send(entry HTTPLogEntry) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var errs []error
	for _, sink := range ms.sinks {
		if err := sink.Send(entry); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple sink errors: %v", errs)
	}

	return nil
}

// Close closes all sinks
func (ms *MultiSink) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var errs []error
	for _, sink := range ms.sinks {
		if err := sink.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple close errors: %v", errs)
	}

	return nil
}

// ConfigLoader loads logging configuration
type ConfigLoader struct {
	mu       sync.RWMutex
	sources  []ConfigSource
	cache    map[string]interface{}
	watchers []ConfigWatcher
}

// ConfigSource represents a configuration source
type ConfigSource interface {
	Load() (map[string]interface{}, error)
	Name() string
}

// ConfigWatcher watches for configuration changes
type ConfigWatcher interface {
	Watch(callback func(config map[string]interface{}))
	Stop()
}

// NewConfigLoader creates a new config loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		sources:  make([]ConfigSource, 0),
		cache:    make(map[string]interface{}),
		watchers: make([]ConfigWatcher, 0),
	}
}

// AddSource adds a configuration source
func (cl *ConfigLoader) AddSource(source ConfigSource) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.sources = append(cl.sources, source)
}

// Load loads configuration
func (cl *ConfigLoader) Load() (map[string]interface{}, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	config := make(map[string]interface{})

	for _, source := range cl.sources {
		sourceConfig, err := source.Load()
		if err != nil {
			return nil, fmt.Errorf("load from %s: %w", source.Name(), err)
		}

		// Merge configurations
		for key, value := range sourceConfig {
			config[key] = value
		}
	}

	cl.cache = config
	return config, nil
}

// Get gets a configuration value
func (cl *ConfigLoader) Get(key string) (interface{}, bool) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	value, ok := cl.cache[key]
	return value, ok
}
