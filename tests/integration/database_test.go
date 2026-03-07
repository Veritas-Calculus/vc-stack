package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Veritas-Calculus/vc-stack/tests/integration"
)

// TestDatabaseConnection verifies that the integration test fixture
// can successfully spin up a PostgreSQL container and connect to it.
func TestDatabaseConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	// Verify database is connected.
	require.NotNil(t, fixture.DB)

	// Verify we can ping the database.
	sqlDB, err := fixture.DB.DB()
	require.NoError(t, err)
	assert.NoError(t, sqlDB.Ping())

	// Verify migrations ran — check that the hosts table exists.
	var count int64
	err = fixture.DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'hosts'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "hosts table should exist after migration")
}

// TestDatabaseDSN verifies the DSN is properly constructed.
func TestDatabaseDSN(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	assert.Contains(t, fixture.DSN, "postgres://")
	assert.Contains(t, fixture.DSN, "vcstack_test")
}
