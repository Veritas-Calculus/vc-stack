package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/internal/management/identity"
	"github.com/Veritas-Calculus/vc-stack/tests/integration"
)

// TestIdentityService verifies the Identity service's core operations
// against a real PostgreSQL database.
func TestIdentityService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	// Create Identity service — this runs AutoMigrate & seeds admin.
	svc, err := identity.NewService(identity.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
		JWT: identity.JWTConfig{
			Secret:           "test-jwt-secret-32chars-min-ok!!",
			ExpiresIn:        3600_000_000_000,  // 1h in nanoseconds
			RefreshExpiresIn: 86400_000_000_000, // 24h
		},
	})
	require.NoError(t, err, "NewService should succeed")
	require.NotNil(t, svc)

	t.Run("default admin exists", func(t *testing.T) {
		// The admin user should have been auto-created.
		resp, err := svc.Login(ctx, &identity.LoginRequest{
			Username: "admin",
			Password: "ChangeMe123!",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken, "should return access token")
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.NotNil(t, resp.User)
		assert.Equal(t, "admin", resp.User.Username)
		assert.True(t, resp.User.IsAdmin)
	})

	t.Run("validate token round-trip", func(t *testing.T) {
		resp, err := svc.Login(ctx, &identity.LoginRequest{
			Username: "admin",
			Password: "ChangeMe123!",
		})
		require.NoError(t, err)

		claims, err := svc.ValidateToken(ctx, resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "admin", claims.Username)
		assert.True(t, claims.IsAdmin)
	})

	t.Run("invalid credentials rejected", func(t *testing.T) {
		_, err := svc.Login(ctx, &identity.LoginRequest{
			Username: "admin",
			Password: "wrong-password",
		})
		assert.Error(t, err)
	})

	t.Run("refresh token flow", func(t *testing.T) {
		resp, err := svc.Login(ctx, &identity.LoginRequest{
			Username: "admin",
			Password: "ChangeMe123!",
		})
		require.NoError(t, err)

		refreshed, err := svc.RefreshAccessToken(ctx, resp.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, refreshed.AccessToken)
		assert.NotEqual(t, resp.AccessToken, refreshed.AccessToken, "should issue new access token")
	})

	t.Run("logout revokes refresh token", func(t *testing.T) {
		resp, err := svc.Login(ctx, &identity.LoginRequest{
			Username: "admin",
			Password: "ChangeMe123!",
		})
		require.NoError(t, err)

		err = svc.Logout(ctx, resp.RefreshToken)
		require.NoError(t, err)

		// Refresh should fail after logout.
		_, err = svc.RefreshAccessToken(ctx, resp.RefreshToken)
		assert.Error(t, err, "refresh should fail after logout")
	})
}
