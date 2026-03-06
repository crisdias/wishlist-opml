# wishlist-opml

Micro serviço que lê uma wishlist da Steam, extrai os game IDs e gera um OPML com os feeds RSS para as notícias de todos os jogos da wishlist.

## Como funciona

1. Recebe uma requisição `GET /wishlist/{steam_user_id}`
2. Resolve o vanity URL para o Steam ID numérico
3. Busca a wishlist via API da Steam (`IWishlistService/GetWishlist/v1`)
4. Obtém o nome de cada jogo via `appdetails` (com cache em memória)
5. Retorna um arquivo OPML com os feeds RSS ordenados alfabeticamente

Cada feed aponta para `https://store.steampowered.com/feeds/news/app/{gameId}/`.

## Build & Run

```bash
# Com Podman (dev)
podman build -t wishlist-opml .
podman run -p 8080:8080 wishlist-opml

# Com Docker (produção)
docker build -t wishlist-opml .
docker run -p 8080:8080 wishlist-opml
```

A porta pode ser configurada via variável de ambiente `PORT` (default: `8080`).

## Uso

```bash
curl http://localhost:8080/wishlist/crisdias -o wishlist.opml
```

O arquivo OPML gerado pode ser importado em qualquer leitor de RSS.

## Stack

- Go (sem dependências externas)
- Multi-stage Dockerfile: compila em `golang:alpine`, executa em `scratch` (~6MB)
