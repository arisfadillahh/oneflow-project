FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /app
ARG NEXT_PUBLIC_APP_NAME="Oneflow.id"
ARG NEXT_PUBLIC_API_BASE_URL=""
ARG NEXT_PUBLIC_WS_BASE_URL=""
ARG NEXT_PUBLIC_WA_GATEWAY_URL=""
ENV NEXT_PUBLIC_APP_NAME=$NEXT_PUBLIC_APP_NAME
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_WS_BASE_URL=$NEXT_PUBLIC_WS_BASE_URL
ENV NEXT_PUBLIC_WA_GATEWAY_URL=$NEXT_PUBLIC_WA_GATEWAY_URL
COPY --from=deps /app/node_modules /app/node_modules
COPY . /app
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
COPY --from=builder /app/.next/standalone /app
COPY --from=builder /app/.next/static /app/.next/static
COPY --from=builder /app/public /app/public
EXPOSE 3000
CMD ["node", "server.js"]
