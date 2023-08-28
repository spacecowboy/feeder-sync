package server

import "github.com/google/uuid"

type CreateChainRequestV1 struct {
	DeviceName string `json:"deviceName"`
}

type JoinChainRequestV1 struct {
	DeviceName string `json:"deviceName"`
}

type JoinChainResponseV1 struct {
	SyncCode string `json:"syncCode"`
	DeviceId int64  `json:"deviceId"`
}

type DeviceMessageV1 struct {
	DeviceId   int64  `json:"deviceId"`
	DeviceName string `json:"deviceName"`
}

type DeviceListResponseV1 struct {
	Devices []DeviceMessageV1 `json:"Devices"`
}

// V2 objects below

type MigrateRequestV2 struct {
	SyncCode   string `json:"syncCode"`
	DeviceId   int64  `json:"deviceId"`
	DeviceName string `json:"deviceName"`
}

type UserDeviceResponseV2 struct {
	UserId     uuid.UUID `json:"userId"`
	DeviceId   uuid.UUID `json:"deviceId"`
	DeviceName string    `json:"deviceName"`
}

type CreateChainRequestV2 struct {
	DeviceName string `json:"deviceName"`
}

type JoinChainRequestV2 struct {
	UserId     uuid.UUID `json:"userId"`
	DeviceName string    `json:"deviceName"`
}
