# Use an official Golang runtime as a parent image
FROM golang:1.21.4

# Set the working directory to /go/src/app
WORKDIR /go/src/app

# Copy the rest of the application source code
COPY . .

# Build the backend binary
RUN go build -o main

# Expose the PORT specified through an environment variable
EXPOSE $PORT

# Command to run the executable
CMD ["./main"]
