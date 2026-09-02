FROM node:22-alpine AS ui
WORKDIR /web
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm ci
COPY web/admin/ .
RUN npm run build

FROM nginx:1.27-alpine
RUN apk add --no-cache openssl \
 && mkdir -p /etc/nginx/tls \
 && openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
      -keyout /etc/nginx/tls/key.pem \
      -out /etc/nginx/tls/cert.pem \
      -subj "/CN=localhost"
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=ui /web/dist /usr/share/nginx/html/admin