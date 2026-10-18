package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"shipment-chat/internal/logistics"
)

func main() {
	client, err := logistics.NewInfraiClient()
	if err != nil {
		log.Fatal(err)
	}
	handler := shipmentHandler(client)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func shipmentHandler(client *logistics.InfraiClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/shipment-events" {
			http.NotFound(w, r)
			return
		}
		var input logistics.DispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		result, err := logistics.DispatchShipmentEvent(r.Context(), client, input)
		if err != nil {
			status := http.StatusBadRequest
			var infraiErr *logistics.InfraiError
			if errors.As(err, &infraiErr) && infraiErr.StatusCode >= 400 && infraiErr.StatusCode < 500 {
				status = infraiErr.StatusCode
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, result)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
