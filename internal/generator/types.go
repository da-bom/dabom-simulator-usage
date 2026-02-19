package generator

// EventEnvelope wraps a payload for Kafka publishing.
// Matches processor-usage EventEnvelope<UsagePayload> Java record.
// subType is omitted when empty (@JsonInclude(NON_NULL) in Java).
type EventEnvelope struct {
	EventID   string       `json:"eventId"`
	EventType string       `json:"eventType"`
	SubType   string       `json:"subType,omitempty"`
	Timestamp string       `json:"timestamp"`
	Payload   UsagePayload `json:"payload"`
}

// UsagePayload matches processor-usage UsagePayload Java record.
type UsagePayload struct {
	EventID    string            `json:"eventId"`
	FamilyID   int64             `json:"familyId"`
	CustomerID int64             `json:"customerId"`
	AppID      string            `json:"appId"`
	BytesUsed  int64             `json:"bytesUsed"`
	Metadata   map[string]string `json:"metadata"`
}

// Family represents a family group with its member customer IDs.
type Family struct {
	ID      int64
	Members []int64
}
