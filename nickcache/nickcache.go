package nickcache

import (
	"log/slog"
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
