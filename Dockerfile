FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/tollgate ./cmd/tollgate

FROM alpine:latest AS runtime
RUN apk add --no-cache ca-certificates
WORKDIR /tollgate
COPY --from=builder /out/tollgate /usr/bin/tollgate
ENTRYPOINT ["tollgate"]
CMD ["-c", "/tollgate/config.yaml"]
