package ldap

import (
	"crypto/tls"
	"fmt"
	"github.com/sethvargo/go-password/password"
	"log"
	"os"
	"strings"

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
}
func ConfigureConnection(base, bind, password string) {
	baseDN = base
	bindDN = bind
	bindPassword = password
}

// Connect - Connect to an LDAP server over TLS on the default port
func Connect() *ldap.Conn {
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
	var conn *ldap.Conn
	if strings.HasSuffix(ldapHost, ":389") {
		log.Printf("warning: creating non-TLS ldap connection to %s", ldapHost)
		if c, err := ldap.Dial("tcp", ldapHost); err != nil {
			log.Fatalf("error connecting to ldap %s: %s", ldapHost, err)
		} else {
			conn = c
		}
	} else {
		log.Printf("debug: creating TLS ldap connection to %s", ldapHost)
		if c, err := ldap.DialTLS("tcp", ldapHost, tlsNoVerify); err != nil {
			log.Fatalf("error: connecting to ldap at %s: %s", ldapHost, err)
		} else {
			conn = c
		}
	}
	return conn
}

// User - A user in ldap
type User struct {
	// Name - Must be the same as a prospectURL
	Name string `json:"name"`
	DN   string `json:"dn"`
	CN   string `json:"cn"`
	User string `json:"user"`
	UID  string `json:"uid"`
	GID  string `json:"gid"`
}

func prospectQuery(prospectURL string) string {
	return fmt.Sprintf("(uid=%s)", prospectURL)
}
func prospectDN(prospectURL string) string {
	return fmt.Sprintf("cn=%s,%s", prospectURL, baseDN)
}

// FindProspect - Locate a prospect
func FindProspect(c *ldap.Conn, prospectURL string) (*User, bool) {
	dn := prospectDN(prospectURL)
	log.Printf("debug: searching %s for %s", baseDN, prospectURL)
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
		prospectQuery(prospectURL),
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
			Name: prospectURL,
			DN:   entry.DN,
			User: entry.GetAttributeValue("uid"),
			UID:  entry.GetAttributeValue("uidNumber"),
			GID:  entry.GetAttributeValue("gidNumber"),
			CN:   entry.GetAttributeValue("cn"),
		}, true
	}
	log.Printf("warning: found %d entries for prospect %s, expected 1", len(result.Entries), prospectURL)
	return nil, false
}

// CreateProspect : Create prospect given an prospectURL
func CreateProspect(c *ldap.Conn, prospectURL string) (*User, bool) {
	var user *User
	dn := prospectDN(prospectURL)

	log.Printf("debug: creating %s for %s", dn, prospectURL)
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
	userPassword, err := password.Generate(16, 4, 2, false, false)
	if err != nil {
		log.Printf("error: failed to generate user password: %s", err)
		return user, false
	}

	prospect := ldap.NewAddRequest(dn, nil)
	prospect.Attribute("uid", []string{prospectURL})
	prospect.Attribute("cn", []string{prospectURL})
	prospect.Attribute("sn", []string{prospectURL})
	prospect.Attribute("uidNumber", []string{uidNumber})
	prospect.Attribute("gidNumber", []string{uidNumber})
	prospect.Attribute("homeDirectory", []string{fmt.Sprintf("/home/%s", prospectURL)})
	prospect.Attribute("objectClass", []string{"organizationalPerson", "posixAccount"})
	prospect.Attribute("userPassword", []string{userPassword})

	if err := c.Add(prospect); err != nil {
		log.Printf("error: failed to create user %s: %s", prospectURL, err)
		return user, false
	}
	user = &User{
		Name: prospectURL,
		DN:   dn,
		User: prospectURL,
		UID:  uidNumber,
		GID:  uidNumber,
		CN:   prospectURL,
	}
	return user, true
}

// DeleteProspect : Delete a prospect given a prospectURL
func DeleteProspect(c *ldap.Conn, prospectURL string) bool {
	dn := prospectDN(prospectURL)
	log.Printf("debug: deleting %s for %s", dn, prospectURL)
	d := ldap.NewDelRequest(dn, nil)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind for delete: %s", err)
		return false
	}
	if err := c.Del(d); err != nil {
		log.Printf("error: failed to delete %s: %s", prospectURL, err)
		return false
	}
	return true
}
