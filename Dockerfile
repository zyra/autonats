FROM golang:1.15-alpine as builder
RUN apk add git make
WORKDIR /root/wd
COPY . .
RUN make -j$(nproc)

FROM alpine
COPY --from=builder /root/wd/bin/autonats_linux_amd64 /usr/bin/autonats
RUN chmod +x /usr/bin/autonats

ENTRYPOINT ["autonats"]