package bot

import (
	"fmt"
	"log/slog"

	"wordle-discord-stats/store"

	"github.com/bwmarrin/discordgo"
)

// Resolver maps a player key (snowflake or fixed nick) to a display name.
type Resolver interface {
	Get(key string) string
}

type Bot struct {
	session  *discordgo.Session
	guildID  string
	store    store.Store
	resolver Resolver
}

var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "stats",
		Description: "Get average Wordle score for a user",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to look up",
				Required:    true,
			},
		},
	},
	{
		Name:        "top",
		Description: "Get top Wordle scores",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "k",
				Description: "Number of users to show",
				Required:    true,
			},
		},
	},
}

func New(token, guildID string, st store.Store, resolver Resolver) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	b := &Bot{session: s, guildID: guildID, store: st, resolver: resolver}
	s.AddHandler(b.handleInteraction)

	if err := s.Open(); err != nil {
		return nil, err
	}

	for _, cmd := range slashCommands {
		if _, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd); err != nil {
			return nil, fmt.Errorf("register command %s: %w", cmd.Name, err)
		}
	}
	slog.Info("registered slash commands", "count", len(slashCommands))

	return b, nil
}

func (b *Bot) Close()                        { b.session.Close() }
func (b *Bot) Session() *discordgo.Session   { return b.session }
func (b *Bot) SetResolver(r Resolver)        { b.resolver = r }

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	switch i.ApplicationCommandData().Name {
	case "stats":
		b.handleStats(s, i)
	case "top":
		b.handleTop(s, i)
	}
}
