# Deploying to AWS

This will build the Lambda, create the APIGateway, APIG integration template, and the APIG lambda integration for `/p/{prospectURLID}`

```
ansible-playbook -K deploy.yml
```

# Preparing Development environment

This does not put web service handlers locally.

```
cd vetzuki
ansible-playbook -K setup_development_environment.yml
ansible-playbook -K schema.yml
```



