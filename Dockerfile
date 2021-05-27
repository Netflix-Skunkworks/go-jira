FROM golang

COPY . /go/src/gopkg.in/Netflix-Skunkworks/go-jira.v1
WORKDIR /go/src/gopkg.in/Netflix-Skunkworks/go-jira.v1

RUN make

ENTRYPOINT [ "./jira" ]
