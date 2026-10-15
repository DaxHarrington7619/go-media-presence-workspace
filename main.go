package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
)

type ingestRequest struct{ AssetID, CreatorID, Channel string }

func ingest(client *InfraiClient, in ingestRequest) (result map[string]any, err error) {
	if err = client.CreateChannel(in.Channel); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, client.DeleteChannel(in.Channel)) }()
	job := map[string]string{"asset_id": in.AssetID, "state": "queued"}
	if err = client.Publish(in.Channel, "asset.queued", job, in.CreatorID); err != nil {
		return nil, err
	}
	var presence PresenceSnapshot
	if err = client.Presence(in.Channel, &presence); err != nil {
		return nil, err
	}
	return map[string]any{"job": job, "online_creators": len(presence.Members), "decision": deliveryDecision(len(presence.Members))}, nil
}

func main() {
	client, err := NewInfraiClient()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var in ingestRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		result, err := ingest(client, in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("media presence service listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
