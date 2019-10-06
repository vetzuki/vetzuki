package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	envAdminWhitelist = "ADMIN_WHITELIST"
	envTeeshAPIKEY    = "TEESH_API_KEY"
)

var (
	adminWhitelist = []string{}
	teeshAPIKey    = ""
)

// JWTClaims : Claims in JWT
type JWTClaims struct {
	Sub      string `json:"sub"`
	Audience string `json:"aud"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func init() {
	if w := os.Getenv(envAdminWhitelist); len(w) > 0 {
		members := strings.Split(w, ",")
		for _, member := range members {
			if len(strings.TrimSpace(member)) == 0 {
				continue
			}
			log.Printf("debug: whitelisting admin %s", member)
			adminWhitelist = append(adminWhitelist, strings.TrimSpace(member))
		}
		log.Printf("info: whitelisted %d admins", len(adminWhitelist))
	}

	teeshAPIKey = os.Getenv(envTeeshAPIKEY)

}
func checkAdminWhitelist(jwtClaims JWTClaims) bool {
	if len(adminWhitelist) == 0 {
		return true
	}
	for _, admin := range adminWhitelist {
		if jwtClaims.Email == admin {
			return true
		}
	}
	log.Printf("warning: %s is not authorized by whitelist", jwtClaims.Email)
	return false
}

// ValidateAPIKey : Validate an API key
func ValidateAPIKey(apiKey string) bool {
	if len(apiKey) == 0 || len(teeshAPIKey) == 0 {
		log.Printf("error: apiKey is %d in size, teesh key is %d. Cannot be like this", len(apiKey), len(teeshAPIKey))
		return false
	}
	return strings.TrimSpace(apiKey) == strings.TrimSpace(teeshAPIKey)
}

// ValidateToken : Validate an auth0 opaque access
// token. Validation returns a JWT profile.
func ValidateToken(accessToken string) (*JWTClaims, bool) {
	accessToken = strings.TrimSpace(accessToken)
	accessTokenValidationURL := "https://vetzuki-poc.auth0.com/userinfo"
	log.Printf("debug: validating token %s against %s", accessToken, accessTokenValidationURL)
	r, err := http.NewRequest("GET", accessTokenValidationURL, nil)
	if err != nil {
		log.Printf("error: unable to create accessToken validation request:%s", err)
		return nil, false
	}
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	r.Header.Add("Content-Type", "application/json")
	log.Printf("debug: making HTTP request %#v", r.Header)
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		log.Printf("error: accessToken validation request failed: %s", err)
		return nil, false
	}

	if response.StatusCode != 200 {
		log.Printf("error: received %d while validating accessToken", response.StatusCode)
		defer response.Body.Close()
		if b, err := ioutil.ReadAll(response.Body); err == nil {
			log.Printf("error: response message: %s", string(b))
		}
		return nil, false
	}
	log.Printf("debug: decoding auth0 response")
	decoder := json.NewDecoder(response.Body)
	var jwtClaims JWTClaims
	if err := decoder.Decode(&jwtClaims); err != nil {
		log.Printf("warning: unable to parse jwt claims: %s", err)
	}
	ok := checkAdminWhitelist(jwtClaims)
	if !ok {
		log.Printf("warning: unauthorized admin %s", jwtClaims.Email)
		return nil, false
	}
	log.Printf("debug: allowing access for %s", jwtClaims.Email)

	return &jwtClaims, true
}

const authorizationHeader = "Authorization"

// ExtractToken : Extract a token from an HTTP.Request header
func ExtractToken(headers map[string]string) (string, bool) {
	log.Printf("debug: extracting token from headers")
	ah, ok := headers[authorizationHeader]
	if !ok {
		log.Printf("warning: no authorization header found")
		return ah, false
	}
	if strings.HasPrefix(ah, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(ah, "Bearer ")), true
	}
	return strings.TrimSpace(ah), true
}
