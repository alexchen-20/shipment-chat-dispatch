package logistics

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeGateway struct {
	verified  bool
	published bool
}

func (f *fakeGateway) VerifySession(context.Context, string) error {
	f.verified = true
	return nil
}

func (f *fakeGateway) IssueChannelToken(context.Context, string, string) ([]byte, error) {
	return []byte(`{"token":"room-token"}`), nil
}

func (f *fakeGateway) PublishShipmentEvent(context.Context, string, ShipmentEvent) error {
	f.published = true
	return nil
}

func TestDispatchShipmentEventDecisions(t *testing.T) {
	valid := ShipmentEvent{ID: "evt-19", ShipmentID: "SHP-42", AccountID: "acme", Type: "shipment.exception", ExceptionCode: "DAMAGED", OccurredAt: time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)}
	tests := []struct {
		name    string
		event   ShipmentEvent
		wantErr error
	}{
		{name: "exception is published when it has a code", event: valid},
		{name: "exception is rejected without a code", event: ShipmentEvent{ID: "evt-20", ShipmentID: "SHP-43", AccountID: "acme", Type: "shipment.exception", OccurredAt: valid.OccurredAt}, wantErr: errors.New("exception events require exception_code")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := &fakeGateway{}
			result, err := DispatchShipmentEvent(context.Background(), gateway, DispatchRequest{SessionID: "ses-1", ClientID: "dispatch-ui", Event: tt.event})
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() || gateway.verified || gateway.published {
					t.Fatalf("error=%v verified=%v published=%v", err, gateway.verified, gateway.published)
				}
				return
			}
			if err != nil || result.Channel != "shipment-SHP-42" || !gateway.verified || !gateway.published {
				t.Fatalf("result=%+v error=%v verified=%v published=%v", result, err, gateway.verified, gateway.published)
			}
		})
	}
}
