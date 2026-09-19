# 04 — Data Modelling
## Jejak Inderasakti: Jelajah Pulau Penyengat

> **Dokumen**: ERD · Database Schema · Data Dictionary · Migration Plan · Data Governance  
> **Versi**: 1.0 · **Tanggal**: 2026-09-19

---

## 1. Ringkasan Data Model

**Prinsip**: Data anak disimpan minimal — tidak ada akun pemain permanen. Identitas pemain hanya berlaku dalam satu room (nama panggilan, sekolah, jenjang, avatar, bahasa). PostgreSQL adalah **sumber kebenaran**; Redis hanya cache volatile.

**6 tabel Postgres + 4 key Redis** cukup untuk seluruh fungsionalitas versi Kamis.

---

## 2. ERD (Entity Relationship Diagram)

```
+----------+       +----------------+       +---------+
| SCHOOLS  |       |  HOST_USERS    |       |QUESTIONS|
|----------|       |----------------|       |---------|
| id (PK)  |       | id (PK)        |       | id (PK) |
| name     |       | email          |       | site    |
| jenjang  |       | password_hash  |       | level   |
| city     |       | name           |       | type    |
| verified |       | created_at     |       | prompt  |
+----+-----+       +-------+--------+       | options |
     |                     |                | explanation |
     |                     |                | active  |
     |            +--------+--------+       +----+----+
     |            |      ROOMS      |            |
     +----------->|-----------------|            |
     | (asal)     | id (PK)         |            |
                  | pin             |            |
                  | host_id (FK)    |            |
                  | jenjang         |            |
                  | short_session   |            |
                  | accuracy_mode   |            |
                  | status          |            |
                  | question_ids[]  |            |
                  | created_at      |            |
                  | started_at      |            |
                  | ended_at        |            |
                  +-------+---------+            |
                          |                      |
                  +-------v---------+            |
                  |  ROOM_PLAYERS   |            |
                  |-----------------|            |
                  | id (PK)         |            |
                  | room_id (FK)    |            |
                  | nickname        |            |
                  | school_id (FK)  |            |
                  | jenjang         |            |
                  | avatar          |            |
                  | lang            |            |
                  | score           |            |
                  | correct_count   |            |
                  | total_ms        |            |
                  | current_index   |            |
                  | streak          |            |
                  | finished_at     |            |
                  +-------+---------+            |
                          |                      |
                  +-------v---------+            |
                  |    ANSWERS      |<-----------+
                  |-----------------|
                  | room_player_id  |
                  | question_id     |
                  | served_at       |
                  | answered_at     |
                  | option_id       |
                  | correct         |
                  | points          |
                  +------ ----------+
```

**Relasi:**
- `SCHOOLS` ||--o{ `ROOM_PLAYERS` (asal sekolah)
- `HOST_USERS` ||--o{ `ROOMS` (membuat)
- `ROOMS` ||--o{ `ROOM_PLAYERS` (berisi)
- `ROOM_PLAYERS` ||--o{ `ANSWERS` (menjawab)
- `QUESTIONS` ||--o{ `ANSWERS` (dijawab)

---

## 3. Database Schema (DDL)

### 3.1 Tabel `schools`

```sql
CREATE TABLE schools (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    jenjang     VARCHAR(20),          -- 'SD', 'SMP', 'SMA', 'Umum'
    city        VARCHAR(100),
    verified    BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Seed khusus
INSERT INTO schools (name, jenjang, verified) VALUES
    ('Sekolah lain', NULL, TRUE),
    ('Umum / General visitor', 'Umum', TRUE);

CREATE INDEX idx_schools_name ON schools USING GIN (to_tsvector('simple', name));
```

### 3.2 Tabel `host_users`

```sql
CREATE TABLE host_users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(72) NOT NULL,  -- bcrypt
    name            VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ
);
```

### 3.3 Tabel `questions`

```sql
CREATE TABLE questions (
    id              VARCHAR(10) PRIMARY KEY,  -- 'M1-01', 'S2-03', dsb.
    site            SMALLINT NOT NULL,        -- 1–5
    level           SMALLINT NOT NULL,        -- 1=mudah, 2=sedang, 3=sulit
    type            VARCHAR(10) NOT NULL,     -- 'mc', 'tf', 'fill'
    prompt          JSONB NOT NULL,           -- {"id": "...", "en": "..."}
    options         JSONB NOT NULL,           -- [{"id":"opt-a","label":{"id":"...","en":"..."},"correct":false}]
    explanation     JSONB NOT NULL,           -- {"id": "...", "en": "..."}
    active          BOOLEAN DEFAULT TRUE,
    review_flag     BOOLEAN DEFAULT FALSE,    -- ditandai jika >80% salah
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_questions_site_level_active ON questions (site, level) WHERE active = TRUE;
```

**ID Convention**: `{level_code}{site}-{seq}` — contoh: `M1-01` = Mudah, Situs 1, soal ke-01; `S2-03` = Sedang, Situs 2, soal ke-03; `Su5-01` = Sulit, Situs 5, soal ke-01

### 3.4 Tabel `rooms`

```sql
CREATE TABLE rooms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pin             CHAR(6) UNIQUE NOT NULL,
    host_id         UUID NOT NULL REFERENCES host_users(id),
    jenjang         VARCHAR(10) NOT NULL,     -- 'SD', 'SMP', 'SMA'
    short_session   BOOLEAN DEFAULT FALSE,
    accuracy_mode   BOOLEAN DEFAULT FALSE,
    status          VARCHAR(10) NOT NULL DEFAULT 'lobby',  -- 'lobby', 'running', 'ended'
    question_ids    TEXT[] NOT NULL,          -- Array 10 atau 15 question_id
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,

    CONSTRAINT chk_status CHECK (status IN ('lobby', 'running', 'ended')),
    CONSTRAINT chk_jenjang CHECK (jenjang IN ('SD', 'SMP', 'SMA'))
);

CREATE INDEX idx_rooms_status ON rooms (status) WHERE status != 'ended';
CREATE INDEX idx_rooms_pin ON rooms (pin) WHERE status != 'ended';
```

### 3.5 Tabel `room_players`

```sql
CREATE TABLE room_players (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id         UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    nickname        VARCHAR(20) NOT NULL,
    school_id       INTEGER REFERENCES schools(id),
    jenjang         VARCHAR(10) NOT NULL,     -- jenjang pemain, bisa beda dari room
    avatar          SMALLINT NOT NULL,        -- 1–12
    lang            CHAR(2) NOT NULL DEFAULT 'id',  -- 'id' atau 'en'
    score           INTEGER NOT NULL DEFAULT 0,
    correct_count   SMALLINT NOT NULL DEFAULT 0,
    total_ms        INTEGER NOT NULL DEFAULT 0,   -- total waktu jawab dalam ms
    current_index   SMALLINT NOT NULL DEFAULT 0,  -- soal berikutnya yang harus dijawab
    streak          SMALLINT NOT NULL DEFAULT 0,  -- benar berturut-turut saat ini
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_room_nickname UNIQUE (room_id, nickname),
    CONSTRAINT chk_avatar CHECK (avatar BETWEEN 1 AND 12),
    CONSTRAINT chk_lang CHECK (lang IN ('id', 'en'))
);

CREATE INDEX idx_room_players_room ON room_players (room_id);
```

### 3.6 Tabel `answers`

```sql
CREATE TABLE answers (
    room_player_id  UUID NOT NULL REFERENCES room_players(id) ON DELETE CASCADE,
    question_id     VARCHAR(10) NOT NULL REFERENCES questions(id),
    served_at       TIMESTAMPTZ NOT NULL,
    answered_at     TIMESTAMPTZ,             -- NULL jika timeout (tidak dijawab)
    option_id       VARCHAR(20),             -- NULL jika timeout
    correct         BOOLEAN,                 -- NULL jika timeout
    points          SMALLINT NOT NULL DEFAULT 0,

    PRIMARY KEY (room_player_id, question_id)  -- mencegah jawab ganda
);

CREATE INDEX idx_answers_player ON answers (room_player_id);
CREATE INDEX idx_answers_question_wrong ON answers (question_id) WHERE correct = FALSE;
-- Index terakhir untuk query: soal yang dijawab salah >80%
```

---

## 4. Redis Key Schema

| Key | Tipe | Nilai | TTL | Keterangan |
|---|---|---|---|---|
| `pin:{pin}` | String | `room_id` | 2 jam | Lookup cepat room dari PIN |
| `room:{id}:lb` | Sorted Set | `room_player_id` → skor integer | 2 jam | Leaderboard; score diurutkan descending; diperbarui tiap jawaban via ZINCRBY |
| `rooms:active` | Set | `room_id` yang belum berakhir | — | Cek batas 5 room; hapus member saat room `ended` |
| `rl:join:{ip}` | Counter | Jumlah percobaan join | 1 menit | Rate limit; INCR + EXPIRE |

**Catatan**: Redis bukan sumber kebenaran. Jika Redis restart, leaderboard dibangun ulang dari tabel `answers` + `room_players`. PIN diambil ulang dari tabel `rooms`.

---

## 5. Data Dictionary

### Tabel `questions`

| Kolom | Tipe | Constraint | Deskripsi |
|---|---|---|---|
| `id` | VARCHAR(10) | PK | Format: `{level}{site}-{seq}`, misal `M1-01` |
| `site` | SMALLINT | 1–5 | 1=Masjid Raya, 2=Makam Engku Putri, 3=Istana Kantor, 4=Gedung Tabib, 5=Perigi Puteri |
| `level` | SMALLINT | 1–3 | 1=Mudah (timer 15s), 2=Sedang (20s), 3=Sulit (25s) |
| `type` | VARCHAR | mc/tf/fill | `mc`=multiple choice, `tf`=true/false, `fill`=isi kalimat |
| `prompt` | JSONB | NOT NULL | `{"id": "teks soal ID", "en": "question text EN"}` |
| `options` | JSONB | NOT NULL | Array: `[{"id":"opt-a","label":{"id":"...","en":"..."},"correct":false}, ...]` |
| `explanation` | JSONB | NOT NULL | `{"id": "penjelasan ID", "en": "explanation EN"}` |
| `active` | BOOLEAN | default TRUE | Soal aktif/tidak (deaktivasi tanpa delete) |
| `review_flag` | BOOLEAN | default FALSE | TRUE jika >80% peserta menjawab salah |

### Tabel `rooms`

| Kolom | Tipe | Constraint | Deskripsi |
|---|---|---|---|
| `status` | VARCHAR(10) | lobby/running/ended | State machine room |
| `question_ids` | TEXT[] | NOT NULL | Ordered array ID soal yang dipilih saat room dibuat (10 atau 15 item) |
| `short_session` | BOOLEAN | — | TRUE = 10 soal (2/situs), FALSE = 15 soal (3/situs) |
| `accuracy_mode` | BOOLEAN | — | TRUE = faktor kecepatan dimatikan (Mode Akurasi untuk SD) |

### Tabel `room_players`

| Kolom | Tipe | Constraint | Deskripsi |
|---|---|---|---|
| `current_index` | SMALLINT | 0–15 | Index ke `question_ids` array; soal berikutnya yang akan ditampilkan |
| `total_ms` | INTEGER | ≥ 0 | Akumulasi `answered_at - served_at` dalam ms (untuk tie-breaking) |
| `streak` | SMALLINT | ≥ 0 | Jumlah benar berturut-turut saat ini (reset ke 0 saat salah) |
| `finished_at` | TIMESTAMPTZ | NULLABLE | Waktu pemain menyelesaikan semua soal; NULL jika belum selesai |

### Tabel `answers`

| Kolom | Tipe | Constraint | Deskripsi |
|---|---|---|---|
| `served_at` | TIMESTAMPTZ | NOT NULL | Waktu server kirim `q.show` ke pemain |
| `answered_at` | TIMESTAMPTZ | NULLABLE | Waktu server terima `q.answer`; NULL jika timeout |
| `option_id` | VARCHAR | NULLABLE | ID opsi yang dipilih; NULL jika timeout |
| `correct` | BOOLEAN | NULLABLE | TRUE/FALSE/NULL (timeout) |
| `points` | SMALLINT | ≥ 0 | Poin yang didapat; 0 jika salah atau timeout |

---

## 6. Contoh Data: questions.json (Seed Format)

```json
[
  {
    "id": "M1-01",
    "site": 1,
    "level": 1,
    "type": "mc",
    "prompt": {
      "id": "Berapa jumlah kubah pada Masjid Raya Sultan Riau?",
      "en": "How many domes does the Masjid Raya Sultan Riau have?"
    },
    "options": [
      {"id": "opt-a", "label": {"id": "9", "en": "9"}, "correct": false},
      {"id": "opt-b", "label": {"id": "13", "en": "13"}, "correct": true},
      {"id": "opt-c", "label": {"id": "17", "en": "17"}, "correct": false},
      {"id": "opt-d", "label": {"id": "21", "en": "21"}, "correct": false}
    ],
    "explanation": {
      "id": "Masjid Raya Sultan Riau memiliki 13 kubah yang direnovasi ke bentuk sekarang pada 1832.",
      "en": "Masjid Raya Sultan Riau has 13 domes, renovated to its current form in 1832."
    },
    "active": true
  }
]
```

---

## 7. Migration Plan

### Strategi
- **Tool**: goose
- **Eksekusi**: Dijalankan otomatis saat startup (`cmd/server/main.go` memanggil `goose.Up()` sebelum listen)
- **File**: Sequential numbered SQL files di `db/migrations/`
- **Convention**: `{seq}_{snake_case_description}.sql`

### File Migrasi

| File | Isi |
|---|---|
| `001_create_schools.sql` | CREATE TABLE schools + index |
| `002_create_host_users.sql` | CREATE TABLE host_users |
| `003_create_questions.sql` | CREATE TABLE questions + indexes |
| `004_create_rooms.sql` | CREATE TABLE rooms + indexes |
| `005_create_room_players.sql` | CREATE TABLE room_players + indexes |
| `006_create_answers.sql` | CREATE TABLE answers + indexes |
| `007_seed_special_schools.sql` | INSERT "Sekolah lain" dan "Umum" |

### Contoh: `003_create_questions.sql`

```sql
-- +goose Up
CREATE TABLE questions (
    id          VARCHAR(10) PRIMARY KEY,
    site        SMALLINT NOT NULL CHECK (site BETWEEN 1 AND 5),
    level       SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 3),
    type        VARCHAR(10) NOT NULL CHECK (type IN ('mc', 'tf', 'fill')),
    prompt      JSONB NOT NULL,
    options     JSONB NOT NULL,
    explanation JSONB NOT NULL,
    active      BOOLEAN DEFAULT TRUE,
    review_flag BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_questions_site_level_active ON questions (site, level) WHERE active = TRUE;

-- +goose Down
DROP TABLE IF EXISTS questions;
DROP INDEX IF EXISTS idx_questions_site_level_active;
```

### Rollback Plan
- goose mendukung `-- +goose Down` di setiap file
- Manual rollback: `goose down-to <version>`
- Backup `pg_dump` dilakukan manual sebelum setiap deploy ke produksi

---

## 8. Komposisi Soal Per Room (Selector Logic)

Server memilih soal saat room dibuat menggunakan tabel komposisi berikut:

### Sesi Normal (15 soal)

| Stage | SD | SMP | SMA |
|---|---|---|---|
| 1 Masjid Raya | M, M, S | M, M, S | M, S, S |
| 2 Makam Engku Putri | M, M, S | M, S, S | M, S, Su |
| 3 Istana Kantor | M, S, S | M, S, Su | M, S, Su |
| 4 Gedung Tabib | M, S, S | M, S, Su | S, S, Su |
| 5 Perigi Puteri | M, S, Su | M, S, Su | S, Su, Su |

### Sesi Singkat (10 soal)
Ambil soal **pertama dan terakhir** dari tiap sel (2 soal per situs).

### Aturan Selector
1. Server mengambil soal acak sesuai komposisi level per situs saat room dibuat
2. Set soal tersimpan sebagai `question_ids[]` di tabel `rooms` — tidak berubah sepanjang sesi
3. Tiap room memakai acakan baru (`rand.New(rand.NewSource(time.Now().UnixNano()))`) — 5 room serentak tidak dapat set yang persis sama
4. Soal yang di-flag `review_flag = TRUE` masih digunakan tetapi ditandai untuk ditinjau panitia

---

## 9. Data Governance & Retention Policy

### 9.1 Data yang Dikumpulkan

| Data | Sumber | Sensitivitas | Berlaku |
|---|---|---|---|
| Nama panggilan | Input peserta | Rendah (pseudonym) | Per room |
| Asal sekolah | Autocomplete + pilihan | Rendah | Per room |
| Jenjang & kelas | Dropdown | Rendah | Per room |
| Avatar | Pilih ilustrasi | Nol | Per room |
| Bahasa | Toggle | Nol | Per room |
| Skor & jawaban | Gameplay | Rendah (no PII) | Per room → agregat |
| Email host | Manual seed | Medium | Permanen |
| Password host (bcrypt) | Manual seed | Tinggi | Permanen |

### 9.2 Retention Schedule

| Tabel | Retensi | Aksi Setelah Retensi |
|---|---|---|
| `room_players` | 12 bulan setelah `rooms.ended_at` | DELETE CASCADE (juga menghapus `answers`) |
| `answers` | 12 bulan (via CASCADE dari `room_players`) | Hapus otomatis |
| `rooms` | 12 bulan setelah `ended_at` | DELETE; rekap sekolah disimpan sebagai agregat terpisah |
| `schools` | Permanen | — |
| `host_users` | Sampai akun di-deaktivasi | Soft delete (tambah `deleted_at`) |
| `questions` | Permanen (deaktivasi via `active = FALSE`) | — |

### 9.3 Implementasi Cleanup

```sql
-- Cron job bulanan: hapus data room yang sudah >12 bulan
DELETE FROM rooms
WHERE ended_at < NOW() - INTERVAL '12 months'
  AND status = 'ended';
-- room_players dan answers terhapus CASCADE
```

### 9.4 Compliance UU No. 27/2022 (PDP)

| Pasal | Kewajiban | Implementasi |
|---|---|---|
| Pasal 20 | Tujuan pengumpulan data jelas | Tertera di halaman join: "Data hanya dipakai untuk sesi ini" |
| Pasal 25 | Persetujuan anak (< 18 tahun) dari orang tua/wali | Lewat izin kegiatan sekolah; host mencentang konfirmasi saat buat room |
| Pasal 27 | Data tidak boleh dibagikan ke pihak ketiga | Data tidak dikirim ke layanan eksternal |
| Pasal 32 | Keamanan data | HTTPS, validasi input, password bcrypt, token per-room |
| Pasal 54 | Pemusnahan data | Retention policy 12 bulan; cron job cleanup |

> **Disclaimer**: Ini bukan nasihat hukum. Panitia sebaiknya memastikan kepatuhan dengan pihak sekolah dan konsultan hukum jika diperlukan.

---

## 10. Query Penting (sqlc)

```sql
-- GetActiveRoomByPIN
SELECT id, jenjang, short_session, accuracy_mode, status, question_ids
FROM rooms
WHERE pin = $1 AND status != 'ended';

-- GetRoomLeaderboard (untuk CSV dan podium)
SELECT
    rp.nickname, s.name AS school_name, rp.jenjang,
    rp.score, rp.correct_count, rp.total_ms,
    RANK() OVER (ORDER BY rp.score DESC, rp.correct_count DESC, rp.total_ms ASC) AS rank
FROM room_players rp
LEFT JOIN schools s ON rp.school_id = s.id
WHERE rp.room_id = $1;

-- GetSchoolLeaderboard (rekap acara)
SELECT
    s.name AS school_name,
    AVG(top5.score) AS avg_score,
    COUNT(top5.id) AS player_count
FROM (
    SELECT rp.id, rp.school_id, rp.score,
           RANK() OVER (PARTITION BY rp.school_id ORDER BY rp.score DESC) AS rnk
    FROM room_players rp
    WHERE rp.finished_at IS NOT NULL
) top5
JOIN schools s ON top5.school_id = s.id
WHERE top5.rnk <= 5
GROUP BY s.id, s.name
HAVING COUNT(top5.id) >= 3
ORDER BY avg_score DESC;

-- FlagQuestionsForReview (cron / post-acara)
UPDATE questions
SET review_flag = TRUE
WHERE id IN (
    SELECT a.question_id
    FROM answers a
    WHERE a.correct = FALSE
    GROUP BY a.question_id
    HAVING COUNT(*) * 1.0 / (SELECT COUNT(*) FROM answers WHERE question_id = a.question_id) > 0.8
);
```
