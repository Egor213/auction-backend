FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build  -o /auction-app ./cmd/app

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /auction-app /auction-app
COPY config/ /config/
COPY migrations/ /migrations/
EXPOSE 8080
ENTRYPOINT ["/auction-app"]