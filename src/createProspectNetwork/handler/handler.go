package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/vetzuki/vetzuki/auth"
	lambdaEvents "github.com/vetzuki/vetzuki/events"
	"github.com/vetzuki/vetzuki/model"
	"log"
	"os"
)

const (
	envTeeshAPIKey = "TEESH_API_KEY"
)

var (
	teeshAPIKey = ""
)

func init() {
	teeshAPIKey = os.Getenv(envTeeshAPIKey)
	if len(teeshAPIKey) == 0 {
		log.Fatalf("fatal: %s must be defined", envTeeshAPIKey)
	}
}

// ProspectNetworkRequest : Request for a network which an EC2 instance will host
type ProspectNetworkRequest struct {
	ProspectURLID string `json:"prospectURLID"`
	EC2InstanceID string `json:"ec2InstanceID"`
}

// Handler : Create a ProspectNetwork for a prospect on an EC2Instance
func Handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	apiKey, ok := auth.ExtractToken(r.Headers)
	if !ok {
		log.Printf("error: failed to find API KEY")
		return lambdaEvents.AccessDenied, fmt.Errorf("error: unable to extract API Key")
	}
	if !auth.ValidateAPIKey(apiKey) {
		log.Printf("error: failed to authenticate API Key %s", apiKey)
		return lambdaEvents.AccessDenied, fmt.Errorf("error: unable to validate client key %s", apiKey)
	}
	var prospectNetworkRequest ProspectNetworkRequest
	err := json.Unmarshal([]byte(r.Body), &prospectNetworkRequest)
	if err != nil {
		log.Printf("error: failed to deserialize prospect network request: %s", err)
		return lambdaEvents.BadRequest, err
	}
	log.Printf("debug: locating prospect %s", prospectNetworkRequest.ProspectURLID)
	prospect, ok := model.GetProspect(prospectNetworkRequest.ProspectURLID)
	if !ok {
		log.Printf("error: prospect %s not found", prospectNetworkRequest.ProspectURLID)
		return lambdaEvents.NotFound, fmt.Errorf("error: prospect %s not found", prospectNetworkRequest.ProspectURLID)
	}
	log.Printf("debug: creating network for %d.%s", prospect.ID, prospect.URL)
	prospectNetwork, ok := model.CreateProspectNetwork(
		prospect.ID,
		prospectNetworkRequest.EC2InstanceID,
	)
	if !ok {
		log.Printf("error: unable to create network for %d.%s", prospect.ID, prospect.URL)
		return lambdaEvents.ServerError, fmt.Errorf("error: unable to create network for %s", prospect.URL)
	}
	prospectNetworkJSON, err := json.Marshal(&prospectNetwork)
	if err != nil {
		log.Printf("error: failed to marshal prospect network: %s", err)
		return lambdaEvents.ServerError, err
	}
	return events.APIGatewayProxyResponse{
		Body:       string(prospectNetworkJSON),
		StatusCode: 200,
	}, nil
}
