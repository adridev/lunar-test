package rockets

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageRepository interface {
	Store(ctx context.Context, message Message) error
	FindByChannel(ctx context.Context, channel uuid.UUID) ([]Message, error)
	FindAfterNumber(ctx context.Context, channel uuid.UUID, number int) ([]Message, error)
}
type MongoMessageRepository struct {
	collection *mongo.Collection
}

func NewMongoMessageRepository(collection *mongo.Collection) *MongoMessageRepository {
	return &MongoMessageRepository{
		collection: collection,
	}
}

func (r MongoMessageRepository) Store(ctx context.Context, message Message) error {

	doc := bson.M{
		"metadata": bson.M{
			"channel":       message.Metadata.Channel.String(),
			"messageNumber": message.Metadata.MessageNumber,
			"messageTime":   message.Metadata.MessageTime,
			"messageType":   message.Metadata.MessageType,
		},
		"message":   message.Message,
		"createdAt": time.Now(),
	}

	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		slog.Error("Error storing message", "error", err)
		return errors.New(StoreMessageError)
	}
	return nil
}

func (r MongoMessageRepository) FindByChannel(ctx context.Context, channel uuid.UUID) ([]Message, error) {
	filter := bson.M{"metadata.channel": channel.String()}
	opts := options.Find().SetSort(bson.D{{Key: "metadata.messageNumber", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode into raw format with string channel
	var rawMessages []struct {
		Metadata struct {
			Channel       string    `bson:"channel"`
			MessageNumber int       `bson:"messageNumber"`
			MessageTime   time.Time `bson:"messageTime"`
			MessageType   string    `bson:"messageType"`
		} `bson:"metadata"`
		Message map[string]interface{} `bson:"message"`
	}
	if err := cursor.All(ctx, &rawMessages); err != nil {
		return nil, err
	}

	// Convert to domain model with UUID
	messages := make([]Message, 0, len(rawMessages))
	for _, raw := range rawMessages {
		parsedUUID, err := uuid.Parse(raw.Metadata.Channel)
		if err != nil {
			return nil, err
		}
		messages = append(messages, Message{
			Metadata: Metadata{
				Channel:       parsedUUID,
				MessageNumber: raw.Metadata.MessageNumber,
				MessageTime:   raw.Metadata.MessageTime,
				MessageType:   raw.Metadata.MessageType,
			},
			Message: raw.Message,
		})
	}

	return messages, nil
}

func (r MongoMessageRepository) FindAfterNumber(ctx context.Context, channel uuid.UUID, number int) ([]Message, error) {
	filter := bson.M{
		"metadata.channel":       channel.String(),
		"metadata.messageNumber": bson.M{"$gt": number},
	}
	opts := options.Find().SetSort(bson.D{{Key: "metadata.messageNumber", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode into raw format with string channel
	var rawMessages []struct {
		Metadata struct {
			Channel       string    `bson:"channel"`
			MessageNumber int       `bson:"messageNumber"`
			MessageTime   time.Time `bson:"messageTime"`
			MessageType   string    `bson:"messageType"`
		} `bson:"metadata"`
		Message map[string]interface{} `bson:"message"`
	}
	if err := cursor.All(ctx, &rawMessages); err != nil {
		return nil, err
	}

	// Convert to domain model with UUID
	messages := make([]Message, 0, len(rawMessages))
	for _, raw := range rawMessages {
		parsedUUID, err := uuid.Parse(raw.Metadata.Channel)
		if err != nil {
			return nil, err
		}
		messages = append(messages, Message{
			Metadata: Metadata{
				Channel:       parsedUUID,
				MessageNumber: raw.Metadata.MessageNumber,
				MessageTime:   raw.Metadata.MessageTime,
				MessageType:   raw.Metadata.MessageType,
			},
			Message: raw.Message,
		})
	}

	return messages, nil
}

type RocketsRepository interface {
	All(ctx context.Context, sortBy *string, order *string) ([]Rocket, error)
	FindByChannel(ctx context.Context, channel uuid.UUID) (*Rocket, error)
	Upsert(ctx context.Context, rocket Rocket) error
}

type MongoRocketsRepository struct {
	collection *mongo.Collection
}

func NewMongoRocketsRepository(collection *mongo.Collection) *MongoRocketsRepository {
	return &MongoRocketsRepository{
		collection: collection,
	}
}

func (m MongoRocketsRepository) All(ctx context.Context, sortBy *string, order *string) ([]Rocket, error) {
	// Build MongoDB sort options
	findOptions := options.Find()
	if sortBy != nil {
		sortOrder := 1 // ascending by default
		if order != nil && *order == "desc" {
			sortOrder = -1
		}
		findOptions.SetSort(bson.D{{Key: *sortBy, Value: sortOrder}})
	}

	cursor, err := m.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode into raw format with string channel
	var rawRockets []struct {
		Channel           string     `bson:"channel"`
		Type              string     `bson:"type"`
		Speed             int        `bson:"speed"`
		Mission           string     `bson:"mission"`
		Status            string     `bson:"status"`
		ExplosionReason   *string    `bson:"explosionReason,omitempty"`
		LastMessageNumber *int       `bson:"lastMessageNumber,omitempty"`
		LastMessageTime   *time.Time `bson:"lastMessageTime,omitempty"`
	}
	if err := cursor.All(ctx, &rawRockets); err != nil {
		return nil, err
	}

	// Convert to domain model with UUID
	rockets := make([]Rocket, 0, len(rawRockets))
	for _, raw := range rawRockets {
		parsedUUID, err := uuid.Parse(raw.Channel)
		if err != nil {
			return nil, err
		}
		rockets = append(rockets, Rocket{
			Channel:           parsedUUID,
			Type:              raw.Type,
			Speed:             raw.Speed,
			Mission:           raw.Mission,
			Status:            raw.Status,
			ExplosionReason:   raw.ExplosionReason,
			LastMessageNumber: raw.LastMessageNumber,
			LastMessageTime:   raw.LastMessageTime,
		})
	}

	return rockets, nil
}

func (m MongoRocketsRepository) FindByChannel(ctx context.Context, channel uuid.UUID) (*Rocket, error) {
	// Decode into raw format with string channel
	var raw struct {
		Channel           string     `bson:"channel"`
		Type              string     `bson:"type"`
		Speed             int        `bson:"speed"`
		Mission           string     `bson:"mission"`
		Status            string     `bson:"status"`
		ExplosionReason   *string    `bson:"explosionReason,omitempty"`
		LastMessageNumber *int       `bson:"lastMessageNumber,omitempty"`
		LastMessageTime   *time.Time `bson:"lastMessageTime,omitempty"`
	}
	err := m.collection.FindOne(ctx, bson.M{"channel": channel.String()}).Decode(&raw)
	if err != nil {
		return nil, err
	}

	// Convert to domain model with UUID
	parsedUUID, err := uuid.Parse(raw.Channel)
	if err != nil {
		return nil, err
	}

	rocket := &Rocket{
		Channel:           parsedUUID,
		Type:              raw.Type,
		Speed:             raw.Speed,
		Mission:           raw.Mission,
		Status:            raw.Status,
		ExplosionReason:   raw.ExplosionReason,
		LastMessageNumber: raw.LastMessageNumber,
		LastMessageTime:   raw.LastMessageTime,
	}
	return rocket, nil
}

func (m MongoRocketsRepository) Upsert(ctx context.Context, rocket Rocket) error {
	filter := bson.M{"channel": rocket.Channel.String()}
	update := bson.M{
		"$set": bson.M{
			"channel":           rocket.Channel.String(),
			"type":              rocket.Type,
			"speed":             rocket.Speed,
			"mission":           rocket.Mission,
			"status":            rocket.Status,
			"explosionReason":   rocket.ExplosionReason,
			"lastMessageNumber": rocket.LastMessageNumber,
			"lastMessageTime":   rocket.LastMessageTime,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		slog.Error("Error updating rocket", "error", err)
		return errors.New(UpdateRocketError)
	}
	return nil
}
