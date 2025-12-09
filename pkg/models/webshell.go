package models

import "time"

// WebShellSession represents a SSH session record.
type WebShellSession struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	SessionID    string     `gorm:"uniqueIndex;size:64;not null" json:"session_id"`
	UserID       *uint      `gorm:"index" json:"user_id,omitempty"`
	Username     string     `gorm:"size:255;not null" json:"username"`
	RemoteHost   string     `gorm:"size:255;not null;index" json:"remote_host"`
	RemotePort   int        `gorm:"not null;default:22" json:"remote_port"`
	RemoteUser   string     `gorm:"size:255;not null" json:"remote_user"`
	AuthMethod   string     `gorm:"size:20;not null" json:"auth_method"`
	Status       string     `gorm:"size:20;not null;default:'active';index" json:"status"`
	StartedAt    time.Time  `gorm:"not null;index" json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	DurationSecs *int       `json:"duration_seconds,omitempty"`
	BytesSent    int64      `gorm:"default:0" json:"bytes_sent"`
	BytesRecv    int64      `gorm:"default:0" json:"bytes_received"`
	CmdCount     int        `gorm:"default:0" json:"commands_count"`
	RecordFile   string     `gorm:"size:512" json:"recording_file,omitempty"`
	ClientIP     string     `gorm:"size:45" json:"client_ip,omitempty"`
	UserAgent    string     `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt    time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null" json:"updated_at"`
}

// TableName specifies the table name for WebShellSession.
func (WebShellSession) TableName() string {
	return "webshell_sessions"
}

// WebShellEvent represents a session event for replay.
type WebShellEvent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SessionID  string    `gorm:"size:64;not null;index" json:"session_id"`
	EventType  string    `gorm:"size:20;not null" json:"event_type"`
	EventTime  time.Time `gorm:"not null;index" json:"event_time"`
	Data       string    `gorm:"type:text" json:"data,omitempty"`
	DataSize   int       `json:"data_size,omitempty"`
	Cols       *int      `json:"cols,omitempty"`
	Rows       *int      `json:"rows,omitempty"`
	TimeOffset int64     `gorm:"not null" json:"time_offset"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
}

// TableName specifies the table name for WebShellEvent.
func (WebShellEvent) TableName() string {
	return "webshell_events"
}
