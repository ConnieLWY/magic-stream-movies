package tokens

import (
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

var (
	accessKey  = []byte(strings.Repeat("a", 32))
	refreshKey = []byte(strings.Repeat("r", 32))
)

func TestGenerateAndValidate(t *testing.T) {
	m := NewManager(accessKey, refreshKey)
	at, rt, err := m.Generate("u1", "USER")
	if err != nil {
		t.Fatal(err)
	}

	claims, err := m.ValidateAccess(at)
	if err != nil {
		t.Fatalf("ValidateAccess: %v", err)
	}
	if claims.UserID != "u1" || claims.Role != "USER" {
		t.Errorf("claims = %+v", claims)
	}
	if _, err := m.ValidateRefresh(rt); err != nil {
		t.Fatalf("ValidateRefresh: %v", err)
	}
}

func TestTokensAreNotInterchangeable(t *testing.T) {
	m := NewManager(accessKey, refreshKey)
	at, rt, _ := m.Generate("u1", "USER")

	if _, err := m.ValidateAccess(rt); err == nil {
		t.Error("refresh token accepted as access token")
	}
	if _, err := m.ValidateRefresh(at); err == nil {
		t.Error("access token accepted as refresh token")
	}
}

func TestGeneratedTokensAreUnique(t *testing.T) {
	m := NewManager(accessKey, refreshKey)
	_, rt1, _ := m.Generate("u1", "USER")
	_, rt2, _ := m.Generate("u1", "USER")
	if rt1 == rt2 {
		t.Error("two refresh tokens issued in the same second are identical; rotation would not work")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	m := NewManager(accessKey, refreshKey)
	m.now = func() time.Time { return time.Now().Add(-AccessTokenTTL - time.Minute) }
	at, _, _ := m.Generate("u1", "USER")

	m.now = time.Now
	if _, err := m.ValidateAccess(at); err == nil {
		t.Error("expired token accepted")
	}
}

func TestRejectsOtherAlgorithmsAndIssuers(t *testing.T) {
	m := NewManager(accessKey, refreshKey)
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))

	cases := map[string]string{}

	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u1", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, ExpiresAt: future,
	}}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	cases["alg none"] = none

	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{UserID: "u1", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, ExpiresAt: future,
	}}).SignedString(accessKey)
	cases["HS512"] = hs512

	wrongIssuer, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "u1", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "someone-else", ExpiresAt: future,
	}}).SignedString(accessKey)
	cases["wrong issuer"] = wrongIssuer

	noExpiry, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "u1", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer,
	}}).SignedString(accessKey)
	cases["no exp"] = noExpiry

	for name, token := range cases {
		if _, err := m.ValidateAccess(token); err == nil {
			t.Errorf("%s: token accepted", name)
		}
	}
}

func TestHash(t *testing.T) {
	if Hash("a") == Hash("b") || Hash("a") != Hash("a") || len(Hash("a")) != 64 {
		t.Error("Hash must be a deterministic SHA-256 hex digest")
	}
}
