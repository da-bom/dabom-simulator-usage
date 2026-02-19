package generator

import (
	"encoding/json"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

func TestGenerate_SchemaCompliance(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := NewFamilyRegistry(100, rng)
	gen := NewEventGenerator(reg, rng)

	for i := 0; i < 100; i++ {
		env := gen.Generate()

		// Envelope fields
		uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		if !uuidRe.MatchString(env.EventID) {
			t.Errorf("eventId %q is not a valid UUID", env.EventID)
		}
		if env.EventType != "DATA_USAGE" {
			t.Errorf("eventType = %q, want DATA_USAGE", env.EventType)
		}
		if env.SubType != "" {
			t.Errorf("subType = %q, want empty", env.SubType)
		}

		// Timestamp format: 2006-01-02T15:04:05.000
		tsRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}$`)
		if !tsRe.MatchString(env.Timestamp) {
			t.Errorf("timestamp %q doesn't match LocalDateTime format", env.Timestamp)
		}

		// Payload fields
		p := env.Payload
		evtRe := regexp.MustCompile(`^evt_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		if !evtRe.MatchString(p.EventID) {
			t.Errorf("payload eventId %q doesn't match evt_UUID format", p.EventID)
		}
		if p.FamilyID < 1 {
			t.Errorf("familyId = %d, want > 0", p.FamilyID)
		}
		if p.CustomerID < 1 {
			t.Errorf("customerId = %d, want > 0", p.CustomerID)
		}
		if p.AppID == "" {
			t.Error("appId is empty")
		}
		if p.BytesUsed <= 0 {
			t.Errorf("bytesUsed = %d, want > 0", p.BytesUsed)
		}
		if p.Metadata["deviceId"] == "" {
			t.Error("metadata.deviceId is empty")
		}
		networkType := p.Metadata["networkType"]
		if networkType != "4G" && networkType != "5G" && networkType != "WIFI" {
			t.Errorf("networkType = %q, want 4G/5G/WIFI", networkType)
		}
	}
}

func TestGenerate_JSONRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := NewFamilyRegistry(10, rng)
	gen := NewEventGenerator(reg, rng)

	env := gen.Generate()
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	// Verify subType is omitted
	if strings.Contains(string(data), `"subType"`) {
		t.Error("subType should be omitted in JSON when empty")
	}

	var decoded EventEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.EventID != env.EventID {
		t.Errorf("roundtrip eventId = %q, want %q", decoded.EventID, env.EventID)
	}
	if decoded.Payload.FamilyID != env.Payload.FamilyID {
		t.Errorf("roundtrip familyId = %d, want %d", decoded.Payload.FamilyID, env.Payload.FamilyID)
	}
	if decoded.Payload.BytesUsed != env.Payload.BytesUsed {
		t.Errorf("roundtrip bytesUsed = %d, want %d", decoded.Payload.BytesUsed, env.Payload.BytesUsed)
	}
}

func TestGenerate_UniqueEventIDs(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := NewFamilyRegistry(10, rng)
	gen := NewEventGenerator(reg, rng)

	seen := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		env := gen.Generate()
		if seen[env.EventID] {
			t.Fatalf("duplicate envelope eventId: %s", env.EventID)
		}
		if seen[env.Payload.EventID] {
			t.Fatalf("duplicate payload eventId: %s", env.Payload.EventID)
		}
		seen[env.EventID] = true
		seen[env.Payload.EventID] = true
	}
}
