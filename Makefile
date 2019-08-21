ANSIBLE_POC_INVENTORY = inventory
ANSIBLE_PLAYBOOKCMD = ansible-playbook
ANSIBLE_POC_MANIFEST = poc.yml
TERRAFORMCMD = ./bin/terraform
TFSTATE_QUERYCMD = ./tfstate_query.py

init:
	$(TERRAFORMCMD) init
createPOCInventory:
	$(eval IP := $(shell $(TFSTATE_QUERYCMD) aws_eip.poc.public_ip | sed -e 's/^.*=//'))
	$(shell rm -f poc_inventory)
	$(file >>poc_inventory,[poc])
	$(file >>poc_inventory,poc_exam_host ansible_host=${IP} ansible_user=ec2-user)
buildPOC: createPOCInventory provisionPOC conifgurePOC

provisionPOC:
	$(TERRAFORMCMD) apply
configurePOC:
	$(ANSIBLE_PLAYBOOKCMD) -i $(ANSIBLE_POC_INVENTORY) $(ANSIBLE_POC_MANIFEST)

# TODO: Version should derive from tag
buildExamContainer:
	docker build -t vetzuki.com/exam:0 .

run: runExamContainer
runExamContainer:
	docker run -d --name exam vetzuki.com/exam:0

clean: docker-kill docker-rm

docker-kill:
	docker kill exam
	
docker-rm:
	docker rm exam
