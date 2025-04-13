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
	ElectricityID string    `gorm:"primaryKey" json:"electricity_id"` // Unique ID for the electricity record
	DeviceID      string    `json:"device_id"`                        // Associated device ID
	Power         float64   `json:"power"`                            // Power consumption in watts
	Voltage       float64   `json:"voltage"`                          // Voltage in volts
	Current       float64   `json:"current"`                          // Current in amperes
	Status        string    `json:"status"`                           // Status of the electricity (e.g., "active", "inactive")
	Timestamp     time.Time `json:"timestamp"`                        // Timestamp of the record
}
