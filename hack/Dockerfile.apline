FROM alpine:3.18

RUN add update --no-cache && apk add xfsprogs-extra sg3_utils lsblk blkid gcompat kmod-libs qemu-img

ADD /bin/extrootfs /usr/bin/
ENTRYPOINT ["/usr/bin/extrootfs"]