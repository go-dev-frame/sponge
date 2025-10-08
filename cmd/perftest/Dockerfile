FROM alpine:latest
LABEL maintainer="zhufuyi <g.zhufuyi@gmail.com>"

# set the time zone to Shanghai
#RUN apk add tzdata  \
#    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
#    && echo "Asia/Shanghai" > /etc/timezone \
#    && apk del tzdata

COPY perftest /app/perftest
RUN chmod +x /app/perftest

# 8888 for collector, 6601 for agent
EXPOSE 8888 6601

WORKDIR /app

ENTRYPOINT ["./perftest"]
