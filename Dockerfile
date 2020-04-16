FROM golang:buster

RUN git clone https://github.com/pyx-partners/dmgd $GOPATH/src/github.com/pyx-partners/dmgd
WORKDIR $GOPATH/src/github.com/pyx-partners/dmgd
RUN go build .

COPY dmgd.conf /root/.dmgd/dmgd.conf

CMD ["./dmgd"]


