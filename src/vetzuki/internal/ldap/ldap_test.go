package ldap

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	ldap "gopkg.in/ldap.v3"
)

var testEnvironment = "development"
var seeds = Seeds{}

type Seeds struct {
	Users []User `json:"ldap_users"`
}

func init() {
	baseDN = fmt.Sprintf("ou=prospects,dc=%s,dc=vetzuki,dc=com", testEnvironment)
	bindDN = fmt.Sprintf("cn=admin,dc=%s,dc=vetzuki,dc=com", testEnvironment)
	bindPassword = testEnvironment

	seedsFile := "../../seeds." + testEnvironment + ".json"
	f, err := os.Open(seedsFile)
	if err != nil {
		panic(fmt.Sprintf("fatal: unable to read seeds file %s: %s", seedsFile, err))
	}
	j := json.NewDecoder(f)
	if err := j.Decode(&seeds); err != nil {
		panic(fmt.Sprintf("fatal: unable to decode seeds: %s", err))
	}
}
func TestFindUser(t *testing.T) {
	ldapConnection, err := ldap.Dial("tcp", "localhost:389")
	if err != nil {
		t.Fatalf("failed to connect to ldap server at localhost: %s", err)
	}

	prospect, ok := FindProspect(ldapConnection, seeds.Users[0].Name)
	fmt.Printf("prospet: %#v\n", prospect)
	if !ok {
		t.Fatalf("Expected to find %s but didn't", seeds.Users[0].Name)
	}
	if prospect.Name != seeds.Users[0].Name {
		t.Fatalf("Expected prospect name to be %s, got %s", seeds.Users[0].Name, prospect.Name)
	}
}
