# Stage 1: Build the client
FROM node:20 AS client-builder

# Set the working directory for the client build
WORKDIR /client

# Copy client code
COPY client/package*.json ./

# Install client dependencies
RUN npm install

# Copy the rest of the client source code
COPY client/ ./

# Build the React client application
RUN npm run build

# Stage 2: Build the server
FROM golang:1.22.3 AS server-builder

# Set the working directory for the Go server
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go server application
RUN go build -o server .

# Stage 3: Final stage with server and client
FROM golang:1.22.3

# Set the working directory
WORKDIR /app

# Copy the built server
COPY --from=server-builder /app/server .

# Copy the built client to ./client/dist
COPY --from=client-builder /client/dist ./client/dist

# Expose the server port
EXPOSE 5000

# Command to run the application
CMD ["./server"]
