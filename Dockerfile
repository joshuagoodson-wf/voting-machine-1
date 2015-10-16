FROM golang

ADD ./Godeps/_workspace/src /go/src/
ADD . /go/src/github.com/joshgoodson/voting-machine

WORKDIR /go/src/github.com/joshgoodson/voting-machine

RUN go install

ENTRYPOINT /go/bin/voting-machine
