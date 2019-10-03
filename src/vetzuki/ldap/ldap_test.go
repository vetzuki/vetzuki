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

	seedsFile := "../seeds." + testEnvironment + ".json"
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
func TestSetProspectPassword(t *testing.T) {
	ldapConnection, err := ldap.Dial("tcp", "localhost:389")
	if err != nil {
		t.Fatalf("failed to connect to ldap server at localhost: %s", err)
	}
	_ = DeleteProspect(ldapConnection, "id1")
	prospect, ok := CreateProspect(ldapConnection, "id1")
	if !ok {
		t.Fatalf("failed to create prospect")
	}
	p, ok := SetProspectPassword(ldapConnection, prospect.CN)
	if !ok {
		t.Fatalf("expected password, got none")
	}
	if len(p) == 0 {
		t.Fatalf("expected a password length > 0, got 0")
	}

}
func TestCreateProspect(t *testing.T) {
	ldapConnection, err := ldap.Dial("tcp", "localhost:389")
	if err != nil {
		t.Fatalf("failed to connect to ldap server at localhost: %s", err)
	}
	prospectID := "testProspect1"
	_ = DeleteProspect(ldapConnection, prospectID)
	prospect, ok := CreateProspect(ldapConnection, prospectID)
	if !ok {
		t.Fatalf("expected to create %s, but failed", prospectID)
	}
	if prospect.Name != prospectID {
		t.Fatalf("expected to %s to equal %s", prospect.Name, prospectID)
	}
	found, foundOK := FindProspect(ldapConnection, prospectID)
	if !foundOK {
		t.Fatalf("expected to find new user %s, but failed", prospectID)
	}
	if found.Name != prospectID {
		t.Fatalf("expected %s to equal prospectID %s", found.Name, prospectID)
	}
}
func TestAddGroupMember(t *testing.T) {
	ldapConnection, err := ldap.Dial("tcp", "localhost:389")
	if err != nil {
		t.Fatalf("failed to connect to ldap server at localhost: %s", err)
	}
	groupName := "docker"
	cn := "cn"
	defer RemoveGroupMember(ldapConnection, groupName, &User{CN: cn})
	dockerGroup, ok := AddGroupMember(ldapConnection, groupName, &User{CN: cn})
	if !ok {
		t.Fatalf("expected to add %s to %s, but failed", cn, groupName)
	}
	found := false
	for _, member := range dockerGroup.Members {
		found = member == cn
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("expected to find %s, but failed", cn)
	}

}
