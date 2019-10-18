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

// Handler : Returns a orospect collection
func Handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	accessToken, ok := auth.ExtractToken(r.Headers)
	if !ok {
		return lambdaEvents.AccessDenied, nil
	}
	claims, ok := auth.ValidateToken(accessToken)
	if !ok {
		return lambdaEvents.AccessDenied, nil
	}
	log.Printf("debug: authorized %s employer", claims.Email)

	employer, ok := model.GetEmployerByEmail(claims.Email)
	if !ok {
		log.Printf("error: unable to find employer %s", claims.Email)
		return lambdaEvents.NotFound, nil
	}
	prospectScores, ok := employer.FindProspectScores()
	if !ok {
		log.Printf("error: unable to find prospect scores for %s", employer.Email)
		return lambdaEvents.NotFound, nil
	}
	prospectScoresJSON, err := json.Marshal(&prospectScores)
	if err != nil {
		log.Printf("error: while serializing prospectScores: %s", err)
		return lambdaEvents.ServerError, err
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(prospectScoresJSON),
	}, nil
}
