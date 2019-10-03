package events

import (
	"github.com/aws/aws-lambda-go/events"
)

var (
	// AccessDenied : 401
	AccessDenied = events.APIGatewayProxyResponse{
		StatusCode: 401,
		Body:       "access denied",
	}
	// NotFound : 404
	NotFound = events.APIGatewayProxyResponse{
		StatusCode: 404,
		Body:       "not found",
	}
	// ServerError : 500
	ServerError = events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       "server error",
	}
	// BadRequest : 400
	BadRequest = events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       "bad request",
	}
)
