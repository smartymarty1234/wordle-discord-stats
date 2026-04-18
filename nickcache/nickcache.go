package nickcache

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type NickCache struct {
	session *discordgo.Session
	guildID string
	mu      sync.RWMutex
	nicks   map[string]string
}

func New(session *discordgo.Session, guildID string) *NickCache {
	return &NickCache{
		session: session,
		guildID: guildID,
		nicks:   map[string]string{},
	}
}

func (c *NickCache) Get(userID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if n, ok := c.nicks[userID]; ok {
		return n
	}
	if strings.HasPrefix(userID, "@") {
		return userID[1:]
	}
	return userID
}

func (c *NickCache) Refresh(getUserIDs func() ([]string, error)) {
	ids, err := getUserIDs()
	if err != nil {
		slog.Error("nickcache: get user IDs", "err", err)
		return
	}

	fresh := make(map[string]string, len(ids))
	for _, id := range ids {
		// Fixed-nick users have IDs like "@sanky panky" — not Discord snowflakes.
		// Their display name is already embedded in the ID; skip the API call.
		if strings.HasPrefix(id, "@") {
			fresh[id] = id[1:]
			continue
		}
		m, err := c.session.GuildMember(c.guildID, id)
		if err != nil {
			slog.Warn("nickcache: lookup failed", "id", id, "err", err)
			fresh[id] = id
			continue
		}
		name := m.Nick
		if name == "" {
			name = m.User.Username
		}
		fresh[id] = name
	}

	c.mu.Lock()
	c.nicks = fresh
	c.mu.Unlock()
	slog.Debug("nickcache: refreshed", "count", len(fresh))
}

// ResolveIdentity returns a canonical display-name key for a userID so that
// results from "@rust cruncher" and the snowflake that the nick cache resolves
// to "rust cruncher" are merged into a single player when scoring.
func (c *NickCache) ResolveIdentity(userID string) string {
	if strings.HasPrefix(userID, "@") {
		return userID[1:]
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if n, ok := c.nicks[userID]; ok {
		return n
	}
	return userID
}

func (c *NickCache) Start(getUserIDs func() ([]string, error), interval time.Duration) {
	c.Refresh(getUserIDs)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.Refresh(getUserIDs)
		}
	}()
}
