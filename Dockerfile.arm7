FROM golang:latest AS builder

WORKDIR /go/src/github.com/macrat/landns

COPY . .

RUN go get -d && \
    GOARCH=arm \
    GOARM=7 \
    go build \
    -tags netgo \
    -installsuffix netgo \
    --ldflags '-extldflags -static' \
    -o /landns \
    .


FROM scratch

COPY --from=builder /landns /landns

EXPOSE 53/udp
EXPOSE 9353/tcp
ENTRYPOINT ["/landns"]
