FROM golang:1.12-alpine as builder
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

FROM alpine:3.9
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/cmd/naiserator/naiserator /app/naiserator
ENV ACCESS_POLICY_ENABLED=false
CMD ["/app/naiserator", "--access-policy-enabled=$ACCESS_POLICY_ENABLED"]
