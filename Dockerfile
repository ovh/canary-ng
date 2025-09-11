# builder
FROM golang:1.24-alpine AS builder

# install git to execute git command within Makefile
RUN apk add make git
WORKDIR /usr/src/app
COPY Makefile .
COPY VERSION .
# copy .git to execute git command within Makefile
COPY .git .git
COPY src src
RUN make


# run
FROM alpine:3.21

COPY --from=builder /usr/src/app/bin/canary-ng /usr/local/bin/canary-ng
COPY docker-entrypoint.sh /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]
