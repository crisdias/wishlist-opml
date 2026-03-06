FROM docker.io/library/golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /wishlist-opml .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /wishlist-opml /wishlist-opml
EXPOSE 8080
ENTRYPOINT ["/wishlist-opml"]
