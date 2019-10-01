package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// JWTClaims : Claims in JWT
type JWTClaims struct {
	Sub      string `json:"sub"`
	Audience string `json:"aud"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// ValidateToken : Validate an auth0 opaque access
// token. Validation returns a JWT profile.
func ValidateToken(accessToken string) bool {
	accessToken = strings.TrimSpace(accessToken)
	accessTokenValidationURL := "https://vetzuki-poc.auth0.com/userinfo"
	log.Printf("debug: validating token %s against %s", accessToken, accessTokenValidationURL)
	r, err := http.NewRequest("GET", accessTokenValidationURL, nil)
	if err != nil {
		log.Printf("error: unable to create accessToken validation request:%s", err)
		return false
	}
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	r.Header.Add("Content-Type", "application/json")
	log.Printf("debug: making HTTP request %#v", r.Header)
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		log.Printf("error: accessToken validation request failed: %s", err)
		return false
	}

	if response.StatusCode != 200 {
		log.Printf("error: received %d while validating accessToken", response.StatusCode)
		defer response.Body.Close()
		if b, err := ioutil.ReadAll(response.Body); err == nil {
			log.Printf("error: response message: %s", string(b))
		}
		return false
	}
	log.Printf("debug: decoding auth0 response")
	decoder := json.NewDecoder(response.Body)
	var jwtClaims JWTClaims
	if err := decoder.Decode(&jwtClaims); err != nil {
		log.Printf("warning: unable to parse jwt claims: %s", err)
	} else {
		log.Printf("debug: allowing access for %s", jwtClaims.Email)
	}
	return true
}
