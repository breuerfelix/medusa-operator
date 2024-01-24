FROM node:20

WORKDIR /app/medusa

RUN apt-get update
RUN apt-get install -y python

RUN npm install -g @medusajs/medusa-cli
RUN medusa new store
WORKDIR /app/medusa/store

COPY medusa-config.js medusa-config.js

EXPOSE 9000
CMD ["/bin/bash", "-c", "medusa migrations run && medusa start"]
