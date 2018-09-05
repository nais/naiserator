FROM golang:1.11-alpine as builder
RUN apk add --no-cache git
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN go get
RUN go test ./...
RUN cd cmd/naiserator && go build -a -installsuffix cgo -o naiserator

FROM alpine:3.5
MAINTAINER Johnny Horvi <johnny.horvi@nav.no>
COPY webproxy.nav.no.cer /usr/local/share/ca-certificates/
RUN  apk add --no-cache ca-certificates
RUN  update-ca-certificates
WORKDIR /app
COPY --from=builder /src/cmd/naiserator/naiserator /app/naiserator
CMD ["/app/naiserator", "--logtostderr=true"]
