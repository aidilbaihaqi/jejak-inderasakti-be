-- +goose Up
CREATE TABLE room_players (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id         UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    nickname        VARCHAR(20) NOT NULL,
    school_id       INTEGER REFERENCES schools(id),
    jenjang         VARCHAR(10) NOT NULL,
    avatar          SMALLINT NOT NULL,
    lang            CHAR(2) NOT NULL DEFAULT 'id',
    score           INTEGER NOT NULL DEFAULT 0,
    correct_count   SMALLINT NOT NULL DEFAULT 0,
    total_ms        INTEGER NOT NULL DEFAULT 0 CHECK (total_ms >= 0),
    current_index   SMALLINT NOT NULL DEFAULT 0 CHECK (current_index BETWEEN 0 AND 15),
    streak          SMALLINT NOT NULL DEFAULT 0 CHECK (streak >= 0),
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_room_nickname UNIQUE (room_id, nickname),
    CONSTRAINT chk_avatar CHECK (avatar BETWEEN 1 AND 12),
    CONSTRAINT chk_lang CHECK (lang IN ('id', 'en'))
);
CREATE INDEX idx_room_players_room ON room_players (room_id);
CREATE INDEX idx_room_players_school ON room_players (school_id) WHERE finished_at IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS room_players;
