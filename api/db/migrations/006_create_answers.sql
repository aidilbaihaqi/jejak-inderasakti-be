-- +goose Up
CREATE TABLE answers (
    room_player_id  UUID NOT NULL REFERENCES room_players(id) ON DELETE CASCADE,
    question_id     VARCHAR(10) NOT NULL REFERENCES questions(id),
    served_at       TIMESTAMPTZ NOT NULL,
    answered_at     TIMESTAMPTZ,
    option_id       VARCHAR(20),
    correct         BOOLEAN,
    points          SMALLINT NOT NULL DEFAULT 0 CHECK (points >= 0),

    PRIMARY KEY (room_player_id, question_id)
);
CREATE INDEX idx_answers_question_wrong ON answers (question_id) WHERE correct = FALSE;

-- +goose Down
DROP TABLE IF EXISTS answers;
