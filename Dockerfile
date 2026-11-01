FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /inventory-server ./cmd/server
FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
RUN mkdir /app/data && chown -R app:app /app
USER app
COPY --from=builder /inventory-server /usr/local/bin/inventory-server
EXPOSE 8080
CMD ["inventory-server"]
