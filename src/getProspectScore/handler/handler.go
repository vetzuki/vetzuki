package handler

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/vetzuki/vetzuki/auth"
	lambdaEvents "github.com/vetzuki/vetzuki/events"
	"github.com/vetzuki/vetzuki/model"
	"log"
)

// Handler : Get prospect score by URLID
func Handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	accessToken, ok := auth.ExtractToken(r.Headers)
	if !ok {
		log.Printf("error: no accessToken found")
		return lambdaEvents.AccessDenied, nil
	}
	claims, ok := auth.ValidateToken(accessToken)
	if !ok {
		log.Printf("error: unauthorized claimant: %s", claims.Email)
		return lambdaEvents.AccessDenied, nil
	}
	prospectURLID := r.PathParameters["prospectURLID"]
	prospect, ok := model.GetProspect(r.PathParameters["prospectURLID"])
	if !ok {
		log.Printf("error: unknown prospect %s", prospectURLID)
		return lambdaEvents.NotFound, nil
	}
	score, ok := prospect.GetScore()
	if !ok {
		log.Printf("error: unable to get prospect %s score", prospectURLID)
		return lambdaEvents.NotFound, nil
	}
	scoreJSON, err := json.Marshal(&score)
	if err != nil {
		log.Printf("error: while marshalling score to JSON: %s", err)
		return lambdaEvents.ServerError, err
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(scoreJSON),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}
