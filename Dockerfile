# Use Go base image
FROM golang:1.23.3

# Set working directory
WORKDIR /app

# Copy and download dependency using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
