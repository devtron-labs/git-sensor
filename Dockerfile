FROM golang:1.20-alpine3.18 AS build-env

RUN apk add --no-cache git gcc musl-dev
RUN apk add --update make
RUN go install github.com/google/wire/cmd/wire@latest
WORKDIR /go/src/github.com/devtron-labs/git-sensor
ADD . /go/src/github.com/devtron-labs/git-sensor/
RUN GOOS=linux make

FROM alpine:3.18
COPY ./git-ask-pass.sh /git-ask-pass.sh
RUN chmod +x /git-ask-pass.sh
RUN apk add --no-cache ca-certificates
RUN apk add git --no-cache
RUN apk add openssh --no-cache
COPY --from=build-env  /go/src/github.com/devtron-labs/git-sensor/git-sensor .
COPY --from=build-env  /go/src/github.com/devtron-labs/git-sensor/scripts/ .

RUN adduser -D devtron
RUN chown -R devtron:devtron ./git-sensor
USER devtron

CMD ["./git-sensor"]
