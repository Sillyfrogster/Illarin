package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const SignedURLLife = 15 * time.Minute

const (
	ExpiresParam   = "expires"
	SignatureParam = "signature"
)

type URLSigner struct {
	secret []byte
}

func NewURLSigner() URLSigner {
	secret := make([]byte, 32)
	rand.Read(secret)
	return URLSigner{secret: secret}
}

func (k URLSigner) Sign(path string, now time.Time) string {
	expires := now.Add(SignedURLLife).Unix()
	query := url.Values{}
	query.Set(ExpiresParam, strconv.FormatInt(expires, 10))
	query.Set(SignatureParam, k.stamp(path, expires))
	return path + "?" + query.Encode()
}

func (k URLSigner) Valid(path, expires, signature string, now time.Time) bool {
	if len(k.secret) == 0 {
		return false
	}
	deadline, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || now.After(time.Unix(deadline, 0)) {
		return false
	}
	return hmac.Equal([]byte(signature), []byte(k.stamp(path, deadline)))
}

func (k URLSigner) stamp(path string, expires int64) string {
	mac := hmac.New(sha256.New, k.secret)
	fmt.Fprintf(mac, "%s\n%d", path, expires)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
