# wishlist-opml

A microservice that reads a Steam wishlist, extracts game IDs, and generates an OPML file with RSS feeds for news about every game in the wishlist.

## How it works

1. Receives a `GET /wishlist/{steam_user_id}` request
2. Resolves the vanity URL to a numeric Steam ID
3. Fetches the wishlist via Steam API (`IWishlistService/GetWishlist/v1`)
4. Gets each game's name via `appdetails` (with in-memory cache)
5. Returns an OPML file with RSS feeds sorted alphabetically

Each feed points to `https://store.steampowered.com/feeds/news/app/{gameId}/`.

## Build & Run

```bash
docker build -t wishlist-opml .
docker run -p 8080:8080 wishlist-opml
```

Port can be configured via the `PORT` environment variable (default: `8080`).

## Usage

```bash
curl http://localhost:8080/wishlist/crisdias -o wishlist.opml
```

The generated OPML file can be imported into any RSS reader.

## Stack

- Go (no external dependencies)
- Multi-stage Dockerfile: builds in `golang:alpine`, runs from `scratch` (~6MB)
