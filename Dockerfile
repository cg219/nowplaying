FROM golang:1.23 AS build
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN apt-get update && apt-get install -y gcc libc-dev
RUN curl -fsSL https://deno.land/install.sh | sh
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY . .
RUN cd frontend && deno install && deno task build && cd ..
RUN go build nowplaying.go
RUN chmod +x /build/nowplaying

FROM ubuntu:latest AS staging
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates
COPY --from=build /build/nowplaying /usr/local/bin/nowplaying
COPY --from=build /build/.env /usr/local/bin/.env
RUN chmod +x /usr/local/bin/nowplaying
EXPOSE 8080
ENTRYPOINT [ "/usr/local/bin/nowplaying" ]

FROM alpine:latest AS alpine
WORKDIR /app
COPY --from=build /build/nowplaying /app
EXPOSE 8080
ENTRYPOINT [ "/app/nowplaying" ]
