# Build stage
FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY internal/ internal/
COPY main.go main.go

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bids-service .

FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bids-service .

ENV MODE=production

# Run the application
CMD ["./bids-service"]

