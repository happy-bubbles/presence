FROM alpine:3.6

MAINTAINER Rinie Romain <romain.rinie@googlemail.com>

RUN apk update \
 && apk add ca-certificates wget unzip \
 && update-ca-certificates

RUN wget https://github.com/happy-bubbles/presence/releases/download/1.8.3/presence_linux_386.zip

RUN unzip presence_linux_386.zip

EXPOSE 5555

COPY ./docker-entrypoint.sh /
ENTRYPOINT ["/docker-entrypoint.sh"]
