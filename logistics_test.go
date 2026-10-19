package main

import "testing"

func TestVerificationState(t *testing.T) {
	tests := []struct {
		name   string
		events []ShipmentEvent
		proof  *ProofOfDelivery
		want   string
	}{{"needs proof", []ShipmentEvent{{Code: "delivered"}}, nil, "pending_review"}, {"ready", []ShipmentEvent{{Code: "delivered"}}, &ProofOfDelivery{URL: "https://files.example/pod.pdf", SHA256: "abc"}, "ready_for_dispatch"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verificationState(tt.events, tt.proof); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
