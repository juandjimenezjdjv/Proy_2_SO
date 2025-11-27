# Etapa 1: Build
FROM golang:1.21-alpine3.19 AS build
WORKDIR /app
COPY . .
RUN go build -o master-bin ./master

# Etapa 2: Runtime
FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/master-bin /app/master
EXPOSE 8080
CMD ["/app/master"]
