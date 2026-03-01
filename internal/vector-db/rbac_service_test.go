package vectordb

import (
	"context"
	"testing"
	"time"
)

func TestRBACService_CheckPermission_ShouldMatchRole(t *testing.T) {
	t.Parallel()

	repo, err := NewSQLiteRepository(setupTestSQLite(t))
	if err != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", err)
	}

	service := NewRBACService(repo)
	key := "rbac-test-key"
	createErr := service.CreateAPIKey(context.Background(), key, "reader")
	if createErr != nil {
		t.Fatalf("CreateAPIKey() error = %v", createErr)
	}

	allowed, err := service.CheckPermission(context.Background(), key, VectorPermissionSearch)
	if err != nil {
		t.Fatalf("CheckPermission(search) error = %v", err)
	}
	if !allowed {
		t.Fatal("CheckPermission(search) = false, want true")
	}

	allowed, err = service.CheckPermission(context.Background(), key, VectorPermissionManage)
	if err != nil {
		t.Fatalf("CheckPermission(manage) error = %v", err)
	}
	if allowed {
		t.Fatal("CheckPermission(manage) = true, want false")
	}

	adminKey := "rbac-admin-key"
	createAdminErr := service.CreateAPIKey(context.Background(), adminKey, "admin")
	if createAdminErr != nil {
		t.Fatalf("CreateAPIKey(admin) error = %v", createAdminErr)
	}
	allowed, err = service.CheckPermission(context.Background(), adminKey, VectorPermissionManage)
	if err != nil {
		t.Fatalf("CheckPermission(admin manage) error = %v", err)
	}
	if !allowed {
		t.Fatal("CheckPermission(admin manage) = false, want true")
	}
}

func TestRBACService_CheckPermission_WhenKeyDisabled_ShouldDeny(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		vectorAPIKeys: map[string]*VectorAPIKey{},
	}
	now := time.Now().UTC()
	hash := hashAPIKey("disabled-key")
	repo.vectorAPIKeys[hash] = &VectorAPIKey{KeyHash: hash, Role: "reader", Enabled: false, CreatedAt: now, UpdatedAt: now}

	service := NewRBACService(repo)
	allowed, err := service.CheckPermission(context.Background(), "disabled-key", VectorPermissionSearch)
	if err != nil {
		t.Fatalf("CheckPermission() error = %v", err)
	}
	if allowed {
		t.Fatal("CheckPermission() = true, want false")
	}
}
