FROM golang:1.22 AS build
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=arm64

RUN apt-get update && apt-get install -y gcc libc-dev
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o nowplaying cmd/nowplaying/main.go

FROM ubuntu:latest AS staging
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates
COPY --from=build /build/nowplaying /usr/local/bin/nowplaying
COPY --from=build /build/.env /usr/local/bin/.env
RUN chmod +x /usr/local/bin/nowplaying
EXPOSE 8080

ENTRYPOINT [ "/usr/local/bin/nowplaying" ]
