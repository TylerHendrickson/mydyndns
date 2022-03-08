# syntax=docker/dockerfile:1

FROM scratch

COPY mydyndns /usr/bin/mydyndns

VOLUME /config

ENTRYPOINT ["/usr/bin/mydyndns"]
