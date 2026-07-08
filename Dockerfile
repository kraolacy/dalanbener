# ---- 1) 构建前端静态页（单文件 dist/index.html）----
FROM node:24-slim AS web
WORKDIR /web
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# ---- 2) 运行：Node 同时提供 API 和前端（用内置 node:sqlite，无原生编译）----
FROM node:24-slim
WORKDIR /app
ENV NODE_ENV=production PORT=3000 DB_PATH=/app/data/dalanshu.db
COPY server/package.json ./
RUN npm install --omit=dev
COPY server/ ./
COPY --from=web /web/dist ./public
EXPOSE 3000
CMD ["node", "server.js"]
