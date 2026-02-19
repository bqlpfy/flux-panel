# 多階段構建：最終映像 ~10MB
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 先複製依賴清單以利用快取
COPY go.mod go.sum ./
RUN go mod download

# 複製原始碼並編譯
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# 最終映像使用 alpine（支援 wget 做健康檢查）
FROM alpine:3.20

# 安裝 CA 憑證 + tzdata（時區）
RUN apk --no-cache add ca-certificates tzdata

# 設定時區
ENV TZ=Asia/Shanghai

# 複製二進位
COPY --from=builder /app/server /server

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --retries=3 --start-period=5s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/server"]
