# Dockerfile to build a goservestars container on Ubuntu

FROM ubuntu
MAINTAINER Robert Butts
#RUN apt-get upate

RUN mkdir -p /data/
RUN mkdir -p /usr/bin/

# port to serve on. Change this if you like
EXPOSE 8081

ADD goservestars /usr/bin/goservestars
ADD hyg.sqlite /data/hyg.sqlite

ENTRYPOINT ["/usr/bin/goservestars", "-t", "sqlite", "-d", "/data/hyg.sqlite"] 
