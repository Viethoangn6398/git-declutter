package config

import "testing"

func TestParseDuration(t *testing.T) {
	d, err := ParseDuration("7d")
	if err != nil || d.Hours() != 7*24 {
		t.Fatalf("7d: %v %v", d, err)
	}
	d, err = ParseDuration("30d")
	if err != nil || d.Hours() != 30*24 {
		t.Fatalf("30d: %v %v", d, err)
	}
}

func TestRetentionForever(t *testing.T) {
	cfg := Defaults()
	cfg.Recovery.Retention = "forever"
	_, forever, err := cfg.RetentionDuration()
	if err != nil || !forever {
		t.Fatalf("forever: %v %v", forever, err)
	}
}

func TestMatchesProtected(t *testing.T) {
	p := Defaults().Protected
	if ok, _ := MatchesProtected(p, "main"); !ok {
		t.Fatal("main should be protected")
	}
	if ok, _ := MatchesProtected(p, "release/1.2"); !ok {
		t.Fatal("release/1.2 should match release/*")
	}
	if ok, _ := MatchesProtected(p, "feature/x"); ok {
		t.Fatal("feature/x should not be protected")
	}
}
