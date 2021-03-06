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
	envGroupsDN      = "GROUPS_DN"
	envBindPassword  = "BIND_PASSWORD"
	envBaseDN        = "BASE_DN"
	envLDAPHost      = "LDAP_HOST"
	envEnvironment   = "VETZUKI_ENVIRONMENT"
)

var (
	ldapHost                = "localhost:389"
	bindDN                  = ""
	bindPassword            = ""
	environment             = "development"
	baseDN                  = fmt.Sprintf("ou=prospect,dc=%s,dc=vetzuki,dc=com", environment)
	groupsDN                = fmt.Sprintf("ou=groups,dc=%s,dc=vetzuki,dc=com", environment)
	tlsNoVerify             = &tls.Config{InsecureSkipVerify: true}
	prospectQueryAttributes = []string{"dn"}
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

// ConfigureConnection : Set the baseDN, bindDN and password
func ConfigureConnection(base, bind, password string) {
	baseDN = base
	bindDN = bind
	bindPassword = password
}

// Connect - Connect to an LDAP server over TLS on the default port
func Connect() *ldap.Conn {
	// TODO: This initialization is messed up
	if b := os.Getenv(envEnvironment); len(b) > 0 {
		environment = b
	}
	if b := os.Getenv(envBindDN); len(b) > 0 {
		bindDN = b
	}
	if b := os.Getenv(envBindPassword); len(b) > 0 {
		bindPassword = b
	}
	if b := os.Getenv(envBaseDN); len(b) > 0 {
		baseDN = b
	}
	if b := os.Getenv(envGroupsDN); len(b) > 0 {
		groupsDN = b
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

// User : A user in ldap
type User struct {
	// Name - Must be the same as a prospectURL
	Name string `json:"name"`
	DN   string `json:"dn"`
	CN   string `json:"cn"`
	User string `json:"user"`
	UID  string `json:"uid"`
	GID  string `json:"gid"`
}

// Group : A Group in LDAP
type Group struct {
	Name    string   `json:"name"`
	DN      string   `json:"dn"`
	CN      string   `json:"cn"`
	Members []string `json:"members"`
}

func prospectQuery(prospectURL string) string {
	return fmt.Sprintf("(uid=%s)", prospectURL)
}
func prospectDN(prospectURL string) string {
	return fmt.Sprintf("cn=%s,%s", prospectURL, baseDN)
}

// FindProspect : Locate a prospect
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
	log.Printf("debug: creating prospect %s in LDAP", prospectURL)
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
	log.Printf("debug: created uid %s for %s", uidNumber, prospectURL)
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
	log.Printf("debug: adding %s to docker group", user.Name)
	if !IsGroupMember(c, "docker", user) {
		_, ok := AddGroupMember(c, "docker", user)
		if !ok {
			log.Printf("error: failed to add user %s to docker group", user.Name)
			return nil, false
		}
	}
	log.Printf("debug: created new prospect %s", prospectURL)
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
func groupDN(groupName string) string {
	return fmt.Sprintf("cn=%s,%s", groupName, groupsDN)
}

// SetProspectPassword : Set the prospect password
func SetProspectPassword(c *ldap.Conn, prospectURL string) (string, bool) {
	log.Printf("debug: setting %s password", prospectURL)
	dn := prospectDN(prospectURL)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind: %s", err)
		return "", false
	}

	log.Printf("debug: generating new password")
	newPassword, err := password.Generate(16, 4, 2, false, false)
	if err != nil {
		log.Printf("error: failed to generate user password for %s: %s", prospectURL, err)
		return "", false
	}
	log.Printf("debug: creating password modify requestfor %s", dn)
	request := ldap.NewPasswordModifyRequest(dn, "", newPassword)
	log.Printf("trace: executing password modification")
	if _, err := c.PasswordModify(request); err != nil {
		log.Printf("error: failed to change user password for %s: %s", prospectURL, err)
		return "", false
	}
	log.Printf("debug: set password for %s", prospectURL)
	return newPassword, true
}

// AddGroupMember : Add a User to a group
func AddGroupMember(c *ldap.Conn, groupName string, user *User) (*Group, bool) {
	log.Printf("debug: adding %s to group %s", user.Name, groupName)
	dn := groupDN(groupName)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind: %s", err)
		return nil, false
	}
	log.Printf("debug: adding memberUID %s to %s", user.UID, dn)
	request := ldap.NewModifyRequest(dn, nil)
	request.Add("memberUid", []string{user.CN})
	err := c.Modify(request)
	if err != nil {
		log.Printf("error: failed to add %s to group %s: %s", user.Name, groupName, err)
		return nil, false
	}
	log.Printf("debug: added %s to %s", user.Name, groupName)
	return GetGroup(c, groupName)
}

// GetGroup : Get a group
func GetGroup(c *ldap.Conn, groupName string) (*Group, bool) {
	log.Printf("debug: getting group %s", groupName)
	dn := groupDN(groupName)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind: %s", err)
		return nil, false
	}
	request := ldap.NewSearchRequest(
		groupsDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		unlimitedSize,
		unlimitedTimeout,
		false,
		fmt.Sprintf("(cn=%s)", groupName),
		[]string{"dn", "cn", "memberUid"},
		nil)
	result, err := c.Search(request)
	if err != nil {
		log.Printf("error: search for group %s failed: %s", dn, err)
		return nil, false
	}
	if len(result.Entries) == 1 {
		entry := result.Entries[0]
		return &Group{
			Name:    entry.GetAttributeValue("cn"),
			CN:      entry.GetAttributeValue("cn"),
			DN:      entry.DN,
			Members: entry.GetAttributeValues("memberUid"),
		}, true
	}
	log.Printf("warning: found %d entries for group %s", len(result.Entries), groupName)
	return nil, false
}

// IsGroupMember : Checking if a user is a group member
func IsGroupMember(c *ldap.Conn, groupName string, user *User) bool {
	log.Printf("debug: checking if %s is a member of %s", user.CN, groupName)
	g, ok := GetGroup(c, groupName)
	if !ok {
		log.Printf("error: no suhc group %s", groupName)
		return false
	}
	for _, member := range g.Members {
		if member == user.CN || member == user.UID {
			return true
		}
	}
	log.Printf("info: %s is not a member of %s", user.CN, groupName)
	return false
}

// RemoveGroupMember : Remove member from group
func RemoveGroupMember(c *ldap.Conn, groupName string, user *User) bool {
	log.Printf("debug: removing %s from %s", user.Name, groupName)
	dn := groupDN(groupName)
	if err := c.Bind(bindDN, bindPassword); err != nil {
		log.Printf("error: failed to bind: %s", err)
		return false
	}
	request := ldap.NewModifyRequest(dn, nil)
	request.Delete("memberUid", []string{user.CN})
	err := c.Modify(request)
	if err != nil {
		log.Printf("error: failed to remove %s from %s: %s", user.CN, groupName, err)
		return false
	}
	return true
}
