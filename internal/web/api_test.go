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

func TestMain(m *testing.M) {
	database.StartSharedContainer(m)
}

func testHandler(t *testing.T) *ClockKeeperServiceHandler {
	t.Helper()

	config := database.CreateTestDatabase(t)

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
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestDeleteScript_BlocksOtherUser(t *testing.T) {
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
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestUpdateScript_OwnerSucceeds(t *testing.T) {
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
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestRandomizeRoles_BlocksOtherUser(t *testing.T) {
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
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestUpdateGameRoles_BlocksOtherUser(t *testing.T) {
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
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestRandomizeRoles_OwnerSucceeds(t *testing.T) {
handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	resp, err := handler.RandomizeRoles(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
		GameId: gameID,
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Game.SelectedRoleIds, "expected roles to be assigned")
}

// --- Game handler tests ---

func TestCreateGame_InvalidPlayerCount(t *testing.T) {
handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("testuser").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Find a system script.
	scriptsResp, err := handler.ListScripts(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.ListScriptsRequest{}))
	require.NoError(t, err)
	var scriptID int64
	for _, s := range scriptsResp.Msg.Scripts {
		if s.IsSystem {
			scriptID = s.Id
			break
		}
	}
	require.NotZero(t, scriptID)

	_, err = handler.CreateGame(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:    scriptID,
		PlayerCount: 3,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateGame_InvalidScript(t *testing.T) {
handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("testuser").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	_, err = handler.CreateGame(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:    99999,
		PlayerCount: 7,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestCreateGame_ReturnsDistribution(t *testing.T) {
handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("testuser").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	scriptsResp, err := handler.ListScripts(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.ListScriptsRequest{}))
	require.NoError(t, err)
	var scriptID int64
	for _, s := range scriptsResp.Msg.Scripts {
		if s.IsSystem {
			scriptID = s.Id
			break
		}
	}
	require.NotZero(t, scriptID)

	resp, err := handler.CreateGame(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:    scriptID,
		PlayerCount: 7,
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Game.Distribution)
	assert.Equal(t, int32(5), resp.Msg.Game.Distribution.Townsfolk)
	assert.Equal(t, int32(0), resp.Msg.Game.Distribution.Outsiders)
	assert.Equal(t, int32(1), resp.Msg.Game.Distribution.Minions)
	assert.Equal(t, int32(1), resp.Msg.Game.Distribution.Demons)
}

func TestRandomizeRoles_ReturnsCorrectCount(t *testing.T) {
handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	resp, err := handler.RandomizeRoles(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
		GameId: gameID,
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Game.SelectedRoleIds, 7, "expected role count to equal player count")
}

func TestUpdateGameRoles_Persists(t *testing.T) {
handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	roles := []string{"washerwoman", "imp"}
	_, err := handler.UpdateGameRoles(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.UpdateGameRolesRequest{
		GameId:          gameID,
		SelectedRoleIds: roles,
	}))
	require.NoError(t, err)

	// Get the game again and verify roles persisted.
	getResp, err := handler.GetGame(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.GetGameRequest{
		Id: gameID,
	}))
	require.NoError(t, err)
	assert.Equal(t, roles, getResp.Msg.Game.SelectedRoleIds)
}

func TestUpdateGameTravellers_Success(t *testing.T) {
handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	// Find a valid traveller ID via ListCharacters.
	charsResp, err := handler.ListCharacters(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.ListCharactersRequest{
		Team: clockkeeperv1.Team_TEAM_TRAVELLER,
	}))
	require.NoError(t, err)
	require.NotEmpty(t, charsResp.Msg.Characters, "expected at least one traveller character")

	travellerID := charsResp.Msg.Characters[0].Id

	resp, err := handler.UpdateGameTravellers(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.UpdateGameTravellersRequest{
		GameId:               gameID,
		SelectedTravellerIds: []string{travellerID},
	}))
	require.NoError(t, err)
	assert.Contains(t, resp.Msg.Game.SelectedTravellerIds, travellerID)
}

func TestUpdateGameTravellers_RejectsNonTraveller(t *testing.T) {
handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	// "washerwoman" is a townsfolk, not a traveller.
	_, err := handler.UpdateGameTravellers(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.UpdateGameTravellersRequest{
		GameId:               gameID,
		SelectedTravellerIds: []string{"washerwoman"},
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateGame_AutoSelectsTravellersFromScript(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("owner2").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Create a custom script with TB roles + 2 travellers.
	scriptResp, err := handler.CreateScript(authedCtx("owner2"), connect.NewRequest(&clockkeeperv1.CreateScriptRequest{
		Name: "TB with Travellers",
		CharacterIds: []string{
			"washerwoman", "librarian", "investigator", "chef", "empath",
			"fortuneteller", "undertaker", "monk", "ravenkeeper", "virgin",
			"slayer", "soldier", "mayor", "butler", "saint", "recluse",
			"drunk", "poisoner", "spy", "scarletwoman", "baron", "imp",
			"thief", "bureaucrat", // travellers
		},
	}))
	require.NoError(t, err)
	scriptID := scriptResp.Msg.Script.Id

	// Create a game with 2 travellers — should select 2 from the script's 2 travellers.
	gameResp, err := handler.CreateGame(authedCtx("owner2"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:       scriptID,
		PlayerCount:    7,
		TravellerCount: 2,
	}))
	require.NoError(t, err)
	g := gameResp.Msg.Game

	assert.Len(t, g.SelectedTravellerIds, 2, "expected 2 travellers auto-selected from script")
	assert.ElementsMatch(t, []string{"thief", "bureaucrat"}, g.SelectedTravellerIds)
	assert.Equal(t, int32(2), g.TravellerCount)

	// Verify all selected IDs have traveller team in character details.
	for _, ch := range g.SelectedTravellerCharacters {
		assert.Equal(t, clockkeeperv1.Team_TEAM_TRAVELLER, ch.Team, "expected traveller team for %s", ch.Id)
	}
}

func TestCreateGame_ZeroTravellersIgnoresScriptTravellers(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("owner2b").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Create a script with travellers.
	scriptResp, err := handler.CreateScript(authedCtx("owner2b"), connect.NewRequest(&clockkeeperv1.CreateScriptRequest{
		Name: "TB with Travellers (zero test)",
		CharacterIds: []string{
			"washerwoman", "librarian", "investigator", "chef", "empath",
			"fortuneteller", "undertaker", "monk", "ravenkeeper", "virgin",
			"slayer", "soldier", "mayor", "butler", "saint", "recluse",
			"drunk", "poisoner", "spy", "scarletwoman", "baron", "imp",
			"thief", "bureaucrat",
		},
	}))
	require.NoError(t, err)

	// Create game with 0 travellers — none should be selected despite script having them.
	gameResp, err := handler.CreateGame(authedCtx("owner2b"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:       scriptResp.Msg.Script.Id,
		PlayerCount:    7,
		TravellerCount: 0,
	}))
	require.NoError(t, err)
	assert.Empty(t, gameResp.Msg.Game.SelectedTravellerIds, "expected 0 travellers when travellerCount is 0")
}

func TestCreateGame_AutoPopulatesFabledFromScript(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("owner3").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Create a custom script with TB roles + a fabled character.
	scriptResp, err := handler.CreateScript(authedCtx("owner3"), connect.NewRequest(&clockkeeperv1.CreateScriptRequest{
		Name: "TB with Fabled",
		CharacterIds: []string{
			"washerwoman", "librarian", "investigator", "chef", "empath",
			"fortuneteller", "undertaker", "monk", "ravenkeeper", "virgin",
			"slayer", "soldier", "mayor", "butler", "saint", "recluse",
			"drunk", "poisoner", "spy", "scarletwoman", "baron", "imp",
			"angel", // fabled
		},
	}))
	require.NoError(t, err)
	scriptID := scriptResp.Msg.Script.Id

	// Create a game — fabled should appear in extra characters.
	gameResp, err := handler.CreateGame(authedCtx("owner3"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:    scriptID,
		PlayerCount: 7,
	}))
	require.NoError(t, err)
	g := gameResp.Msg.Game

	assert.Contains(t, g.ExtraCharacterIds, "angel", "expected fabled character in extra characters")
}

func TestRandomizeRoles_IncludesTravellersFromScript(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("owner4").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Create a custom script with TB roles + travellers.
	scriptResp, err := handler.CreateScript(authedCtx("owner4"), connect.NewRequest(&clockkeeperv1.CreateScriptRequest{
		Name: "TB with Travellers for Randomize",
		CharacterIds: []string{
			"washerwoman", "librarian", "investigator", "chef", "empath",
			"fortuneteller", "undertaker", "monk", "ravenkeeper", "virgin",
			"slayer", "soldier", "mayor", "butler", "saint", "recluse",
			"drunk", "poisoner", "spy", "scarletwoman", "baron", "imp",
			"thief", "bureaucrat",
		},
	}))
	require.NoError(t, err)
	scriptID := scriptResp.Msg.Script.Id

	// Create game with 2 travellers and randomize roles.
	gameResp, err := handler.CreateGame(authedCtx("owner4"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:       scriptID,
		PlayerCount:    7,
		TravellerCount: 2,
	}))
	require.NoError(t, err)

	resp, err := handler.RandomizeRoles(authedCtx("owner4"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
		GameId: gameResp.Msg.Game.Id,
	}))
	require.NoError(t, err)
	g := resp.Msg.Game

	assert.Len(t, g.SelectedTravellerIds, 2, "expected 2 travellers after randomize")
	assert.ElementsMatch(t, []string{"thief", "bureaucrat"}, g.SelectedTravellerIds)
	assert.Len(t, g.SelectedRoleIds, 7, "expected 7 roles for 7 players")
}

func TestRandomizeRoles_RespectsExistingTravellerCount(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("owner5").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Create a script with 3 travellers (thief, bureaucrat, gunslinger are all TB travellers).
	scriptResp, err := handler.CreateScript(authedCtx("owner5"), connect.NewRequest(&clockkeeperv1.CreateScriptRequest{
		Name: "TB 3 Travellers",
		CharacterIds: []string{
			"washerwoman", "librarian", "investigator", "chef", "empath",
			"fortuneteller", "undertaker", "monk", "ravenkeeper", "virgin",
			"slayer", "soldier", "mayor", "butler", "saint", "recluse",
			"drunk", "poisoner", "spy", "scarletwoman", "baron", "imp",
			"thief", "bureaucrat", "gunslinger",
		},
	}))
	require.NoError(t, err)

	// Create game with only 1 traveller.
	gameResp, err := handler.CreateGame(authedCtx("owner5"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:       scriptResp.Msg.Script.Id,
		PlayerCount:    8,
		TravellerCount: 1,
	}))
	require.NoError(t, err)
	assert.Len(t, gameResp.Msg.Game.SelectedTravellerIds, 1, "expected 1 traveller at creation")

	// Randomize — should still have only 1 traveller, not 3.
	resp, err := handler.RandomizeRoles(authedCtx("owner5"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
		GameId: gameResp.Msg.Game.Id,
	}))
	require.NoError(t, err)
	g := resp.Msg.Game

	assert.Len(t, g.SelectedTravellerIds, 1, "randomize should respect existing traveller count, not add all from script")
	assert.Equal(t, int32(1), g.TravellerCount, "traveller count should not change after randomize")
}

func TestRandomizeRoles_ShufflesTravellers(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("owner6").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Script with 5 TB travellers, picking 1 — should not always be the same.
	scriptResp, err := handler.CreateScript(authedCtx("owner6"), connect.NewRequest(&clockkeeperv1.CreateScriptRequest{
		Name: "TB 5 Travellers shuffle test",
		CharacterIds: []string{
			"washerwoman", "librarian", "investigator", "chef", "empath",
			"fortuneteller", "undertaker", "monk", "ravenkeeper", "virgin",
			"slayer", "soldier", "mayor", "butler", "saint", "recluse",
			"drunk", "poisoner", "spy", "scarletwoman", "baron", "imp",
			"thief", "bureaucrat", "gunslinger", "beggar", "scapegoat",
		},
	}))
	require.NoError(t, err)

	gameResp, err := handler.CreateGame(authedCtx("owner6"), connect.NewRequest(&clockkeeperv1.CreateGameRequest{
		ScriptId:       scriptResp.Msg.Script.Id,
		PlayerCount:    7,
		TravellerCount: 1,
	}))
	require.NoError(t, err)
	gameID := gameResp.Msg.Game.Id

	// Randomize 20 times and collect the selected traveller each time.
	seen := make(map[string]bool)
	for range 20 {
		resp, err := handler.RandomizeRoles(authedCtx("owner6"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
			GameId: gameID,
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Game.SelectedTravellerIds, 1)
		seen[resp.Msg.Game.SelectedTravellerIds[0]] = true
	}

	assert.Greater(t, len(seen), 1, "expected different travellers across 20 randomizations, but always got the same one")
}

func TestGetSetupChecklist_ReturnsSteps(t *testing.T) {
handler := testHandler(t)
	_, gameID := createTestGame(t, handler)

	// Randomize roles first so the checklist has something to work with.
	_, err := handler.RandomizeRoles(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.RandomizeRolesRequest{
		GameId: gameID,
	}))
	require.NoError(t, err)

	resp, err := handler.GetSetupChecklist(authedCtx("owner"), connect.NewRequest(&clockkeeperv1.GetSetupChecklistRequest{
		GameId: gameID,
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.Steps, "expected setup checklist to have steps")
}

func TestGetDistribution_Valid(t *testing.T) {
handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("testuser").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	resp, err := handler.GetDistribution(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.GetDistributionRequest{
		PlayerCount: 7,
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Distribution)
	assert.Equal(t, int32(5), resp.Msg.Distribution.Townsfolk)
	assert.Equal(t, int32(0), resp.Msg.Distribution.Outsiders)
	assert.Equal(t, int32(1), resp.Msg.Distribution.Minions)
	assert.Equal(t, int32(1), resp.Msg.Distribution.Demons)
}

func TestGetDistribution_InvalidCount(t *testing.T) {
handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("testuser").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	_, err = handler.GetDistribution(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.GetDistributionRequest{
		PlayerCount: 3,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetScript_BlocksOtherUser(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	userA, err := handler.db.User.Create().SetUsername("userA").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("userB").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// User A creates a script.
	script, err := handler.db.Script.Create().
		SetName("Private Script").
		SetCharacterIds([]string{"washerwoman"}).
		SetUserID(userA.ID).
		Save(ctx)
	require.NoError(t, err)

	// User B tries to read it.
	_, err = handler.GetScript(authedCtx("userB"), connect.NewRequest(&clockkeeperv1.GetScriptRequest{
		Id: int64(script.ID),
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGetScript_OwnerSucceeds(t *testing.T) {
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

	resp, err := handler.GetScript(authedCtx("userA"), connect.NewRequest(&clockkeeperv1.GetScriptRequest{
		Id: int64(script.ID),
	}))
	require.NoError(t, err)
	assert.Equal(t, "My Script", resp.Msg.Script.Name)
}

func TestGetScript_SystemScriptAccessible(t *testing.T) {
	handler := testHandler(t)
	ctx := context.Background()

	hash, err := HashPassword("pass")
	require.NoError(t, err)
	_, err = handler.db.User.Create().SetUsername("testuser").SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	// Find a system script.
	listResp, err := handler.ListScripts(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.ListScriptsRequest{}))
	require.NoError(t, err)

	var systemID int64
	for _, s := range listResp.Msg.Scripts {
		if s.IsSystem {
			systemID = s.Id
			break
		}
	}
	require.NotZero(t, systemID)

	resp, err := handler.GetScript(authedCtx("testuser"), connect.NewRequest(&clockkeeperv1.GetScriptRequest{
		Id: systemID,
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.Script.IsSystem)
}

func TestUpdateGameRoles_RejectsUnknownCharacter(t *testing.T) {
	handler := testHandler(t)
	ownerName, gameID := createTestGame(t, handler)

	_, err := handler.UpdateGameRoles(authedCtx(ownerName), connect.NewRequest(&clockkeeperv1.UpdateGameRolesRequest{
		GameId:          gameID,
		SelectedRoleIds: []string{"washerwoman", "nonexistent_character"},
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
