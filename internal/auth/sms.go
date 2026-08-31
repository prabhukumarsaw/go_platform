package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// SMSProvider defines the interface for dispatching phone OTP SMS.
type SMSProvider interface {
	SendOTP(ctx context.Context, phone, otp string) error
}

// ─── Console Provider (Local / Dev) ─────────────

type ConsoleSMSProvider struct {
	logger zerolog.Logger
}

func NewConsoleSMSProvider(logger zerolog.Logger) *ConsoleSMSProvider {
	return &ConsoleSMSProvider{logger: logger}
}

func (c *ConsoleSMSProvider) SendOTP(ctx context.Context, phone, otp string) error {
	c.logger.Info().
		Str("phone", phone).
		Str("otp", otp).
		Msg("🔑 [SMS OTP DISPATCH - CONSOLE DEV] Verification Code: " + otp)
	return nil
}

// ─── MSG91 Provider (India DLT Compliant) ───────

type MSG91Provider struct {
	authKey    string
	templateID string
	httpClient *http.Client
	logger     zerolog.Logger
}

func NewMSG91Provider(authKey, templateID string, logger zerolog.Logger) *MSG91Provider {
	return &MSG91Provider{
		authKey:    authKey,
		templateID: templateID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (m *MSG91Provider) SendOTP(ctx context.Context, phone, otp string) error {
	// Standardize phone for India (e.g. +91)
	cleanPhone := strings.TrimPrefix(phone, "+")
	if len(cleanPhone) == 10 {
		cleanPhone = "91" + cleanPhone
	}

	payload := map[string]interface{}{
		"template_id": m.templateID,
		"short_url":   "0",
		"recipients": []map[string]interface{}{
			{
				"mobiles": cleanPhone,
				"otp":     otp,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://control.msg91.com/api/v5/flow/", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("authkey", m.authKey)
	req.Header.Set("content-type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("msg91 dispatch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("msg91 returned HTTP error %d", resp.StatusCode)
	}

	m.logger.Info().Str("phone", cleanPhone).Msg("OTP SMS dispatched via MSG91")
	return nil
}

// ─── Twilio Provider (Global Fallback) ──────────

type TwilioProvider struct {
	accountSID string
	authToken  string
	fromNumber string
	httpClient *http.Client
	logger     zerolog.Logger
}

func NewTwilioProvider(accountSID, authToken, fromNumber string, logger zerolog.Logger) *TwilioProvider {
	return &TwilioProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (t *TwilioProvider) SendOTP(ctx context.Context, phone, otp string) error {
	msgData := url.Values{}
	msgData.Set("To", phone)
	msgData.Set("From", t.fromNumber)
	msgData.Set("Body", fmt.Sprintf("Your BharatVani verification code is: %s. Valid for 5 minutes.", otp))

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(msgData.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("twilio dispatch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio returned HTTP error %d", resp.StatusCode)
	}

	t.logger.Info().Str("phone", phone).Msg("OTP SMS dispatched via Twilio")
	return nil
}
