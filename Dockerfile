# ---- 1) 构建前端单文件（vite-plugin-singlefile 产物为 dist/index.html）----
FROM node:24-slim AS web
WORKDIR /web
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# ---- 2) 构建 Go 后端（gin + gorm + mysql + redis）----
FROM golang:1.23 AS gobuild
WORKDIR /src
COPY server-go/ ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /dalanshu .

# ---- 3) 运行 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=web /web/dist ./dist
COPY --from=gobuild /dalanshu /app/dalanshu
ENV PORT=8080 \
    STATIC_DIR=/app/dist \
    GIN_MODE=release \
    DB_DRIVER=mysql
EXPOSE 8080
CMD ["/app/dalanshu"]
