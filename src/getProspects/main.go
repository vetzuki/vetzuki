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

// Handler : Handle request from APIGateway
func Handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("debug: get prospects with query: %#v", r.QueryStringParameters)
	accessToken, ok := auth.ExtractToken(r.Headers)
	if !ok {
		return eventAccessDenied, fmt.Errorf("access denied")
	}
	claims, ok := auth.ValidateToken(accessToken)
	if !ok {
		return eventAccessDenied, fmt.Errorf("access denied")
	}
	employer, ok := model.GetEmployerByEmail(claims.Email)
	if !ok {
		log.Printf("error: failed to find employer %s", claims.Email)
		return eventServerError, fmt.Errorf("unable to find employer %s", claims.Email)
	}
	prospects, ok := employer.FindProspects()
	if !ok {
		log.Printf("error: unable to find prospects for employer %s", employer.Email)
		return eventServerError, fmt.Errorf("unable to find prospects for empler %s", employer.Email)
	}
	log.Printf("debug: founds %d prospects for %s", len(prospects), employer.Email)
	prospectsJSON, err := json.Marshal(prospects)
	if err != nil {
		log.Printf("error: failed to marshal prospects: %s", err)
		return eventServerError, fmt.Errorf("unable to render prospects for employer %s", employer.Email)
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(prospectsJSON),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
