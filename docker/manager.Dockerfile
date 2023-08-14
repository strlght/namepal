FROM --platform=$BUILDPLATFORM golang:1.20-alpine as builder

COPY --from=tonistiigi/xx:golang / /
ARG TARGETPLATFORM
RUN apk add --no-cache musl-dev git gcc
ADD . /src
WORKDIR /src
ENV GO111MODULE=on
RUN cd cmd/manager && go env && go build -v

FROM alpine:3.18.3
LABEL org.opencontainers.image.source https://github.com/strlght/namepal

WORKDIR /app/
COPY --from=builder /src/cmd/manager/manager .

ENTRYPOINT ["/app/manager"]

