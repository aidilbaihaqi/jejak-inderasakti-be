-- +goose Up
CREATE TABLE rooms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pin             CHAR(6) UNIQUE NOT NULL,
    host_id         UUID NOT NULL REFERENCES host_users(id),
    jenjang         VARCHAR(10) NOT NULL,
    short_session   BOOLEAN NOT NULL DEFAULT FALSE,
    accuracy_mode   BOOLEAN NOT NULL DEFAULT FALSE,
    status          VARCHAR(10) NOT NULL DEFAULT 'lobby',
    question_ids    TEXT[] NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,

    CONSTRAINT chk_status CHECK (status IN ('lobby', 'running', 'ended')),
    CONSTRAINT chk_jenjang CHECK (jenjang IN ('SD', 'SMP', 'SMA'))
);
CREATE INDEX idx_rooms_status ON rooms (status) WHERE status != 'ended';
CREATE INDEX idx_rooms_host ON rooms (host_id);

-- +goose Down
DROP TABLE IF EXISTS rooms;
