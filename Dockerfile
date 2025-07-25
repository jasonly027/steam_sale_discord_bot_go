FROM golang:1.24.5-alpine3.22 as builder

WORKDIR /usr/src/app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o /usr/bin/app

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /usr/bin/app /app

CMD ["/app"]
