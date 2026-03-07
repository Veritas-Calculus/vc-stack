// Package models defines shared data models for the VC Stack system.
package models

import (
	"database/sql/driver"
	"encoding/json"
)

// JSONMap is a custom type for PostgreSQL JSONB fields.
// It implements sql.Scanner and driver.Valuer for automatic serialization.
type JSONMap map[string]interface{}

// Scan implements the sql.Scanner interface.
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface.
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(j)
}
