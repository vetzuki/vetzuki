# Exam image
FROM ubuntu:latest

RUN apt update && apt dist-upgrade -y && apt install -y openssh-server iproute2 net-tools
ADD docker-entrypoint.sh /usr/local/bin/
ADD exam_sshd_config /etc/ssh/sshd_config
RUN rm -rf /etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_dsa_key

RUN mkdir /root/.ssh
ADD exam_container_rsa.pub /root/.ssh/authorized_keys
RUN chmod 0700 /root/.ssh && chmod 0664 /root/.ssh/authorized_keys

# Add proctor user
RUN useradd -ms /bin/bash proctor

EXPOSE 22
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/usr/sbin/sshd", "-D"]
