
FROM alpine:3.12.3

LABEL maintainer="eirture@gmail.com"

ARG WALLE_VERSION="v0.0.1"

ADD https://bizseer-public.oss-cn-beijing.aliyuncs.com/release/walle/${WALLE_VERSION}/walle /usr/local/bin/walle

RUN chmod a+x /usr/local/bin/walle
