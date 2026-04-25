FROM golang:1.26-alpine AS frontend-builder
RUN apk add --no-cache nodejs npm pnpm
WORKDIR /frontend
COPY website/ .
RUN pnpm install --frozen-lockfile && pnpm build

FROM golang:1.26-alpine AS go-builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /frontend/dist ./website/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /slot-art ./cmd/server/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=go-builder /slot-art .
COPY config.example.yaml config.yaml
EXPOSE 8080
CMD ["./slot-art"]
