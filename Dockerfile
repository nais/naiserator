FROM alpine:3.5
MAINTAINER Johnny Horvi <johnny.horvi@nav.no>

COPY webproxy.nav.no.cer /usr/local/share/ca-certificates/
RUN  apk add --no-cache ca-certificates
RUN  update-ca-certificates

WORKDIR /app

COPY naiserator .

CMD /app/naiserator --logtostderr=true
