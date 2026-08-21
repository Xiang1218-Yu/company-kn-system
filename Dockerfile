# ---- Frontend build stage ----
# Build the React app with a reproducible Node toolchain. The output dist/ is
# copied into the final image and served by the Go binary.
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# ---- Backend build stage ----
# Compile the Go server. Using the slim image keeps the build context small;
# the final image only needs the resulting binary plus the migrations and the
# compiled frontend.
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
# CGO is disabled so the binary is fully static and runs on the scratch-like
# final image without a C toolchain.
RUN CGO_ENABLED=0 go build -o /out/kn-server ./cmd/server

# ---- Final runtime stage ----
# A single small image: the static Go binary, migrations, the compiled frontend,
# and a CA bundle so HTTPS calls to LLM providers work.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 kn && \
    mkdir -p /app/uploads && chown -R kn:kn /app
WORKDIR /app
COPY --from=backend /out/kn-server /app/kn-server
COPY migrations/ /app/migrations/
COPY --from=frontend /app/frontend/dist /app/web
USER kn
ENV APP_ENV=production \
    SERVER_PORT=8080 \
    STORAGE_PROVIDER=local \
    STORAGE_LOCAL_DIR=/app/uploads \
    EMBEDDER_PROVIDER=mock \
    LLM_PROVIDER=mock \
    QUEUE_PROVIDER=memory
EXPOSE 8080
# The Go server serves both the API (/api/v1) and the SPA (everything else).
ENTRYPOINT ["/app/kn-server"]
