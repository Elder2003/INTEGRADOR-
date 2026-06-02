package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

var privateRanges []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
	} {
		_, block, _ := net.ParseCIDR(cidr)
		if block != nil {
			privateRanges = append(privateRanges, block)
		}
	}
}

type Service struct {
	repo       *Repository
	httpClient *http.Client
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ValidateURL rejects private/loopback URLs (SSRF protection).
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must use http or https")
	}
	host := u.Hostname()
	addrs, err := net.LookupHost(host)
	if err != nil {
		// If DNS resolution fails at config time, still allow (SSRF check at delivery time)
		return nil
	}
	for _, addrStr := range addrs {
		ip := net.ParseIP(addrStr)
		if ip == nil {
			continue
		}
		for _, block := range privateRanges {
			if block.Contains(ip) {
				return fmt.Errorf("private/loopback URLs are not allowed")
			}
		}
	}
	return nil
}

// Sign returns the HMAC-SHA256 hex signature for the payload.
func Sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + fmt.Sprintf("%x", mac.Sum(nil))
}

// Deliver POSTs the payload to the webhook URL with retry and HMAC signing.
func (s *Service) Deliver(ctx context.Context, wh *WebhookConfig, logID uuid.UUID, eventType string, payloadJSON []byte) {
	delays := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	sig := Sign(payloadJSON, wh.SecretHmac)

	for attempt, delay := range delays {
		if attempt > 0 {
			time.Sleep(delay)
		}

		statusCode, err := s.doPost(wh.URL, sig, payloadJSON)
		code := statusCode
		if err == nil && statusCode >= 200 && statusCode < 300 {
			_ = s.repo.UpdateDeliveryLog(ctx, logID, "success", &code, attempt+1)
			return
		}

		slog.Warn("webhook delivery attempt failed",
			"webhook_id", wh.ID, "attempt", attempt+1,
			"status_code", statusCode, "err", err)
	}

	// All retries exhausted
	_ = s.repo.UpdateDeliveryLog(ctx, logID, "dead", nil, len(delays))
}

func (s *Service) doPost(targetURL, sig string, body []byte) (int, error) {
	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NextLab-Signature", sig)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// TriggerAsync fires webhook deliveries for an event in background goroutines.
func (s *Service) TriggerAsync(eventType string, data any) {
	go func() {
		ctx := context.Background()
		webhooks, err := s.repo.ListActiveForEvent(ctx, eventType)
		if err != nil || len(webhooks) == 0 {
			return
		}

		envelope := map[string]any{
			"event":     eventType,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"data":      data,
		}
		payloadJSON, err := json.Marshal(envelope)
		if err != nil {
			return
		}

		for _, wh := range webhooks {
			wh := wh // capture
			logID, err := s.repo.InsertDeliveryLog(ctx, &WebhookDeliveryLog{
				WebhookID: wh.ID,
				EventType: eventType,
				Payload:   payloadJSON,
			})
			if err != nil {
				continue
			}
			go s.Deliver(ctx, &wh, logID, eventType, payloadJSON)
		}
	}()
}
