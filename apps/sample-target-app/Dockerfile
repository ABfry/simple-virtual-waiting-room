FROM nginx:1.27-alpine

COPY ./index.html /usr/share/nginx/html/index.html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -qO- http://127.0.0.1/ || exit 1
