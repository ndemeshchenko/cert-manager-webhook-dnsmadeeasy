# Cert-manager ACME webhook for DNSMadeEasy

Cert-manager ACME DNS webhook provider for DNSMadeEasy.

## Requirements
-   [go](https://golang.org/) >= 1.13.0
-   [helm](https://helm.sh/) >= v3.0.0
-   [kubernetes](https://kubernetes.io/) >= v1.14.0
-   [cert-manager](https://cert-manager.io/) >= 0.12.0

## Installation

### cert-manager

Follow the [instructions](https://cert-manager.io/docs/installation/) using the cert-manager documentation to install it within your cluster.

### Webhook

#### Using public helm chart
```bash
helm repo add cert-manager-webhook-dnsmadeeasy https://ndemeshchenko.github.io/cert-manager-webhook-dnsmadeeasy
helm install --namespace cert-manager cert-manager-webhook-dnsmadeeasy cert-manager-webhook-dnsmadeeasy/cert-manager-webhook-dnsmadeeasy
```

#### From local checkout

```bash
helm install --namespace cert-manager cert-manager-webhook-dnsmadeeasy deploy/cert-manager-webhook-dnsmadeeasy
```
**Note**: The kubernetes resources used to install the Webhook should be deployed within the same namespace as the cert-manager.

To uninstall the webhook run
```bash
helm uninstall --namespace cert-manager cert-manager-webhook-dnsmadeeasy
```

## Issuer

Create a `ClusterIssuer` or `Issuer` resource as following:
```yaml
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    # The ACME server URL
    server: https://acme-staging-v02.api.letsencrypt.org/directory

    # Email address used for ACME registration
    email: mail@example.com # REPLACE THIS WITH YOUR EMAIL!!!

    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-staging

    solvers:
      - dns01:
          webhook:
            groupName: acme.yourdomain.here
            solverName: dnsmadeeasy
            config:
              secretName: dnsmadeeasy-secret
              zoneName: example.com # YOUR DOMAIN HERE
              apiURL: https://api.dnsmadeeasy.com/V2.0
```

### Credentials
In order to access the DNSMadeEasy API, the webhook needs an API-Key and Secret-Key tokens.

If you choose another name for the secret than `dnsmadeeasy-secret`, ensure you modify the value of `secretName` in the `[Cluster]Issuer`.

The secret for the example above will look like this:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dnsmadeeasy-secret
type: Opaque
data:
  api-key: your-key-base64-encoded
  secret-key: your-secret-base64-encoded
```

### Create a certificate

Finally you can create certificates, for example:

```yaml
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: example-cert
  namespace: cert-manager
spec:
  commonName: example.com
  dnsNames:
    - example.com
  issuerRef:
    name: letsencrypt-staging
    kind: ClusterIssuer
  secretName: example-cert
```

## Development

### Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite,
else they will have undetermined behaviour when used with cert-manager.

**It is essential that you configure and run the test suite when creating a
DNS01 webhook.**

First, you need to have DNSMAdeEasy account with access to DNS control panel and API access available. You need to create API token and have a registered and verified DNS zone there.
Then you need to replace `zoneName` parameter at `testdata/dnsmadeeasy/config.json` file with actual one.
You also must encode your api token into base64 and put the hash into `testdata/dnsmadeeasy/dnsmadeeasy-secret.yml` file.

You can then run the test suite with:

```bash
# first install necessary binaries (only required once)
PLATFORM=linux|darwin
./scripts/fetch-test-binaries.sh $PLATFORM
# then run the tests
TEST_ZONE_NAME=example.com. make verify
```


## DNSMadeEasy API documentation

https://api-docs.dnsmadeeasy.com/#5b98221f-37e9-4845-a349-5e959241b4a5