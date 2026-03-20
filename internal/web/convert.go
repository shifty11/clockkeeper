package web

import (
	"github.com/loomi-labs/clockkeeper/ent"
	"github.com/loomi-labs/clockkeeper/ent/game"
	clockkeeperv1 "github.com/loomi-labs/clockkeeper/gen/clockkeeper/v1"
	"github.com/loomi-labs/clockkeeper/internal/botc"
)

var teamToProto = map[botc.Team]clockkeeperv1.Team{
	botc.TeamTownsfolk: clockkeeperv1.Team_TEAM_TOWNSFOLK,
	botc.TeamOutsider:  clockkeeperv1.Team_TEAM_OUTSIDER,
	botc.TeamMinion:    clockkeeperv1.Team_TEAM_MINION,
	botc.TeamDemon:     clockkeeperv1.Team_TEAM_DEMON,
	botc.TeamTraveller: clockkeeperv1.Team_TEAM_TRAVELLER,
	botc.TeamFabled:    clockkeeperv1.Team_TEAM_FABLED,
	botc.TeamLoric:     clockkeeperv1.Team_TEAM_LORIC,
}

var protoToTeam = map[clockkeeperv1.Team]botc.Team{
	clockkeeperv1.Team_TEAM_TOWNSFOLK: botc.TeamTownsfolk,
	clockkeeperv1.Team_TEAM_OUTSIDER:  botc.TeamOutsider,
	clockkeeperv1.Team_TEAM_MINION:    botc.TeamMinion,
	clockkeeperv1.Team_TEAM_DEMON:     botc.TeamDemon,
	clockkeeperv1.Team_TEAM_TRAVELLER: botc.TeamTraveller,
	clockkeeperv1.Team_TEAM_FABLED:    botc.TeamFabled,
	clockkeeperv1.Team_TEAM_LORIC:     botc.TeamLoric,
}

var gameStateToProto = map[game.State]clockkeeperv1.GameState{
	game.StateSetup:      clockkeeperv1.GameState_GAME_STATE_SETUP,
	game.StateInProgress: clockkeeperv1.GameState_GAME_STATE_IN_PROGRESS,
	game.StateCompleted:  clockkeeperv1.GameState_GAME_STATE_COMPLETED,
}

func characterToProto(c *botc.Character) *clockkeeperv1.Character {
	return &clockkeeperv1.Character{
		Id:                 c.ID,
		Name:               c.Name,
		Team:               teamToProto[c.Team],
		Edition:            c.Edition,
		Ability:            c.Ability,
		Flavor:             c.Flavor,
		Setup:              c.Setup,
		Reminders:          c.Reminders,
		RemindersGlobal:    c.RemindersGlobal,
		FirstNightReminder: c.FirstNightReminder,
		OtherNightReminder: c.OtherNightReminder,
		FirstNight:         int32(c.FirstNight),
		OtherNight:         int32(c.OtherNight),
	}
}

func characterToProtoWithJinxes(c *botc.Character, registry *botc.Registry) *clockkeeperv1.Character {
	proto := characterToProto(c)
	jinxes := registry.Jinxes(c.ID)
	if len(jinxes) > 0 {
		proto.Jinxes = make([]*clockkeeperv1.CharacterJinx, len(jinxes))
		for i, j := range jinxes {
			name := j.ID
			if target, ok := registry.Character(j.ID); ok {
				name = target.Name
			}
			proto.Jinxes[i] = &clockkeeperv1.CharacterJinx{
				CharacterId:   j.ID,
				CharacterName: name,
				Reason:        j.Reason,
			}
		}
	}
	return proto
}

func charactersToProto(chars []*botc.Character) []*clockkeeperv1.Character {
	result := make([]*clockkeeperv1.Character, len(chars))
	for i, c := range chars {
		result[i] = characterToProto(c)
	}
	return result
}

func entScriptToProto(s *ent.Script, registry *botc.Registry) *clockkeeperv1.Script {
	proto := &clockkeeperv1.Script{
		Id:           int64(s.ID),
		Name:         s.Name,
		Edition:      s.Edition,
		CharacterIds: s.CharacterIds,
		IsSystem:     s.IsSystem,
	}
	if registry != nil {
		proto.Characters = charactersToProto(registry.Characters(s.CharacterIds))
	}
	return proto
}

func entGameToProto(g *ent.Game, registry *botc.Registry) *clockkeeperv1.Game {
	chars := registry.Characters(g.SelectedRoles)
	travellerChars := registry.Characters(g.SelectedTravellers)
	extraChars := registry.Characters(g.ExtraCharacters)

	var dist *clockkeeperv1.RoleDistribution
	if d, err := botc.DistributionForPlayerCount(g.PlayerCount); err == nil {
		adjusted, _ := botc.ApplySetupModifiers(d, chars)
		dist = &clockkeeperv1.RoleDistribution{
			Townsfolk: int32(adjusted.Townsfolk),
			Outsiders: int32(adjusted.Outsiders),
			Minions:   int32(adjusted.Minions),
			Demons:    int32(adjusted.Demons),
		}
	}

	// Collect reminder tokens from regular, traveller, and extra characters.
	var tokens []*clockkeeperv1.ReminderToken
	allChars := make([]*botc.Character, 0, len(chars)+len(travellerChars)+len(extraChars))
	allChars = append(allChars, chars...)
	allChars = append(allChars, travellerChars...)
	allChars = append(allChars, extraChars...)
	for _, c := range allChars {
		for _, r := range c.Reminders {
			tokens = append(tokens, &clockkeeperv1.ReminderToken{
				CharacterId:   c.ID,
				CharacterName: c.Name,
				Text:          r,
			})
		}
		for _, r := range c.RemindersGlobal {
			tokens = append(tokens, &clockkeeperv1.ReminderToken{
				CharacterId:   c.ID,
				CharacterName: c.Name,
				Text:          r,
			})
		}
	}

	return &clockkeeperv1.Game{
		Id:                          int64(g.ID),
		ScriptId:                    int64(g.ScriptID),
		PlayerCount:                 int32(g.PlayerCount),
		TravellerCount:              int32(g.TravellerCount),
		SelectedRoleIds:             g.SelectedRoles,
		SelectedTravellerIds:        g.SelectedTravellers,
		ExtraCharacterIds:           g.ExtraCharacters,
		State:                       gameStateToProto[g.State],
		Distribution:                dist,
		SelectedCharacters:          charactersToProto(chars),
		SelectedTravellerCharacters: charactersToProto(travellerChars),
		ExtraCharacterDetails:       charactersToProto(extraChars),
		ReminderTokens:              tokens,
	}
}
