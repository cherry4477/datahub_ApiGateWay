FROM golang:1.6.0

ENV TIME_ZONE=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TIME_ZONE /etc/localtime && echo $TIME_ZONE > /etc/timezone

COPY . /go/src/github.com/asiainfoLDP/datahub_ApiGateWay

WORKDIR /go/src/github.com/asiainfoLDP/datahub_ApiGateWay

RUN go build

EXPOSE 8092

CMD ["sh", "-c", "./datahub_ApiGateWay"]