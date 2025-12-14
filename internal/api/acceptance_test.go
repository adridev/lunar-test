package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adrianrios/lunar-test/internal/rockets"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestRocketStateGeneration_InOrder(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	channelID := uuid.New()

	// Post message 1: RocketLaunched
	msg1 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload1 RocketMessage_Message
	err := msgPayload1.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Falcon-9",
		LaunchSpeed: 500,
		Mission:     "ARTEMIS",
	})
	require.NoError(t, err)
	msg1.Message = msgPayload1

	body1, _ := json.Marshal(msg1)
	req1 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Check rocket state after first message
	rockets, err := rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Len(t, rockets, 1)
	assert.Equal(t, channelID, rockets[0].Channel)
	assert.Equal(t, "Falcon-9", rockets[0].Type)
	assert.Equal(t, 500, rockets[0].Speed)
	assert.Equal(t, "ARTEMIS", rockets[0].Mission)
	assert.Equal(t, "active", rockets[0].Status)
	assert.NotNil(t, rockets[0].LastMessageNumber)
	assert.Equal(t, 1, *rockets[0].LastMessageNumber)

	// Post message 2: RocketSpeedIncreased
	msg2 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 2,
			MessageTime:   time.Now(),
			MessageType:   RocketSpeedIncreased,
		},
	}
	var msgPayload2 RocketMessage_Message
	err = msgPayload2.FromRocketSpeedIncreasedPayload(RocketSpeedIncreasedPayload{By: 3000})
	require.NoError(t, err)
	msg2.Message = msgPayload2

	body2, _ := json.Marshal(msg2)
	req2 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Check rocket state after second message
	rockets, err = rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Len(t, rockets, 1)
	assert.Equal(t, 3500, rockets[0].Speed) // 500 + 3000
	assert.Equal(t, 2, *rockets[0].LastMessageNumber)

	// Post message 3: RocketSpeedDecreased
	msg3 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 3,
			MessageTime:   time.Now(),
			MessageType:   RocketSpeedDecreased,
		},
	}
	var msgPayload3 RocketMessage_Message
	err = msgPayload3.FromRocketSpeedDecreasedPayload(RocketSpeedDecreasedPayload{By: 500})
	require.NoError(t, err)
	msg3.Message = msgPayload3

	body3, _ := json.Marshal(msg3)
	req3 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body3))
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)

	// Check final rocket state
	rockets, err = rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Len(t, rockets, 1)
	assert.Equal(t, 3000, rockets[0].Speed) // 500 + 3000 - 500
	assert.Equal(t, 3, *rockets[0].LastMessageNumber)

	// Test ListRockets endpoint
	reqList := httptest.NewRequest(http.MethodGet, "/rockets", nil)
	recList := httptest.NewRecorder()
	handler.ServeHTTP(recList, reqList)
	assert.Equal(t, http.StatusOK, recList.Code)

	var listResp struct {
		Rockets []Rocket `json:"rockets"`
	}
	err = json.Unmarshal(recList.Body.Bytes(), &listResp)
	require.NoError(t, err)
	require.Len(t, listResp.Rockets, 1)
	assert.Equal(t, 3000, listResp.Rockets[0].Speed)

	// Test GetRocket endpoint
	reqGet := httptest.NewRequest(http.MethodGet, "/rockets/"+channelID.String(), nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	assert.Equal(t, http.StatusOK, recGet.Code)

	var rocket Rocket
	err = json.Unmarshal(recGet.Body.Bytes(), &rocket)
	require.NoError(t, err)
	assert.Equal(t, channelID, rocket.Channel)
	assert.Equal(t, 3000, rocket.Speed)
}

func TestRocketStateGeneration_OutOfOrder(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	channelID := uuid.New()

	// Post message 3 first
	msg3 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 3,
			MessageTime:   time.Now(),
			MessageType:   RocketSpeedDecreased,
		},
	}
	var msgPayload3 RocketMessage_Message
	err := msgPayload3.FromRocketSpeedDecreasedPayload(RocketSpeedDecreasedPayload{By: 500})
	require.NoError(t, err)
	msg3.Message = msgPayload3

	body3, _ := json.Marshal(msg3)
	req3 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body3))
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)

	// No rocket should be created yet
	rockets, err := rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Len(t, rockets, 0)

	// Post message 1
	msg1 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload1 RocketMessage_Message
	err = msgPayload1.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Falcon-9",
		LaunchSpeed: 500,
		Mission:     "ARTEMIS",
	})
	require.NoError(t, err)
	msg1.Message = msgPayload1

	body1, _ := json.Marshal(msg1)
	req1 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// rocket has message number 1
	rockets, err = rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Len(t, rockets, 1)
	assert.Equal(t, *rockets[0].LastMessageNumber, 1)

	// Post message 2 - now sequence is complete
	msg2 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 2,
			MessageTime:   time.Now(),
			MessageType:   RocketSpeedIncreased,
		},
	}
	var msgPayload2 RocketMessage_Message
	err = msgPayload2.FromRocketSpeedIncreasedPayload(RocketSpeedIncreasedPayload{By: 3000})
	require.NoError(t, err)
	msg2.Message = msgPayload2

	body2, _ := json.Marshal(msg2)
	req2 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Now rocket should be created with all messages applied in order
	rockets, err = rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Len(t, rockets, 1)
	assert.Equal(t, channelID, rockets[0].Channel)
	assert.Equal(t, 3000, rockets[0].Speed) // 500 + 3000 - 500 (applied in correct order despite arrival order)
	assert.Equal(t, 3, *rockets[0].LastMessageNumber)
}

func TestRocketExploded(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	channelID := uuid.New()

	// Launch rocket
	msg1 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload1 RocketMessage_Message
	_ = msgPayload1.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Falcon-9",
		LaunchSpeed: 500,
		Mission:     "ARTEMIS",
	})
	msg1.Message = msgPayload1

	body1, _ := json.Marshal(msg1)
	req1 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body1))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Explode rocket
	msg2 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 2,
			MessageTime:   time.Now(),
			MessageType:   RocketExploded,
		},
	}
	var msgPayload2 RocketMessage_Message
	_ = msgPayload2.FromRocketExplodedPayload(RocketExplodedPayload{
		Reason: "PRESSURE_VESSEL_FAILURE",
	})
	msg2.Message = msgPayload2

	body2, _ := json.Marshal(msg2)
	req2 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Check rocket state
	rockets, err := rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Len(t, rockets, 1)
	assert.Equal(t, "exploded", rockets[0].Status)
	assert.NotNil(t, rockets[0].ExplosionReason)
	assert.Equal(t, "PRESSURE_VESSEL_FAILURE", *rockets[0].ExplosionReason)
}

func TestMissionChanged(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	channelID := uuid.New()

	// Launch rocket
	msg1 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload1 RocketMessage_Message
	_ = msgPayload1.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Falcon-9",
		LaunchSpeed: 500,
		Mission:     "ARTEMIS",
	})
	msg1.Message = msgPayload1

	body1, _ := json.Marshal(msg1)
	req1 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body1))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Change mission
	msg2 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID,
			MessageNumber: 2,
			MessageTime:   time.Now(),
			MessageType:   RocketMissionChanged,
		},
	}
	var msgPayload2 RocketMessage_Message
	_ = msgPayload2.FromRocketMissionChangedPayload(RocketMissionChangedPayload{
		NewMission: "SHUTTLE_MIR",
	})
	msg2.Message = msgPayload2

	body2, _ := json.Marshal(msg2)
	req2 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Check rocket state
	rockets, err := rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Len(t, rockets, 1)
	assert.Equal(t, "SHUTTLE_MIR", rockets[0].Mission)
	assert.Equal(t, 2, *rockets[0].LastMessageNumber)
}

func TestMultipleRockets(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	// Launch rocket 1
	channelID1 := uuid.New()
	msg1 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID1,
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload1 RocketMessage_Message
	_ = msgPayload1.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Falcon-9",
		LaunchSpeed: 500,
		Mission:     "ARTEMIS",
	})
	msg1.Message = msgPayload1

	body1, _ := json.Marshal(msg1)
	req1 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body1))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Launch rocket 2
	channelID2 := uuid.New()
	msg2 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID2,
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload2 RocketMessage_Message
	_ = msgPayload2.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Starship",
		LaunchSpeed: 1000,
		Mission:     "MARS",
	})
	msg2.Message = msgPayload2

	body2, _ := json.Marshal(msg2)
	req2 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Check both rockets exist
	rockets, err := rocketsRepository.All(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Len(t, rockets, 2)

	// Check we can get each rocket individually
	reqGet1 := httptest.NewRequest(http.MethodGet, "/rockets/"+channelID1.String(), nil)
	recGet1 := httptest.NewRecorder()
	handler.ServeHTTP(recGet1, reqGet1)
	assert.Equal(t, http.StatusOK, recGet1.Code)

	var rocket1 Rocket
	json.Unmarshal(recGet1.Body.Bytes(), &rocket1)
	assert.Equal(t, "Falcon-9", rocket1.Type)
	assert.Equal(t, "ARTEMIS", rocket1.Mission)

	reqGet2 := httptest.NewRequest(http.MethodGet, "/rockets/"+channelID2.String(), nil)
	recGet2 := httptest.NewRecorder()
	handler.ServeHTTP(recGet2, reqGet2)
	assert.Equal(t, http.StatusOK, recGet2.Code)

	var rocket2 Rocket
	json.Unmarshal(recGet2.Body.Bytes(), &rocket2)
	assert.Equal(t, "Starship", rocket2.Type)
	assert.Equal(t, "MARS", rocket2.Mission)
}

func TestListRockets_SortBySpeed(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	// Create 3 rockets with different speeds
	rockets := []struct {
		Channel uuid.UUID
		Type    string
		Speed   int
		Mission string
	}{
		{uuid.New(), "Falcon-9", 500, "ARTEMIS"},
		{uuid.New(), "Starship", 1000, "MARS"},
		{uuid.New(), "Atlas-V", 300, "ISS"},
	}

	for _, rkt := range rockets {
		msg := RocketMessage{
			Metadata: MessageMetadata{
				Channel:       rkt.Channel,
				MessageNumber: 1,
				MessageTime:   time.Now(),
				MessageType:   RocketLaunched,
			},
		}
		var msgPayload RocketMessage_Message
		_ = msgPayload.FromRocketLaunchedPayload(RocketLaunchedPayload{
			Type:        rkt.Type,
			LaunchSpeed: rkt.Speed,
			Mission:     rkt.Mission,
		})
		msg.Message = msgPayload

		body, _ := json.Marshal(msg)
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Test ascending sort
	reqAsc := httptest.NewRequest(http.MethodGet, "/rockets?sortBy=speed&order=asc", nil)
	recAsc := httptest.NewRecorder()
	handler.ServeHTTP(recAsc, reqAsc)
	assert.Equal(t, http.StatusOK, recAsc.Code)

	var respAsc struct {
		Rockets []Rocket `json:"rockets"`
	}
	json.Unmarshal(recAsc.Body.Bytes(), &respAsc)
	require.Len(t, respAsc.Rockets, 3)
	assert.Equal(t, 300, respAsc.Rockets[0].Speed)
	assert.Equal(t, 500, respAsc.Rockets[1].Speed)
	assert.Equal(t, 1000, respAsc.Rockets[2].Speed)

	// Test descending sort
	reqDesc := httptest.NewRequest(http.MethodGet, "/rockets?sortBy=speed&order=desc", nil)
	recDesc := httptest.NewRecorder()
	handler.ServeHTTP(recDesc, reqDesc)
	assert.Equal(t, http.StatusOK, recDesc.Code)

	var respDesc struct {
		Rockets []Rocket `json:"rockets"`
	}
	json.Unmarshal(recDesc.Body.Bytes(), &respDesc)
	require.Len(t, respDesc.Rockets, 3)
	assert.Equal(t, 1000, respDesc.Rockets[0].Speed)
	assert.Equal(t, 500, respDesc.Rockets[1].Speed)
	assert.Equal(t, 300, respDesc.Rockets[2].Speed)
}

func TestListRockets_SortByType(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	// Create rockets with different types
	rocketTypes := []string{"Starship", "Atlas-V", "Falcon-9"}
	for i, rType := range rocketTypes {
		msg := RocketMessage{
			Metadata: MessageMetadata{
				Channel:       uuid.New(),
				MessageNumber: 1,
				MessageTime:   time.Now(),
				MessageType:   RocketLaunched,
			},
		}
		var msgPayload RocketMessage_Message
		_ = msgPayload.FromRocketLaunchedPayload(RocketLaunchedPayload{
			Type:        rType,
			LaunchSpeed: 500 + (i * 100),
			Mission:     "TEST",
		})
		msg.Message = msgPayload

		body, _ := json.Marshal(msg)
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Test ascending sort by type
	reqAsc := httptest.NewRequest(http.MethodGet, "/rockets?sortBy=type&order=asc", nil)
	recAsc := httptest.NewRecorder()
	handler.ServeHTTP(recAsc, reqAsc)
	assert.Equal(t, http.StatusOK, recAsc.Code)

	var respAsc struct {
		Rockets []Rocket `json:"rockets"`
	}
	json.Unmarshal(recAsc.Body.Bytes(), &respAsc)
	require.Len(t, respAsc.Rockets, 3)
	assert.Equal(t, "Atlas-V", respAsc.Rockets[0].Type)
	assert.Equal(t, "Falcon-9", respAsc.Rockets[1].Type)
	assert.Equal(t, "Starship", respAsc.Rockets[2].Type)
}

func TestListRockets_SortByStatus(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	// Create active rocket
	msg1 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       uuid.New(),
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload1 RocketMessage_Message
	_ = msgPayload1.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Falcon-9",
		LaunchSpeed: 500,
		Mission:     "ACTIVE",
	})
	msg1.Message = msgPayload1
	body1, _ := json.Marshal(msg1)
	req1 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body1))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Create exploded rocket
	channelID2 := uuid.New()
	msg2 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID2,
			MessageNumber: 1,
			MessageTime:   time.Now(),
			MessageType:   RocketLaunched,
		},
	}
	var msgPayload2 RocketMessage_Message
	_ = msgPayload2.FromRocketLaunchedPayload(RocketLaunchedPayload{
		Type:        "Starship",
		LaunchSpeed: 1000,
		Mission:     "EXPLODED",
	})
	msg2.Message = msgPayload2
	body2, _ := json.Marshal(msg2)
	req2 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Explode the second rocket
	msg3 := RocketMessage{
		Metadata: MessageMetadata{
			Channel:       channelID2,
			MessageNumber: 2,
			MessageTime:   time.Now(),
			MessageType:   RocketExploded,
		},
	}
	var msgPayload3 RocketMessage_Message
	_ = msgPayload3.FromRocketExplodedPayload(RocketExplodedPayload{Reason: "TEST"})
	msg3.Message = msgPayload3
	body3, _ := json.Marshal(msg3)
	req3 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body3))
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)

	// Test ascending sort by status
	reqAsc := httptest.NewRequest(http.MethodGet, "/rockets?sortBy=status&order=asc", nil)
	recAsc := httptest.NewRecorder()
	handler.ServeHTTP(recAsc, reqAsc)
	assert.Equal(t, http.StatusOK, recAsc.Code)

	var respAsc struct {
		Rockets []Rocket `json:"rockets"`
	}
	json.Unmarshal(recAsc.Body.Bytes(), &respAsc)
	require.Len(t, respAsc.Rockets, 2)
	assert.Equal(t, "active", string(respAsc.Rockets[0].Status))
	assert.Equal(t, "exploded", string(respAsc.Rockets[1].Status))
}

func TestListRockets_NoSort(t *testing.T) {
	mongoClient, cleanup := setupMongoDB(t)
	defer cleanup()

	db := mongoClient.Database("rockets_test")
	messagesCollection := db.Collection("messages")
	rocketsCollection := db.Collection("rockets")

	messagesRepository := rockets.NewMongoMessageRepository(messagesCollection)
	rocketsRepository := rockets.NewMongoRocketsRepository(rocketsCollection)
	handler := setupHanler(messagesRepository, rocketsRepository)

	// Create a couple of rockets
	for i := 0; i < 2; i++ {
		msg := RocketMessage{
			Metadata: MessageMetadata{
				Channel:       uuid.New(),
				MessageNumber: 1,
				MessageTime:   time.Now(),
				MessageType:   RocketLaunched,
			},
		}
		var msgPayload RocketMessage_Message
		_ = msgPayload.FromRocketLaunchedPayload(RocketLaunchedPayload{
			Type:        "Falcon-9",
			LaunchSpeed: 500,
			Mission:     "TEST",
		})
		msg.Message = msgPayload

		body, _ := json.Marshal(msg)
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Test without sorting parameters
	req := httptest.NewRequest(http.MethodGet, "/rockets", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Rockets []Rocket `json:"rockets"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Rockets, 2)
}

func setupMongoDB(t *testing.T) (*mongo.Client, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:8.0",
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "test",
			"MONGO_INITDB_ROOT_PASSWORD": "test",
			"MONGO_INITDB_DATABASE":      "rockets_test",
		},
		WaitingFor: wait.ForLog("Waiting for connections").
			WithStartupTimeout(60 * time.Second),
	}

	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := mongoContainer.Host(ctx)
	require.NoError(t, err)

	port, err := mongoContainer.MappedPort(ctx, "27017")
	require.NoError(t, err)

	connStr := "mongodb://test:test@" + host + ":" + port.Port()

	clientOptions := options.Client().ApplyURI(connStr)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	require.NoError(t, err)

	err = mongoClient.Ping(ctx, nil)
	require.NoError(t, err)

	cleanup := func() {
		_ = mongoClient.Disconnect(ctx)
		_ = mongoContainer.Terminate(ctx)
	}

	return mongoClient, cleanup
}

func setupHanler(messagesRepository *rockets.MongoMessageRepository, rocketsRepository *rockets.MongoRocketsRepository) http.Handler {
	messagesService := rockets.NewResequencerMessageService(messagesRepository, rocketsRepository)
	rocketsService := rockets.NewRocketsServiceImpl(rocketsRepository)
	api := NewRocketsAPI(messagesService, rocketsService)
	handler := HandlerFromMux(api, chi.NewRouter())
	return handler
}
