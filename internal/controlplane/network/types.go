package network

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// FixedIP represents an IP assignment for a port.
type FixedIP struct {
	IP       string `json:"ip"`
	SubnetID string `json:"subnet_id,omitempty"`
}

// FixedIPList is a JSON-stored list of FixedIP
type FixedIPList []FixedIP

// Value implements driver.Valuer (store as JSON)
func (f FixedIPList) Value() (driver.Value, error) {
	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner (read from JSON)
func (f *FixedIPList) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, f)
	case string:
		return json.Unmarshal([]byte(v), f)
	case nil:
		*f = nil
		return nil
	default:
		return fmt.Errorf("unsupported type for FixedIPList: %T", src)
	}
}
