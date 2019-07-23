TERRAFORMCMD = ./bin/terraform

init:
	$(TERRAFORMCMD) init
buildPOC:
	$(TERRAFORMCMD) apply
