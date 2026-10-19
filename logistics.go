package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type ShipmentEvent struct {
	Code string `json:"code"`
	At   string `json:"at"`
}
type ProofOfDelivery struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}
type Signup struct {
	Email      string           `json:"email"`
	ShipmentID string           `json:"shipment_id"`
	Events     []ShipmentEvent  `json:"events"`
	Proof      *ProofOfDelivery `json:"proof_of_delivery,omitempty"`
}
type signupReply struct {
	ShipmentID string `json:"shipment_id"`
	MessageID  string `json:"message_id"`
	State      string `json:"state"`
}

func verificationState(events []ShipmentEvent, proof *ProofOfDelivery) string {
	if len(events) == 0 || proof == nil {
		return "pending_review"
	}
	return "ready_for_dispatch"
}

func signupHandler(client *emailClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var in Signup
		if json.NewDecoder(r.Body).Decode(&in) != nil || !strings.Contains(in.Email, "@") || in.ShipmentID == "" {
			http.Error(w, "invalid signup", 400)
			return
		}
		link := "https://logistics.example/verify?shipment_id=" + in.ShipmentID
		id, err := client.send(context.Background(), in.Email, link)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		_, _ = client.events(context.Background(), id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(signupReply{ShipmentID: in.ShipmentID, MessageID: id, State: verificationState(in.Events, in.Proof)})
	}
}
