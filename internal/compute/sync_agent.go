package compute

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// SyncAgentRegInfo holds registration information for the node.
type SyncAgentRegInfo struct {
	NodeInfo  NodeInfo
	Port      int
	ZoneID    string
	ClusterID string
}

// SyncAgent handles periodic heartbeats and node synchronization.
type SyncAgent struct {
	client   *ControllerClient
	logger   *zap.Logger
	interval time.Duration
	regInfo  SyncAgentRegInfo
	stopCh   chan struct{}
}

// NewSyncAgent creates a new sync agent for the compute node.
func NewSyncAgent(client *ControllerClient, _ interface{}, logger *zap.Logger, interval time.Duration) *SyncAgent {
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &SyncAgent{
		client:   client,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// SetRegistrationInfo sets the metadata for this node.
func (s *SyncAgent) SetRegistrationInfo(info SyncAgentRegInfo) {
	s.regInfo = info
}

// Start begins the heartbeat loop.
func (s *SyncAgent) Start() {
	s.logger.Info("SyncAgent started", zap.Duration("interval", s.interval))
	go s.run()
}

// Stop terminates the heartbeat loop.
func (s *SyncAgent) Stop() {
	close(s.stopCh)
}

func (s *SyncAgent) run() {
	// 1. Perform initial registration.
	ctx := context.Background()
	_, err := s.client.RegisterNode(ctx, s.regInfo.NodeInfo, s.regInfo.Port, s.regInfo.ZoneID, s.regInfo.ClusterID)
	if err != nil {
		s.logger.Error("Initial registration failed", zap.Error(err))
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sendHeartbeat()
		case <-s.stopCh:
			return
		}
	}
}

func (s *SyncAgent) sendHeartbeat() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Collect and report Host Info (Heartbeat)
	info := CollectNodeInfo(s.logger)
	payload := map[string]interface{}{
		"hostname":      info.Hostname,
		"ip_address":    info.IPAddress,
		"cpus_total":    info.CPUCores,
		"ram_mb_total":  info.RAMMB,
		"disk_gb_total": info.DiskGB,
		"status":        "online",
		"type":          info.HypervisorType,
	}

	if err := s.client.ReportHeartbeat(ctx, payload); err != nil {
		s.logger.Warn("Heartbeat report failed", zap.Error(err))
	}

	// 2. Collect and report performance metrics (Telemetry)
	// Here we reuse info for simple case, but could collect dynamic usage
	perfMetrics := map[string]interface{}{
		"cpu_usage_percent": 15.5, // Placeholder for dynamic collection
		"ram_used_mb":       info.RAMMB / 2,
		"timestamp":         time.Now().Unix(),
	}

	if err := s.client.PushMetrics(ctx, s.client.nodeID, perfMetrics); err != nil {
		s.logger.Warn("Metrics push failed", zap.Error(err))
	}
}
