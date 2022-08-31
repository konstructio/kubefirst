FROM squidfunk/mkdocs-material:latest as build

ADD docs /docs/docs
ADD mkdocs.yml /docs/mkdocs.yml
RUN mkdocs build -d "/pub"

FROM nginx:1.17.8-alpine
RUN rm -rf /usr/share/nginx/html
COPY --from=build /pub /usr/share/nginx/html
