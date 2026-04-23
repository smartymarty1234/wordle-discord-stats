package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wordle-discord-stats/bot"
	"wordle-discord-stats/daemon"
	"wordle-discord-stats/nickcache"
	"wordle-discord-stats/store"
)

func main() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "1" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	token := mustEnv("DISCORD_TOKEN")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	channelID := os.Getenv("DISCORD_CHANNEL_ID")
	botUserID := os.Getenv("WORDLE_BOT_USER_ID")
	imgparseBin := envOr("IMGPARSE_BIN", "/app/imgparse")

	st := store.NewFileStore(envOr("RESULTS_FILE", "wordle_results.json"))

	b, err := bot.New(token, guildID, st, nil)
	if err != nil {
		slog.Error("bot init failed", "err", err)
		os.Exit(1)
	}
	defer b.Close()

	nc := nickcache.New(b.Session(), guildID)
	nc.Start(time.Hour)
	st.SetResolver(nc)
	b.SetResolver(nc)

	if channelID != "" && botUserID != "" {
		cursor := daemon.NewFileCursor(envOr("CURSOR_FILE", "cursor.txt"))
		d := daemon.New(b.Session(), channelID, botUserID, imgparseBin, cursor, st, nc)
		go d.Run()
	} else {
		slog.Info("DISCORD_CHANNEL_ID or WORDLE_BOT_USER_ID not set, daemon disabled")
	}

	slog.Info("bot running, press ctrl+c to exit")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error(key + " not set")
		os.Exit(1)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
