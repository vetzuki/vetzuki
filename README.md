VetZuki lets employers vet prospects before spending time on costly phone interviews. It is unbiased and allows a candidate to be free to do work like they would in a real-world situation.


## Design Documents

* [Processes and diagrams](https://docs.google.com/presentation/d/1pjBTq506C-PzuDA8mbtWvYD32PWxfomcmMjJwMGKrbE/edit#slide=id.p)
* [Stories and features](https://docs.google.com/document/d/1LnkD2yBv7YxoU-yW1Sgu4tGjlYaKoRUYMt5UrUKDGmw/edit)

# Building and Deploying

Running `make examContainer` or `make proctorContainer` will create the base containers. Executing `eval $(aws ecr get-login --no-include-email)` followed by `make pushExamContainer` and `make pushProspectContainer` allows the POC EC2 container to have access to the containers.

A final step in preparing an environment is creating the `own_config` binary. The makefile allows `make ownConfig` to accomplish this. Following this with `make configurePOC` will install of the freshly built components.

# Source layout

## /vetzuki-host

Create the EC2 environment to run software

```
terraform poc.tf
```

Create the software environment on the EC2 environment. This also generates self-signed certificates using the `role/ca`.

```
ansible-playbook -i inventory poc.yml
```

Create the software environment and reset the LDAP seeds

```
ansible-playbook -i inventory poc.yml -e '{"drop_prospects_ou":true}'
```
## /src

Database schema, AWS deployment descriptor, and development environment setup playbooks. More details available in [README](src/README.md)

## /exam

Includes `Dockerfile` for building the examination container. This is what a prospect logs in to.

## /proctor

Includes `Dockerfile` for building the proctor container. This is what monitors the exam container for completion.

## /src/vetzuki

The Go codebase for vetzuki, `github.com/vetzuki/vetzuki` is housed here. It should be linked into the `$GOPATH` to run go tools on the codebase. It has its own [README](src/vetzuki/README.md).

## /src/ui

VetZuki's UI code is stored here and built with `ansible-playbook build.yml`. Deploying the environment to run the code is handled by `terraform api_gateway.tf`. More detail is available in the sub-project's [README](src/ui/README.md).
