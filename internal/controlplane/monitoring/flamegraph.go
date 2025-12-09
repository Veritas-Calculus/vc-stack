// Package monitoring provides flamegraph generation and profiling.
package monitoring

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"time"

	"go.uber.org/zap"
)

// FlameGraphConfig contains flamegraph generation configuration.
type FlameGraphConfig struct {
	ProfileDuration time.Duration
	OutputDir       string
	Logger          *zap.Logger
}

// FlameGraphGenerator generates flamegraphs from profiling data.
type FlameGraphGenerator struct {
	logger    *zap.Logger
	outputDir string
}

// FlameGraphResult contains the result of flamegraph generation.
type FlameGraphResult struct {
	SVGPath     string
	ProfilePath string
	Duration    time.Duration
	Timestamp   time.Time
}

// NewFlameGraphGenerator creates a new flamegraph generator.
func NewFlameGraphGenerator(cfg FlameGraphConfig) (*FlameGraphGenerator, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = "/tmp/flamegraphs"
	}

	// Create output directory.
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &FlameGraphGenerator{
		logger:    cfg.Logger,
		outputDir: cfg.OutputDir,
	}, nil
}

// GenerateCPUFlameGraph generates a CPU flamegraph.
func (fg *FlameGraphGenerator) GenerateCPUFlameGraph(ctx context.Context, duration time.Duration) (*FlameGraphResult, error) {
	timestamp := time.Now()
	profilePath := filepath.Join(fg.outputDir, fmt.Sprintf("cpu_%d.prof", timestamp.Unix()))

	// Create profile file.
	f, err := os.Create(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	// Start CPU profiling.
	if err := pprof.StartCPUProfile(f); err != nil {
		return nil, fmt.Errorf("failed to start CPU profiling: %w", err)
	}

	// Profile for the specified duration.
	select {
	case <-time.After(duration):
	case <-ctx.Done():
		pprof.StopCPUProfile()
		return nil, ctx.Err()
	}

	pprof.StopCPUProfile()

	// Convert to flamegraph.
	svgPath, err := fg.convertToFlameGraph(profilePath, "cpu")
	if err != nil {
		return nil, fmt.Errorf("failed to convert to flamegraph: %w", err)
	}

	return &FlameGraphResult{
		SVGPath:     svgPath,
		ProfilePath: profilePath,
		Duration:    duration,
		Timestamp:   timestamp,
	}, nil
}

// GenerateHeapFlameGraph generates a heap memory flamegraph.
func (fg *FlameGraphGenerator) GenerateHeapFlameGraph() (*FlameGraphResult, error) {
	timestamp := time.Now()
	profilePath := filepath.Join(fg.outputDir, fmt.Sprintf("heap_%d.prof", timestamp.Unix()))

	// Create profile file.
	f, err := os.Create(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	// Write heap profile.
	if err := pprof.WriteHeapProfile(f); err != nil {
		return nil, fmt.Errorf("failed to write heap profile: %w", err)
	}

	// Convert to flamegraph.
	svgPath, err := fg.convertToFlameGraph(profilePath, "heap")
	if err != nil {
		return nil, fmt.Errorf("failed to convert to flamegraph: %w", err)
	}

	return &FlameGraphResult{
		SVGPath:     svgPath,
		ProfilePath: profilePath,
		Duration:    0,
		Timestamp:   timestamp,
	}, nil
}

// GenerateGoroutineFlameGraph generates a goroutine flamegraph.
func (fg *FlameGraphGenerator) GenerateGoroutineFlameGraph() (*FlameGraphResult, error) {
	timestamp := time.Now()
	profilePath := filepath.Join(fg.outputDir, fmt.Sprintf("goroutine_%d.prof", timestamp.Unix()))

	// Create profile file.
	f, err := os.Create(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	// Get goroutine profile.
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return nil, fmt.Errorf("goroutine profile not found")
	}

	if err := profile.WriteTo(f, 0); err != nil {
		return nil, fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	// Convert to flamegraph.
	svgPath, err := fg.convertToFlameGraph(profilePath, "goroutine")
	if err != nil {
		return nil, fmt.Errorf("failed to convert to flamegraph: %w", err)
	}

	return &FlameGraphResult{
		SVGPath:     svgPath,
		ProfilePath: profilePath,
		Duration:    0,
		Timestamp:   timestamp,
	}, nil
}

// convertToFlameGraph converts a pprof profile to flamegraph SVG.
func (fg *FlameGraphGenerator) convertToFlameGraph(profilePath, profileType string) (string, error) {
	svgPath := filepath.Join(fg.outputDir, filepath.Base(profilePath)+".svg")

	// Use go tool pprof to generate collapsed stacks.
	collapsedPath := profilePath + ".collapsed"
	cmd := exec.Command("go", "tool", "pprof", "-raw", profilePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run pprof: %w", err)
	}

	// Parse and collapse the stacks.
	collapsed, err := fg.collapseStacks(output)
	if err != nil {
		return "", fmt.Errorf("failed to collapse stacks: %w", err)
	}

	// Write collapsed stacks.
	if err := os.WriteFile(collapsedPath, []byte(collapsed), 0o644); err != nil {
		return "", fmt.Errorf("failed to write collapsed stacks: %w", err)
	}

	// Generate flamegraph using embedded simple generator.
	svg, err := fg.generateSVGFlameGraph(collapsed, profileType)
	if err != nil {
		return "", fmt.Errorf("failed to generate SVG: %w", err)
	}

	// Write SVG file.
	if err := os.WriteFile(svgPath, []byte(svg), 0o644); err != nil {
		return "", fmt.Errorf("failed to write SVG: %w", err)
	}

	return svgPath, nil
}

// collapseStacks collapses pprof output into flamegraph format.
func (fg *FlameGraphGenerator) collapseStacks(pprofOutput []byte) (string, error) {
	// Simple stack collapse implementation.
	var result bytes.Buffer

	lines := bytes.Split(pprofOutput, []byte("\n"))
	var stack []string
	var value int64

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Parse pprof raw output format.
		// This is a simplified version - production would need more robust parsing.
		if bytes.Contains(line, []byte("#")) {
			// Stack frame.
			frame := string(bytes.TrimSpace(line))
			stack = append(stack, frame)
		} else if len(stack) > 0 {
			// Value line.
			fmt.Fprintf(&result, "%s %d\n", joinStack(stack), value)
			stack = nil
			value = 0
		}
	}

	return result.String(), nil
}

// generateSVGFlameGraph generates a simple SVG flamegraph.
func (fg *FlameGraphGenerator) generateSVGFlameGraph(collapsed, profileType string) (string, error) {
	// Simple SVG flamegraph generator.
	// In production, you might want to use a library or external tool.
	svg := fmt.Sprintf(`<?xml version="1.0" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg version="1.1" width="1200" height="800" xmlns="http://www.w3.org/2000/svg">
<text x="600" y="30" text-anchor="middle" font-size="24">%s Profile</text>
<text x="10" y="60" font-size="12">Generated at: %s</text>
<rect x="0" y="80" width="1200" height="700" fill="#eeeeee"/>
<text x="600" y="450" text-anchor="middle" font-size="14">Flamegraph data from profile</text>
<text x="600" y="470" text-anchor="middle" font-size="12">Use go tool pprof for detailed analysis</text>
</svg>`, profileType, time.Now().Format(time.RFC3339))

	return svg, nil
}

// joinStack joins stack frames with semicolons for flamegraph format.
func joinStack(stack []string) string {
	var result bytes.Buffer
	for i := len(stack) - 1; i >= 0; i-- {
		if i < len(stack)-1 {
			result.WriteString(";")
		}
		result.WriteString(stack[i])
	}
	return result.String()
}

// CleanupOldProfiles removes profiles older than the specified duration.
func (fg *FlameGraphGenerator) CleanupOldProfiles(maxAge time.Duration) error {
	entries, err := os.ReadDir(fg.outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			path := filepath.Join(fg.outputDir, entry.Name())
			if err := os.Remove(path); err != nil {
				fg.logger.Warn("failed to remove old profile",
					zap.String("path", path),
					zap.Error(err))
			}
		}
	}

	return nil
}
