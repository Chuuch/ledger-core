// Package Auth
package auth

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Config struct {
	Issuer string
}

type Authenticator struct {
	issuer string
	keySet *oidc.RemoteKeySet
}

type Claims struct {
	Issuer string `json:"iss"`
	Subject string `json:"sub"`
	AZP string `json:"azp"`
	Expiry int64 `json:"exp"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

type ctxKey int

const claimsKey ctxKey = 1

func NewAuthenticator(ctx context.Context, cfg Config) (*Authenticator, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("auth: issuer is required")
	}
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("auth: oidc provider: %w", err)
	}

	var disc struct {
		Issuer string `json:"issuer"`
		JWKSURL string `json:"jwks_uri"`
	}
	if err := provider.Claims(&disc); err != nil {
		return nil, fmt.Errorf("auth: discovery claims: %w", err)
	}
	if disc.Issuer == "" || disc.JWKSURL == "" {
		return nil, fmt.Errorf("auth: discovery missing issuer or jwks_uri")
	}

	return &Authenticator{
		issuer: disc.Issuer,
		keySet: oidc.NewRemoteKeySet(ctx, disc.JWKSURL),
	}, nil
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "missing bearer token", "")
			return
		}
		claims, err := a.verifyAccessToken(r.Context(), raw)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "invalid token", err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Authenticator) verifyAccessToken(ctx context.Context, raw string) (Claims, error) {
	payload, err := a.keySet.VerifySignature(ctx, raw)
	if err != nil {
		return Claims{}, fmt.Errorf("signature: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("claims: %w", err)
	}
	if claims.Issuer != a.issuer {
		return Claims{}, fmt.Errorf("issuer mismatch: token=%q discovery=%q", claims.Issuer, a.issuer)
	}
	if claims.Expiry == 0 {
		return Claims{}, fmt.Errorf("missing exp")
	}
	if time.Now().UTC().After(time.Unix(claims.Expiry, 0).UTC()) {
		return Claims{}, fmt.Errorf("token expired")
	}
	return claims, nil
}

func RequireRoles (roles ...string) func(http.Handler) http.Handler {
	needed := roles
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "")
				return
			}
			if !hasAnyRole(claims.RealmAccess.Roles, needed) {
				writeAuthError(w, http.StatusForbidden, "insufficient role", "")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}

func bearerToken(h string) (string, bool) {
	h = strings.TrimSpace(h)
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	tok := strings.TrimSpace(parts[1])
	return tok, tok != ""
}

func hasAnyRole(have, need []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, r := range have {
		set[r] = struct{}{}
	}
	for _, n := range need {
		if _, ok := set[n]; ok {
			return true
		}
	}
	return false
}

func writeAuthError(w http.ResponseWriter, status int, msg, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]string{"error": msg}
	if detail != "" {
		body["detail"] = detail
	}
	_ = json.MarshalWrite(w, body)
}
