FROM alpine:latest
LABEL maintainer="zhufuyi <g.zhufuyi@gmail.com>"

COPY perftest /app/perftest
RUN chmod +x /app/perftest

# 8888 for collector, 6601 for agent
EXPOSE 8888 6601

WORKDIR /app

ENTRYPOINT ["./perftest"]
