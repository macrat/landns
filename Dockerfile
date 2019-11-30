FROM golang:latest AS builder

ARG arch="amd64"
ARG armv
WORKDIR /go/src/github.com/macrat/landns

COPY . .

RUN go get -d && CGO_ENABLED=0 GOARCH=${arch} GOARM=${armv} go build -o /landns .


FROM scratch

COPY --from=builder /landns /landns

EXPOSE 53/udp
EXPOSE 9353/tcp
ENTRYPOINT ["/landns"]
