# wordle-discord-stats

A discord bot that reads from the NYT Wordle game's discord output and shows how well you're doing against your friends

## Features

- **Bot slash commands** — `/stats <user>` for a player's all-time average and rank, `/top <k>` for the leaderboard
- **DNF = 7** — failed puzzles (X/6) are scored as 7, keeping them comparable to real scores
- **Fixed nicknames** — faulty, nickname (not discord snowflake tagged) records are resolved to the corresponding user
- **Streaks, Elo, averages** — tracks current and all-time streaks, round-robin Elo ratings, and per-player score averages
- **Daily daemon** — posts a summary to the channel automatically each day

## Running

### Locally

Build the OCR binary, then run the bot:

```sh
cd imgparse && cargo build --release && cd ..
source values.sh && go run main.go
```

`values.sh` exports the required env vars (see `values.yaml` for the full list). It is gitignored so you can put real tokens in it safely.

### Docker

```sh
docker build -t wordle-bot .

docker run --rm \
  -e DISCORD_TOKEN=... \
  -e DISCORD_GUILD_ID=... \
  -e DISCORD_CHANNEL_ID=... \
  -e WORDLE_BOT_USER_ID=... \
  -v $(pwd)/data:/data \
  wordle-bot
```

State (results + cursor) is persisted in the `/data` volume.
