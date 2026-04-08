# Build stage
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY internal/ internal/
COPY main.go main.go

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -installsuffix cgo -o bids-service .

FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bids-service .

ENV MODE=production

# Run the application
CMD ["./bids-service"]

