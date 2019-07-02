# Start by building the application.
FROM golang:1.12-alpine3.10 AS build
WORKDIR /grepples
COPY . .
RUN apk add --no-cache git
RUN CGO_ENABLED=0 GOOS=linux go build

# Now copy it into our base image.
FROM alpine:3.10
RUN apk add --no-cache ca-certificates
COPY --from=build /grepples/grepples /grepples
USER 1000
ENTRYPOINT ["/grepples"]
