package main

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/jetstack/cert-manager/test/acme/dns"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
	fqdn string
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	fqdn = GetRandomString(19) + "." + zone
	fmt.Printf("GetRandomString fqdn is: %s", fqdn)

	fixture := dns.NewFixture(&dnsmadeeasySolver{},
		dns.SetResolvedZone(zone),
		dns.SetResolvedFQDN(fqdn),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/dnsmadeeasy"),
		dns.SetBinariesPath("kubebuilder/bin"),
	)

	fixture.RunConformance(t)
}

func GetRandomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
