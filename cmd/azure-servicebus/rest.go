package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/hitenpratap/mcp-azure-service-bus/internal/servicebus"
)

func main1() {
	sbClient, err := servicebus.NewClient("config/config.yaml")
	if err != nil {
		log.Fatalf("failed to create service bus client: %v", err)
	}

	http.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		var from, to *time.Time
		if f := r.URL.Query().Get("from"); f != "" {
			t, err := time.Parse(time.RFC3339, f)
			if err == nil {
				from = &t
			}
		}
		if tStr := r.URL.Query().Get("to"); tStr != "" {
			t, err := time.Parse(time.RFC3339, tStr)
			if err == nil {
				to = &t
			}
		}
		msgs, err := sbClient.ListMessages(r.Context(), from, to)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(msgs)
	})

	http.HandleFunc("/deadletters", func(w http.ResponseWriter, r *http.Request) {
		var from, to *time.Time
		if f := r.URL.Query().Get("from"); f != "" {
			t, err := time.Parse(time.RFC3339, f)
			if err == nil {
				from = &t
			}
		}
		if tStr := r.URL.Query().Get("to"); tStr != "" {
			t, err := time.Parse(time.RFC3339, tStr)
			if err == nil {
				to = &t
			}
		}
		msgs, err := sbClient.ListDeadLetters(r.Context(), from, to)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(msgs)
	})

	http.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		seqStr := r.URL.Query().Get("seq")
		if seqStr == "" {
			http.Error(w, "missing seq parameter", http.StatusBadRequest)
			return
		}
		seq, err := strconv.ParseInt(seqStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid seq parameter", http.StatusBadRequest)
			return
		}
		deadLetter := r.URL.Query().Get("deadletter") == "true"

		msg, err := sbClient.FetchMessage(r.Context(), seq, deadLetter)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Prepare a response struct with exported fields
		resp := struct {
			SequenceNumber int64                  `json:"sequenceNumber"`
			EnqueuedTime   string                 `json:"enqueuedTime"`
			Body           string                 `json:"body"`
			Properties     map[string]interface{} `json:"properties"`
			SystemProps    map[string]interface{} `json:"systemProperties"`
		}{
			SequenceNumber: *msg.SequenceNumber,
			EnqueuedTime:   msg.EnqueuedTime.Format(time.RFC3339),
			Body:           string(msg.Body),
			Properties:     msg.ApplicationProperties,
			SystemProps: map[string]interface{}{
				"MessageID":      msg.MessageID,
				"ContentType":    msg.ContentType,
				"CorrelationID":  msg.CorrelationID,
				"Subject":        msg.Subject,
				"To":             msg.To,
				"ReplyTo":        msg.ReplyTo,
				"ReplyToSession": msg.ReplyToSessionID,
				"SessionID":      msg.SessionID,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
