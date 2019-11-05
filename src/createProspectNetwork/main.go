package main
/* 
createProspectnetwork Lambda

* Create a ProspectNetwork in the DB
* Increment network counter in Redis
* Increment SSH port counter in Redis
*/

import (
    "./handler"
    "github.com/aws/aws-lambda-go/lambda"
)

func main() {
    lambda.Start(handler.Handler)
}
