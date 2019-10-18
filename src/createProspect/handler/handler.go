package handler

/*
createProspect Lambda

* create a Prospect in the DB
* create a Prospect in LDAP
* send an email to the Prospect with their link

*/
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	// "github.com/aws/aws-sdk-go/service/s3"
	"github.com/vetzuki/vetzuki/auth"
	"github.com/vetzuki/vetzuki/model"
	"log"
)

// ScreeningRequest : Information required to define a screening
type ScreeningRequest struct {
	Name       string `json:"name"`
	Role       string `json:"role"`
	Email      string `json:"email"`
	EmployerID int64  `json:"employerID"`
}

// ScreeningResponse : Response body payload for a success
type ScreeningResponse struct {
	EmployerProspect *model.EmployerProspect `json:"employerProspect"`
	Name             string                  `json:"name"`
	Email            string                  `json:"email"`
}

const (
	defaultExam = int64(1)
	sender      = "hello@poc.vetzuki.com"
	redeemURL   = "https://www.poc.vetzuki.com/p"
	htmlMessage = `
	<html>
	<body>
	Someone's excited to get to know you! There's a screening waiting for you.
	<p>Follow the link below to get started.
	<p><a href="%s/%s">%s/%s</a>
	</body>
	</html>`
	textMessage = `
	Copy and paste this link: %s/%s`
)

var (
	invalidPayload = events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       "invalid payload",
	}
	serverError = events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       "server error",
	}
	unauthorizedError = events.APIGatewayProxyResponse{
		StatusCode: 403,
		Body:       "unauthorized",
	}
)

// sendEmail : Send greeting email to a prospect
func sendEmail(email, prospectURL string) bool {
	log.Printf("debug: preparing email to %s", email)
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
				aws.String(email),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(fmt.Sprintf(htmlMessage, redeemURL, prospectURL, redeemURL, prospectURL)),
				},
				Text: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(fmt.Sprintf(textMessage, redeemURL, prospectURL)),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charSet),
				Data:    aws.String("You've got a screening from VetZuki!"),
			},
		},
		Source: aws.String(sender),
	}
	result, err := emailService.SendEmail(emailMessage)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				log.Printf("error: while sending email to %s: %v, %v", email, ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				log.Printf("error: while sending email to %s: %v, %v", email, ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				log.Printf("error: while sending email to %s: %v, %v", email, ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				log.Printf("error: while sending email to %s: %v", email, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Printf("error: while sending email to %s: %v", email, err.Error())
		}
	}
	log.Printf("info: sent email to %s: %s", email, result.GoString())
	return true
}

// Handler : Creates a DB record, LDAP user, and notifies the Prospect of a screening
func Handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	accessToken, ok := auth.ExtractToken(r.Headers)
	if !ok {
		log.Printf("error: Missing token")
		return unauthorizedError, fmt.Errorf("invalid or missing token")
	}
	if _, ok := auth.ValidateToken(accessToken); !ok {
		return unauthorizedError, fmt.Errorf("invalid token")
	}

	var screening ScreeningRequest
	log.Printf("debug: create prospect request %#v\n", r.Body)
	err := json.Unmarshal([]byte(r.Body), &screening)
	log.Printf("debug: creating employer %d screening for %s", screening.EmployerID, screening.Email)
	if err != nil {
		log.Printf("error: failed to unmarshal JSON: %s", err)
		return invalidPayload, err
	}

	log.Printf("debug : creating %s for employer %d", screening.Email, screening.EmployerID)
	employerProspect, ok := model.CreateEmployerProspect(
		screening.EmployerID,
		defaultExam,
		screening.Name,
		screening.Email,
		screening.Role,
	)
	if !ok {
		log.Printf("error: unable to create screening for %s with employer %d", screening.Email, screening.EmployerID)
		return serverError, err
	}
	if !sendEmail(screening.Email, employerProspect.Prospect.URL) {
		log.Printf("error: failed to send email to %s", screening.Email)
	}
	response := ScreeningResponse{
		EmployerProspect: employerProspect,
		Name:             screening.Name,
		Email:            screening.Email,
	}
	body := bytes.Buffer{}
	encoder := json.NewEncoder(&body)
	log.Printf("debug: creating screening response")
	if err := encoder.Encode(&response); err != nil {
		log.Printf("error : unable to encode prospect %s for employer %d to json: %s", screening.Email, screening.EmployerID, err)
		return serverError, err
	}
	log.Printf("debug: responding with %s", body.String())
	return events.APIGatewayProxyResponse{
		Body:       body.String(),
		StatusCode: 200,
	}, nil
}
