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
RUN cd cmd/nebula && go build -a -installsuffix cgo -o nebula

FROM alpine:3.14
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/cmd/nebula/nebula /app/nebula
CMD ["/app/nebula"]
