FROM debian:jessie

MAINTAINER Rinie Romain <romain.rinie@googlemail.com>

RUN apt-get update && apt-get install -y mosquitto mosquitto-clients wget unzip

RUN wget https://github.com/happy-bubbles/presence/releases/download/1.8.0/presence_linux_386.zip

RUN unzip presence_linux_386.zip

EXPOSE 5555

COPY ./docker-entrypoint.sh /
ENTRYPOINT ["/docker-entrypoint.sh"]
