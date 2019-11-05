package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	createProspect "github.com/vetzuki/createProspect/handler"
	createProspectNetwork "github.com/vetzuki/createProspectNetwork/handler"
	createScore "github.com/vetzuki/createScore/handler"
	employerLogin "github.com/vetzuki/employerLogin/handler"
	getProspect "github.com/vetzuki/getProspect/handler"
	getProspectScore "github.com/vetzuki/getProspectScore/handler"
	getProspectScores "github.com/vetzuki/getProspectScores/handler"
	getProspects "github.com/vetzuki/getProspects/handler"
	updateExamState "github.com/vetzuki/updateExamState/handler"
	"io/ioutil"
	"net/http"
)

// LambdaHandler : Wrap a lambda handler as an http.HandleFunc
type LambdaHandler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

var authorizedHeaders = map[string]string{
	"Authorization": "developmentToken",
	"Content-Type":  "application/json",
}

func init() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("warning: failed to locate .env\n")
	}
}

func handlerWrapper(handler LambdaHandler, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, _ := ioutil.ReadAll(r.Body)
	request := events.APIGatewayProxyRequest{
		PathParameters: mux.Vars(r),
		Body:           string(body),
		Headers:        authorizedHeaders,
	}
	response, err := handler(context.TODO(), request)
	if err != nil {
		fmt.Fprint(w, err)
		w.Write([]byte(fmt.Sprintf("%d", response.StatusCode)))
		return
	}
	fmt.Fprint(w, response.Body)
	w.WriteHeader(response.StatusCode)
}
func mkWrapperHandler(handlerMap map[string]LambdaHandler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("debug: %s %s\n", r.Method, r.URL.Path)
		for method, lambdaHandler := range handlerMap {
			if r.Method == method {
				handlerWrapper(lambdaHandler, w, r)
				return
			}
		}
		fmt.Printf("error: %s %s not supported\n", r.Method, r.URL.Path)
		w.WriteHeader(404)
	}
}
func main() {
	router := mux.NewRouter().StrictSlash(true)
	// PUT: /api/prospects/{prospectURLID}
	router.HandleFunc("/api/exams/{prospectURLID}",
		mkWrapperHandler(map[string]LambdaHandler{
			"PUT": updateExamState.Handler,
		}))
	// POST: /api/prospects
	router.HandleFunc("/api/prospects", mkWrapperHandler(map[string]LambdaHandler{
		"POST": createProspect.Handler,
		"GET":  getProspects.Handler,
	}))
	// POST : /api/networks
	router.HandleFunc("/api/networks", mkWrapperHandler(map[string]LambdaHandler{
		"POST": createProspectNetwork.Handler,
	}))
	// POST: /api/login
	router.HandleFunc("/api/login", mkWrapperHandler(map[string]LambdaHandler{
		"POST": employerLogin.Handler,
	}))
	// POST: /api/scores
	// GET: /api/scores
	router.HandleFunc("/api/scores", mkWrapperHandler(map[string]LambdaHandler{
		"POST": createScore.Handler,
		"GET":  getProspectScores.Handler,
	}))
	// GET: /api/scores/{prospectURLID}
	router.HandleFunc("/api/scores/{prospectURLID}", mkWrapperHandler(map[string]LambdaHandler{
		"GET": getProspectScore.Handler,
	}))
	// GET: /p/{prospectURLID}
	router.HandleFunc("/p/{prospectURLID}", mkWrapperHandler(map[string]LambdaHandler{
		"GET": getProspect.Handler,
	}))
	router.HandleFunc("/validateToken", func(w http.ResponseWriter, r *http.Request) {
		claims := map[string]string{
			"email": "admin@localhost",
			"name":  "admin",
			"sub":   "sub",
			"aud":   "aud",
		}
		encoder := json.NewEncoder(w)
		err := encoder.Encode(claims)
		if err != nil {
			fmt.Printf("error: while encoding claims: %s\n", err)
		}
		// w.WriteHeader(200)
	})
	http.ListenAndServe(":9000", router)
}
