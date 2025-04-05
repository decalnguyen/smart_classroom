package models

type SenSorData struct {
	DeviceID   string  `gorm:"primaryKey" json:"device_id"`
	DeviceType string  `json:"device_type"`
	Value      float64 `json:"value"`
	Status     string  `json:"status"`
}

type Sensor struct {
	DeviceID   string `gorm:"primaryKey" json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
	Location   string `json:"location"`
	Status     string `json:"status"`
}

type Device struct {
	DeviceID   string `gorm:"primaryKey" json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
	Location   string `json:"location"`
	Status     string `json:"status"`
}
