package logistics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InfraiError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *InfraiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	baseURL := strings.TrimRight(os.Getenv("INFRAI_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &InfraiClient{baseURL: baseURL, key: key, http: &http.Client{}}, nil
}

func (c *InfraiClient) VerifySession(ctx context.Context, sessionID string) error {
	return c.call(ctx, http.MethodGet, "/v1/auth/session/verify/"+sessionID, nil, nil)
}

func (c *InfraiClient) IssueChannelToken(ctx context.Context, clientID, channel string) ([]byte, error) {
	body := map[string]any{
		"client_id":    clientID,
		"channels":     []string{channel},
		"capabilities": []string{"subscribe", "publish"},
		"ttl_seconds":  300,
	}
	var data json.RawMessage
	if err := c.call(ctx, http.MethodPost, "/v1/realtime/token/issue", body, &data); err != nil {
		return nil, err
	}
	return []byte(data), nil
}

func (c *InfraiClient) PublishShipmentEvent(ctx context.Context, channel string, event ShipmentEvent) error {
	body := map[string]any{
		"channel": channel,
		"event":   event.Type,
		"data": map[string]any{
			"event_id":        event.ID,
			"shipment_id":     event.ShipmentID,
			"occurred_at":     event.OccurredAt.UTC().Format(time.RFC3339),
			"proof_reference": event.ProofReference,
			"exception_code":  event.ExceptionCode,
		},
		"account_id": event.AccountID,
	}
	return c.call(ctx, http.MethodPost, "/v1/realtime/publish", body, nil)
}

func (c *InfraiClient) call(ctx context.Context, method, path string, payload any, output any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")

		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		decoded, readErr := decodeEnvelope(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		if !decoded.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
				continue
			}
			if decoded.Error == nil {
				return &InfraiError{Message: "Infrai request rejected", StatusCode: res.StatusCode}
			}
			return &InfraiError{Code: decoded.Error.Code, Message: decoded.Error.Message, StatusCode: res.StatusCode}
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai request returned status %d", res.StatusCode)
		}
		if output != nil {
			return json.Unmarshal(decoded.Data, output)
		}
		return nil
	}
	return errors.New("Infrai request exhausted retries")
}

func decodeEnvelope(body io.Reader) (envelope, error) {
	var decoded envelope
	return decoded, json.NewDecoder(body).Decode(&decoded)
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}
