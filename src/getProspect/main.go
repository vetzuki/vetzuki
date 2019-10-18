package main

/*
getProspect Lambda

* get the prospect URL from the ID parameter of the request
* get the prospect from RDS
* get prospect template from s3://vetzuki.templates/prospect.template
* compile template with Prospect information

*/
import (
	"./handler"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler.Handler)
}
