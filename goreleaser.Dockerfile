# syntax=docker/dockerfile:1

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY mydyndns /usr/bin/mydyndns
VOLUME /config
ENTRYPOINT ["/usr/bin/mydyndns"]
