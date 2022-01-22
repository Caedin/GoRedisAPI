FROM golang:1.17.5-alpine3.15 as build-env

RUN mkdir /build
WORKDIR /build
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . . 

# Rebuild binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o ./bin/ ./...

FROM scratch
WORKDIR /app
COPY --from=build-env /build/bin /app
COPY server.crt server.crt
COPY server.key server.key 

ENTRYPOINT ["/app/PropertyWebAppAPI"]