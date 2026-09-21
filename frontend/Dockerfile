# syntax=docker/dockerfile:1

# --- deps + build stage ---
FROM node:20-alpine AS build
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# Baked in at build time: Next.js inlines NEXT_PUBLIC_* vars into both the
# client bundle and the server render code, so this must be set here (not
# just as a runtime env var) to point Server Components at the backend
# service's Docker Compose network name.
ARG NEXT_PUBLIC_API_BASE_URL=http://backend:8080
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
RUN npm run build

# --- runtime stage ---
FROM node:20-alpine AS runtime
WORKDIR /app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=build /src/public ./public
COPY --from=build --chown=appuser:appgroup /src/.next/standalone ./
COPY --from=build --chown=appuser:appgroup /src/.next/static ./.next/static

USER appuser
ENV PORT=3000
EXPOSE 3000
CMD ["node", "server.js"]
