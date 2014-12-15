FROM ubuntu:latest

ENV DEBIAN_FRONTEND noninteractive

# SYSTEM
RUN locale-gen en_US en_US.UTF-8
RUN apt-get update -y
RUN apt-get install -qq curl wget \
    python-software-properties wget \
    software-properties-common && \
    apt-get clean

# JAVA
RUN add-apt-repository -y ppa:webupd8team/java
RUN apt-get update -y
RUN echo oracle-java7-installer shared/accepted-oracle-license-v1-1 select true | /usr/bin/debconf-set-selections && \
    apt-get install -qq oracle-java7-installer && \
    apt-get clean

RUN apt-get install -y nodejs npm vim

# DOCKER ENTRIES
EXPOSE 8083 8086 1935
ENTRYPOINT ["/start"]

# WOWZA
RUN cd /tmp && \
    wget -q http://www.wowza.com/downloads/WowzaStreamingEngine-4-1-1/WowzaStreamingEngine-4.1.1.deb.bin && \
    chmod +x /tmp/WowzaStreamingEngine*.bin && \
    echo "yes\nrushmore\nrushmore\nrushmore\nET1A4-NDYy8-KjKQB-wRuTc-DHufk-VZFWu-7N9jERExTxp3\nyes\n" > config.txt && \
    /tmp/WowzaStreamingEngine*.bin < config.txt && \
    rm -f /tmp/WowzaStreamingEngine*.bin config.txt && \
    touch /usr/local/WowzaStreamingEngine/conf/Server.license && \
    chmod 777 /usr/local/WowzaStreamingEngine/conf/Server.license
# cd /usr/local/WowzaStreamingEngine/bin
# ./startup.sh

ADD ./start /
ADD ./wowza /
ADD ./Application.xml /
