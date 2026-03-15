package web

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	clockkeeperv1 "github.com/loomi-labs/clockkeeper/gen/clockkeeper/v1"
	"github.com/loomi-labs/clockkeeper/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHandler(t *testing.T) *ClockKeeperServiceHandler {
	t.Helper()

	config := database.SetupPostgreSQLContainer(t)
	migrator := database.NewMigrator(t, config)
	migrator.ApplyN(t, -1)

	client, _, err := database.NewClient(config)
	if err != nil {
		t.Fatalf("failed to create ent client: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	auth := NewAuthInterceptor("test-jwt-secret")

	return &ClockKeeperServiceHandler{
		config: &Config{JWTSecretKey: "test-jwt-secret"},
		db:     client,
		auth:   auth,
	}
}

func TestLogin_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	// Create a user.
	hash, err := HashPassword("password123")
	require.NoError(t, err)
	_, err = handler.db.User.Create().
		SetUsername("testuser").
		SetPasswordHash(hash).
		Save(ctx)
	require.NoError(t, err)

	// Login.
	resp, err := handler.Login(ctx, connect.NewRequest(&clockkeeperv1.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Token)

	// Verify returned token is valid.
	err = handler.auth.validate("Bearer " + resp.Msg.Token)
	assert.NoError(t, err)
}

func TestLogin_WrongPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("correct")
	require.NoError(t, err)
	_, err = handler.db.User.Create().
		SetUsername("testuser").
		SetPasswordHash(hash).
		Save(ctx)
	require.NoError(t, err)

	_, err = handler.Login(ctx, connect.NewRequest(&clockkeeperv1.LoginRequest{
		Username: "testuser",
		Password: "wrong",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestLogin_NonexistentUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	_, err := handler.Login(ctx, connect.NewRequest(&clockkeeperv1.LoginRequest{
		Username: "nobody",
		Password: "password",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
