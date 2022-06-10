# syntax=docker/dockerfile:1

FROM gcr.io/distroless/static
COPY mydyndns /usr/bin/mydyndns
VOLUME /config
ENTRYPOINT ["/usr/bin/mydyndns"]
