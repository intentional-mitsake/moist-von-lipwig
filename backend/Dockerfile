# Stage 1: Build
FROM golang:latest AS builder
WORKDIR /app

# copy all the dependencied and download
COPY go.mod go.sum ./
RUN go mod download

# copy everything
COPY . .

# build start from main.go
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o main ./cmd/api/main.go

# hwat htis does is to take the built binary and put it in the alpine image
FROM alpine:latest
WORKDIR /root/

# copy the binary
COPY --from=builder /app/main .

COPY --from=builder /app/templates ./templates
COPY --from=builder /app/pkg ./pkg
COPY --from=builder /app/.env .

# create the uploads folder
RUN mkdir -p ./uploads

# expose the port 8848
EXPOSE 8080
# run the binary
CMD ["./main"]