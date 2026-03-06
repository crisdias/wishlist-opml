# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Steam API Details

- Resolve vanity URL → Steam ID: parse HTML from `https://store.steampowered.com/wishlist/id/{user}/` (regex for 17-digit ID)
- Wishlist (no API key needed): `https://api.steampowered.com/IWishlistService/GetWishlist/v1/?steamid={steamid}`
- App name: `https://store.steampowered.com/api/appdetails?appids={id}&filters=basic`
- RSS feed: `https://store.steampowered.com/feeds/news/app/{gameId}/`

## Architecture

Single-file Go service (`main.go`), no external dependencies. In-memory cache for Steam IDs and app names (permanent — these don't change). Wishlist is always fetched fresh.

## Dev Environment

- Use `podman` (not docker) for local builds
- Dockerfile uses `docker.io/library/` prefix for podman compatibility

## Language

Project documentation is in Portuguese (pt-BR). Commit messages and code comments should be in English.
