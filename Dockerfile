# Start by building the application.
FROM golang:1.11-alpine3.8 AS build
WORKDIR /grepples
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor

# Now copy it into our base image.
FROM alpine:3.8
RUN apk add --no-cache ca-certificates
COPY --from=build /grepples/grepples /grepples
ENTRYPOINT ["/grepples"]
