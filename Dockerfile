FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/enginewhy ./cmd/enginewhy

FROM alpine:latest AS runtime
RUN apk add --no-cache ca-certificates
WORKDIR /enginewhy
COPY --from=builder /out/enginewhy /usr/bin/enginewhy
ENTRYPOINT ["enginewhy"]
CMD ["-c", "/enginewhy/config.yaml"]
