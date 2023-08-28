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

// TODO

type ReadMarkV1 struct {
	Timestamp int64  `json:"timestamp"`
	Encrypted string `json:"encrypted"`
}

type GetReadMarksResponseV1 struct {
	ReadMarks []ReadMarkV1 `json:"readMarks"`
}

type SendReadMarkV1 struct {
	Encrypted string `json:"encrypted"`
}

type SendReadMarksResponseV1 struct {
	ReadMarks []SendReadMarkV1 `json:"items"`
}

/*
type GetFeedsResponse = {
  hash: number;
  encrypted: string;
};

type UpdateFeedsRequest = {
  contentHash: number;
  encrypted: string;
};

type UpdateFeedsResponse = {
  hash: number;
};
*/
