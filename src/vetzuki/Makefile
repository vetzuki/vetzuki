VERSION=0.0.0

test: setup go-test

setup:
	cd .. ; \
	ansible-playbook -K setup_development_environment.yml ; \
	ansible-playbook -e '{"db_seed": true, "db_reset": true}' schema.yml 

go-test:
	go test -v auth/ ldap/ model/
