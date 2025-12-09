-- Migration: Add WebShell session recording tables
-- Created: 2025-12-09

-- WebShell sessions table
CREATE TABLE IF NOT EXISTS webshell_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INTEGER,
    username VARCHAR(255) NOT NULL,
    remote_host VARCHAR(255) NOT NULL,
    remote_port INTEGER NOT NULL DEFAULT 22,
    remote_user VARCHAR(255) NOT NULL,
    auth_method VARCHAR(20) NOT NULL,
    
    -- Session status
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, closed, error
    
    -- Timing information
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    duration_seconds INTEGER,
    
    -- Statistics
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    commands_count INTEGER DEFAULT 0,
    
    -- Recording file path
    recording_file VARCHAR(512),
    
    -- Metadata
    client_ip VARCHAR(45),
    user_agent TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Session events table (detailed command log)
CREATE TABLE IF NOT EXISTS webshell_events (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(20) NOT NULL, -- input, output, resize, connect, disconnect
    event_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Event data
    data TEXT,
    data_size INTEGER,
    
    -- For resize events
    cols INTEGER,
    rows INTEGER,
    
    -- Timing offset from session start (milliseconds)
    time_offset BIGINT NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_webshell_sessions_user_id ON webshell_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_webshell_sessions_started_at ON webshell_sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_webshell_sessions_remote_host ON webshell_sessions(remote_host);
CREATE INDEX IF NOT EXISTS idx_webshell_sessions_status ON webshell_sessions(status);
CREATE INDEX IF NOT EXISTS idx_webshell_events_session_id ON webshell_events(session_id);
CREATE INDEX IF NOT EXISTS idx_webshell_events_event_time ON webshell_events(event_time);

-- Add foreign key
ALTER TABLE webshell_events 
    ADD CONSTRAINT fk_webshell_events_session 
    FOREIGN KEY (session_id) 
    REFERENCES webshell_sessions(session_id) 
    ON DELETE CASCADE;

-- Comments
COMMENT ON TABLE webshell_sessions IS 'WebShell SSH session records for audit and replay';
COMMENT ON TABLE webshell_events IS 'Detailed events for WebShell session replay';
COMMENT ON COLUMN webshell_sessions.session_id IS 'Unique session identifier (UUID)';
COMMENT ON COLUMN webshell_sessions.recording_file IS 'Path to asciicast recording file';
COMMENT ON COLUMN webshell_events.time_offset IS 'Milliseconds from session start';
