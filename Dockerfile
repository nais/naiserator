FROM alpine:3.5
MAINTAINER Johnny Horvi <johnny.horvi@nav.no>

WORKDIR /app

COPY naiserator .

CMD /app/naiserator --logtostderr=true
