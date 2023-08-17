package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func FeederServer(w http.ResponseWriter, r *http.Request) {
	userDevice := RegisterNewUser()

	response, _ := json.Marshal(userDevice)
	fmt.Fprint(w, string(response))
}

func RegisterNewUser() UserDevice {
	return UserDevice{
		UserId:   uuid.New(),
		DeviceId: uuid.New(),
	}
}

type UserDevice struct {
	UserId   uuid.UUID
	DeviceId uuid.UUID
}
