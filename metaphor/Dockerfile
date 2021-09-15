FROM node:12.18.4 AS build
WORKDIR /app
ADD package*.json ./
RUN npm ci
ADD ./ .
ENV PORT=3000
EXPOSE $PORT
RUN npm run build


FROM node:12.18.4-alpine3.12 AS production
ENV PORT=3000
ENV DOTENV_CONFIG_PATH="/vault/secrets/.env"
EXPOSE $PORT

USER node
WORKDIR /app

COPY --from=build --chown=node:node /app/dist /app/dist
COPY --from=build --chown=node:node /app/views /app/dist/views
COPY --from=build --chown=node:node /app/node_modules /app/node_modules
# add any new deployable directories and files from the build stage here

CMD ["node", "dist/server.js"]
