package transport

import (
	"log"
	"net/http"
	"os"
)

// GET /v1/whatsapp/webhook — Meta (Facebook) webhook verification endpoint.
//
// When configuring the WhatsApp Business API webhook, Meta sends a GET request
// containing a challenge integer and a verify_token. The server must echo back
// the challenge if the token matches our environment variable exactly.
func (s *Server) handleVerifyWhatsAppWebhook(w http.ResponseWriter, r *http.Request) {
	expectedToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
	if expectedToken == "" {
		expectedToken = "turbo_tax_ninja_2026" // Default fallback for ease of setup
	}

	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == expectedToken {
		log.Printf("whatsapp: webhook verified successfully")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	w.WriteHeader(http.StatusForbidden)
}

// POST /v1/whatsapp/webhook — Meta (Facebook) webhook messaging endpoint.
//
// This is called by Meta whenever a user sends a message to our WhatsApp Number.
func (s *Server) handleReceiveWhatsAppMessage(w http.ResponseWriter, r *http.Request) {
	// Acknowledge the message instantly to prevent Meta from retrying
	w.WriteHeader(http.StatusOK)
	
	// Payload handling to be implemented alongside the bot flow...
}
