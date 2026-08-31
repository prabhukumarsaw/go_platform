package iam

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// EvalContext carries the request-level attributes needed by the ABAC evaluator.
type EvalContext struct {
	IPAddress string
	UserAgent string
	RequestID string
	Now       time.Time
	DeviceType string // "mobile", "desktop", "tablet"
}

// evaluateABAC checks all active ABAC policies for a user-tenant pair.
// Returns true if ALL policies pass (AND-logic).
// If the user has no ABAC policies, this returns true (no constraints = pass).
func evaluateABAC(policies []ABACPolicy, evalCtx *EvalContext) bool {
	if len(policies) == 0 {
		return true
	}

	for _, p := range policies {
		if !evaluatePolicy(p, evalCtx) {
			return false
		}
	}
	return true
}

// evaluatePolicy dispatches to the appropriate checker based on the attribute type.
func evaluatePolicy(p ABACPolicy, ctx *EvalContext) bool {
	switch strings.ToUpper(p.Attribute) {
	case "TIME_RANGE":
		return checkTimeRange(p.Value, ctx.Now)
	case "IP_WHITELIST":
		return checkIPWhitelist(p.Value, ctx.IPAddress)
	case "DEVICE_TYPE":
		return checkDeviceType(p.Value, ctx.DeviceType)
	case "DISTRICT_RESTRICT":
		// District restrictions are enforced at the SQL query level, not here.
		// The evaluator passes this through; the content module adds WHERE clauses.
		return true
	default:
		// Unknown attribute type — deny by default (secure fail).
		return false
	}
}

// ─── Individual ABAC checks ─────────────────────

// checkTimeRange verifies the current time is within an allowed window.
// Expected JSON: {"from": "09:00", "to": "20:00"}
func checkTimeRange(value []byte, now time.Time) bool {
	var tr struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(value, &tr); err != nil {
		return false
	}

	fromTime, err := time.Parse("15:04", tr.From)
	if err != nil {
		return false
	}
	toTime, err := time.Parse("15:04", tr.To)
	if err != nil {
		return false
	}

	currentMinutes := now.Hour()*60 + now.Minute()
	fromMinutes := fromTime.Hour()*60 + fromTime.Minute()
	toMinutes := toTime.Hour()*60 + toTime.Minute()

	if fromMinutes <= toMinutes {
		// Same-day range (e.g., 09:00 to 20:00)
		return currentMinutes >= fromMinutes && currentMinutes <= toMinutes
	}
	// Overnight range (e.g., 22:00 to 06:00)
	return currentMinutes >= fromMinutes || currentMinutes <= toMinutes
}

// checkIPWhitelist verifies the request IP is within allowed CIDR ranges.
// Expected JSON: {"ips": ["10.0.0.0/8", "192.168.1.0/24"]}
func checkIPWhitelist(value []byte, ipAddr string) bool {
	var wl struct {
		IPs []string `json:"ips"`
	}
	if err := json.Unmarshal(value, &wl); err != nil {
		return false
	}

	if len(wl.IPs) == 0 {
		return true // no whitelist = allow all
	}

	requestIP := net.ParseIP(ipAddr)
	if requestIP == nil {
		return false
	}

	for _, cidr := range wl.IPs {
		// Support both CIDR notation and plain IPs
		if !strings.Contains(cidr, "/") {
			if cidr == ipAddr {
				return true
			}
			continue
		}

		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(requestIP) {
			return true
		}
	}
	return false
}

// checkDeviceType verifies the request device type is in the allowed list.
// Expected JSON: {"allowed": ["desktop", "mobile"]}
func checkDeviceType(value []byte, deviceType string) bool {
	var dt struct {
		Allowed []string `json:"allowed"`
	}
	if err := json.Unmarshal(value, &dt); err != nil {
		return false
	}

	if len(dt.Allowed) == 0 {
		return true // no restriction
	}

	for _, allowed := range dt.Allowed {
		if strings.EqualFold(allowed, deviceType) {
			return true
		}
	}
	return false
}

// decisionString returns "GRANTED" or "DENIED" based on the boolean.
func decisionString(allowed bool) string {
	if allowed {
		return "GRANTED"
	}
	return "DENIED"
}

// FormatAction builds the canonical action string "menu.ACTION" from
// a menu name and action verb.
func FormatAction(menu, action string) string {
	return fmt.Sprintf("%s.%s", menu, action)
}
