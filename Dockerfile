# Use the official Go image as a base
FROM golang:1.20-alpine

# Set the working directory
WORKDIR /app

# Copy the go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the rest of the source code to the working directory
COPY . .

# Install the binary
RUN go install github.com/corlinp/victor@latest

# Set the entrypoint to the victor binary
ENTRYPOINT ["victor", "--data-dir", "/data", "--host", "0.0.0.0:6723"]

# Expose the port for the service
EXPOSE 6723
