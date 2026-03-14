package vm

import (
	"sync"
	"testing"
)

func TestNewVNCPortAllocator(t *testing.T) {
	alloc := NewVNCPortAllocator(nil)
	if alloc == nil {
		t.Fatal("expected non-nil allocator")
	}
	if alloc.basePort != 5900 {
		t.Errorf("expected base port 5900, got %d", alloc.basePort)
	}
	if alloc.maxPort != 5999 {
		t.Errorf("expected max port 5999, got %d", alloc.maxPort)
	}
}

func TestVNCPortAllocator_AllocateAndRelease(t *testing.T) {
	alloc := NewVNCPortAllocator(nil)

	port1, err := alloc.Allocate("vm-1")
	if err != nil {
		t.Fatalf("Allocate vm-1: %v", err)
	}
	if port1 < 5900 || port1 > 5999 {
		t.Errorf("port out of range: %d", port1)
	}

	port2, err := alloc.Allocate("vm-2")
	if err != nil {
		t.Fatalf("Allocate vm-2: %v", err)
	}
	if port2 == port1 {
		t.Error("expected different ports for different VMs")
	}

	// Same VM should return same port.
	port1Again, err := alloc.Allocate("vm-1")
	if err != nil {
		t.Fatalf("re-Allocate vm-1: %v", err)
	}
	if port1Again != port1 {
		t.Errorf("expected same port %d, got %d", port1, port1Again)
	}

	// Release and re-allocate.
	alloc.Release("vm-1")
	if alloc.PortFor("vm-1") != 0 {
		t.Error("expected 0 after release")
	}
}

func TestVNCPortAllocator_PortFor(t *testing.T) {
	alloc := NewVNCPortAllocator(nil)

	if p := alloc.PortFor("nonexistent"); p != 0 {
		t.Errorf("expected 0 for nonexistent VM, got %d", p)
	}

	port, _ := alloc.Allocate("vm-x")
	if got := alloc.PortFor("vm-x"); got != port {
		t.Errorf("expected %d, got %d", port, got)
	}
}

func TestVNCPortAllocator_Concurrent(t *testing.T) {
	alloc := NewVNCPortAllocator(nil)
	var wg sync.WaitGroup
	ports := make([]int, 20)
	errs := make([]error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p, err := alloc.Allocate(string(rune('a'+idx)) + "-vm")
			ports[idx] = p
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent allocate %d failed: %v", i, err)
		}
	}

	// All ports should be unique.
	seen := map[int]bool{}
	for _, p := range ports {
		if p != 0 && seen[p] {
			t.Errorf("duplicate port: %d", p)
		}
		seen[p] = true
	}
}

func TestVM_Struct(t *testing.T) {
	vm := VM{
		ID:       "test-id",
		Name:     "test-vm",
		VCPUs:    4,
		MemoryMB: 8192,
		DiskGB:   100,
		Status:   "running",
		Power:    "running",
	}
	if vm.ID != "test-id" {
		t.Errorf("expected test-id, got %q", vm.ID)
	}
	if vm.MemoryMB != 8192 {
		t.Errorf("expected 8192 MB, got %d", vm.MemoryMB)
	}
}

func TestCreateVMRequest_Fields(t *testing.T) {
	req := CreateVMRequest{
		Name:     "my-vm",
		VCPUs:    2,
		MemoryMB: 4096,
		DiskGB:   50,
		Image:    "ubuntu-24.04",
		UEFI:     true,
		TPM:      true,
		Nics: []Nic{
			{MAC: "aa:bb:cc:dd:ee:ff", PortID: "port-1"},
		},
		RootRBDImage:     "pool/image",
		SSHAuthorizedKey: "ssh-rsa AAAA...",
		UserData:         "#cloud-config\n",
	}

	if req.Name != "my-vm" {
		t.Errorf("expected name my-vm, got %q", req.Name)
	}
	if !req.UEFI {
		t.Error("expected UEFI=true")
	}
	if len(req.Nics) != 1 {
		t.Fatalf("expected 1 nic, got %d", len(req.Nics))
	}
	if req.Nics[0].PortID != "port-1" {
		t.Errorf("expected port-1, got %q", req.Nics[0].PortID)
	}
}

func TestNodeStatus_Fields(t *testing.T) {
	ns := NodeStatus{
		CPUsTotal:   32,
		CPUsUsed:    16,
		RAMMBTotal:  65536,
		RAMMBUsed:   32000,
		DiskGBTotal: 1000,
		DiskGBUsed:  500,
		UptimeSec:   86400,
		LoadAvg1:    2.5,
	}

	if ns.CPUsTotal != 32 {
		t.Errorf("expected 32, got %d", ns.CPUsTotal)
	}
	if ns.LoadAvg1 != 2.5 {
		t.Errorf("expected 2.5, got %f", ns.LoadAvg1)
	}
}
