# ---- build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod ./
COPY . .
# resolve deps + build both binaries (network available at build time)
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api  ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/api  /app/api
COPY --from=build /out/seed /app/seed
EXPOSE 8080
CMD ["/app/api"]
