package notifications

import (
	"context"
	"encoding/base64"
	"net"
	"testing"
)

func TestSecretCipherRoundTripAndMasking(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	cipher, err := NewSecretCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("webhook-secret")
	encrypted, err := cipher.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	if string(encrypted) == string(secret) || MaskSecret(secret) == string(secret) {
		t.Fatal("secret was returned without protection")
	}
	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil || string(decrypted) != string(secret) {
		t.Fatalf("secret round-trip failed: %v", err)
	}
	encrypted[len(encrypted)-1] ^= 1
	if _, err := cipher.Decrypt(encrypted); err == nil {
		t.Fatal("tampered ciphertext must fail authentication")
	}
}

func TestBlockedIPRanges(t *testing.T) {
	for _, address := range []string{
		"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.169.254",
		"100.64.0.1", "192.0.2.1", "198.18.0.1", "::1", "fc00::1", "fe80::1",
		"2001:db8::1", "224.0.0.1", "168.63.129.16",
	} {
		if !blockedIP(net.ParseIP(address)) {
			t.Errorf("expected %s to be blocked", address)
		}
	}
	for _, address := range []string{"1.1.1.1", "2606:4700:4700::1111"} {
		if blockedIP(net.ParseIP(address)) {
			t.Errorf("expected %s to be allowed", address)
		}
	}
}

type staticResolver []net.IPAddr

func (r staticResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r, nil
}

func TestWebhookTargetRejectsPrivateResolutionAndHTTPProduction(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	client := WebhookClient{Resolver: staticResolver{{IP: net.ParseIP("10.0.0.1")}}}
	if err := client.Send(context.Background(), "https://hooks.example.test/event", []byte(`{}`), secret, true); err == nil {
		t.Fatal("expected private DNS result to be blocked")
	}
	client.Resolver = staticResolver{{IP: net.ParseIP("1.1.1.1")}}
	if err := client.Send(context.Background(), "http://hooks.example.test/event", []byte(`{}`), secret, true); err == nil {
		t.Fatal("production webhook must require HTTPS")
	}
	if err := client.Send(context.Background(), "https://hooks.example.test/event", []byte(`{}`), []byte("short"), true); err == nil {
		t.Fatal("short webhook signing secrets must be rejected")
	}
}
