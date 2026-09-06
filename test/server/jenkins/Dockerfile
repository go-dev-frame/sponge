FROM jenkins/jenkins:2.370
LABEL maintainer="zhufuyi g.zhufuyi@gmail.com"
USER root

RUN apt-get update -y && apt-get dist-upgrade -y && apt-get install -y apt-utils make
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone

RUN curl -L -o go1.27.1.tar.gz https://go.dev/dl/go1.27.1.linux-amd64.tar.gz \
    && tar zxf go1.27.1.tar.gz \
    && rm -f go1.27.1.tar.gz \
    && mv -f go /opt/

RUN echo 'export GOROOT="/opt/go"' >> ~/.bashrc \
    && echo 'export GOPATH=$HOME/go' >> ~/.bashrc \
    && echo 'export GOBIN=$GOROOT/bin' >> ~/.bashrc \
    && echo 'export PATH=$PATH:$GOBIN' >> ~/.bashrc
