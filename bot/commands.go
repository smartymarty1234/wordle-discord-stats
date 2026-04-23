package bot

import (
	"fmt"

	"wordle-discord-stats/store"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	userID := opts["user"].UserValue(nil).ID

	result, err := b.store.Query(store.Query{
		Kind:     store.KindAvgAllTime,
		Selector: store.SelectorPlayer,
		Player:   userID,
	})
	var msg string
	if err != nil {
		msg = fmt.Sprintf("error: %v", err)
	} else {
		e := result.Entries[0]
		msg = fmt.Sprintf("%s — avg %.2f, rank #%d", e.Name, e.Value, e.Rank)
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
}

func (b *Bot) handleTop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	k := int(opts["k"].IntValue())

	result, err := b.store.Query(store.Query{
		Kind:     store.KindAvgAllTime,
		Selector: store.SelectorTopK,
		K:        k,
	})
	var msg string
	if err != nil {
		msg = fmt.Sprintf("error: %v", err)
	} else {
		msg = store.FormatEntries(result.Entries)
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
