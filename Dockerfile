# Stage 1 — build the React frontend
FROM node:20-alpine AS frontend
WORKDIR /web
COPY web/package*.json ./
RUN npm install --prefer-offline 2>/dev/null || npm install
COPY web/ ./
RUN npm run build

# Stage 2 — build the Go binary
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

# Stage 3 — minimal runtime image
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/api        ./api
COPY --from=backend /app/rules      ./rules
COPY --from=frontend /web/dist      ./web/dist
ENV WEB_DIST=/app/web/dist
EXPOSE 8080
CMD ["./api"]
