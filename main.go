package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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

type dnsmadeeasySolver struct {
	client *kubernetes.Clientset
}

type dnsmadeeasyConfig struct {
	APIKey            string `json:"apiKey"`
	APITokenSecretRef string `json:"apiTokenSecretRef"`
	TTL               *int   `json:"ttl"`
}

func (c *dnsmadeeasySolver) Name() string {
	return "dnsmadeeasy"
}

func (c *dnsmadeeasySolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("call function Present: namespace=%s, zone=%s, fqdn=%s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN)
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Errorf("Failed to log config %v: %v", ch.Config, err)
		return err
	}
	klog.Info(cfg)
	return nil
}

func (c *dnsmadeeasySolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Errorf("Failed to log config %v: %v", ch.Config, err)
		return err
	}
	klog.Info(cfg)
	return nil
}

func (c *dnsmadeeasySolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	client, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		klog.Errorf("Failed to new kubernetes client: %v", err)
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
