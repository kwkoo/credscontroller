FROM docker.io/vault:1.3.2

COPY docker-entrypoint.sh /usr/local/bin/

RUN chown -R :0 /vault \
  && \
  chmod -R g+rwx /vault \
  && \
  chmod 755 /usr/local/bin/docker-entrypoint.sh \
  && \
  chown :0 /usr/local/bin/docker-entrypoint.sh

USER 1001