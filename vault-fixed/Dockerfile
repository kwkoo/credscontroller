FROM docker.io/vault:0.11.1

COPY docker-entrypoint.sh /usr/local/bin/

RUN chown -R :0 /vault && chmod 755 /usr/local/bin/docker-entrypoint.sh