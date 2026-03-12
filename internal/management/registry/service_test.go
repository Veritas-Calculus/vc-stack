package registry

import (
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupImageTest(t *testing.T) *Service {
	t.Helper()
	db, _ := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestCreateRepo(t *testing.T) {
	svc := setupImageTest(t)
	r, err := svc.CreateRepo(1, &CreateRepoRequest{Name: "myproject/nginx", Visibility: "public"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Visibility != "public" {
		t.Errorf("expected public, got %q", r.Visibility)
	}
}

func TestListRepos(t *testing.T) {
	svc := setupImageTest(t)
	svc.CreateRepo(1, &CreateRepoRequest{Name: "repo1"})
	svc.CreateRepo(1, &CreateRepoRequest{Name: "repo2"})
	repos, _ := svc.ListRepos(0)
	if len(repos) != 2 {
		t.Errorf("expected 2, got %d", len(repos))
	}
}

func TestPushTag(t *testing.T) {
	svc := setupImageTest(t)
	r, _ := svc.CreateRepo(1, &CreateRepoRequest{Name: "tag-repo"})
	tag, err := svc.PushTag(r.ID, &PushTagRequest{Tag: "v1.0.0", Digest: "sha256:abc123", SizeBytes: 102400})
	if err != nil {
		t.Fatal(err)
	}
	if tag.Tag != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %q", tag.Tag)
	}
	if tag.Architecture != "amd64" {
		t.Errorf("expected amd64, got %q", tag.Architecture)
	}
}

func TestPushTagUpsert(t *testing.T) {
	svc := setupImageTest(t)
	r, _ := svc.CreateRepo(1, &CreateRepoRequest{Name: "upsert-repo"})
	svc.PushTag(r.ID, &PushTagRequest{Tag: "latest", Digest: "sha256:old"})
	updated, _ := svc.PushTag(r.ID, &PushTagRequest{Tag: "latest", Digest: "sha256:new"})
	if updated.Digest != "sha256:new" {
		t.Errorf("expected sha256:new, got %q", updated.Digest)
	}
}

func TestDeleteRepo(t *testing.T) {
	svc := setupImageTest(t)
	r, _ := svc.CreateRepo(1, &CreateRepoRequest{Name: "del-repo"})
	svc.PushTag(r.ID, &PushTagRequest{Tag: "v1"})
	svc.DeleteRepo(r.ID)
	_, err := svc.GetRepo(r.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestTagCount(t *testing.T) {
	svc := setupImageTest(t)
	r, _ := svc.CreateRepo(1, &CreateRepoRequest{Name: "count-repo"})
	svc.PushTag(r.ID, &PushTagRequest{Tag: "v1"})
	svc.PushTag(r.ID, &PushTagRequest{Tag: "v2"})
	svc.PushTag(r.ID, &PushTagRequest{Tag: "latest"})

	repos, _ := svc.ListRepos(0)
	for _, repo := range repos {
		if repo.ID == r.ID && repo.TagCount != 3 {
			t.Errorf("expected tag count 3, got %d", repo.TagCount)
		}
	}
}
