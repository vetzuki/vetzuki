## Environment Setup

From the parent directory, use Ansible

```
ansible-playbook setup_development_environment.yml
```

This will create the LDAP, Redis, and Postgres containers. It will also seed the LDAP database using data in `seeds.$ENV.json`.

LDAP testing makes use of the `seeds.$ENV.json` file.

### Database

To provision the schema use the Ansible playbooks in the praent directory.

```
ansible-playbook schema.yml
```

To reset the database

```
ansible-playbook -e '{"db_reset":true}' schema.yml
```

To seed the database (drops all table data):

```
ansible-playbook -e '{"db_seed": true}' schema.yml
```

To reset and seed the database

```
ansible-playbook -e '{"db_reset": true, "db_seed": true}
```