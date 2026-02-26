package generator

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventGenerator produces EventEnvelope instances.
type EventGenerator struct {
	registry     *FamilyRegistry
	rng          *rand.Rand
	fixedTargets []FixedTarget
	mu           sync.RWMutex
}

// NewEventGenerator creates a new generator with the given family registry.
func NewEventGenerator(registry *FamilyRegistry, rng *rand.Rand) *EventGenerator {
	return &EventGenerator{
		registry: registry,
		rng:      rng,
	}
}

// Generate creates a single EventEnvelope.
// If fixed targets are set, picks from them; otherwise uses random family/member.
func (g *EventGenerator) Generate() EventEnvelope {
	g.mu.RLock()
	targets := g.fixedTargets
	g.mu.RUnlock()

	if len(targets) > 0 {
		t := targets[g.rng.IntN(len(targets))]
		cid := t.CustomerIDs[g.rng.IntN(len(t.CustomerIDs))]
		return g.generateFor(t.FamilyID, cid)
	}

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

// SetFixedTargets sets fixed family/customer targets. When set, Generate() uses these instead of random selection.
func (g *EventGenerator) SetFixedTargets(targets []FixedTarget) {
	g.mu.Lock()
	g.fixedTargets = targets
	g.mu.Unlock()
}

// ClearFixedTargets removes fixed targets, returning to random mode.
func (g *EventGenerator) ClearFixedTargets() {
	g.mu.Lock()
	g.fixedTargets = nil
	g.mu.Unlock()
}

// FixedTargets returns the current fixed targets (nil if random mode).
func (g *EventGenerator) FixedTargets() []FixedTarget {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.fixedTargets
}
