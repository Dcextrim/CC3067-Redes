FROM node:22-alpine

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

COPY data ./data
COPY mcp ./mcp
COPY herramientas.js remote-server.js ./

ENV NODE_ENV=production
ENV PORT=8080
ENV KAESER_LECTURAS_PATH=/tmp/kaeser-lecturas.json

EXPOSE 8080

USER node

CMD ["node", "remote-server.js"]
