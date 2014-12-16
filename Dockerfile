FROM golang:1.3-onbuild

# Skip the annoying "+ exec app" dumped on STDOUT during `go-wrapper run`
RUN sed -i -e 's/set -x; //' /usr/local/bin/go-wrapper
