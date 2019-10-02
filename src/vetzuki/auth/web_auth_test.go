package auth

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"testing"
)

func TestExtractToken(t *testing.T) {
	expectation := "token"
	apiProxyIntegration := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", expectation),
		},
	}
	if token, ok := ExtractToken(apiProxyIntegration.Headers); !ok || token != expectation {
		t.Fatalf("expected api proxy integration token %s, got %s", expectation, token)
	}

}
