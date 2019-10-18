package main

/*
createProspect Lambda

* create a Prospect in the DB
* create a Prospect in LDAP
* send an email to the Prospect with their link

*/
import (
	"./handler"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler.Handler)
}
