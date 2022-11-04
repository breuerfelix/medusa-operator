FROM node:17.1.0 as builder

WORKDIR /usr/app

RUN apt-get update
RUN yarn add sharp
RUN yarn global add gatsby-cli
RUN git clone https://github.com/medusajs/admin
WORKDIR /usr/app/admin

RUN yarn install
ENV GATSBY_MEDUSA_BACKEND_URL=SEARCH_AND_REPLACE_ME
RUN gatsby build

FROM nginx
EXPOSE 80 
COPY --from=builder /usr/app/admin/public /usr/share/nginx/html

CMD ["/bin/sh", "-c", "find /usr/share/nginx/html -type f -print0 | xargs -0 sed -i \"s|SEARCH_AND_REPLACE_ME|$MEDUSA_URL|g\"; nginx -g 'daemon off;'"]
