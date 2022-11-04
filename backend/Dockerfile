FROM node:17.1.0

WORKDIR /app/medusa

RUN apt-get update
RUN apt-get install -y python
RUN npm install -g npm

RUN npm install -g @medusajs/medusa-cli
RUN medusa new store --seed
WORKDIR /app/medusa/store

COPY medusa-config.js medusa-config.js

EXPOSE 9000
CMD ["/bin/bash", "-c", "medusa migrations run && medusa start"]
