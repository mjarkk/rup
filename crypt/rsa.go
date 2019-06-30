package crypt

//
// This file has a number of functions to make rsa easier
//

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
)

// RSAPrivToString transforms the private key into bytes
func RSAPrivToString(priv *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
}

// RSAPubToString transforms the public key into bytes
func RSAPubToString(pub *rsa.PublicKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(pub)})
}

// RSAParsePriv parses a rsa private key into *rsa.PrivateKey
func RSAParsePriv(priv []byte) (*rsa.PrivateKey, error) {
	privBlock, _ := pem.Decode(priv)
	key, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, err
	}
	key.Precompute()
	return key, nil
}

// RSAParsePub parses a rsa public key into *rsa.PublicKey
func RSAParsePub(pub []byte) (*rsa.PublicKey, error) {
	pubBlock, _ := pem.Decode(pub)
	return x509.ParsePKCS1PublicKey(pubBlock.Bytes)
}

// RSAGenKey genrates a private key and a public key
func RSAGenKey(bits int) (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	key.Precompute()
	return key, nil
}

// RSADecrypt decrypt a rsa message
func RSADecrypt(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, data, []byte{})
}

// RSAEncrypt encrypts data using the public key
// Note: The message can't be longer than the key size
func RSAEncrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, data, []byte{})
}
