FROM golang:1.22-alpine AS builder

ARG SERVICE

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/service ./cmd/${SERVICE}

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/service /service
ENTRYPOINT ["/service"]
