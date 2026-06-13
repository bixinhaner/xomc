package e2e

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"testing"
	"time"
)

// loginResult is the parsed {data,msg,ret} envelope of an encrypted-login attempt.
type loginResult struct {
	status int
	data   map[string]interface{} // the "data" object of the unified response envelope
	raw    map[string]interface{}
}

// fetchLoginPublicKey GETs /api/v1/auth/public-key and returns (keyID, RSA public key).
func fetchLoginPublicKey(t *testing.T) (string, *rsa.PublicKey) {
	t.Helper()
	resp, err := newClient().Get(baseURL() + "/api/v1/auth/public-key")
	if err != nil {
		t.Fatalf("fetch public-key: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public-key status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data struct {
			KeyID     string `json:"key_id"`
			PublicKey string `json:"public_key"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode public-key: %v", err)
	}
	block, _ := pem.Decode([]byte(env.Data.PublicKey))
	if block == nil {
		t.Fatalf("public-key PEM decode failed")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("public key is not RSA (got %T)", pub)
	}
	return env.Data.KeyID, rsaPub
}

// encryptedLogin performs the RSA-OAEP(SHA-256) encrypted login flow the app requires
// (plaintext password login is disabled outside HTTPS/true-localhost — T-0120). It
// mirrors the frontend / smoke-suite flow: fetch a fresh public key, encrypt a
// {password, ts, nonce} payload, then POST {username, encrypted_password, key_id}.
// Each call uses a fresh nonce + timestamp so the ReplayGuard never rejects it.
func encryptedLogin(t *testing.T, username, password string) loginResult {
	t.Helper()
	keyID, pub := fetchLoginPublicKey(t)

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("nonce: %v", err)
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"password": password,
		"ts":       time.Now().Unix(),
		"nonce":    hex.EncodeToString(nonce),
	})
	cipher, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, payload, nil)
	if err != nil {
		t.Fatalf("rsa encrypt: %v", err)
	}
	reqBody, _ := json.Marshal(map[string]string{
		"username":           username,
		"encrypted_password": base64.StdEncoding.EncodeToString(cipher),
		"key_id":             keyID,
	})
	resp, err := newClient().Post(baseURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("login post: %v", err)
	}
	defer resp.Body.Close()

	res := loginResult{status: resp.StatusCode, raw: map[string]interface{}{}}
	_ = json.NewDecoder(resp.Body).Decode(&res.raw)
	if d, ok := res.raw["data"].(map[string]interface{}); ok {
		res.data = d
	}
	return res
}
