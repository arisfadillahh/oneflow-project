FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
COPY standalone/ /app/
COPY static/ /app/.next/static/
COPY public/ /app/public/
EXPOSE 3000
CMD ["node", "server.js"]
