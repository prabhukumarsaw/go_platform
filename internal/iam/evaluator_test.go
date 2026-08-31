package iam

import (
	"testing"
	"time"
)

func TestABACTimeRange(t *testing.T) {
	policy := []byte(`{"from":"09:00","to":"18:00"}`)

	// Test within window (14:30)
	midday := time.Date(2026, 9, 1, 14, 30, 0, 0, time.UTC)
	if !checkTimeRange(policy, midday) {
		t.Errorf("expected 14:30 to be within 09:00-18:00 window")
	}

	// Test before window (08:00)
	early := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	if checkTimeRange(policy, early) {
		t.Errorf("expected 08:00 to be outside 09:00-18:00 window")
	}

	// Test after window (20:00)
	late := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	if checkTimeRange(policy, late) {
		t.Errorf("expected 20:00 to be outside 09:00-18:00 window")
	}
}

func TestABACIPWhitelist(t *testing.T) {
	policy := []byte(`{"ips":["192.168.1.0/24", "10.0.0.5"]}`)

	// Subnet match
	if !checkIPWhitelist(policy, "192.168.1.55") {
		t.Errorf("expected 192.168.1.55 to match subnet 192.168.1.0/24")
	}

	// Exact IP match
	if !checkIPWhitelist(policy, "10.0.0.5") {
		t.Errorf("expected 10.0.0.5 to match exact IP")
	}

	// Unlisted IP
	if checkIPWhitelist(policy, "172.16.0.1") {
		t.Errorf("expected 172.16.0.1 to be denied")
	}
}

func TestABACDeviceType(t *testing.T) {
	policy := []byte(`{"allowed":["desktop","tablet"]}`)

	if !checkDeviceType(policy, "desktop") {
		t.Errorf("expected desktop to be allowed")
	}
	if checkDeviceType(policy, "mobile") {
		t.Errorf("expected mobile to be blocked")
	}
}

func TestFormatAction(t *testing.T) {
	action := FormatAction("articles", "PUBLISH")
	if action != "articles.PUBLISH" {
		t.Errorf("expected 'articles.PUBLISH', got '%s'", action)
	}
}
