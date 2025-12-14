package rockets

import (
	"time"

	"github.com/google/uuid"
)

type Metadata struct {
	Channel       uuid.UUID `json:"channel"`
	MessageNumber int       `json:"messageNumber"`
	MessageTime   time.Time `json:"messageTime"`
	MessageType   string    `json:"messageType"`
}

type Message struct {
	Metadata Metadata               `json:"metadata"`
	Message  map[string]interface{} `json:"message"`
}

type Rocket struct {
	Channel           uuid.UUID  `json:"channel"`
	Type              string     `json:"type"`
	Speed             int        `json:"speed"`
	Mission           string     `json:"mission"`
	Status            string     `json:"status"`
	ExplosionReason   *string    `json:"explosionReason,omitempty"`
	LastMessageNumber *int       `json:"lastMessageNumber,omitempty"`
	LastMessageTime   *time.Time `json:"lastMessageTime,omitempty"`
}
