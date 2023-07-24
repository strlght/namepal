FROM --platform=$BUILDPLATFORM golang:1.20-alpine as builder

COPY --from=tonistiigi/xx:golang / /
ARG TARGETPLATFORM
RUN apk add --no-cache musl-dev git gcc
ADD . /src
WORKDIR /src
ENV GO111MODULE=on
RUN cd cmd/manager && go env && go build -v

FROM alpine:latest

WORKDIR /app/
COPY --from=builder /src/cmd/manager/manager .

ENTRYPOINT ["/app/manager"]

