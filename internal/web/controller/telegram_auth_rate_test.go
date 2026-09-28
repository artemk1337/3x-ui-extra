package controller

import "testing"

func TestTelegramAuthRateLimitIPGroupsIPv6ByPrefix(t *testing.T) {
	first := telegramAuthRateLimitIP("2001:db8:1:2::1")
	second := telegramAuthRateLimitIP("2001:db8:1:2::ffff")
	if first != second {
		t.Fatalf("same IPv6 /64 has distinct rate keys: %q, %q", first, second)
	}
	if first == telegramAuthRateLimitIP("2001:db8:1:3::1") {
		t.Fatal("different IPv6 /64 shares a rate key")
	}
	if got := telegramAuthRateLimitIP("192.0.2.1"); got != "192.0.2.1" {
		t.Fatalf("IPv4 rate key = %q", got)
	}
}
