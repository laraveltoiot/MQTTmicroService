package metrics

import (
	"net/http"
	"sync"
	"time"

	"MQTTmicroService/internal/logger"
)

// Metrics holds the metrics for the MQTT microservice
type Metrics struct {
	// Message metrics
	PublishedMessages   int64
	ReceivedMessages    int64
	FailedPublishes     int64
	SubscriptionCount   int64
	
	// Connection metrics
	ConnectionAttempts  int64
	ConnectionFailures  int64
	ConnectionSuccesses int64
	Disconnections      int64
	
	// API metrics
	APIRequests         int64
	APIErrors           int64
	
	// Performance metrics
	PublishLatency      []time.Duration
	SubscribeLatency    []time.Duration
	
	// Last updated timestamp
	LastUpdated         time.Time
	
	// Mutex for thread safety
	mu                  sync.RWMutex
	
	// Logger
	logger              *logger.Logger
}

// New creates a new metrics instance
func New(log *logger.Logger) *Metrics {
	return &Metrics{
		PublishLatency:   make([]time.Duration, 0, 100),
		SubscribeLatency: make([]time.Duration, 0, 100),
		LastUpdated:      time.Now(),
		logger:           log,
	}
}

// IncrementPublishedMessages increments the published messages counter
func (m *Metrics) IncrementPublishedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishedMessages++
	m.LastUpdated = time.Now()
}

// IncrementReceivedMessages increments the received messages counter
func (m *Metrics) IncrementReceivedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReceivedMessages++
	m.LastUpdated = time.Now()
}

// IncrementFailedPublishes increments the failed publishes counter
func (m *Metrics) IncrementFailedPublishes() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedPublishes++
	m.LastUpdated = time.Now()
}

// SetSubscriptionCount sets the subscription count
func (m *Metrics) SetSubscriptionCount(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SubscriptionCount = count
	m.LastUpdated = time.Now()
}

// IncrementConnectionAttempts increments the connection attempts counter
func (m *Metrics) IncrementConnectionAttempts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectionAttempts++
	m.LastUpdated = time.Now()
}

// IncrementConnectionFailures increments the connection failures counter
func (m *Metrics) IncrementConnectionFailures() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectionFailures++
	m.LastUpdated = time.Now()
}

// IncrementConnectionSuccesses increments the connection successes counter
func (m *Metrics) IncrementConnectionSuccesses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectionSuccesses++
	m.LastUpdated = time.Now()
}

// IncrementDisconnections increments the disconnections counter
func (m *Metrics) IncrementDisconnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Disconnections++
	m.LastUpdated = time.Now()
}

// IncrementAPIRequests increments the API requests counter
func (m *Metrics) IncrementAPIRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.APIRequests++
	m.LastUpdated = time.Now()
}

// IncrementAPIErrors increments the API errors counter
func (m *Metrics) IncrementAPIErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.APIErrors++
	m.LastUpdated = time.Now()
}

// AddPublishLatency adds a publish latency measurement
func (m *Metrics) AddPublishLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Keep only the last 100 measurements
	if len(m.PublishLatency) >= 100 {
		m.PublishLatency = m.PublishLatency[1:]
	}
	
	m.PublishLatency = append(m.PublishLatency, latency)
	m.LastUpdated = time.Now()
}

// AddSubscribeLatency adds a subscribe latency measurement
func (m *Metrics) AddSubscribeLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Keep only the last 100 measurements
	if len(m.SubscribeLatency) >= 100 {
		m.SubscribeLatency = m.SubscribeLatency[1:]
	}
	
	m.SubscribeLatency = append(m.SubscribeLatency, latency)
	m.LastUpdated = time.Now()
}

// GetMetrics returns the current metrics
func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Calculate average latencies
	var avgPublishLatency, avgSubscribeLatency time.Duration
	
	if len(m.PublishLatency) > 0 {
		var total time.Duration
		for _, latency := range m.PublishLatency {
			total += latency
		}
		avgPublishLatency = total / time.Duration(len(m.PublishLatency))
	}
	
	if len(m.SubscribeLatency) > 0 {
		var total time.Duration
		for _, latency := range m.SubscribeLatency {
			total += latency
		}
		avgSubscribeLatency = total / time.Duration(len(m.SubscribeLatency))
	}
	
	return map[string]interface{}{
		"messages": map[string]int64{
			"published": m.PublishedMessages,
			"received":  m.ReceivedMessages,
			"failed":    m.FailedPublishes,
		},
		"subscriptions": m.SubscriptionCount,
		"connections": map[string]int64{
			"attempts":  m.ConnectionAttempts,
			"failures":  m.ConnectionFailures,
			"successes": m.ConnectionSuccesses,
			"disconnections": m.Disconnections,
		},
		"api": map[string]int64{
			"requests": m.APIRequests,
			"errors":   m.APIErrors,
		},
		"latency": map[string]string{
			"publish":   avgPublishLatency.String(),
			"subscribe": avgSubscribeLatency.String(),
		},
		"last_updated": m.LastUpdated.Format(time.RFC3339),
	}
}

// MetricsMiddleware is a middleware that increments the API requests counter
func (m *Metrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.IncrementAPIRequests()
		next.ServeHTTP(w, r)
	})
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.PublishedMessages = 0
	m.ReceivedMessages = 0
	m.FailedPublishes = 0
	m.SubscriptionCount = 0
	m.ConnectionAttempts = 0
	m.ConnectionFailures = 0
	m.ConnectionSuccesses = 0
	m.Disconnections = 0
	m.APIRequests = 0
	m.APIErrors = 0
	m.PublishLatency = make([]time.Duration, 0, 100)
	m.SubscribeLatency = make([]time.Duration, 0, 100)
	m.LastUpdated = time.Now()
	
	m.logger.Info("Metrics reset")
}