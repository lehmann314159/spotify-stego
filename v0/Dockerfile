FROM golang:1.26 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o indexer ./cmd/indexer

FROM alpine:latest
WORKDIR /app
COPY --from=builder /build/server .
COPY --from=builder /build/indexer .
COPY templates/ templates/
COPY static/ static/
VOLUME /app/data
ENV DB_PATH=/app/data/tracks.db
ENV ADDR=:8080
EXPOSE 8080
CMD ["./server"]
