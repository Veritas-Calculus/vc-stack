package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm/qemu"
)

// qemuDriver is a lightweight driver that manages QEMU processes directly.
// This is intentionally minimal: it supports local image files, seed ISOs.
// and basic tap networking attached to br-int. It's enabled by building.
// with `-tags qemu`.
type qemuDriver struct {
	cfg       Config
	runDir    string // runtime directory for pid and artifacts
	qemuDrv   *qemu.Driver
	useNewDrv bool // use new qemu driver
	vncPorts  *VNCPortAllocator
}

func newDriver(cfg Config) (Driver, error) {
	rd := "/var/run/vc-compute"
	if env := os.Getenv("VC_LITE_RUN_DIR"); env != "" {
		rd = env
	}
	if cfg.QEMURunDir != "" {
		rd = cfg.QEMURunDir
	}
	if err := os.MkdirAll(rd, 0o750); err != nil { // #nosec
		return nil, fmt.Errorf("create run dir: %w", err)
	}

	d := &qemuDriver{cfg: cfg, runDir: rd, vncPorts: NewVNCPortAllocator(cfg.Logger)}

	// Use new QEMU driver if enabled.
	if cfg.UseQEMU {
		cfgDir := cfg.QEMUCfgDir
		if cfgDir == "" {
			cfgDir = "/etc/vc-compute/vms"
		}
		tmplDir := cfg.QEMUTmplDir
		if tmplDir == "" {
			tmplDir = "/etc/vc-compute/templates"
		}

		qemuDrv, err := qemu.NewDriver(cfg.Logger, rd, cfgDir, tmplDir)
		if err != nil {
			return nil, fmt.Errorf("create qemu driver: %w", err)
		}
		d.qemuDrv = qemuDrv
		d.useNewDrv = true
	}

	return d, nil
}

// helper: pid file path.
func (d *qemuDriver) pidPath(id string) string { return filepath.Join(d.runDir, id+".pid") }
func (d *qemuDriver) tapName(id string) string {
	// avoid panic if id shorter than 8.
	clean := id
	if len(clean) > 8 {
		clean = clean[:8]
	}
	return "tap" + clean
}
func (d *qemuDriver) seedPath(id string) string { return filepath.Join(d.runDir, id+"-seed.iso") }

func (d *qemuDriver) metaPath(id string) string { return filepath.Join(d.runDir, id+".meta.json") }

type vmMeta struct {
	PID     int    `json:"pid"`
	QMP     string `json:"qmp"`
	Seed    string `json:"seed"`
	Tap     string `json:"tap"`
	Image   string `json:"image"`
	Created int64  `json:"created"`
	VNC     string `json:"vnc"`
}

// validateVMID checks that id is safe for use in file paths.
func validateVMID(id string) (string, error) {
	id = filepath.Base(id)
	if id == "." || id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, "/\\") {
		return "", fmt.Errorf("invalid VM id")
	}
	return id, nil
}

// detectQEMUBinary returns the appropriate qemu-system binary for the host architecture.
func detectQEMUBinary() string {
	return qemu.DetectQEMUBinary()
}

// safePrefix returns at most n characters from s, safe from bounds panic.
func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// local sanitizeName similar to libvirt helper: make a filesystem/hypervisor friendly name.
func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, " ", "-")
	b := strings.Builder{}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.Trim(out, ".-")
	return out
}
