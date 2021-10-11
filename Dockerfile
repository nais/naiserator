FROM golang:1.17-alpine as builder
RUN apk add --no-cache git make curl
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY go.* /src/
RUN go mod download
COPY . /src
RUN mkdir -p /usr/local/kubebuilder
RUN make kubebuilder
RUN go test ./...
RUN cd cmd/naiserator && go build -a -installsuffix cgo -o naiserator
RUN cd cmd/naiserator_webhook && go build -a -installsuffix cgo -o naiserator_webhook

FROM alpine:3.14
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/cmd/naiserator/naiserator /app/naiserator
COPY --from=builder /src/cmd/naiserator_webhook/naiserator_webhook /app/naiserator_webhook
CMD ["/app/naiserator"]
