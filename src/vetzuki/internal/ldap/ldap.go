package ldap

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"

	ldap "gopkg.in/ldap.v3"
)

const (
	unlimitedTimeout = 0
	unlimitedSize    = 0
	tlsPort          = "636"
	envBindDN        = "BIND_DN"
	envBindPassword  = "BIND_PASSWORD"
	envBaseDN        = "BASE_DN"
	envLDAPHost      = "LDAP_HOST"
)

var (
	ldapHost                = "localhost:389"
	bindDN                  = ""
	bindPassword            = ""
	baseDN                  = "ou=prospect,dc=vetzuki,dc=com"
	tlsNoVerify             = &tls.Config{InsecureSkipVerify: true}
	prospectQueryAttributes = []string{"dn"}
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	if b := os.Getenv(envBindDN); len(b) > 0 {
		bindDN = b
	}
	if b := os.Getenv(envBindPassword); len(b) > 0 {
		bindPassword = b
	}
	if b := os.Getenv(envBaseDN); len(b) > 0 {
		baseDN = b
	}
	if b := os.Getenv(envLDAPHost); len(b) > 0 {
		ldapHost = b
	}
}

// Connect - Connect to an LDAP server over TLS on the default port
func Connect(hostname string) *ldap.Conn {
	conn, err := ldap.DialTLS("tcp", net.JoinHostPort(hostname, tlsPort), tlsNoVerify)
	if err != nil {
		log.Fatalf("error: connecting to ldap at %s: %s", hostname, err)
	}
	return conn
}

// User - A user in ldap
type User struct {
	// Name - Must be the same as a prospectID
	Name string `json:"name"`
	DN   string `json:"dn"`
	CN   string `json:"cn"`
	User string `json:"user"`
	UID  string `json:"uid"`
	GID  string `json:"gid"`
}

func prospectQuery(prospectID string) string {
	return fmt.Sprintf("(uid=%s)", prospectID)
}
func prospectDN(prospectID string) string {
	return fmt.Sprintf("cn=%s,%s", prospectID, baseDN)
}

// FindProspect - Locate a prospect
func FindProspect(c *ldap.Conn, prospectID string) (*User, bool) {
	dn := prospectDN(prospectID)
	log.Printf("debug: searching %s for %s", baseDN, prospectID)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind: %s", err)
		return nil, false
	}
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		unlimitedSize,
		unlimitedTimeout,
		false,
		prospectQuery(prospectID),
		[]string{"dn", "cn", "uid", "uidNumber", "gidNumber"},
		nil)

	result, err := c.Search(searchRequest)
	if err != nil {
		log.Printf("error: search for %s failed: %s", dn, err)
		return nil, false
	}
	if len(result.Entries) == 1 {
		entry := result.Entries[0]
		return &User{
			Name: prospectID,
			DN:   entry.DN,
			User: entry.GetAttributeValue("uid"),
			UID:  entry.GetAttributeValue("uidNumber"),
			GID:  entry.GetAttributeValue("gidNumber"),
			CN:   entry.GetAttributeValue("cn"),
		}, true
	}
	log.Printf("warning: found %d entries for prospect %s, expected 1", len(result.Entries), prospectID)
	return nil, false
}

// CreateProspect : Create prospect given an prospectID
func CreateProspect(c *ldap.Conn, prospectID string) (*User, bool) {
	var user *User
	dn := prospectDN(prospectID)

	log.Printf("debug: creating %s for %s", dn, prospectID)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind %s", err)
		return user, false
	}
	n, err := NextUID()
	if err != nil {
		log.Printf("error: failed to get next UID: %s", err)
		return user, false
	}
	uidNumber := fmt.Sprintf("%d", n)

	prospect := ldap.NewAddRequest(dn, nil)
	prospect.Attribute("uid", []string{prospectID})
	prospect.Attribute("cn", []string{prospectID})
	prospect.Attribute("sn", []string{prospectID})
	prospect.Attribute("uidNumber", []string{uidNumber})
	prospect.Attribute("gidNumber", []string{uidNumber})
	prospect.Attribute("homeDirectory", []string{fmt.Sprintf("/home/%s", prospectID)})
	prospect.Attribute("objectClass", []string{"organizationalPerson", "posixAccount"})

	if err := c.Add(prospect); err != nil {
		log.Printf("error: failed to create user %s: %s", prospectID, err)
		return user, false
	}
	user = &User{
		Name: prospectID,
		DN:   dn,
		User: prospectID,
		UID:  uidNumber,
		GID:  uidNumber,
		CN:   prospectID,
	}
	return user, true
}

// DeleteProspect : Delete a prospect given a prospectID
func DeleteProspect(c *ldap.Conn, prospectID string) bool {
	dn := prospectDN(prospectID)
	log.Printf("debug: deleting %s for %s", dn, prospectID)
	d := ldap.NewDelRequest(dn, nil)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind for delete: %s", err)
		return false
	}
	if err := c.Del(d); err != nil {
		log.Printf("error: failed to delete %s: %s", prospectID, err)
		return false
	}
	return true
}
