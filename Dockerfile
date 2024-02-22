# Start with a base Go image to build your application
FROM golang:1.18-alpine AS builder

# Install git for fetching Go dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy the Go modules manifests
COPY src/backend/go.mod src/backend/go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the go source files
COPY src/backend .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# Continue with a smaller Python base image for the runtime container
FROM python:3.9-alpine

WORKDIR /app

# Copy the built Go binary from the builder stage
COPY --from=builder /app/server /app/

# Copy the frontend files to the production image
COPY src/backend/public /app/public

# Copy the Python script
COPY python/fetch_chat.py /app/python/

# Install Python dependencies
COPY python/requirements.txt /app/python/
RUN pip install --no-cache-dir -r /app/python/requirements.txt && \
    rm -rf /app/python/requirements.txt

# Expose the port the app runs on
EXPOSE 8080

# Set environment variables for the Go application
ENV PYTHONPATH=/app/python

# Specify a non-root user to run the app
RUN adduser -D myuser
USER myuser

# Run the web service on container startup
CMD ["/app/server"]
