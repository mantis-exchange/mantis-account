# ---------- builder ----------
FROM golang:1.24-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /mantis-account ./cmd/account

# ---------- runtime ----------
FROM gcr.io/distroless/static-debian12

COPY --from=builder /mantis-account /mantis-account

EXPOSE 50053

CMD ["/mantis-account"]
