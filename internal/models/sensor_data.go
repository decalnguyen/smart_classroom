package models

import "time"

type SenSorData struct {
	DeviceID   string    `gorm:"primaryKey" json:"device_id"`
	DeviceType string    `json:"device_type"`
	Value      float64   `json:"value"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}

type Sensor struct {
	DeviceID   string    `gorm:"primaryKey" json:"device_id"`
	DeviceName string    `json:"device_name"`
	DeviceType string    `json:"device_type"`
	Location   string    `json:"location"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}

type Device struct {
	DeviceID   string `gorm:"primaryKey" json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
	Location   string `json:"location"`
	Status     string `json:"status"`
}

type Electricity struct {
	DeviceID   string    `gorm:"primaryKey" json:"device_id"` // Associated device ID
	DeviceName string    `json:"device_name"`                 // Name of the device
	DeviceType string    `json:"device_type"`                 // Type of the device (e.g., "electricity meter")
	Value      float64   `json:"value"`                       // Electricity value (e.g., kWh)
	Status     string    `json:"status"`                      // Status of the electricity (e.g., "active", "inactive")
	Timestamp  time.Time `json:"timestamp"`                   // Timestamp of the record
}
