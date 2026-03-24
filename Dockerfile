FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /usr/local/bin/ringclaw .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /usr/local/bin/ringclaw /usr/local/bin/ringclaw

VOLUME /root/.ringclaw
ENTRYPOINT ["ringclaw"]
CMD ["start"]
