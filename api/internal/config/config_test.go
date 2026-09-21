package config

import "testing"

func TestJoinLimit(t *testing.T) {
	tests := []struct {
		name, env, value string
		want             int
		wantErr          bool
	}{
		{"production default", "production", "", 10, false},
		{"development default is off", "development", "", 0, false},
		{"explicit value wins in development", "development", "3", 3, false},
		{"explicit zero disables", "production", "0", 0, false},
		{"not a number", "production", "many", 0, true},
		{"negative", "production", "-1", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RATE_LIMIT_JOIN_PER_MIN", tt.value)
			got, err := joinLimit(tt.env)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("joinLimit(%q) = %d, %v; want %d, error=%v", tt.env, got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestLoadReadsProxyAndRequiresDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Error("missing DATABASE_URL should fail")
	}
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("TRUST_PROXY", "true")
	t.Setenv("ENV", "production")
	t.Setenv("RATE_LIMIT_JOIN_PER_MIN", "")
	cfg, err := Load()
	if err != nil || !cfg.TrustProxy || cfg.JoinLimitPerMinute != 10 {
		t.Errorf("cfg = %+v, err = %v", cfg, err)
	}
}
