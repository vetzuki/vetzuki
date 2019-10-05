PROJECT_ROOT = $(HOME)/development/vetZuki

ANSIBLE_POC_INVENTORY = inventory
ANSIBLE_PLAYBOOKCMD = ansible-playbook
ANSIBLE_POC_MANIFEST = poc.yml
TERRAFORMCMD = ./bin/terraform
TFSTATE_QUERYCMD = ./tfstate_query.py

EXAM_ECR = 588487667149.dkr.ecr.us-west-2.amazonaws.com/vetzuki-exam
PROCTOR_ECR = 588487667149.dkr.ecr.us-west-2.amazonaws.com/vetzuki-proctor
EXAM_CONTAINER_RELEASE = 0.1
PROCTOR_CONTAINER_RELEASE = 0.1

init:
	$(TERRAFORMCMD) init
# Provisioning
createPOCInventory:
	$(eval IP := $(shell $(TFSTATE_QUERYCMD) aws_eip.poc.public_ip | sed -e 's/^.*=//'))
	$(shell rm -f poc_inventory)
	$(file >>poc_inventory,[poc])
	$(file >>poc_inventory,poc_exam_host ansible_host=${IP} ansible_user=ec2-user)

buildPOC: createPOCInventory provisionPOC configurePOC

provisionPOC:
	$(TERRAFORMCMD) apply

configurePOC:
	$(ANSIBLE_PLAYBOOKCMD) -i $(ANSIBLE_POC_INVENTORY) $(ANSIBLE_POC_MANIFEST)

# Container build and release
# TODO: Version should derive from tag
buildAll: examContainer proctorContainer

examContainer: 
	docker build -t $(EXAM_ECR):$(EXAM_CONTAINER_RELEASE) $(PROJECT_ROOT)/exam/

proctorContainer:
	docker build -t $(PROCTOR_ECR):$(PROCTOR_CONTAINER_RELEASE) $(PROJECT_ROOT)/proctor/

pushAll: pushExamContainer pushProctorContainer

pushExamContainer: examContainer
	docker push $(EXAM_ECR):$(EXAM_CONTAINER_RELEASE)

pushProctorContainer: proctorContainer
	docker push $(PROCTOR_ECR):$(PROCTOR_CONTAINER_RELEASE)

ecrLogin:
	aws ecr get-login --no-include-email

# Testing features
run: runExamContainer 
runExamContainer:
	docker run -d --name exam vetzuki.com/exam:$(EXAM_CONTAINER_RELEASE)

runProctorContainer:
	docker run -d --name proctor vetzuki.com/proctor:$(PROCTOR_CONTAINER_RELEASE)

clean: docker-kill docker-rm

docker-kill:
	docker kill exam
	
docker-rm:
	docker rm exam

ownConfig:
	cd $(PROJECT_ROOT)/vetzuki-host
	GOOS=linux GOARCH=386 go build -o own_config own_config.go
