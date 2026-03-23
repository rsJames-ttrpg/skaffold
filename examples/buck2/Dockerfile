FROM golang:1.22 AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 go build -o /app/main main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/main /main
ENTRYPOINT ["/main"]
