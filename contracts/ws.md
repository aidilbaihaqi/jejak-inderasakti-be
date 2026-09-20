# WebSocket Contract

**FROZEN as of 2026-09-19** — change only via a contract PR. REST is in `openapi.yaml`.

- Endpoint: `/ws?token=<player_token>` for a player (token from `POST /api/rooms/{pin}/join`), or `/ws?token=<host JWT>&room=<room id>` for the host (`room` is the `id` returned by `POST /api/rooms`; the host must own that room).
- Auth failures: invalid token → HTTP 401 and unknown/ended room → HTTP 404 (JSON `{code, message}`) before the upgrade; a token that is valid but not a member of the room (kicked player, other host) is closed with WebSocket close code `4401`.
- Envelope: `{"t": "<type>", "d": {...}}` (`d` omitted when there is no data).
- Keepalive: `ping` / `pong` every 20 s.
- Content language: `prompt`, `label`, `explanation` are sent in the **player's** language only.
- The answer key is never in `q.show`. `correct_option_id` is only in `q.result`, after an answer is received.

## Client → Server

| `t` | Sender | `d` | Notes |
|---|---|---|---|
| `q.next` | Player | — | Served only if the previous question was answered or its deadline passed; a skipped question is recorded as 0 points |
| `q.answer` | Player | `{"question_id": "M1-01", "option_id": "opt-b"}` | `option_id: null` when time ran out. Rejected if not served, already answered, or past deadline |
| `host.start` | Host | — | Start the session |
| `host.end` | Host | — | Force-end the session |
| `host.kick` | Host | `{"player_id": "<uuid>"}` | Remove a player |
| `ping` | Both | — | |

Deadline = `served_at + limit_ms + 1000 ms`.

## Server → Client

| `t` | To | `d` |
|---|---|---|
| `room.state` | Player | `{status, current_index, score, streak, players[]}` — on connect/reconnect; the active question follows via `q.show` if not past deadline |
| `room.started` | All | — |
| `q.show` | Player | `{index, total, site, level, prompt, options: [{id, label}], limit_ms}` — `total` is the number of questions in the session (10 or 15). Option order is shuffled per player (except true/false). After a reconnect `limit_ms` is the time left |
| `q.result` | Player | `{correct, correct_option_id, explanation, points, score, streak, finished}` — `finished` is true after the last question |
| `lb.update` | All | `{rankings: [{rank, nickname, school, score, correct_count}]}` — at most 1×/s per room |
| `room.ended` | All | `{podium: [{rank, nickname, avatar, score}], school_lb: [...]}` — on all finished, `host.end`, or 12 min after start |
| `player.joined` | Host | `{id, nickname, avatar, school, lang}` |
| `player.kicked` | Kicked player | `{reason}` |
| `error` | Player | `{code, message, ref}` — a rejected command; `ref` is the command type (see below) |
| `pong` | Both | — |

## Examples

```json
{"t": "q.show", "d": {"index": 0, "site": 1, "level": 1,
  "prompt": "Berapa jumlah kubah pada Masjid Raya Sultan Riau?",
  "options": [{"id": "opt-a", "label": "9"}, {"id": "opt-b", "label": "13"}],
  "limit_ms": 15000}}

{"t": "q.answer", "d": {"question_id": "M1-01", "option_id": "opt-b"}}

{"t": "q.result", "d": {"correct": true, "correct_option_id": "opt-b",
  "explanation": "Masjid Raya Sultan Riau memiliki 13 kubah ...",
  "points": 583, "score": 583, "streak": 1}}
```

## Gameplay rules (added 21 Sep, additive to the frozen contract)

- Flow per player: `q.next` -> `q.show` -> `q.answer` -> `q.result` -> `q.next` ... Each player paces themselves; there is no clock sync (ADR-002).
- Time is measured by the server: `served_at` is set when `q.show` is sent, `answered_at` when `q.answer` arrives.
- Deadline = `served_at + limit_ms + 1000 ms`. An answer after the deadline, or `option_id: null`, is stored as a 0-point timeout and answered with a normal `q.result` (`correct: false`).
- `q.next` while the current question is unanswered and before its deadline is rejected with `QUESTION_IN_PROGRESS`. After the deadline it records a 0-point timeout for the old question and serves the next one.
- `lb.update` is sent to everyone at most once per second per room, ranked by score, then correct answers, then total answer time.
- The room ends (`room.ended`) when every player has answered all questions, on `host.end`, or 12 minutes after `host.start`.
- On reconnect the server sends `room.state` (with the player's `current_index`, `score`, `streak`) and, if a question is still open, `q.show` again with the remaining `limit_ms`. If its deadline passed while disconnected it is recorded as a timeout first.

### `error` codes

| `code` | `ref` | Meaning |
|---|---|---|
| `ROOM_NOT_STARTED` | `q.next`, `q.answer` | The host has not started the session |
| `QUESTION_IN_PROGRESS` | `q.next` | Answer the current question first |
| `ALREADY_FINISHED` | `q.next` | The player answered every question |
| `NOT_SERVED` | `q.answer` | `question_id` is not the active question |
| `ALREADY_ANSWERED` | `q.answer` | The question was already answered; the first answer is kept |
| `INVALID_OPTION` | `q.answer` | `option_id` does not belong to the question |
| `INVALID_REQUEST` | `q.answer` | Malformed payload |
| `INTERNAL` | `q.answer` | The answer could not be saved; the client may retry |
