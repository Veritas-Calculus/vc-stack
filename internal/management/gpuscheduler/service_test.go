package gpuscheduler

import (
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setup(t *testing.T) *Service {
	t.Helper()
	db, _ := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestRegisterGPU(t *testing.T) {
	svc := setup(t)
	gpu, err := svc.RegisterGPU(1, "NVIDIA A100 80GB", "nvidia", 81920, "0000:3b:00.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if gpu.VRAMMB != 81920 {
		t.Errorf("expected 81920, got %d", gpu.VRAMMB)
	}
	if !gpu.MIGCapable {
		t.Error("expected MIG capable")
	}
}

func TestDefaultProfiles(t *testing.T) {
	svc := setup(t)
	profiles, _ := svc.ListProfiles()
	if len(profiles) != 5 {
		t.Errorf("expected 5 default profiles, got %d", len(profiles))
	}
}

func TestCreateVGPU(t *testing.T) {
	svc := setup(t)
	gpu, _ := svc.RegisterGPU(1, "A100", "nvidia", 81920, "0000:3b:00.0", true)
	vgpu, err := svc.CreateVGPU(gpu.ID, "1g.10gb")
	if err != nil {
		t.Fatal(err)
	}
	if vgpu.ProfileName != "1g.10gb" {
		t.Errorf("expected 1g.10gb, got %q", vgpu.ProfileName)
	}
	if vgpu.Status != "free" {
		t.Errorf("expected free, got %q", vgpu.Status)
	}
}

func TestVGPUCapacityLimit(t *testing.T) {
	svc := setup(t)
	gpu, _ := svc.RegisterGPU(1, "A100", "nvidia", 81920, "0000:3b:00.0", true)
	// 7g.80gb profile allows MaxPerGPU=1.
	svc.CreateVGPU(gpu.ID, "7g.80gb")
	_, err := svc.CreateVGPU(gpu.ID, "7g.80gb")
	if err == nil {
		t.Error("expected capacity error")
	}
}

func TestAllocateRelease(t *testing.T) {
	svc := setup(t)
	gpu, _ := svc.RegisterGPU(1, "A100", "nvidia", 81920, "0000:3b:00.0", true)
	vgpu, _ := svc.CreateVGPU(gpu.ID, "2g.20gb")
	if err := svc.AllocateVGPU(vgpu.ID, 42); err != nil {
		t.Fatal(err)
	}
	vgpus, _ := svc.ListVGPUs(gpu.ID)
	if len(vgpus) == 0 {
		t.Fatal("expected at least 1 vgpu")
	}
	if vgpus[0].Status != "allocated" {
		t.Errorf("expected allocated, got %q", vgpus[0].Status)
	}

	svc.ReleaseVGPU(vgpu.ID)
	vgpus, _ = svc.ListVGPUs(gpu.ID)
	if vgpus[0].Status != "free" {
		t.Errorf("expected free after release, got %q", vgpus[0].Status)
	}
}

func TestInvalidProfile(t *testing.T) {
	svc := setup(t)
	gpu, _ := svc.RegisterGPU(1, "A100", "nvidia", 81920, "0000:3b:00.0", true)
	_, err := svc.CreateVGPU(gpu.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for invalid profile")
	}
}
