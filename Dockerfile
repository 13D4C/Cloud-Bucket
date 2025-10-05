FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
RUN npm run build

FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./main.go

FROM alpine:latest
WORKDIR /app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app
USER appuser
COPY --from=backend-builder /app/server .
COPY --from=frontend-builder /app/dist ./frontend/build
EXPOSE 8080
CMD ["./server"]
