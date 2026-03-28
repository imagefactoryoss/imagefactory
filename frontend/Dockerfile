# Multi-stage Dockerfile for React frontend
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

ARG VITE_APP_VERSION
ARG VITE_APP_BUILD_DATE
ARG VITE_PRODUCT_INFO_LAST_SYNC
ENV VITE_APP_VERSION=$VITE_APP_VERSION
ENV VITE_APP_BUILD_DATE=$VITE_APP_BUILD_DATE
ENV VITE_PRODUCT_INFO_LAST_SYNC=$VITE_PRODUCT_INFO_LAST_SYNC

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build || npx vite build

# Shared runtime base
FROM nginx:alpine AS runtime-base

# Copy custom nginx config
COPY nginx.conf /etc/nginx/nginx.conf
COPY scripts/40-runtime-config.sh /docker-entrypoint.d/40-runtime-config.sh
RUN chmod +x /docker-entrypoint.d/40-runtime-config.sh

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]

# Runtime image that uses container-built dist
FROM runtime-base AS runtime
COPY --from=builder /app/dist /usr/share/nginx/html

# Runtime image that uses locally-built dist from build context
FROM runtime-base AS runtime-local
COPY dist /usr/share/nginx/html
