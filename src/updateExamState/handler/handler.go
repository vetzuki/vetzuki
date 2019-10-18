package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/vetzuki/vetzuki/auth"
	lambdaEvents "github.com/vetzuki/vetzuki/events"
	"github.com/vetzuki/vetzuki/model"
	"log"
	"os"
)

const (
	envTeeshAPIKey                = "TEESH_API_KEY"
	envVetzukiEnvironment         = "VETZUKI_ENVIRONMENT"
	emailProspectExamCompleteHTML = `
	<html>
	<body>
	You finished it, you got it done.
	<h1>Next Steps</h1>
	<p>We've let %s know and they'll reach out to you.
	<p>Updates will be posted to %s
	</body>
	</html>
	`  // employer.Name, https://www....prospect.URL
	emailEmployerExamCompleteHTML = `
	<html>
	<head>
	<style>
	.examScore {
		background-color: #a3cfcd;
		text-align: center;
		vertical-align: middle;
		font-family: sans-serif;
		border: 20px solid #677381;
	}
	body {
		background-color: #f9f871;
	}
	</style>
	</head>
	<body>
	<p>%s has completed the screening for the %s role with a score of
	<div class="examScore">
	  <h1>%d</h1>
	</div>
	</body>
	</html>`  // prospect.Name, prospect.Role, score.Score
	emailProspectExamCompleteText = `
	You finished it, you got it done.
	
	# Next Steps

	We've let %s know and they'll reach out to you.
	
	Updates will be posted to %s
	`
	emailEmployerExamCompleteText = `
	%s has completed the screening for the %s role with a score of %d.
	`
)

// ExamStateUpdate : Update the state of a Prospect URL
type ExamStateUpdate struct {
	ProspectURL    string `json:"prospectURL"`
	ScreeningState int    `json:"screeningState"`
}

var (
	teeshAPIKey        = ""
	emailSender        = "hello@poc.vetzuki.com"
	vetzukiProspectURL = "https://www.poc.vetzuki.com/p/"
)

func init() {
	teeshAPIKey = os.Getenv(envTeeshAPIKey)
	if len(teeshAPIKey) == 0 {
		log.Fatalf("fatal: %s must be defined", envTeeshAPIKey)
	}
}

// SendProspectEmail : Send an email to the prospect based on the screening state
func SendProspectEmail(p *model.Prospect, employer *model.Employer, screeningState int) bool {
	var htmlBody, textBody, subject string
	prospectURL := vetzukiProspectURL + p.URL
	switch screeningState {
	case model.ScreeningStateComplete:
		subject = "Mark it! - Vetzuki Exam Complete"
		htmlBody = fmt.Sprintf(emailProspectExamCompleteHTML, employer.Name, prospectURL)
		textBody = fmt.Sprintf(emailProspectExamCompleteText, employer.Name, prospectURL)
	default:
		return false
	}
	return sendEmail(emailSender, p.Email, subject, htmlBody, textBody)
}

// SendEmployerEmail : Send an email to the employer based on the screening state
func SendEmployerEmail(e *model.Employer, prospect *model.Prospect, screeningState int) bool {
	var htmlBody, textBody, subject string
	switch screeningState {
	case model.ScreeningStateComplete:
		score, ok := prospect.GetScore()
		if !ok {
			log.Printf("error: failed to get score for %s", prospect.Email)
			return false
		}
		subject = fmt.Sprintf("%s has completed their VetZuki screening", prospect.Name)
		htmlBody = fmt.Sprintf(emailEmployerExamCompleteHTML, prospect.Name, prospect.Role, int64(score.Score*100.0))
		textBody = fmt.Sprintf(emailEmployerExamCompleteText, prospect.Name, prospect.Role, int64(score.Score*100.0))
	default:
		return false
	}
	return sendEmail(emailSender, e.Email, subject, htmlBody, textBody)
}

func sendEmail(sender, recipient, subject, htmlBody, textBody string) bool {
	log.Printf("debug: preparing email to %s", recipient)
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		log.Printf("error: unable to create aws session: %s", err)
		return false
	}
	charSet := "UTF-8"
	emailService := ses.New(awsSession)
	emailMessage := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(htmlBody),
				},
				Text: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(textBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(sender),
	}
	result, err := emailService.SendEmail(emailMessage)
	if err != nil {
		log.Printf("error: failed to send message from %s to %s: %s", sender, recipient, subject)
		return false
	}
	log.Printf("debug: sent message to %s: %s", recipient, result)
	return true
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
	employer, ok := model.GetEmployer(prospect.EmployerID)
	if !ok {
		log.Printf("error: failed to find prospect %s employer", prospect.URL)
	}
	prospect.ScreeningState = examStateUpdate.ScreeningState
	ok = prospect.SetScreeningState(examStateUpdate.ScreeningState)
	if !ok {
		return lambdaEvents.ServerError,
			fmt.Errorf("error: failed to update screening state to %d for %s",
				examStateUpdate.ScreeningState,
				prospectURLID)
	}
	if !SendProspectEmail(prospect, employer, examStateUpdate.ScreeningState) {
		log.Printf("warning: failed to send prospect email to %s", prospect.Email)
	}
	if !SendEmployerEmail(employer, prospect, examStateUpdate.ScreeningState) {
		log.Printf("warning: failed to send employer email to %s", employer.Email)
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
