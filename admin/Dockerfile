FROM node:17.1.0 as builder

WORKDIR /usr/app

#ENV NODE_OPTIONS=--openssl-legacy-provider

RUN apt-get update
RUN yarn add sharp
RUN yarn global add gatsby-cli
RUN git clone https://github.com/medusajs/admin
WORKDIR /usr/app/admin

RUN yarn install

env GATSBY_MEDUSA_BACKEND_URL=http://localhost:9000
RUN gatsby build

FROM nginx

EXPOSE 80 

COPY --from=builder /usr/app/admin/public /usr/share/nginx/html

ENTRYPOINT ["nginx", "-g", "daemon off;"]
