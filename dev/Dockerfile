FROM golang:1.23-alpine AS builder

WORKDIR /build

ADD go.mod go.sum /build/
RUN go mod download

ADD . /build/
RUN ls -al

RUN go build ./cmd/fleeting-plugin-hetzner

FROM gitlab/gitlab-runner:alpine-v17.4.0

COPY --from=builder /build/fleeting-plugin-hetzner /bin/fleeting-plugin-hetzner
