// monitor/monitor.go
package monitor

import (
	"expvar"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	OnlinePlayers    prometheus.Gauge
	ActiveRooms      prometheus.Gauge
	MessagesReceived prometheus.Counter
	MessageLatency   prometheus.Histogram
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		OnlinePlayers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "online_players",
			Help:      "Number of online players",
		}),
		ActiveRooms: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "active_rooms",
			Help:      "Number of active rooms",
		}),
		MessagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "messages_received_total",
			Help:      "Total number of messages received",
		}),
		MessageLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_latency_seconds",
			Help:      "Message processing latency",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
		}),
	}

	prometheus.MustRegister(
		m.OnlinePlayers,
		m.ActiveRooms,
		m.MessagesReceived,
		m.MessageLatency,
	)

	return m
}

type Monitor struct {
	metrics      *Metrics
	startTime    time.Time
	requestCount int64
	mutex        sync.Mutex
}

func NewMonitor(namespace string) *Monitor {
	return &Monitor{
		metrics:   NewMetrics(namespace),
		startTime: time.Now(),
	}
}

func (m *Monitor) StartServer(addr string) {
	http.Handle("/metrics", promhttp.Handler())

	// 添加expvar指标
	expvar.Publish("uptime", expvar.Func(func() interface{} {
		return time.Since(m.startTime).Seconds()
	}))

	expvar.Publish("requests", expvar.Func(func() interface{} {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		return m.requestCount
	}))

	go http.ListenAndServe(addr, nil)
}

func (m *Monitor) IncOnlinePlayers() {
	m.metrics.OnlinePlayers.Inc()
}

func (m *Monitor) DecOnlinePlayers() {
	m.metrics.OnlinePlayers.Dec()
}

func (m *Monitor) SetActiveRooms(count int) {
	m.metrics.ActiveRooms.Set(float64(count))
}

func (m *Monitor) IncMessagesReceived() {
	m.metrics.MessagesReceived.Inc()
	m.mutex.Lock()
	m.requestCount++
	m.mutex.Unlock()
}

func (m *Monitor) ObserveMessageLatency(duration time.Duration) {
	m.metrics.MessageLatency.Observe(duration.Seconds())
}
