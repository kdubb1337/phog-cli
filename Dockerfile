FROM gcr.io/distroless/static-debian12
COPY phog /usr/local/bin/phog
ENTRYPOINT ["/usr/local/bin/phog"]
