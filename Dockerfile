FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG CMD=gateway
RUN go build -o /out/app ./cmd/${CMD}

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/app .
COPY configs/gateway.yaml ./configs/gateway.yaml
EXPOSE 8080 8081 8082
CMD ["./app"]