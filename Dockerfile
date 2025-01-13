# Use the official Go image
FROM golang:1.22.3

# Set the working directory
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o server .

# Expose the port
EXPOSE 5000

# Command to run the application
CMD ["./server"]
