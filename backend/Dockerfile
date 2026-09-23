FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/ ./cmd/...


FROM alpine:3.22

# Без них не выйдет ни одного HTTPS-запроса: ни в Telegram, ни за погодой.
RUN apk add --no-cache ca-certificates

COPY --from=build /out/ /usr/local/bin/

CMD ["bot"]
