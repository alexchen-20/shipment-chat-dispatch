package logistics

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ShipmentEvent struct {
	ID             string    `json:"id"`
	ShipmentID     string    `json:"shipment_id"`
	AccountID      string    `json:"account_id"`
	Type           string    `json:"type"`
	OccurredAt     time.Time `json:"occurred_at"`
	ProofReference string    `json:"proof_reference"`
	ExceptionCode  string    `json:"exception_code"`
}

type DispatchRequest struct {
	SessionID string        `json:"session_id"`
	ClientID  string        `json:"client_id"`
	Event     ShipmentEvent `json:"event"`
}

type DispatchResult struct {
	Channel string `json:"channel"`
	Token   any    `json:"token"`
}

type RealtimeGateway interface {
	VerifySession(context.Context, string) error
	IssueChannelToken(context.Context, string, string) ([]byte, error)
	PublishShipmentEvent(context.Context, string, ShipmentEvent) error
}

func DispatchShipmentEvent(ctx context.Context, gateway RealtimeGateway, request DispatchRequest) (DispatchResult, error) {
	if err := validateEvent(request.Event); err != nil {
		return DispatchResult{}, err
	}
	if request.SessionID == "" || request.ClientID == "" {
		return DispatchResult{}, errors.New("session_id and client_id are required")
	}
	if err := gateway.VerifySession(ctx, request.SessionID); err != nil {
		return DispatchResult{}, fmt.Errorf("verify session: %w", err)
	}
	channel := "shipment-" + request.Event.ShipmentID
	token, err := gateway.IssueChannelToken(ctx, request.ClientID, channel)
	if err != nil {
		return DispatchResult{}, fmt.Errorf("issue channel token: %w", err)
	}
	if err := gateway.PublishShipmentEvent(ctx, channel, request.Event); err != nil {
		return DispatchResult{}, fmt.Errorf("publish shipment event: %w", err)
	}
	return DispatchResult{Channel: channel, Token: string(token)}, nil
}

func validateEvent(event ShipmentEvent) error {
	if event.ID == "" || event.ShipmentID == "" || event.AccountID == "" || event.Type == "" || event.OccurredAt.IsZero() {
		return errors.New("event id, shipment_id, account_id, type, and occurred_at are required")
	}
	if event.Type == "shipment.exception" && event.ExceptionCode == "" {
		return errors.New("exception events require exception_code")
	}
	if event.Type == "shipment.delivered" && event.ProofReference == "" {
		return errors.New("delivered events require proof_reference")
	}
	return nil
}
