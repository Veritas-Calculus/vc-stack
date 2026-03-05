package compute

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestListImages_FilterByVisibility verifies visibility filtering.
func TestListImages_FilterByVisibility(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "public-img", UUID: "pub-1", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "private-img", UUID: "priv-1", Visibility: "private", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?visibility=public", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "public-img") {
		t.Error("should contain public image")
	}
	if strings.Contains(w.Body.String(), "private-img") {
		t.Error("should NOT contain private image")
	}
}

// TestListImages_FilterByOSType verifies OS type filtering.
func TestListImages_FilterByOSType(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "ubuntu", UUID: "u-1", OSType: "linux", OSVersion: "ubuntu-22.04", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "win2022", UUID: "w-1", OSType: "windows", OSVersion: "win-2022", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?os_type=linux", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ubuntu") {
		t.Error("should contain linux image")
	}
	if strings.Contains(w.Body.String(), "win2022") {
		t.Error("should NOT contain windows image")
	}
}

// TestListImages_FilterByCategory verifies category filtering (CloudStack-style).
func TestListImages_FilterByCategory(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "featured-img", UUID: "feat-1", Category: "featured", Visibility: "public", Status: "active", OwnerID: 1, Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "user-img", UUID: "user-1", Category: "user", Visibility: "private", Status: "active", OwnerID: 1, Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?category=featured", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "featured-img") {
		t.Error("should contain featured image")
	}
	if strings.Contains(w.Body.String(), "user-img") {
		t.Error("should NOT contain user image when filtering featured")
	}
}

// TestListImages_Search verifies name search.
func TestListImages_Search(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "ubuntu-22.04-server", UUID: "s-1", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "centos-9", UUID: "s-2", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?search=ubuntu", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "ubuntu-22.04-server") {
		t.Error("should find ubuntu by search")
	}
	if strings.Contains(w.Body.String(), "centos-9") {
		t.Error("should NOT find centos when searching for ubuntu")
	}
}

// TestGetImage_IncludesInstanceCount verifies instance count in getImage.
func TestGetImage_IncludesInstanceCount(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "count-img", UUID: "cnt-1", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Instance{Name: "vm-a", ImageID: 1, Status: "active", FlavorID: 0})
	db.Create(&Instance{Name: "vm-b", ImageID: 1, Status: "active", FlavorID: 0})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/cnt-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"instance_count\":2") {
		t.Error("should report 2 instances")
	}
}

// TestUpdateImage_HTTP verifies image metadata update.
func TestUpdateImage_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "upd-img", UUID: "upd-1", Visibility: "private", OSType: "linux", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"visibility":"public","category":"featured","os_version":"ubuntu-22.04"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/images/upd-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"visibility\":\"public\"") {
		t.Error("visibility should be updated to public")
	}
	if !strings.Contains(w.Body.String(), "featured") {
		t.Error("category should be updated to featured")
	}
}

// TestDeleteImage_ProtectedBlocked verifies protected images cannot be deleted.
func TestDeleteImage_ProtectedBlocked(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "protected-img", UUID: "prot-1", Protected: true, Status: "active", OwnerID: 1, Category: "user", Visibility: "public", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images/prot-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteImage_InUseBlocked verifies images in use by instances cannot be deleted.
func TestDeleteImage_InUseBlocked(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "in-use-img", UUID: "inuse-1", Status: "active", OwnerID: 1, Category: "user", Visibility: "public", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Instance{Name: "vm-using", ImageID: 1, Status: "active", FlavorID: 0})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images/inuse-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "instance_count") {
		t.Error("should report instance count")
	}
}
