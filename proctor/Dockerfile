# Exam image
FROM ubuntu:latest

RUN apt update && apt dist-upgrade -y && apt install -y openssh-server iproute2 net-tools curl bash iputils-ping


RUN mkdir /app
ADD ping_monitor /app

ENTRYPOINT ["/app/ping_monitor"]
CMD ["/app/ping_monitor"]
