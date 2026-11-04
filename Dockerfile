# Build the pinned Go binary with `make build` before building this image.
FROM scratch
COPY bin/tsunami /tsunami
COPY bin/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65532:65532
ENTRYPOINT ["/tsunami"]
