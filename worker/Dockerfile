FROM bikenow.vkw.tu-dresden.de/priobike/priobike-nginx:v1.0

WORKDIR /data

RUN chown -R nginx:nginx /data

# Install httpd-tools for htpasswd
RUN apt-get update && apt-get install -y apache2-utils

COPY ./default.conf /etc/nginx/conf.d/default.conf

# Create htpasswd file and start nginx
CMD htpasswd -bc /etc/nginx/.htpasswd $BASIC_AUTH_USER $BASIC_AUTH_PASS && nginx -g 'daemon off;'