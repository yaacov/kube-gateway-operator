/*
Copyright 2021 Yaacov Zamir.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocgatev1beta1 "github.com/rh-fieldwork/kube-gateway-operator/api/v1beta1"
)

// Secret is a
func Secret(s *ocgatev1beta1.GateServer) (*corev1.Secret, error) {
	labels := map[string]string{
		"app": s.Name,
	}

	bitSize := 4096

	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		return nil, err
	}

	publicKeyBytes, err := encodePublicKeyToPEM(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	privateKeyBytes := encodePrivateKeyToPEM(privateKey)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JWTSecretName,
			Namespace: s.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"cert.pem": publicKeyBytes,
			"key.pem":  privateKeyBytes,
		},
	}

	return secret, nil
}

// JWTSecretName is the name of the secret holding the private and public SSH keys
// for authenticating the JWT access codes.
const JWTSecretName = "kube-gateway-jwt-secret"

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

// encodePublicKeyToPEM encodes public Key from RSA to PEM format
func encodePublicKeyToPEM(pubkey *rsa.PublicKey) ([]byte, error) {
	// Get ASN.1 DER format
	pubDER, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return nil, err
	}

	// pem.Block
	pubBlock := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: pubDER,
	}

	// public key in PEM format
	pubPEM := pem.EncodeToMemory(&pubBlock)

	return pubPEM, nil
}

// generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// returns in the format "ssh-rsa ..."
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	return pubKeyBytes, nil
}
