# oc-gate-operator

![alt gopher network](https://raw.githubusercontent.com/yaacov/oc-gate/main/web/public/network-side.png)

Operate the [oc-gate](https://github.com/yaacov/oc-gate) service on a cluster.

[![Go Report Card](https://goreportcard.com/badge/github.com/yaacov/oc-gate-operator)](https://goreportcard.com/report/github.com/yaacov/oc-gate-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

(gopher network image - [egonelbre/gophers](https://github.com/egonelbre/gophers))

## Deploy

``` bash
# Clone repository
git clone git@github.com:yaacov/oc-gate-operator.git
cd oc-gate-operator

# Create namespace and add the private/public key secret used to generate tokens
oc new-project oc-gate-operator-system
oc create -n oc-gate-operator-system secret generic oc-gate-jwt-secret --from-file=test/cert.pem --from-file=test/key.pem

# Deoploy
oc create -f deploy
```

### Disconnected clusters

``` bash
# Edit the operator image in operator-controller-manager yaml file.
vim deploy/apps_v1_deployment_oc-gate-operator-controller-manager.yaml
```

#### GateToken demo:

[![asciicast](https://asciinema.org/a/397136.svg)](https://asciinema.org/a/397136)

### Remove deplyment

```bash
# Un-Deploy
oc delete -f deploy
```

## Usage

### Setting up the [oc-gate](https://github.com/yaacov/oc-gate) service is done using GateService CRD

Available fields are:

- route is: the the gate proxy server.
- api-url: is the k8s API url, defalut value is "https://kubernetes.default.svc".
- admin-role: is the verbs athorization role of the service (reader/admin), defalut value is "reader".
- admin-resources: is a comma separated list of resources athorization role of the service, defalut value is "" (allow all).
- admin-namespaced: determain if the athorization role of the service is namespaced, defalut value is false.
- passthrough: determain if  the tokens acquired from OAuth2 server directly to k8s API, defalut value is false.
- image: is the oc gate proxy image to use, defalut value is "quay.io/yaacov/oc-gate:latest".

Creating a service requires a secret holding a RSA public-key for sighing the token in the namespace of the service (secret name: oc-gate-jwt-secret).

### Requesting a token for [oc-gate](https://github.com/yaacov/oc-gate) service is done using GateToken CRD

Available fields are:

- match-path: string (required), match-path is a regular expresion used to validate API request path, API requests matching this pattern will be validated by the token. This field may not be empty.
- match-method: string, a comma separeted list of allowed http methods, defoult is "GET,OPTIONS"
- duration-sec: int, duration-sec is the duration in sec the token will be validated since it's invocation. Defalut value is 3600s (1h).
- from: string, from is time of token invocation, the token will not validate before this time, the token duration will start from this time. Defalut to token object creation time.

Creating a token requires a secret holding a RSA private-key for sighing the token in the namespace of the token (secret name: oc-gate-jwt-secret), once token is ready it will be available in the GateToken status.

## Example GateToken CR

Note: the token signiture requires a secret holding the private key in the same namespace, see the [deploy](#deploy) section for how to create the secret.

```yaml
apiVersion: ocgate.yaacov.com/v1beta1
kind: GateToken
metadata:
  name: gatetoken-sample
  namespace: oc-gate
spec:
  match-path: ^/k8s/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/my-vm/vnc
```

## Example GateServer CR

Note: the server signiture authentication requires a secret holding the public key in the same namespace, see the [deploy](#deploy) section for how to create the secret.

```yaml
apiVersion: ocgate.yaacov.com/v1beta1
kind: GateServer
metadata:
  name: gateserver-sample
  namespace: oc-gate
spec:
  route: test-proxy.apps.ostest.test.metalkube.org
```

### Set the image field on disconnected clusters

On disconnected clusters use the optional image field in the GateServer CRD.

```yaml
apiVersion: ocgate.yaacov.com/v1beta1
kind: GateServer
metadata:
  name: gateserver-sample
  namespace: oc-gate
spec:
  # image is optional field for disconnected clusters
  image: quay.io/yaacov/oc-gate:v0.0.1
  route: test-proxy.apps.ostest.test.metalkube.org
```
