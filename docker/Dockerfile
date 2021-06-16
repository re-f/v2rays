FROM golang:1.14.15-alpine3.13 
LABEL maintainer "V2Fly Community <dev@v2fly.org>"

# install v2rayS
WORKDIR /etc
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add git
RUN git clone https://gitee.com/erica_z/v2rays.git /etc/v2rays
WORKDIR /etc/v2rays/
ENV GOPROXY="goproxy.cn,goproxy.io,direct"
RUN pwd && go build -o /usr/local/bin

ARG TARGETPLATFORM
ARG TAG
COPY v2ray.sh /tmp/v2ray.sh

WORKDIR /tmp
RUN set -ex \
	&& apk add --no-cache tzdata openssl ca-certificates \
	&& mkdir -p /etc/v2ray /usr/local/share/v2ray /var/log/v2ray \
	&& chmod +x /tmp/v2ray.sh \
	&& /tmp/v2ray.sh "${TARGETPLATFORM}" "${TAG}"

# CMD [ "/usr/bin/v2ray", "-config", "/etc/v2ray/config.json" ]
ENTRYPOINT ["v2rayS", "server", "--config","/etc/v2rays/config.json","--template","/etc/v2rays/config.json.tmpl","--subscribeUrl"] 

