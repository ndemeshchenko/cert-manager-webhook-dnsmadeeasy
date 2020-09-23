package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
	"github.com/ndemeshchenko/cert-manager-webhook-dnsmadeeasy/internal"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

const (
	defaultTTL = 600
)

// GroupName ...
var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName, &dnsmadeeasySolver{})
}

type dnsmadeeasyConfig struct {
	// APIKey    string `json:"apiKey"`
	SecretRef string `json:"secretName"`
	APIURL    string `json:"apiURL"`
	ZoneName  string `json:"zoneName"`
	TTL       *int   `json:"ttl"`
}

type dnsmadeeasySolver struct {
	client *kubernetes.Clientset
}

func (c *dnsmadeeasySolver) Name() string {
	return "dnsmadeeasy"
}

func (c *dnsmadeeasySolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("call function Present: namespace=%s, zone=%s, fqdn=%s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN)
	config, err := clientConfig(c, ch)
	if err != nil {
		klog.Errorf("Failed to log config %v: %v", ch.Config, err)
		return err
	}
	addTXTRecord(config, ch)
	return nil
}

func (c *dnsmadeeasySolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	config, err := clientConfig(c, ch)
	if err != nil {
		klog.Errorf("Failed to log config %v: %v", ch.Config, err)
		return err
	}

	removeTXTRecord(config, ch)
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *dnsmadeeasySolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	client, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		klog.Errorf("Failed to initialize new kubernetes client: %v", err)
		return err
	}

	c.client = client

	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (dnsmadeeasyConfig, error) {
	ttl := defaultTTL
	cfg := dnsmadeeasyConfig{TTL: &ttl}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func addTXTRecord(config internal.Config, ch *v1alpha1.ChallengeRequest) {
	name := recordName(ch.ResolvedFQDN)

	domainID, err := searchDomainID(config)
	if err != nil {
		klog.Errorf("%v", err)
	}

	url := fmt.Sprintf("%s/dns/managed/%s/records", config.APIURL, domainID)
	payloadStr := fmt.Sprintf(`{"value":"%s", "ttl":120, "type":"TXT", "name":"%s"}`, ch.Key, name)

	add, err := callDNSProviderAPI(url, "POST", bytes.NewBuffer([]byte(payloadStr)), config)

	if err != nil {
		klog.Error(err)
	}
	klog.Infof("Added TXT record result: %s", string(add))

	return
}

func removeTXTRecord(config internal.Config, ch *v1alpha1.ChallengeRequest) {
	name := recordName(ch.ResolvedFQDN)

	domainID, err := searchDomainID(config)
	if err != nil {
		klog.Errorf("%v", err)
	}

	recordID, err := searchRecordID(name, domainID, config)
	if err != nil {
		klog.Errorf("unable to fetch DNS record ID %v", err)
	}

	if recordID == "" || recordID == "0" {
		return
	}
	url := fmt.Sprintf("%s/dns/managed/%s/records/%s", config.APIURL, domainID, recordID)

	remove, err := callDNSProviderAPI(url, "DELETE", nil, config)
	if err != nil {
		klog.Error(err)
	}

	klog.Infof("TXT record has been removed. Result: %s", string(remove))

	return
}

func clientConfig(c *dnsmadeeasySolver, ch *v1alpha1.ChallengeRequest) (internal.Config, error) {
	var config internal.Config
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return config, err
	}

	config.ZoneName = cfg.ZoneName
	config.APIURL = cfg.APIURL

	secretName := cfg.SecretRef
	sec, err := c.client.CoreV1().Secrets(ch.ResourceNamespace).Get(secretName, metav1.GetOptions{})

	if err != nil {
		return config, fmt.Errorf("unable to get secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	apiKey, err := stringFromSecretData(&sec.Data, "api-key")
	config.APIKey = apiKey

	if err != nil {
		return config, fmt.Errorf("unable to get api-key from secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	secretKey, err := stringFromSecretData(&sec.Data, "secret-key")
	config.SecretKey = secretKey
	if err != nil {
		return config, fmt.Errorf("unable to get secret-key from secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	return config, nil
}

func callDNSProviderAPI(url string, method string, body io.Reader, config internal.Config) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return []byte{}, fmt.Errorf("Unable to execute request %v", err)
	}

	addAuthHeaders(req, config)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			klog.Fatal(err)
		}
	}()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		return respBody, nil
	}

	errText := "Error calling API status:" + resp.Status + " url: " + url + " method: " + method
	return nil, errors.New(errText)
}

func addAuthHeaders(r *http.Request, config internal.Config) {
	ts := time.Now().UTC().Format("Mon, 2 Jan 2006 15:04:05 MST")

	r.Header.Add("x-dnsme-apiKey", config.APIKey)
	r.Header.Add("x-dnsme-requestDate", ts)
	r.Header.Add("Content-Type", "application/json")

	mac := hmac.New(sha1.New, []byte(config.SecretKey))
	mac.Write([]byte(ts))
	sk := hex.EncodeToString(mac.Sum(nil))
	r.Header.Add("x-dnsme-hmac", sk)
}

func searchDomainID(config internal.Config) (string, error) {
	url := fmt.Sprintf("%s/dns/managed", config.APIURL)

	// Get Zone configuration
	domainResponse, err := callDNSProviderAPI(url, "GET", nil, config)
	if err != nil {
		return "", fmt.Errorf("Unable to get zone info %v", err)
	}

	//Unmarshall response
	domains := internal.DomainResponse{}
	readErr := json.Unmarshal(domainResponse, &domains)

	if readErr != nil {
		return "", fmt.Errorf("unable to unmarshal response %v", readErr)
	}

	if domains.TotalPages != 1 {
		return "", fmt.Errorf("wrong number of zones in response %d must be exactly = 1", domains.TotalPages)
	}

	for _, v := range domains.Data {
		if v.Name == config.ZoneName {
			return strconv.Itoa(v.ID), nil
		}
	}

	return "", fmt.Errorf("DNS domain %s not found", config.ZoneName)

}

func searchRecordID(recordName, domainID string, config internal.Config) (string, error) {
	url := fmt.Sprintf("%s/dns/managed/%s/records?recordName=%s&type=TXT", config.APIURL, domainID, strings.ToLower(recordName))

	recordResponse, err := callDNSProviderAPI(url, "GET", nil, config)
	if err != nil {
		return "", fmt.Errorf("Unable to get records info %v", err)
	}

	//Unmarshal response
	records := internal.RecordResponse{}
	readErr := json.Unmarshal(recordResponse, &records)
	if readErr != nil {
		return "", fmt.Errorf("unable to unmarshal response %v", readErr)
	}

	if records.TotalRecords != 1 {
		return "", fmt.Errorf("wrong number of records in response. %d must be exactly == 1", records.TotalRecords)
	}

	for _, v := range records.Data {
		if v.Name == strings.ToLower(recordName) {
			return strconv.Itoa(v.ID), nil
		}
	}

	return "", fmt.Errorf("DNS record %s not found", recordName)
}

func stringFromSecretData(secretData *map[string][]byte, key string) (string, error) {
	data, ok := (*secretData)[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret data", key)
	}
	return string(data), nil
}

func recordName(fqdn string) string {
	return strings.Split(fqdn, ".")[0]
}
