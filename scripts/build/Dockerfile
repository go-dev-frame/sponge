FROM alpine:latest
MAINTAINER zhufuyi "g.zhufuyi@gmail.com"

# todo generate dockerfile code for http or grpc here
# delete the templates code start

# add curl, used in the http service check, if deployed in k8s, can be installed without
RUN apk add curl

# add grpc_health_probe for health check of grpc service
COPY grpc_health_probe /bin/grpc_health_probe
RUN chmod +x /bin/grpc_health_probe

COPY configs/ /app/configs/
COPY serverNameExample /app/serverNameExample
RUN chmod +x /app/serverNameExample

# http port, grpc service can be ignored
EXPOSE 8080

# delete the templates code end

WORKDIR /app

CMD ["./serverNameExample", "-c", "configs/serverNameExample.yml"]
# if you use the Configuration Center, serverNameExample.yml is changed to the Configuration Center configuration.
#CMD ["./serverNameExample", "-c", "configs/serverNameExample.yml", "-enable-cc"]
