package lite

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// LiteMetrics provides simple counters and duration summaries with Prometheus text rendering.
type LiteMetrics struct {
	mu       sync.Mutex
	counters map[string]uint64
	sumsMs   map[string]float64
	counts   map[string]uint64
	lastsMs  map[string]float64
	started  time.Time
}

func NewLiteMetrics() *LiteMetrics {
	return &LiteMetrics{
		counters: map[string]uint64{},
		sumsMs:   map[string]float64{},
		counts:   map[string]uint64{},
		lastsMs:  map[string]float64{},
		started:  time.Now(),
	}
}

func (m *LiteMetrics) Inc(name string, delta uint64) {
	m.mu.Lock()
	m.counters[name] += delta
	m.mu.Unlock()
}

func (m *LiteMetrics) ObserveMs(name string, ms float64) {
	m.mu.Lock()
	m.sumsMs[name] += ms
	m.counts[name] += 1
	m.lastsMs[name] = ms
	m.mu.Unlock()
}

func (m *LiteMetrics) RenderProm() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := "# HELP vc_lite_uptime_seconds Uptime of vc-lite process in seconds\n# TYPE vc_lite_uptime_seconds gauge\n"
	out += fmt.Sprintf("vc_lite_uptime_seconds %d\n", int(time.Since(m.started).Seconds()))

	// counters.
	counterKeys := make([]string, 0, len(m.counters))
	for k := range m.counters {
		counterKeys = append(counterKeys, k)
	}
	sort.Strings(counterKeys)
	for _, k := range counterKeys {
		out += fmt.Sprintf("# TYPE vc_lite_%s counter\nvc_lite_%s %d\n", k, k, m.counters[k])
	}

	// summaries (avg and last)
	sumKeys := make([]string, 0, len(m.sumsMs))
	for k := range m.sumsMs {
		sumKeys = append(sumKeys, k)
	}
	sort.Strings(sumKeys)
	for _, k := range sumKeys {
		cnt := float64(m.counts[k])
		avg := 0.0
		if cnt > 0 {
			avg = m.sumsMs[k] / cnt
		}
		out += fmt.Sprintf("# HELP vc_lite_%s_ms Average operation duration in milliseconds\n# TYPE vc_lite_%s_ms gauge\nvc_lite_%s_ms %0.3f\n", k+"_avg", k+"_avg", k+"_avg", avg)
		out += fmt.Sprintf("# HELP vc_lite_%s_ms Last operation duration in milliseconds\n# TYPE vc_lite_%s_ms gauge\nvc_lite_%s_ms %0.3f\n", k+"_last", k+"_last", k+"_last", m.lastsMs[k])
	}
	return out
}

// common metric keys.
const (
	MVMCreateTotal   = "vm_create_total"
	MVMDeleteTotal   = "vm_delete_total"
	MVMCreateMs      = "vm_create"
	MVMDeleteMs      = "vm_delete"
	MRbdCloneTotal   = "rbd_clone_total"
	MRbdFlattenTotal = "rbd_flatten_total"
	MRbdResizeTotal  = "rbd_resize_total"
	MRbdRmTotal      = "rbd_rm_total"
	MRbdCloneMs      = "rbd_clone"
	MRbdFlattenMs    = "rbd_flatten"
	MRbdResizeMs     = "rbd_resize"
	MRbdRmMs         = "rbd_rm"
)

// global metrics wiring so the driver can emit metrics without circular deps.
var defaultMetrics *LiteMetrics

func SetMetrics(m *LiteMetrics) { defaultMetrics = m }
func GetMetrics() (*LiteMetrics, bool) {
	if defaultMetrics == nil {
		return nil, false
	}
	return defaultMetrics, true
}
