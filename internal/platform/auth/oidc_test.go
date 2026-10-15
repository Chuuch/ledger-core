package auth

import "testing"

func TestBearerToken(t *testing.T) {
	t.Parallel()
	tok, ok := bearerToken("Bearer abc.def.ghi")
	if !ok || tok != "abc.def.ghi" {
		t.Fatalf("got %q %v", tok, ok)
	}
	if _, ok := bearerToken("Basic x"); ok {
		t.Fatal("expected false")
	}
}

func TestHasAnyRole(t *testing.T) {
	t.Parallel()
	if !hasAnyRole([]string{"account-read", "x"}, []string{"account-write", "account-read"}) {
		t.Fatal("expected match")
	}
	if hasAnyRole([]string{"account-read"}, []string{"account-write"}) {
		t.Fatal("expected no match")
	}
}
