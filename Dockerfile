FROM golang:1.12-stretch AS builder
ENV GO111MODULE on
ENV CGO_ENABLED 0
ENV GOFLAGS -mod=vendor
COPY . /go/src
WORKDIR /go/src
RUN go build -v -o goyoubbs -ldflags "-s -w" .

# ===========================

FROM scratch
LABEL maintainer="ohko <ohko@qq.com>"
COPY --from=builder /go/src/goyoubbs /
COPY static/ /static/
COPY config/config.yaml /config/config.yaml
COPY view/ /view/
WORKDIR /
EXPOSE 8088
ENTRYPOINT ["/goyoubbs"]
