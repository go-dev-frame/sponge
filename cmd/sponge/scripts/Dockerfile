FROM alpine:latest
MAINTAINER zhufuyi "g.zhufuyi@gmail.com"

USER root

COPY .sponge /root/.sponge
COPY sponge /usr/local/bin/sponge
RUN chmod +x /usr/local/bin/sponge

EXPOSE 24631

WORKDIR /app

ENTRYPOINT ["sponge", "run"]
