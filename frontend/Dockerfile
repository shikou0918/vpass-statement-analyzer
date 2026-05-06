FROM node:22-bookworm

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN chown -R node:node /app

EXPOSE 5173

USER node

CMD ["npm", "run", "dev:docker"]
