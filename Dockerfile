FROM docker.io/library/golang:1.15-alpine as builder

MAINTAINER John Bass <john_bass2@cable.comcast.com>

WORKDIR /src

ARG VERSION
ARG GITCOMMIT
ARG BUILDTIME


RUN apk add --no-cache --no-progress \
    ca-certificates \
    make \
    git \
    openssh \
    gcc \
    libc-dev \
    upx

RUN go get github.com/geofffranks/spruce/cmd/spruce && chmod +x /go/bin/spruce
COPY . .
RUN make test release

FROM alpine:3.12.1

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/dms /src/dms.yaml /src/deploy/packaging/entrypoint.sh /go/bin/spruce /src/Dockerfile /src/NOTICE /src/LICENSE /src/CHANGELOG.md /
COPY --from=builder /src/deploy/packaging/dms.yaml /tmp/dms.yaml

RUN mkdir /etc/dms/ && touch /etc/dms/dms.yaml && chmod 666 /etc/dms/dms.yaml

USER nobody

ENTRYPOINT ["/entrypoint.sh"]

EXPOSE 11000

CMD ["/dms"]
