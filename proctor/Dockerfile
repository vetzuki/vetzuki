# Exam image
FROM ubuntu:latest
RUN apt update -o "Acquire::Check-Date=no" && \
    apt dist-upgrade -y -o "Acquire::Check-Date=no" && \
    apt install -y openssh-server iproute2 net-tools curl bash iputils-ping

RUN mkdir /app
ADD ping_monitor /app

ENTRYPOINT ["/app/ping_monitor"]
CMD ["/app/ping_monitor"]
