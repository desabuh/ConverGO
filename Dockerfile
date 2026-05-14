FROM golang:1.22.3 as build-stage
WORKDIR /ConverGo
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o out

FROM build-stage AS test-stage

RUN make test

# fast dev image (skip unit tests validation)
FROM gcr.io/distroless/base-debian11 AS dev-release-stage

WORKDIR /

COPY --from=build-stage /ConverGo/out ConverGo

ENTRYPOINT ["/ConverGo"]


FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=test-stage /ConverGo/out ConverGo


ENV COMMAND_FILE=/test_domain_concurrent/command.txt

ENV DNS=peer

ENV PEER_ID=1
ENV PEER_NAME=John
ENV LOCAL_PORT=8080



EXPOSE 8080


ENTRYPOINT ["/ConverGo"]

