# Start by building the application.
FROM golang:1.13-alpine AS build
ADD . /src
RUN chown -R 250:users /src
USER 250
WORKDIR /src
ENV GOCACHE=/tmp/.go-cache
RUN go build
USER root
RUN chown 0:0 /src/grepples


# Now copy it into our base image.
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build --chown=0:0 /src/grepples /grepples
USER 250
ENTRYPOINT ["/grepples"]
