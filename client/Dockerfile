# Etapa 1: Build
FROM golang:1.21-alpine3.19 AS build
WORKDIR /app
COPY . .
RUN go build -o client ./client

# Etapa 2: Runtime
FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/client /app/client
ENTRYPOINT ["/app/client"]
