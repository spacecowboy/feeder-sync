package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func FeederServer(w http.ResponseWriter, r *http.Request) {
	userDevice := UserDevice{
		UserId:   uuid.New(),
		DeviceId: uuid.New(),
	}

	response, _ := json.Marshal(userDevice)
	fmt.Fprint(w, string(response))
}

type UserDevice struct {
	UserId   uuid.UUID
	DeviceId uuid.UUID
}
