# Deploying to AWS

This will build the Lambda, create the APIGateway, APIG integration template, and the APIG lambda integration for `/p/{prospectURLID}`

```
ansible-playbook -K deploy.yml
```

# Deploy database schema

This deploys the schema locally and to remote environments, like POC.

```
ansible-playbook -i inventory schema.yml
```

# Preparing Development environment

This does not put up web service handlers locally.

```
cd vetzuki
# Create containers and seed LDAP
ansible-playbook -K setup_development_environment.yml
# Create database and seed as needed
ansible-playbook -K schema.yml
```

# Testing locally

After completing a new Lambda handler, or to integration test the UI with real handlers, the development server should be used.

```
cd devServer && go run main.go
```

This will launch the development server on port `9000` with mock authentication credentials for the `admin` user with the email `admin@localhost`.

