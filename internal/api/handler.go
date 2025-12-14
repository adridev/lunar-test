package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/adrianrios/lunar-test/internal/rockets"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ ServerInterface = (*RocketsAPI)(nil)

type RocketsAPI struct {
	messagesService rockets.MessageService
	rocketsService  *rockets.RocketsService
}

func NewRocketsAPI(messagesService rockets.MessageService, rocketsService *rockets.RocketsService) *RocketsAPI {
	return &RocketsAPI{messagesService: messagesService, rocketsService: rocketsService}
}

func (a RocketsAPI) PostMessage(w http.ResponseWriter, r *http.Request) {
	var message rockets.Message
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := a.messagesService.Ingest(r.Context(), message); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a RocketsAPI) ListRockets(w http.ResponseWriter, r *http.Request, params ListRocketsParams) {
	var sortBy *string
	var order *string

	if params.SortBy != nil {
		sortVal := string(*params.SortBy)
		sortBy = &sortVal
	}
	if params.Order != nil {
		orderVal := string(*params.Order)
		order = &orderVal
	}

	rockets, err := a.rocketsService.GetAll(r.Context(), sortBy, order)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	apiRockets := make([]Rocket, 0, len(rockets))
	for _, rkt := range rockets {
		apiRockets = append(apiRockets, toAPIRocket(rkt))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Rockets []Rocket `json:"rockets"`
	}{Rockets: apiRockets})
}

func (a RocketsAPI) GetRocket(w http.ResponseWriter, r *http.Request, channel openapi_types.UUID) {
	rkt, err := a.rocketsService.GetByChannel(r.Context(), uuid.UUID(channel))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toAPIRocket(*rkt))
}

func toAPIRocket(r rockets.Rocket) Rocket {
	var lastNum *int
	if r.LastMessageNumber != nil {
		v := *r.LastMessageNumber
		lastNum = &v
	}
	var lastTime *time.Time
	if r.LastMessageTime != nil {
		v := *r.LastMessageTime
		lastTime = &v
	}
	return Rocket{
		Channel:           openapi_types.UUID(r.Channel),
		Type:              r.Type,
		Speed:             r.Speed,
		Mission:           r.Mission,
		Status:            RocketStatus(r.Status),
		ExplosionReason:   r.ExplosionReason,
		LastMessageNumber: lastNum,
		LastMessageTime:   lastTime,
	}
}
