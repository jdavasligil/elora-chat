# Start with a base Go image to build your application
FROM golang:1.18 AS builder

# Set the working directory outside $GOPATH to enable Go modules
WORKDIR /app

# Copy the Go modules manifests
COPY src/backend/go.mod src/backend/go.sum ./
# Download Go module dependencies and tidy up the modules
RUN go mod download && go mod tidy

# Copy the go source files
COPY src/backend .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o server .

# Continue with a Python base image for the runtime container
FROM python:3.9-slim

# Set the working directory in the container
WORKDIR /app

# Copy the built Go binary from the builder stage
COPY --from=builder /app/server /app/

# Copy the frontend files to the production image
COPY src/backend/public /app/public

# Copy the Python script and requirements
COPY python/fetch_chat.py python/requirements.txt /app/python/

# Install Python dependencies
RUN pip install --no-cache-dir -r /app/python/requirements.txt

# Expose the port the app runs on
EXPOSE 8080

# Set environment variables for the Go application
ENV PYTHONPATH=/app/python

# Run the web service on container startup
CMD ["/app/server"]
