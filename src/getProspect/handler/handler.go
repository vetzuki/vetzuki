package handler

/*
getProspect Lambda

* get the prospect URL from the ID parameter of the request
* get the prospect from RDS
* get prospect template from s3://vetzuki.templates/prospect.template
* compile template with Prospect information

*/
import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"strings"
	// "github.com/aws/aws-sdk-go/service/s3"
	lambdaEvents "github.com/vetzuki/vetzuki/events"
	"github.com/vetzuki/vetzuki/model"
	"html/template"
	"log"
	"os"
)

const (
	templateObject = "s3://vetzuki.templates/prospect.template"
	envSSHURL      = "SSH_URL"
)

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta http-equiv="x-ua-compatible" content="ie=edge" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>{{ .Title }}</title>
	<style>
	body {
	  background-color: #f9f871;
	  display: flex;
	  flex-direction: column;
	  flex-wrap: nowrap;
	  align-items: stretch;
	}
	div#prospect-greeting {
	  width: 80%;
	  height: 33%;
	  margin-left: auto;
	  margin-right: auto;
	  margin-bottom: 5px;
	  font-family: sans-serif;
	}
	div#prospect-ssh-url {
		flex-grow: 0;
        margin-left: auto;
        margin-right: auto;
        margin-bottom: 5px;
        background-color: #fff3fa;
        font-family: monospace;
        font-weight: bold;
        text-align: center;
	}
	div#prospect-ssh-password {
		flex-grow: 0;
        margin-left: auto;
        margin-right: auto;
        margin-bottom: 5px;
        background-color: #fff3fa;
        font-family: monospace;
		font-weight: bold;
        text-align: center;
	}
	div#prospect-instructions {
		font-family: sans-serif;
        width: 80%;
        margin-left: auto;
        margin-right: auto;
        height: 20%;
        margin-bottom: 5px;
	}
	div#footer {
		flex: 0 1 50px;
        font-family: sans-serif;
        color: #fff3fa;
        width: 80%;
        margin-left: auto;
        margin-right: auto;
        background-color: #677381
	}
	div#header {
		justify-content: right;
		display: flex;
		flex-direction: row;
		flex: 0 1 auto;
		font-family: sans-serif;
		color: #fff3fa;
		width: 80%;
		margin-left: auto;
		margin-right: auto;
		background-color: #677381
	}
  </style>
  <script>
  function selectText(event) {
	let node =event.target;
	let text = "";
    if (document.body.createTextRange) {
        const range = document.body.createTextRange();
        range.moveToElementText(node);
        text = range.select();
    } else if (window.getSelection) {
        const selection = window.getSelection();
        const range = document.createRange();
        range.selectNodeContents(node);
        selection.removeAllRanges();
        text = selection.addRange(range);
    } else {
        console.warn("Could not select text in node: Unsupported browser.");
	}
	try {
		document.execCommand("copy");
	} catch (e) {
		console.log("failed to copy text")
	}
  }
  </script>
</head>

<body>
<div id="header">
  {{ range .Links }}
    <a href="{{ .HREF }}">{{ .Name }}</a>
  {{ end }}
</div>
<div id="prospect-greeting">Hello {{ .Name }}</div>
<div  id="prospect-instructions">
Connect to the server below to complete
your application for {{ .Role }} at {{ .Employer }}
</div>
<div id="prospect-ssh-url">ssh {{ .SSHURL }}</div>
<div id="prospect-instructions">
	Use the following password to connect
</div>
<div id="prospect-ssh-password">{{ .Password }}</div>
<div id="footer">{{ .Footer }}</div>
<script>
	document.getElementById('prospect-ssh-url').onclick = selectText
	document.getElementById('prospect-ssh-password').onclick = selectText
</script>
</body>
</html>`

var sshURL = "ssh.vetzuki.com"

func init() {
	if b := os.Getenv(envSSHURL); len(b) > 0 {
		sshURL = b
	}
}

// Redemption : Contains the URL identifier for a prospect
type Redemption struct {
	ProspectURLID string `json:"prospectURLID"`
}

// Link : HTTP link
type Link struct {
	HREF, Name string
}

// Page : Structure of the page
type Page struct {
	Title, Name, Role, Employer string
	SSHURL                      string
	Password                    string
	Footer                      string
	Links                       []Link
}

func getTemplate() *template.Template {
	t := template.Must(
		template.New("page").Parse(pageTemplate),
	)
	return t
}

func buildPage(r Redemption, prospect *model.Prospect, password string) (*Page, error) {
	page := &Page{
		Title:    "VetZuki",
		Name:     prospect.Name,
		Role:     prospect.Role,
		Employer: prospect.EmployerName,
		SSHURL:   fmt.Sprintf("%s@%s", prospect.URL, sshURL),
		Password: strings.TrimSpace(password),
		Footer:   "Copyright VetZuki 2019. All rights reserved.",
	}
	return page, nil
}
func render(t *template.Template, p *Page) (string, error) {
	b := &bytes.Buffer{}
	err := t.Execute(b, p)
	if err != nil {
		log.Printf("error: while rendering: %s", err)
		return "", err
	}
	return b.String(), nil
}

// Handler : Takes care of handling the request
// returns a compiled HTML template
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	r := Redemption{ProspectURLID: request.PathParameters["prospectURLID"]}
	log.Printf("debug: redeeming URL for %s", r.ProspectURLID)

	t := getTemplate()
	prospect, ok := model.GetProspect(r.ProspectURLID)
	if !ok {
		log.Printf("error: failed to find prospect %s", r.ProspectURLID)
		return lambdaEvents.NotFound, fmt.Errorf("404: no such URL")
	}
	if !prospect.SetScreeningState(model.ScreeningStateConfirmed) {
		log.Printf("error: failed to set screening state to confirmed for %s", prospect.URL)
		return lambdaEvents.ServerError, fmt.Errorf("500: server error")
	}
	password, ok := prospect.SetPassword()
	if !ok {
		log.Printf("error: failed to reset prospect password %s", prospect.URL)
		return lambdaEvents.ServerError, fmt.Errorf("error: failed to reset password")
	}
	page, err := buildPage(r, prospect, password)
	if err != nil {
		log.Printf("error: while building page: %s", err)
		return lambdaEvents.ServerError, fmt.Errorf("500: error rendering")
	}
	content, err := render(t, page)
	if err != nil {
		log.Printf("error: while rendering page: %s", err)
		return lambdaEvents.ServerError, fmt.Errorf("500: Error rendering")
	}
	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         map[string]string{"Content-Type": "text/html"},
		Body:            content,
		IsBase64Encoded: false,
	}, nil
}
