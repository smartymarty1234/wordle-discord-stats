package bot

import (
	"fmt"

	"wordle-discord-stats/store"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	userID := opts["user"].UserValue(nil).ID
	sinceDay := 0
	if v, ok := opts["since_day"]; ok {
		sinceDay = int(v.IntValue())
	}
	scoringType := store.ScoringAverage
	if v, ok := opts["scoring_type"]; ok {
		scoringType = store.ScoringType(v.StringValue())
	}

	result, err := b.store.QueryStats(userID, sinceDay, scoringType, b.nicks.ResolveIdentity)
	var msg string
	if err != nil {
		msg = fmt.Sprintf("error: %v", err)
	} else {
		msg = fmt.Sprintf("%s — avg %.2f, rank #%d", b.nicks.Get(userID), result.AvgScore, result.Rank)
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
}

func (b *Bot) handleTop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	k := int(opts["k"].IntValue())
	sinceDay := 0
	if v, ok := opts["since_day"]; ok {
		sinceDay = int(v.IntValue())
	}
	scoringType := store.ScoringAverage
	if v, ok := opts["scoring_type"]; ok {
		scoringType = store.ScoringType(v.StringValue())
	}

	results, err := b.store.QueryTop(k, sinceDay, scoringType, b.nicks.ResolveIdentity)
	var msg string
	if err != nil {
		msg = fmt.Sprintf("error: %v", err)
	} else {
		msg = store.FormatTop(results)
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
}

func optionMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	m := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(opts))
	for _, o := range opts {
		m[o.Name] = o
	}
	return m
}
