# Start from the latest golang base image
FROM golang:1.20.5

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Build the Go app
RUN go build -o cmd/hotelito/hotelito cmd/hotelito/main.go

# Expose port 8088 to the outside
EXPOSE 8080

# Command to run the executable
CMD ["./cmd/hotelito/hotelito"]