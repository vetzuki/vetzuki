# Exam image
FROM ubuntu:latest
RUN apt update -o "Acquire::Check-Date=no" && \
  apt dist-upgrade -o "Acquire::Check-Date=no" -y && \
  apt install -y openssh-server iproute2 net-tools update-motd
ADD docker-entrypoint.sh /usr/local/bin/
ADD sshd_config /etc/ssh/sshd_config
RUN rm -rf /etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_dsa_key

RUN mkdir /root/.ssh
ADD rsa.pub /root/.ssh/authorized_keys
RUN chmod 0700 /root/.ssh && chmod 0600 /root/.ssh/authorized_keys

# Clear out all existing MOTD information
RUN rm -rf /etc/update-motd.d/* /etc/motd /etc/issue /etc/legal && update-motd

# Add proctor user
RUN useradd -ms /bin/bash proctor

EXPOSE 22
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/usr/sbin/sshd", "-D"]
