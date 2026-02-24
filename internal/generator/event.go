package generator

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

// EventGenerator produces EventEnvelope instances.
type EventGenerator struct {
	registry *FamilyRegistry
	rng      *rand.Rand
}

// NewEventGenerator creates a new generator with the given family registry.
func NewEventGenerator(registry *FamilyRegistry, rng *rand.Rand) *EventGenerator {
	return &EventGenerator{
		registry: registry,
		rng:      rng,
	}
}

// Generate creates a single EventEnvelope with a random family/member.
func (g *EventGenerator) Generate() EventEnvelope {
	family := g.registry.RandomFamily()
	customerID := g.registry.RandomMember(family)
	return g.generateFor(family.ID, customerID)
}

func (g *EventGenerator) generateFor(familyID, customerID int64) EventEnvelope {
	appID := PickApp(g.rng)
	bytesUsed := GenerateBytesUsed(g.rng, appID)
	networkType := PickNetworkType(g.rng)
	deviceID := GenerateDeviceID(g.rng)

	envelopeID := uuid.New().String()

	// Java LocalDateTime format: no timezone
	ts := time.Now().Format("2006-01-02T15:04:05.000")

	return EventEnvelope{
		EventID:   envelopeID,
		EventType: "DATA_USAGE",
		Timestamp: ts,
		Payload: UsagePayload{
			FamilyID:   familyID,
			CustomerID: customerID,
			AppID:      appID,
			BytesUsed:  bytesUsed,
			Metadata: map[string]string{
				"deviceId":    deviceID,
				"networkType": networkType,
			},
		},
	}
}

// Registry returns the underlying FamilyRegistry.
func (g *EventGenerator) Registry() *FamilyRegistry {
	return g.registry
}
