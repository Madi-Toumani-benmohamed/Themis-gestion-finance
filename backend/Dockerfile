# ============================================================================
# Image unique : build du frontend + du backend, servis par le meme binaire.
# themis sert ./static (SPA avec fallback) ET /api sur le meme port :8080.
# Un seul objet a deployer, aucune desynchronisation possible front/back.
#
# Le contexte de build est la RACINE du repo (voir docker-compose.yml) :
#   docker compose build --no-cache app
# ============================================================================

# --- 1. build du frontend ---
FROM node:22-alpine AS frontend
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- 2. build du backend ---
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /themis ./cmd/server

# --- 3. runtime (OCR : poppler + tesseract) ---
FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends poppler-utils tesseract-ocr tesseract-ocr-fra ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -r -u 10001 themis
WORKDIR /app
# binaire du backend
COPY --from=build /themis /app/themis
# interface compilee, servie par le backend sur /
COPY --from=frontend /web/dist /app/static
USER themis
EXPOSE 8080
ENTRYPOINT ["/app/themis"]
