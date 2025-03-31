package models

type SenSorData struct {
	DeviceID    string  `gorm:"primaryKey" json:"device_id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
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
