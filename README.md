VetZuki lets employers vet prospects before spending time on costly phone interviews. It is unbiased and allows a candidate to be free to do work like they would in a real-world situation.


## Design Documents

* [Processes and diagrams](https://docs.google.com/presentation/d/1pjBTq506C-PzuDA8mbtWvYD32PWxfomcmMjJwMGKrbE/edit#slide=id.p)
* [Stories and features](https://docs.google.com/document/d/1LnkD2yBv7YxoU-yW1Sgu4tGjlYaKoRUYMt5UrUKDGmw/edit)

# Building and Deploying

Running `make examContainer` or `make proctorContainer` will create the base containers. Executing `eval $(aws ecr get-login --no-include-email)` followed by `make pushExamContainer` and `make pushProspectContainer` allows the POC EC2 container to have access to the containers.

A final step in preparing an environment is creating the `own_config` binary. The makefile allows `make ownConfig` to accomplish this. Following this with `make configurePOC` will install of the freshly built components.

