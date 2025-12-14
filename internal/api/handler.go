package api

import (
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type RocketsAPI struct {
}

func (a RocketsAPI) PostMessage(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}

func (a RocketsAPI) ListRockets(w http.ResponseWriter, r *http.Request, params ListRocketsParams) {
	//TODO implement me
	panic("implement me")
}

func (a RocketsAPI) GetRocket(w http.ResponseWriter, r *http.Request, channel openapi_types.UUID) {
	//TODO implement me
	panic("implement me")
}
