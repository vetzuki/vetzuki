# VettingWare - VooC

Finds the right technical candidate

# POC 

## Give this machine an IP address
* Log commands issued
* Log time between commands
* Log time from login until the IP is up

### Feature wants
* Time limit
* Preferred answer forms

## Interactive SSH Component

* Use EC2 instance as proxy to Container
* EC2 auth against LDAP
* Create audit log and `tee` prospect to Container

```mermaid
sequenceDiagram
  Note left of Prospect: Auth with LDAP
  Prospect ->> ProspectURL: Connect with SSH client
  ProspectURL ->> NetworkLB: Connect to EC2 instance
  NetworkLB ->> EC2DockerHost: Login using teesh to Docker Container
  Note left of Container: ec2:///tmp/ssh-$USER.log
  EC2DockerHost ->> Container: Begin interview
  ```
### Prerequsites

* Build docker container
* On EC2 host: `docker network create Prospect.ShortURL`
* Docker launch: `docker run -d --net Prospect.ShortURL --net ProxyNet`

ProxyNet allows the EC2 instance to connect the user to the Docker container.

### Networking

  * Docker container network 10.0.0.0/8
  * Create container on LDAP auth
  * Create home on LDAP auth
  * Create SSH keypair on LDAP auth
  * Mount public key as `container:///root/.ssh/authorized_keys`
  * Name container `Prospect.ShortURL` without scheme
  * Store IP of launched container in `$prospectContainerIP`
  * Begin `ping` of `10.0.1.1/24` which candidate must assign
  * When `ping` is up, `docker exec Prospect.shortURL wall "Way to go. Session will end immediately."`
  * SSH to `root@$prospectContainerIP` using tee to log to `/tmp/$prospect.ShortURL-$propsectContainerIP.log`
  * In `api://dockerd`, when container `Prospect.ShortURL` has been alive for `30m`, kill it


### Technical Design
```mermaid
sequenceDiagram
  participant Candidate
  participant InterviewLancherLambda
  participant bit.ly/OneTimeURL
  participant Employer
  Employer->Candidate: Send bit.ly/OneTimeURL
  Candidate->http://bit.ly/XoX: GET http://bit.ly/XoX
```

# MVP

* Use Kubernetes to have N spare containers launched
* Remove apt, dpkg, wget, curl, lynx, vim, nano, cat from the system

## Budget

| Item | Monthly Cost | Notes |
|------|--------------|-------|
| t3.micro Host | 7.736 | Hosts up to 100 docker containers (100*8MB) |
| NetworkLB | $31.15 | Estimated at 3.686 NLCU (2MB per session) |