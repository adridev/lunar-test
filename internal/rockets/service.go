package rockets

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type RocketsService struct {
	repository RocketsRepository
}

func NewRocketsServiceImpl(repository RocketsRepository) *RocketsService {
	return &RocketsService{repository: repository}
}

func (r RocketsService) GetAll(ctx context.Context, sortBy *string, order *string) ([]Rocket, error) {
	return r.repository.All(ctx, sortBy, order)
}

func (r RocketsService) GetByChannel(ctx context.Context, channel uuid.UUID) (*Rocket, error) {
	return r.repository.FindByChannel(ctx, channel)
}

type MessageService interface {
	Ingest(ctx context.Context, message Message) error
	Process(ctx context.Context, message Message) error
}

type ResequencerMessageService struct {
	messageRepository MessageRepository
	rocketsRepository RocketsRepository
	channelMutexes    map[uuid.UUID]*sync.Mutex
	mutexLock         sync.Mutex
}

func NewResequencerMessageService(messageRepository MessageRepository, rocketsRepository RocketsRepository) *ResequencerMessageService {
	return &ResequencerMessageService{
		messageRepository: messageRepository,
		rocketsRepository: rocketsRepository,
		channelMutexes:    make(map[uuid.UUID]*sync.Mutex),
	}
}

func (m *ResequencerMessageService) Ingest(ctx context.Context, message Message) error {
	if err := m.messageRepository.Store(ctx, message); err != nil {
		return err
	}

	if err := m.Process(ctx, message); err != nil {
		return err
	}

	return nil
}

func (m *ResequencerMessageService) getChannelMutex(channel uuid.UUID) *sync.Mutex {
	m.mutexLock.Lock()
	defer m.mutexLock.Unlock()

	if _, exists := m.channelMutexes[channel]; !exists {
		m.channelMutexes[channel] = &sync.Mutex{}
	}
	return m.channelMutexes[channel]
}

func (m *ResequencerMessageService) Process(ctx context.Context, message Message) error {
	// Lock channel
	channelMutex := m.getChannelMutex(message.Metadata.Channel)
	channelMutex.Lock()
	defer channelMutex.Unlock()

	var rocket *Rocket
	var err error
	if message.Metadata.MessageNumber == 1 {
		rocket, err = buildRocketState(message.Metadata.Channel, []Message{message})
		if err != nil {
			slog.Error("error building rocket state", "error", err)
			return errors.New(ProcessMessageError)
		}
	} else {
		rocket, err = m.rocketsRepository.FindByChannel(ctx, message.Metadata.Channel)
		if err != nil {
			slog.Error("error getting rocket from db", "error", err)
		}
	}

	if rocket == nil {
		return nil
	}

	messages, err := m.messageRepository.FindAfterNumber(ctx, message.Metadata.Channel, *rocket.LastMessageNumber)
	if err != nil {
		slog.Error("error finding messages after last message number", "error", err)
		return errors.New(ProcessMessageError)
	}

	for _, msg := range messages {
		if msg.Metadata.MessageNumber == *rocket.LastMessageNumber+1 {
			err = applyMessage(rocket, msg)
			if err != nil {
				slog.Error("error applying message", "error", err)
				return errors.New(ProcessMessageError)
			}
		} else {
			break
		}
	}

	// Persist rocket
	if err := m.rocketsRepository.Upsert(ctx, *rocket); err != nil {
		slog.Error("error persisting rocket", "error", err)
		return errors.New(ProcessMessageError)
	}

	return nil
}

func buildRocketState(channelID uuid.UUID, messages []Message) (*Rocket, error) {
	rocket := &Rocket{
		Channel: channelID,
		Status:  "active",
	}

	// Apply each messages
	for _, msg := range messages {
		if err := applyMessage(rocket, msg); err != nil {
			return nil, err
		}
	}

	return rocket, nil
}

func applyMessage(rocket *Rocket, msg Message) error {
	// Update metadata from the current message
	msgNum := msg.Metadata.MessageNumber
	rocket.LastMessageNumber = &msgNum
	msgTime := msg.Metadata.MessageTime
	rocket.LastMessageTime = &msgTime

	switch msg.Metadata.MessageType {
	case "RocketLaunched":
		rocketType, _ := msg.Message["type"].(string)
		launchSpeed, _ := msg.Message["launchSpeed"].(float64)
		mission, _ := msg.Message["mission"].(string)

		rocket.Type = rocketType
		rocket.Speed = int(launchSpeed)
		rocket.Mission = mission
		rocket.Status = "active"

	case "RocketSpeedIncreased":
		by, ok := msg.Message["by"].(float64)
		if !ok {
			return errors.New("invalid RocketSpeedIncreased message")
		}
		rocket.Speed += int(by)

	case "RocketSpeedDecreased":
		by, ok := msg.Message["by"].(float64)
		if !ok {
			return errors.New("invalid RocketSpeedDecreased message")
		}
		rocket.Speed -= int(by)

	case "RocketExploded":
		reason, _ := msg.Message["reason"].(string)
		rocket.ExplosionReason = &reason
		rocket.Status = "exploded"

	case "RocketMissionChanged":
		newMission, ok := msg.Message["newMission"].(string)
		if !ok {
			return errors.New("invalid RocketMissionChanged message")
		}
		rocket.Mission = newMission

	default:
		return errors.New("unknown message type")
	}

	return nil
}
