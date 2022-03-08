# syntax=docker/dockerfile:1

FROM scratch

COPY mydyndns /usr/bin/mydyndns

ENTRYPOINT ["/usr/bin/mydyndns"]
