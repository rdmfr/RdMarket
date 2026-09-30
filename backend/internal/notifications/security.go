package notifications

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	webhookTimeout       = 10 * time.Second
	webhookResponseLimit = 64 * 1024
)

var blockedNetworks = func() []*net.IPNet {
	var networks []*net.IPNet
	for _, cidr := range []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
		"2001:db8::/32", "2001::/23", "2002::/16",
	} {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		networks = append(networks, network)
	}
	return networks
}()

type SecretCipher struct {
	aead cipher.AEAD
}

func NewSecretCipher(base64Key string) (*SecretCipher, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("application encryption key must be base64-encoded 32-byte data")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create secret cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create secret cipher: %w", err)
	}
	return &SecretCipher{aead: aead}, nil
}

func (c *SecretCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("create encryption nonce: %w", err)
	}
	return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *SecretCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.aead.NonceSize() {
		return nil, errors.New("encrypted secret is too short")
	}
	nonce, content := ciphertext[:c.aead.NonceSize()], ciphertext[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, content, nil)
	if err != nil {
		return nil, errors.New("encrypted secret authentication failed")
	}
	return plaintext, nil
}

func MaskSecret(secret []byte) string {
	if len(secret) == 0 {
		return ""
	}
	return "********"
}

type DNSResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type WebhookClient struct {
	Resolver DNSResolver
	Timeout  time.Duration
}

func (c WebhookClient) Send(ctx context.Context, target string, payload, secret []byte, production bool) error {
	if len(secret) < 32 {
		return fmt.Errorf("webhook signing secret must be at least 32 bytes")
	}
	if !json.Valid(payload) {
		return fmt.Errorf("webhook payload must be valid JSON")
	}
	parsed, err := url.ParseRequestURI(target)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("webhook URL is invalid")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && !(scheme == "http" && !production) {
		return fmt.Errorf("webhook URL must use HTTPS")
	}
	port := parsed.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("webhook URL port is invalid")
	}
	resolver := c.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	host := parsed.Hostname()
	addresses, err := resolveAndValidate(ctx, resolver, host)
	if err != nil {
		return err
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = webhookTimeout
	}
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		DialContext: func(dialCtx context.Context, network, address string) (net.Conn, error) {
			dialHost, dialPort, splitErr := net.SplitHostPort(address)
			if splitErr != nil || !strings.EqualFold(dialHost, host) || dialPort != port {
				return nil, fmt.Errorf("webhook dial target changed")
			}
			return dialer.DialContext(dialCtx, network, net.JoinHostPort(addresses[0].String(), port))
		},
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}
	client := &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-RdMarket-Timestamp", strconv.FormatInt(time.Now().UTC().Unix(), 10))
	request.Header.Set("X-RdMarket-Signature", webhookSignature(secret, request.Header.Get("X-RdMarket-Timestamp"), payload))
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, webhookResponseLimit+1))
	if err != nil {
		return fmt.Errorf("read webhook response: %w", err)
	}
	if len(responseBody) > webhookResponseLimit {
		return fmt.Errorf("webhook response exceeds size limit")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned HTTP %d", response.StatusCode)
	}
	return nil
}

func resolveAndValidate(ctx context.Context, resolver DNSResolver, host string) ([]net.IP, error) {
	if parsed := net.ParseIP(host); parsed != nil {
		if blockedIP(parsed) {
			return nil, fmt.Errorf("webhook host resolves to a blocked address")
		}
		return []net.IP{parsed}, nil
	}
	resolved, err := resolver.LookupIPAddr(ctx, host)
	if err != nil || len(resolved) == 0 {
		return nil, fmt.Errorf("webhook host could not be resolved")
	}
	addresses := make([]net.IP, 0, len(resolved))
	for _, address := range resolved {
		if blockedIP(address.IP) {
			return nil, fmt.Errorf("webhook host resolves to a blocked address")
		}
		addresses = append(addresses, address.IP)
	}
	return addresses, nil
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, network := range blockedNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return ip.Equal(net.ParseIP("168.63.129.16"))
}

func webhookSignature(secret []byte, timestamp string, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
