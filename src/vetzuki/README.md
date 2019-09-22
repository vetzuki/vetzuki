## Environment Setup

From the project root, use ansible:

```
ansible-playbook setup_development_environment.yml
```

### Database

To provision the schema
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