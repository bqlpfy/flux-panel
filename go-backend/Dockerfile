# 多階段構建：最終映像 ~20MB
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 先複製依賴清單以利用快取
COPY go.mod go.sum ./
RUN go mod download

# 複製原始碼並編譯
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# 最終映像使用 scratch (零基礎映像)
FROM scratch

# 複製 SSL 憑證（用於 HTTPS 請求）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 複製二進位
COPY --from=builder /app/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
