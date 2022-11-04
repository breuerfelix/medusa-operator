FROM node:17.1.0 as builder

WORKDIR /usr/app

RUN apt-get update
RUN yarn add sharp
RUN yarn global add gatsby-cli
RUN git clone https://github.com/medusajs/admin
WORKDIR /usr/app/admin

RUN yarn install
ENV GATSBY_MEDUSA_BACKEND_URL=https://medusa.burban.me
RUN gatsby build

FROM nginx
EXPOSE 80 
COPY --from=builder /usr/app/admin/public /usr/share/nginx/html

# TODO substitute backend_url
ENTRYPOINT ["nginx", "-g", "daemon off;"]
