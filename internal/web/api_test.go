package web

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	clockkeeper "github.com/loomi-labs/clockkeeper"
	clockkeeperv1 "github.com/loomi-labs/clockkeeper/gen/clockkeeper/v1"
	"github.com/loomi-labs/clockkeeper/internal/botc"
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

	registry, err := botc.NewRegistry(clockkeeper.RolesJSON, clockkeeper.JinxesJSON, clockkeeper.NightSheetJSON)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	return &ClockKeeperServiceHandler{
		config:   &Config{JWTSecretKey: "test-jwt-secret"},
		db:       client,
		auth:     auth,
		registry: registry,
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
	_, err = handler.auth.validate("Bearer " + resp.Msg.Token)
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

// authedCtx returns a context with the given username set for auth.
func authedCtx(username string) context.Context {
	return context.WithValue(context.Background(), usernameKey, username)
}

func TestListScripts_IncludesSystemScripts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	// Create a user and a user-owned script.
	hash, err := HashPassword("pass")
	require.NoError(t, err)
	u, err := handler.db.User.Create().
		SetUsername("testuser").
		SetPasswordHash(hash).
		Save(ctx)
	require.NoError(t, err)

	_, err = handler.db.Script.Create().
		SetName("My Custom Script").
		SetCharacterIds([]string{"washerwoman"}).
		SetUserID(u.ID).
		Save(ctx)
	require.NoError(t, err)

	// List scripts as this user.
	resp, err := handler.ListScripts(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.ListScriptsRequest{}))
	require.NoError(t, err)

	// Should include 3 system scripts + 1 user script.
	assert.Len(t, resp.Msg.Scripts, 4)

	systemCount := 0
	userCount := 0
	for _, s := range resp.Msg.Scripts {
		if s.IsSystem {
			systemCount++
		} else {
			userCount++
		}
	}
	assert.Equal(t, 3, systemCount, "expected 3 system scripts")
	assert.Equal(t, 1, userCount, "expected 1 user script")
}

func TestUpdateScript_BlocksSystemScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	// Create a user for auth context.
	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().
		SetUsername("testuser").
		SetPasswordHash(hash).
		Save(ctx)
	require.NoError(t, err)

	// Find a system script via listing.
	resp, err := handler.ListScripts(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.ListScriptsRequest{}))
	require.NoError(t, err)

	var systemID int64
	for _, s := range resp.Msg.Scripts {
		if s.IsSystem {
			systemID = s.Id
			break
		}
	}
	require.NotZero(t, systemID, "expected at least one system script")

	_, err = handler.UpdateScript(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.UpdateScriptRequest{
		Id:   systemID,
		Name: "Hacked",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestDeleteScript_BlocksSystemScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	// Create a user for auth context.
	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().
		SetUsername("testuser").
		SetPasswordHash(hash).
		Save(ctx)
	require.NoError(t, err)

	// Find a system script via listing.
	resp, err := handler.ListScripts(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.ListScriptsRequest{}))
	require.NoError(t, err)

	var systemID int64
	for _, s := range resp.Msg.Scripts {
		if s.IsSystem {
			systemID = s.Id
			break
		}
	}
	require.NotZero(t, systemID, "expected at least one system script")

	_, err = handler.DeleteScript(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.DeleteScriptRequest{
		Id: systemID,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

// --- Script ownership tests ---

func TestUpdateScript_BlocksOtherUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	// Create two users.
	hash, err := HashPassword("pass")
	require.NoError(t, err)
	userA, err := handler.db.User.Create().SetUsername("userA").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("userB").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// User A creates a script.
	script, err := handler.db.Script.Create().
		SetName("A's Script").
		SetCharacterIds([]string{"washerwoman"}).
		SetUserID(userA.ID).
		Save(ctx)
	require.NoError(t, err)

	// User B tries to update it.
	_, err = handler.UpdateScript(authedCtx("userB"), connect.NewRequest(&clockkeeperv1.UpdateScriptRequest{
		Id:   int64(script.ID),
		Name: "Hacked",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestDeleteScript_BlocksOtherUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	userA, err := handler.db.User.Create().SetUsername("userA").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("userB").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	script, err := handler.db.Script.Create().
		SetName("A's Script").
		SetCharacterIds([]string{"washerwoman"}).
		SetUserID(userA.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = handler.DeleteScript(authedCtx("userB"), connect.NewRequest(&clockkeeperv1.DeleteScriptRequest{
		Id: int64(script.ID),
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestUpdateScript_OwnerSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	userA, err := handler.db.User.Create().SetUsername("userA").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	script, err := handler.db.Script.Create().
		SetName("My Script").
		SetCharacterIds([]string{"washerwoman"}).
		SetUserID(userA.ID).
		Save(ctx)
	require.NoError(t, err)

	resp, err := handler.UpdateScript(authedCtx("userA"), connect.NewRequest(&clockkeeperv1.UpdateScriptRequest{
		Id:   int64(script.ID),
		Name: "Renamed",
	}))
	require.NoError(t, err)
	assert.Equal(t, "Renamed", resp.Msg.Script.Name)
}

func TestDeleteScript_OwnerSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	userA, err := handler.db.User.Create().SetUsername("userA").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	script, err := handler.db.Script.Create().
		SetName("My Script").
		SetCharacterIds([]string{"washerwoman"}).
		SetUserID(userA.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = handler.DeleteScript(authedCtx("userA"), connect.NewRequest(&clockkeeperv1.DeleteScriptRequest{
		Id: int64(script.ID),
	}))
	require.NoError(t, err)
}

// --- Game ownership tests ---

// createTestGame is a helper that creates a user, a script, and a game owned by that user.
func createTestGame(t *testing.T, handler *ClockKeeperServiceHandler) (ownerUsername string, gameID int64) {
	t.Helper()
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("owner").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Create a script via the handler (uses system script).
	scriptsResp, err := handler.ListScripts(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.ListScriptsRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, scriptsResp.Msg.Scripts)

	var scriptID int64
	for _, s := range scriptsResp.Msg.Scripts {
		if s.IsSystem {
			scriptID = s.Id
			break
		}
	}
	require.NotZero(t, scriptID)

	gameResp, err := handler.CreateGame(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:    scriptID,
		PlayerCount: 7,
	}))
	require.NoError(t, err)

	return "owner", gameResp.Msg.Game.Id
}

func TestCreateGame_SetsOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	// Verify owner can access the game.
	resp, err := handler.GetGame(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.GetGameRequest{
		Id: gameID,
	}))
	require.NoError(t, err)
	assert.Equal(t, int32(7), resp.Msg.Game.PlayerCount)
}

func TestGetGame_BlocksOtherUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()
	_, gameID := createTestGame(t, handler)

	// Create another user.
	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("attacker").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	_, err = handler.GetGame(authedCtx("attacker"), connect.NewRequest(&clockkeeperv1.GetGameRequest{
		Id: gameID,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestRandomizeRoles_BlocksOtherUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()
	_, gameID := createTestGame(t, handler)

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("attacker").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	_, err = handler.RandomizeRoles(authedCtx("attacker"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
		GameId: gameID,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestUpdateGameRoles_BlocksOtherUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	ctx := context.Background()
	_, gameID := createTestGame(t, handler)

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("attacker").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	_, err = handler.UpdateGameRoles(authedCtx("attacker"), connect.NewRequest(&clockkeeperv1.UpdateGameRolesRequest{
		GameId:          gameID,
		SelectedRoleIds: []string{"washerwoman"},
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestRandomizeRoles_OwnerSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	resp, err := handler.RandomizeRoles(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
		GameId: gameID,
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Game.SelectedRoleIds, "expected roles to be assigned")
}
