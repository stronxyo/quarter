# Privacy

Quarter is a program that runs on your computer. It has no account, no server of its own, no analytics and no tracking. This page lists everything it reads, keeps and asks for.

## What Quarter reads on your computer

From the Steam client's own folder, read only:

- your Steam libraries and the games in them (names, sizes, install state, downloads and updates in progress);
- the accounts signed in on this computer, to know your public Steam ID;
- your playtime, last played dates and achievement progress, as Steam keeps them;
- Steam's logs, for the times you played (your play history), downloads in progress, and games Steam couldn't show in its library;
- Steam's cache of game logos.

Quarter never reads or asks for your Steam password, and never needs a Steam Web API key.

## What Quarter keeps

Only on your computer, in Quarter's data folder:

| File | What it holds |
|---|---|
| `config.json` | Your favourites, sort, filter and settings |
| `library.json` | Your library as last scanned, so Quarter opens instantly |
| `store.json` | Game details from the Steam store (developer, publisher, year, genres, description, website, achievement count) |
| `logos/` | Game logos downloaded from Steam's servers, for games Steam has none of on your computer (at most 3,000, the oldest removed first) |
| `sessions.json` | Your play sessions, read from Steam's log, so your history outlives it |
| `replay.json` | Your past years from Steam Replay, only if you added them |
| `quarter.log` | Errors and warnings, for troubleshooting |

The data folder is `%APPDATA%\quarter` on Windows, `~/Library/Application Support/quarter` on macOS and `~/.config/quarter` on Linux.

## What Quarter asks the internet for

| Asked of | What is sent | When | What for |
|---|---|---|---|
| Steam store (`store.steampowered.com`) | A game's Steam app number | In the background: once a week per game at most, once a day for games the store doesn't know | The game's details |
| Steam's image servers (`shared.akamai.steamstatic.com`) | A game's Steam app number | For games without a logo on your computer | The game's logo |
| A link-shortening service (for example `t.co`) | The short link the store gives as a game's website | Only for such games | Where the link leads, so Quarter shows the official site (the site itself is not visited) |
| Steam Web API (`api.steampowered.com`) | Your public Steam ID and a year | Only after you press `a` in history and say yes | Your past years from Steam Replay (only years you set to Public) |
| Your web browser | The game's website or SteamDB page, or Quarter's Ko-fi page | Only when you press `w`, `b` or `c`, or click the website or "buy me a coffee" | Opening the page |

Nothing else is sent, to anyone. Quarter doesn't report usage, crashes or anything about you.

Actions like playing, installing and uninstalling are handed to the Steam client on your computer through `steam://` links; Steam does the rest under its own terms.

## Steam data

Game names, details and logos come from Steam and belong to Valve and the games' publishers. Quarter shows them as they are, without any guarantee, and is not affiliated with or endorsed by Valve.

## Deleting everything

- Your past years: press `a` in history and say yes; `replay.json` is deleted.
- Everything: close Quarter and delete its data folder (above). Nothing is kept anywhere else.

## Questions

Open an issue at [github.com/stronxyo/quarter/issues](https://github.com/stronxyo/quarter/issues).
