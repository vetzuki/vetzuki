package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vetzuki/vetzuki/auth"
	"github.com/vetzuki/vetzuki/model"
	"log"
)

var (
	eventAccessDenied = events.APIGatewayProxyResponse{
		StatusCode: 401,
		Body:       "access denied",
	}

	eventNotFound = events.APIGatewayProxyResponse{
		StatusCode: 404,
		Body:       "not found",
	}
	eventServerError = events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       "server error",
	}
)

// Handler : Handle API Gateway integration events
func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	accessToken, ok := auth.ExtractToken(event.Headers)
	if !ok {
		return eventAccessDenied, fmt.Errorf(eventAccessDenied.Body)
	}
	// Blocking call
	claims, ok := auth.ValidateToken(accessToken)
	if !ok {
		return eventAccessDenied, fmt.Errorf(eventAccessDenied.Body)
	}

	employer, ok := model.GetEmployerByEmail(claims.Email)
	if !ok {
		return eventNotFound, fmt.Errorf(eventNotFound.Body)
	}

	employerJSON, err := json.Marshal(employer)
	if err != nil {
		log.Printf("error: failed to create employer JSON: %s", err)
		return eventServerError, err
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(employerJSON),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
