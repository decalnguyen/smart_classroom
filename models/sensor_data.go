package models

type SenSorData struct {
	DeviceID    string  `json:"device_id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}
