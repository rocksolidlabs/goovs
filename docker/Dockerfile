FROM golang:1.6.0

RUN go get -u github.com/tools/godep
#RUN mkdir -p /go/src/github.com/kopwei/
COPY files/restore_dep.sh /files/restore_dep.sh
COPY files/runtest.py /runtest.py

ENTRYPOINT ["/usr/bin/python", "/runtest.py"]