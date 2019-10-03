package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vetzuki/vetzuki/auth"
	lambdaEvents "github.com/vetzuki/vetzuki/events"
	"github.com/vetzuki/vetzuki/model"
	"log"
	"os"
)

const (
	envTeeshAPIKey = "TEESH_API_KEY"
)

// ExamStateUpdate : Update the state of a Prospect URL
type ExamStateUpdate struct {
	ProspectURL    string `json:"prospectURL"`
	ScreeningState int    `json:"screeningState"`
}

var teeshAPIKey = ""

func init() {
	teeshAPIKey = os.Getenv(envTeeshAPIKey)
	if len(teeshAPIKey) == 0 {
		log.Fatalf("fatal: %s must be defined", envTeeshAPIKey)
	}
}

// Handler : Handle an ExamStateUpdate request
func Handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	apiKey, ok := auth.ExtractToken(r.Headers)
	if !ok {
		return lambdaEvents.AccessDenied,
			fmt.Errorf("error: unable to extract API Key")
	}
	if teeshAPIKey != apiKey {
		return lambdaEvents.AccessDenied,
			fmt.Errorf("error: unable to validate client key %s", apiKey)
	}
	prospectURLID := r.PathParameters["prospectURLID"]
	if len(prospectURLID) == 0 {
		return lambdaEvents.BadRequest, fmt.Errorf("erro: missing prospectURLID parameter")
	}

	var examStateUpdate ExamStateUpdate
	err := json.Unmarshal([]byte(r.Body), &examStateUpdate)
	if err != nil {
		return lambdaEvents.BadRequest,
			fmt.Errorf("error: failed to decode json: %s", err)
	}
	prospect, ok := model.GetProspect(prospectURLID)
	if !ok {
		return lambdaEvents.NotFound,
			fmt.Errorf("error: unable to find prospect %s", prospectURLID)
	}
	prospect.ScreeningState = examStateUpdate.ScreeningState
	ok = prospect.SetScreeningState(examStateUpdate.ScreeningState)
	if !ok {
		return lambdaEvents.ServerError,
			fmt.Errorf("error: failed to update screening state to %d for %s",
				examStateUpdate.ScreeningState,
				prospectURLID)
	}
	prospectJSON, err := json.Marshal(&prospect)
	if err != nil {
		log.Printf("error: failed to marshal prospect: %s", err)
		return lambdaEvents.ServerError,
			fmt.Errorf("error: failed to render")
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(prospectJSON),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
