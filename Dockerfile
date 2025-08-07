FROM golang:1.20.6-alpine3.18 as builder

WORKDIR /data/PrometheusAlert

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache gcc g++ sqlite-libs make git

ENV GO111MODULE on

ENV GOPROXY https://goproxy.io

COPY . /data/PrometheusAlert

RUN make build

# -----------------------------------------------------------------------------
FROM alpine:3.18

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache tzdata sqlite-libs curl sqlite && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata && \
	mkdir -p /app/logs

HEALTHCHECK --start-period=10s --interval=20s --timeout=3s --retries=3 \
    CMD curl -fs http://localhost:8080/health || exit 1

WORKDIR /app

COPY --from=builder /data/PrometheusAlert/PrometheusAlert .

COPY db/PrometheusAlertDB.db /opt/PrometheusAlertDB.db

COPY conf/app-example.conf conf/app.conf

COPY db db

COPY static static

COPY views views

COPY docker-entrypoint.sh docker-entrypoint.sh

ENTRYPOINT [ "/bin/sh", "/app/docker-entrypoint.sh" ]
