FROM golang:1.16-alpine as builder
RUN apk add --no-cache git make curl
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN go get
RUN make kubebuilder
RUN go test ./...
RUN cd cmd/naiserator && go build -a -installsuffix cgo -o naiserator

FROM alpine:3.13
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/cmd/naiserator/naiserator /app/naiserator
CMD ["/app/naiserator"]
