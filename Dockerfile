FROM golang:latest

RUN wget https://github.com/koofr/go-pin/releases/download/v1.10/go-pin.sh -O /usr/bin/go-pin && \
    chmod +x /usr/bin/go-pin

RUN mkdir -p /go/src/github.com/sroze/kubernetes-load-balancer-proxifier
ADD . /go/src/github.com/sroze/kubernetes-load-balancer-proxifier/

WORKDIR /go/src/github.com/sroze/kubernetes-load-balancer-proxifier

RUN cat versions | go-pin reset
RUN go get
RUN go build -o main .

CMD ["/app/main"]
