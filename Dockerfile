FROM golang:1.23-alpine AS build

WORKDIR /app
COPY . .

ENV CGO_ENABLED=0
RUN go build -o anyflip-downloader .

FROM alpine:latest
COPY --from=build /app/anyflip-downloader /usr/local/bin/anyflip-downloader

WORKDIR /data
ENTRYPOINT ["anyflip-downloader"]
CMD ["--help"]