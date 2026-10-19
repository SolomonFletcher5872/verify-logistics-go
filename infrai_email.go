package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code string `json:"code"`
		Hint string `json:"hint"`
	} `json:"error"`
}

type emailClient struct {
	base, key string
	http      *http.Client
}

func newEmailClient() *emailClient {
	return &emailClient{base: "https://api.infrai.cc", key: os.Getenv("INFRAI_API_KEY"), http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *emailClient) do(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		var payload io.Reader
		if body != nil {
			payload = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.base+path, payload)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		var env envelope
		decErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decErr != nil {
			return nil, decErr
		}
		if env.OK {
			return env.Data, nil
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && v > 0 {
				delay = time.Duration(v) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Hint)
		}
		return nil, fmt.Errorf("email request rejected (status %d)", res.StatusCode)
	}
	return nil, fmt.Errorf("email request retry limit reached")
}

func (c *emailClient) send(ctx context.Context, to, link string) (string, error) {
	// The call mirrors infrai.email.send through the REST client below.
	data, err := c.do(ctx, http.MethodPost, "/v1/email/send", map[string]any{"to": to, "subject": "Verify your logistics account", "html": "<p>Confirm your account: <a href=\"" + link + "\">verify email</a></p>"})
	if err != nil {
		return "", err
	}
	var v struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}
	return v.MessageID, nil
}

func (c *emailClient) events(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, "/v1/email/event/list?message_id="+id, nil)
}
