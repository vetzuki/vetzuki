package model

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/vetzuki/vetzuki/ldap"
	"os"
	"strings"
	"testing"
)

func init() {
	testEnvironment := "development"
	if e := os.Getenv("VETZUKI_ENVIRONMENT"); len(e) > 0 {
		switch e {
		case "development":
		case "test":
			testEnvironment = e
		}
	}
	// configure LDAP
	baseDN := fmt.Sprintf("ou=prospects,dc=%s,dc=vetzuki,dc=com", testEnvironment)
	bindDN := fmt.Sprintf("cn=admin,dc=%s,dc=vetzuki,dc=com", testEnvironment)
	bindPassword := testEnvironment
	ldap.ConfigureConnection(baseDN, bindDN, bindPassword)
	if err := godotenv.Load("../.env." + testEnvironment); err != nil {
		fmt.Printf("failed to load .env")
	}
	Connect()
}

func TestNextPort(t *testing.T) {
	ec2InstanceID := "i-123abc"
	_, _ = redisConnection().HDel(ec2InstanceID, sshPortField).Result()

	for i := range []int{1, 2, 3} {
		port, ok := NextPort(ec2InstanceID)
		if !ok {
			t.Fatalf("expected to get a fresh port but failed")
		}
		if expected := baseSSHPort + i; port != expected {
			t.Fatalf("expected port %d, got %d", expected, port)
		}
	}
}

func TestNextNetwork(t *testing.T) {

	tests := map[string][]int{
		"i-123abc": []int{0, 1},
		"i-234abc": []int{0, 1, 2},
	}

	for ec2InstanceID, ips := range tests {
		_, _ = redisConnection().HDel(ec2InstanceID, "network").Result()
		for i := range ips {
			base10Bits, ok := NextNetwork(ec2InstanceID)
			if !ok {
				t.Fatalf("expected to get a new network but failed")
			}
			if expected := int64(0 + i); base10Bits != expected {
				t.Fatalf("expected %d bits, got %d", expected, base10Bits)
			}
		}
	}
}

func TestCreateProspectNetwork(t *testing.T) {

	prospectID := int64(1)
	ec2InstanceID := "i-123abc"
	network, ok := CreateProspectNetwork(prospectID, ec2InstanceID)
	if !ok {
		t.Fatalf("expected to create a prospect network but failed")
	}
	if network == nil {
		t.Fatalf("expectd a prospect network instance, got nil")
	}
	networkID := strings.Join(
		strings.Split(network.Network, ".")[0:3],
		".")
	if network.Mask != "255.255.255.0" {
		t.Fatalf("expected a class C mask, got %s", network.Mask)
	}
	if !strings.HasPrefix(network.ExamContainerIP, networkID) || !strings.HasSuffix(network.GatewayIP, ".1") {
		t.Fatalf("expected a gateway IP at the base of the network, got %s", network.GatewayIP)
	}
	if !strings.HasPrefix(network.ExamContainerIP, networkID) || !strings.HasSuffix(network.ExamContainerIP, ".2") {
		t.Fatalf("expected a examContainerIP of NETWORK.2, got %s", network.ExamContainerIP)
	}
}
