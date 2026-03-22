package secrets

import "testing"

func TestGenerate(t *testing.T) {
	s, err := Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(s.JWTSecret) != 64 {
		t.Errorf("JWTSecret length = %d, want 64", len(s.JWTSecret))
	}
	if len(s.JWTRefreshSecret) != 64 {
		t.Errorf("JWTRefreshSecret length = %d, want 64", len(s.JWTRefreshSecret))
	}
	if len(s.SessionSecret) != 64 {
		t.Errorf("SessionSecret length = %d, want 64", len(s.SessionSecret))
	}
	if len(s.EncryptionKey) != 32 {
		t.Errorf("EncryptionKey length = %d, want 32", len(s.EncryptionKey))
	}
	if s.RSAPublicKey == "" {
		t.Error("RSAPublicKey should not be empty")
	}
	if s.RSAPrivateKey == "" {
		t.Error("RSAPrivateKey should not be empty")
	}
	if len(s.DBPasswordMain) != 24 {
		t.Errorf("DBPasswordMain length = %d, want 24", len(s.DBPasswordMain))
	}
	if s.DBPasswordMain == s.DBPasswordMonitoring {
		t.Error("DB passwords should be unique")
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	s1, _ := Generate()
	s2, _ := Generate()
	if s1.JWTSecret == s2.JWTSecret {
		t.Error("two Generate calls should produce different JWTSecret")
	}
}
