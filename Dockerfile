ARG ALIPNE_VERSION=3.21
ARG GO_VERSION=1.24.0

FROM golang:${GO_VERSION}-alpine${ALIPNE_VERSION} AS builder
WORKDIR /app

RUN --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download

COPY . .
RUN go build -o moniepoint_storage_engine_server cmd/rest/main.go

FROM alpine:${ALIPNE_VERSION}
WORKDIR /app

RUN apk --no-cache add curl

COPY --from=builder /app/moniepoint_storage_engine_server .

ENV HTTP_PORT=8080
ENV DATA_PATH=/app/data
ENV REPLICATION_MODE=leader

RUN mkdir ${DATA_PATH}

EXPOSE ${HTTP_PORT}

CMD ["/app/moniepoint_storage_engine_server"]
