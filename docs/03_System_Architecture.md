# 03 — Arsitektur Sistem
## Jejak Inderasakti: Jelajah Pulau Penyengat

> **Dokumen**: C4 Model · Sequence Diagram · API Contract · WebSocket Spec · ADR  
> **Versi**: 1.0 · **Tanggal**: 2026-09-19

---

## 1. Ringkasan Arsitektur

**Rekomendasi**: Satu binary Go (REST + WebSocket + mesin game) di belakang Caddy, PostgreSQL sebagai sumber kebenaran, Redis untuk PIN/leaderboard/rate-limit — **4 container Docker Compose** di satu VPS.

Arsitektur sengaja dibuat sekecil mungkin agar dapat di-deploy dan diuji di VPS pada **2026-09-24** dengan tim 1 FE + 1 BE + 1 desainer dalam 6 hari.

| Parameter | Nilai | Dampak ke Desain |
|---|---|---|
| Peserta per room | Maks. 15 | Satu goroutine per room sudah cukup |
| Room bersamaan | Maks. 5 | Maks. 75 koneksi WebSocket; room ke-6 ditolak |
| Durasi sesi | 5–10 menit, 15 soal | Batas keras room 12 menit |
| Koneksi | Online saja | Tanpa mode offline; cukup reconnect otomatis |
| Server | VPS 2 vCPU / 4 GB RAM | Jauh di atas kebutuhan 75 pemain |

**Target kualitas**: ACK jawaban p95 < 300ms · halaman awal < 2.5 detik di 4G · 0 error pada uji beban 75 pemain

---

## 2. C4 Model

### 2.1 Level 1 — System Context

```
+-----------------------------+      +-----------------------------+
|  Peserta (Browser HP)       |----->|                             |
|  SD/SMP/SMA/Umum            |      |   Jejak Inderasakti         |
+-----------------------------+      |   Web Application           |
                                     |                             |
+-----------------------------+      |   Game kuis multiplayer     |
|  Host / Guru (Laptop)       |----->|   real-time berbasis web    |
|  Operator room              |      |   untuk Pulau Penyengat     |
+-----------------------------+      +----------------------------+
                                              |
                                      [Internet / HTTPS]
                                              |
                                    +---------v---------+
                                    |  VPS Ubuntu LTS   |
                                    |  Singapura/Jakarta|
                                    +-------------------+
```

### 2.2 Level 2 — Container Diagram

```
+---------------------------------------------------------------+
|                    VPS (2 vCPU / 4GB RAM)                    |
|                                                               |
|  +---------------+     +---------------------------------+   |
|  |    Caddy      |     |         Go API Server           |   |
|  |  (Reverse     |---->|  REST + WebSocket + Game Engine |   |
|  |   Proxy +     |     |  :8080                          |   |
|  |   HTTPS)      |     |                                 |   |
|  |               |     |  - net/http (routing)           |   |
|  |  :80, :443    |     |  - gorilla/websocket            |   |
|  |               |     |  - internal/game (mesin room)   |   |
|  | /api, /ws ->  |     |  - sqlc (query type-safe)       |   |
|  |   Go API      |     |  - goose (migrasi)              |   |
|  |               |     |  - go-redis v9                  |   |
|  | /* -> FE      |     |  - log/slog (JSON)              |   |
|  |   static      |     +-----------+---------------------+   |
|  +---------------+                 |           |             |
|                                    v           v             |
|                           +--------+--+ +------+------+      |
|                           |PostgreSQL | |   Redis 7   |      |
|                           |   17      | |             |      |
|                           | Sumber    | | PIN, LB,    |      |
|                           | kebenaran | | rate-limit  |      |
|                           +-----------+ +-------------+      |
+---------------------------------------------------------------+

+---------------------------------------------------------------+
|                    Client (Browser HP)                        |
|   Vite + React + TypeScript (SPA statis)                     |
|   Tailwind CSS + Framer Motion + lottie-react                |
|   Zustand (state) + react-i18next (bilingual)                |
|   Served oleh Caddy dari /srv/web                            |
+---------------------------------------------------------------+
```

### 2.3 Level 3 — Component Diagram (Go API)

```
internal/
├── http/
│   ├── router.go          # Routing: POST /api/rooms, GET /api/schools, dll.
│   ├── auth.go            # Middleware JWT, login host
│   ├── rooms.go           # Handler buat room, join, results.csv
│   ├── schools.go         # Handler autocomplete sekolah
│   └── leaderboard.go     # Handler rekap sekolah
│
├── ws/
│   ├── hub.go             # WebSocket hub, mapping token → conn
│   ├── pump.go            # Read pump + Write pump per koneksi
│   └── message.go         # Parsing pesan {t, d}
│
├── game/
│   ├── room.go            # Goroutine room, channel event, state machine
│   ├── scoring.go         # Fungsi Score(ScoreInput) int
│   ├── selector.go        # Pemilihan soal acak per jenjang + situs
│   └── registry.go        # Map room_id → *Room goroutine
│
└── store/
    ├── db.go              # Pool pgx v5
    ├── queries/           # Kode Go hasil sqlc
    └── redis.go           # go-redis v9 wrapper
```

### 2.4 Level 4 — Code (Fungsi Penilaian)

```go
package game

import "math"

var base = map[int]float64{1: 500, 2: 750, 3: 1000}
const netGraceMs = 1000 // toleransi jaringan setelah timer habis

type ScoreInput struct {
    Level        int
    Correct      bool
    ResponseMs   int64 // answered_at - served_at, diukur server
    LimitMs      int64
    GraceMs      int64 // masa baca: 2000, SD 3000
    StreakBefore int   // benar berturut-turut sebelum soal ini
    AccuracyMode bool
}

func Score(in ScoreInput) int {
    if !in.Correct || in.ResponseMs > in.LimitMs+netGraceMs {
        return 0
    }
    r := min(in.ResponseMs, in.LimitMs)
    speed := 1.0
    if !in.AccuracyMode {
        tEff := max(0, r-in.GraceMs)
        speed = 1 - float64(tEff)/float64(in.LimitMs-in.GraceMs)
    }
    pts := int(math.Round(base[in.Level] * (0.6 + 0.4*speed)))
    return pts + min(50*in.StreakBefore, 250)
}
```

---

## 3. Tech Stack

| Lapisan | Pilihan | Alasan |
|---|---|---|
| Backend | Go + `net/http` standar | Tanpa framework; lebih sedikit yang bisa salah |
| WebSocket | gorilla/websocket | Contoh dan dokumentasi paling banyak; pola read/write pump sudah baku |
| Database | PostgreSQL 17 + pgx v5 | Driver Postgres paling matang di Go |
| Query | sqlc | Tulis SQL biasa, kode Go type-safe dihasilkan otomatis |
| Migrasi | goose | File SQL berurutan, dijalankan saat startup |
| Cache | Redis 7 + go-redis v9 | PIN → room, leaderboard (sorted set), rate limit |
| Log | `log/slog` (JSON) | Standar library |
| Frontend | Vite + React + TypeScript | SPA statis tanpa SSR; build dilayani Caddy |
| UI | Tailwind CSS + Framer Motion + lottie-react | Cepat untuk UI berwarna dan animasi maskot |
| State & i18n | Zustand + react-i18next | Ringan; file `id.json` dan `en.json` |
| Proxy & TLS | Caddy | HTTPS otomatis + file server FE |
| Deploy | Docker Compose | Satu perintah untuk naik/turun |

**Tidak dipakai di versi Kamis**: Kubernetes, message queue, CDN, object storage, service worker.

---

## 4. Sequence Diagram — Alur Satu Room

```
Host          Go API        Postgres/Redis     Peserta
 |                |                |               |
 |--POST /api/rooms-->             |               |
 |            |--simpan room + 15 question_ids-->  |
 |            |--SET pin:{pin} Redis (TTL 2h)-->   |
 |<--{pin, qr}----|                |               |
 |                |                |               |
 |                |           |--POST /api/rooms/{pin}/join-->
 |                |           |<--insert room_players--------|
 |                |<--token pemain-->                        |
 |                |                |               |         |
 |                |                |    <--WS connect (token)|
 |                |                |        |                |
 |--host.start--->|                |        |                |
 |                |--room.started->|        |                |
 |                |                |  <--q.next              |
 |                |--catat served_at------->|                |
 |                |--q.show (tanpa kunci)-->|                |
 |                |                |  <--q.answer            |
 |                |--simpan jawaban, ZINCRBY->               |
 |                |--q.result (kunci, poin, skor)-->         |
 |<--lb.update----|                |        |                |
 |      [loop 15 soal]             |        |                |
 |                |--room.ended (podium)-->|                 |
```

---

## 5. State Machine Room

```
        CREATE
           |
           v
        [lobby] <-- peserta join/leave
           |
       host.start
           |
           v
        [running] <-- q.next / q.answer / ping-pong
           |
    (salah satu kondisi berikut:)
    1. Semua peserta selesai (current_index = jumlah soal)
    2. Host tekan Akhiri (host.end)
    3. 12 menit sejak started_at
           |
           v
        [ended] --> room.ended dikirim ke semua
```

---

## 6. REST API Contract

| Method | Path | Auth | Fungsi | Response |
|---|---|---|---|---|
| POST | `/api/auth/login` | — | Login host, kembalikan JWT | `{token}` |
| POST | `/api/rooms` | Host JWT | Buat room; 409 bila 5 room aktif | `{id, pin, qr_url}` |
| GET | `/api/rooms/{pin}` | — | Cek room ada dan sisa slot | `{status, slots_left, jenjang}` |
| POST | `/api/rooms/{pin}/join` | — | Masuk room; 409 bila penuh (15) atau nama dipakai | `{player_token}` |
| GET | `/api/schools?q=` | — | Autocomplete sekolah | `[{id, name, jenjang}]` |
| GET | `/api/rooms/{id}/results.csv` | Host JWT | Ekspor hasil CSV | `text/csv` |
| GET | `/api/leaderboard/schools` | — | Rekap sekolah | `[{school, avg_score, player_count}]` |
| GET | `/api/healthz` | — | Health check | `{status: "ok"}` |

### Error Codes
| HTTP | Kode | Arti |
|---|---|---|
| 401 | UNAUTHORIZED | JWT tidak valid atau kedaluwarsa |
| 409 | ROOM_FULL | Room sudah 15 peserta |
| 409 | MAX_ROOMS | Sudah ada 5 room aktif |
| 409 | NICKNAME_TAKEN | Nama sudah dipakai di room ini |
| 404 | ROOM_NOT_FOUND | PIN tidak ditemukan atau sudah berakhir |
| 429 | RATE_LIMITED | > 10 percobaan join/menit per IP |

---

## 7. WebSocket Contract

**Endpoint**: `/ws?token=<player_token>`  
**Format pesan**: `{"t": "<type>", "d": {...}}`  
**Keepalive**: ping/pong tiap 20 detik

### 7.1 Client → Server

| Type | Pengirim | Data | Keterangan |
|---|---|---|---|
| `q.next` | Peserta | — | Minta soal berikutnya |
| `q.answer` | Peserta | `{question_id, option_id}` | Jawab soal; `option_id: null` bila waktu habis |
| `host.start` | Host | — | Mulai sesi |
| `host.end` | Host | — | Akhiri sesi paksa |
| `host.kick` | Host | `{player_id}` | Kick peserta dari room |
| `ping` | Keduanya | — | Keepalive |

### 7.2 Server → Client

| Type | Penerima | Data | Keterangan |
|---|---|---|---|
| `room.state` | Peserta | `{status, current_index, score, streak, players[]}` | Dikirim saat connect/reconnect |
| `q.show` | Peserta | `{index, site, level, prompt, options[{id, label}], limit_ms}` | Soal tanpa kunci jawaban |
| `q.result` | Peserta | `{correct, correct_option_id, explanation, points, score, streak}` | Hasil jawaban |
| `lb.update` | Semua di room | `{rankings: [{rank, nickname, school, score, correct_count}]}` | Maks. 1×/detik |
| `room.started` | Semua di room | — | Sesi dimulai |
| `room.ended` | Semua di room | `{podium: [{rank, nickname, avatar, score}], school_lb[]}` | Sesi berakhir |
| `player.joined` | Host | `{id, nickname, avatar, school, lang}` | Peserta baru masuk lobby |
| `player.kicked` | Peserta yang di-kick | `{reason}` | Peserta di-kick |
| `pong` | Keduanya | — | Respons keepalive |

### 7.3 Aturan Server

1. `q.next` hanya dilayani bila soal sebelumnya sudah dijawab atau melewati deadline; soal terlewat dicatat 0 poin
2. Deadline = `served_at + limit_ms + 1000ms` (toleransi jaringan)
3. Reconnect: token masih valid → server kirim `room.state` dan soal aktif bila belum lewat deadline
4. Room otomatis berakhir saat semua peserta selesai, host menekan Akhiri, atau 12 menit setelah dimulai
5. Satu room = satu goroutine yang memproses event dari channel — state room tidak perlu mutex; **wajib lulus `go test -race`**
6. Rate limit join: 10×/menit per IP; PIN 6 digit dihapus dari Redis saat room berakhir
7. Leaderboard update dibatasi 1×/detik untuk menghindari flood ke host

---

## 8. Bilingual Architecture

| Bagian | Cara |
|---|---|
| Teks UI | react-i18next; file `web/src/i18n/id.json` dan `en.json`; kunci datar `lobby.waiting` |
| Konten soal | Kolom JSONB `{id, en}` di tabel `questions`; server hanya kirim bahasa pemain dalam `q.show` dan `q.result` |
| Nama diri | Tetap Melayu (Rumah Sotoh, Gurindam Dua Belas) dengan penjelas singkat di versi EN |
| Ilustrasi | Tanpa teks di dalam gambar; satu aset untuk dua bahasa |

---

## 9. Struktur Repository

```
jejak-inderasakti/
├── AGENTS.md            # Konteks untuk AI agent: stack, perintah, konvensi
├── docs/                # 3 dokumen sumber + 10 dokumen ini
├── contracts/
│   ├── openapi.yaml     # REST API contract (sumber kebenaran FE-BE)
│   └── ws.md            # WebSocket messages + contoh JSON
├── api/                 # Go Backend
│   ├── cmd/
│   │   ├── server/      # main.go: start HTTP server
│   │   └── seed/        # main.go: seed DB dari questions.json
│   ├── internal/
│   │   ├── http/        # Handler REST, middleware
│   │   ├── ws/          # WebSocket hub, read/write pump
│   │   ├── game/        # Room goroutine, scoring, question selector
│   │   └── store/       # sqlc generated code, redis wrapper
│   └── db/
│       ├── migrations/  # goose SQL files
│       └── queries/     # SQL queries untuk sqlc
├── web/                 # Vite + React Frontend
│   └── src/
│       ├── pages/       # P1–P10, H1–H5
│       ├── components/  # Button, AnswerTile, TimerBar, dll.
│       ├── game/        # WebSocket client, state machine
│       └── i18n/        # id.json, en.json
├── seed/
│   ├── questions.json   # Bank soal 60+6 bilingual ID/EN
│   └── schools.csv      # Daftar sekolah
├── deploy/
│   ├── docker-compose.yml
│   └── Caddyfile
└── loadtest/
    └── room.js          # k6 load test script
```

---

## 10. ADR (Architecture Decision Records)

### ADR-001: Go + net/http (tanpa framework)

| | |
|---|---|
| **Tanggal** | 2026-09-19 |
| **Status** | Accepted |
| **Konteks** | Tim kecil (1 BE) dengan waktu 6 hari; perlu cepat setup dan mudah dibantu AI agent |
| **Keputusan** | Gunakan Go standard library `net/http` dengan routing manual (method + path param) tanpa framework external seperti Gin atau Echo |
| **Konsekuensi (+)** | Lebih sedikit dependensi = lebih sedikit yang bisa salah; AI agent lebih mudah generate kode standar |
| **Konsekuensi (-)** | Tidak ada middleware built-in; perlu tulis helper untuk param parsing dan error response |
| **Alternatif yang ditolak** | Gin (lebih banyak magic), Echo (unnecessary untuk scope ini) |

### ADR-002: Tempo per Peserta (bukan serentak seperti Kahoot)

| | |
|---|---|
| **Tanggal** | 2026-09-19 |
| **Status** | Accepted |
| **Konteks** | Peserta dengan sinyal lemah tertinggal satu soal penuh jika serentak; sinkronisasi jam antar-HP sulit |
| **Keputusan** | Setiap peserta berjalan dengan temponya sendiri; server mencatat `served_at` dan `answered_at`; tidak ada sinkronisasi jam |
| **Konsekuensi (+)** | Peserta sinyal lemah tidak tertinggal soal; lebih sedikit kompleksitas backend |
| **Konsekuensi (-)** | Leaderboard tidak bisa menunjukkan "siapa yang menjawab lebih cepat saat ini" karena setiap peserta di soal berbeda |
| **Alternatif yang ditolak** | Mode serentak (Kahoot): butuh NTP sync atau timestamp server terpusat; +1 hari kerja backend |

### ADR-003: Satu Goroutine per Room (Channel-based)

| | |
|---|---|
| **Tanggal** | 2026-09-19 |
| **Status** | Accepted |
| **Konteks** | State room (player list, current questions, scores) diakses concurrent dari banyak koneksi WS |
| **Keputusan** | Satu goroutine per room yang menerima semua event via channel; goroutine ini adalah satu-satunya yang menulis state room — tidak perlu mutex |
| **Konsekuensi (+)** | Tidak ada data race; mudah ditest dengan `-race`; pola sederhana |
| **Konsekuensi (-)** | Goroutine leak jika room tidak dibersihkan; butuh timeout dan cleanup |
| **Mitigasi** | Context cancellation + timer 12 menit; registry room dengan cleanup saat `ended` |

### ADR-004: PostgreSQL sebagai Sumber Kebenaran, Redis sebagai Cache

| | |
|---|---|
| **Tanggal** | 2026-09-19 |
| **Status** | Accepted |
| **Konteks** | State room harus survive jika Go process restart; leaderboard butuh update cepat (1×/detik) |
| **Keputusan** | Semua jawaban dan skor tersimpan di PostgreSQL; Redis dipakai hanya untuk PIN lookup, leaderboard sorted set, dan rate limit counter |
| **Konsekuensi (+)** | Jika Go restart, state room dapat dibangun ulang dari `rooms`, `room_players`, `answers`; leaderboard update O(log n) di Redis |
| **Konsekuensi (-)** | Setiap jawaban butuh satu write ke PG + ZINCRBY ke Redis |
| **Alternatif yang ditolak** | Redis-only (kehilangan data jika Redis restart + tidak durable) |

### ADR-005: Docker Compose (bukan Kubernetes)

| | |
|---|---|
| **Tanggal** | 2026-09-19 |
| **Status** | Accepted |
| **Konteks** | Hanya 75 pemain; VPS 2 vCPU / 4GB; timeline 6 hari |
| **Keputusan** | Docker Compose dengan 4 service: caddy, api, postgres, redis |
| **Konsekuensi (+)** | Deploy: `git pull && docker compose up -d --build`; mudah dipahami semua anggota tim |
| **Konsekuensi (-)** | Tidak ada auto-scaling, self-healing pod, atau rolling update |
| **Mitigasi** | Backup pg_dump harian; manual rollback jika perlu; Kubernetes bisa menyusul post-launch |

### ADR-006: Scoring di Server, Kunci Jawaban Tidak Dikirim ke Client

| | |
|---|---|
| **Tanggal** | 2026-09-19 |
| **Status** | Accepted |
| **Konteks** | Klien bisa di-inspect di browser developer tools; peserta bisa cheat jika kunci ada di payload |
| **Keputusan** | `q.show` tidak mengandung field `correct`; semua kalkulasi poin dilakukan di server; `q.result` baru dikirim setelah jawaban diterima |
| **Konsekuensi (+)** | Tidak bisa cheat dengan inspect network; poin fair |
| **Konsekuensi (-)** | Feedback (highlight opsi benar) hanya bisa muncul setelah round-trip ke server |

---

## 11. Keamanan & Privasi

| Area | Kontrol |
|---|---|
| **Transport** | HTTPS wajib via Caddy (TLS otomatis Let's Encrypt) |
| **Host auth** | bcrypt (cost 12) + JWT RS256; rahasia hanya di `.env` |
| **Peserta auth** | Token HMAC per-room; hanya valid untuk satu room |
| **Input validation** | Semua input divalidasi di server (nama, PIN, option_id) |
| **Rate limit** | 10 percobaan join/menit per IP; key `rl:join:{ip}` di Redis TTL 1 menit |
| **CORS** | SPA di subdomain terpisah dari API (`penyengatadventure.tech` vs `api.penyengatadventure.tech`), satu Caddy yang sama; allow-list eksplisit via `CORS_ALLOWED_ORIGINS`, dipakai juga untuk cek `Origin` saat WebSocket upgrade |
| **Data minimisasi** | Nama panggilan, sekolah, jenjang, avatar, bahasa — tanpa email/HP/tgl lahir/foto |
| **Moderasi** | Filter nama kasar ID + Melayu; host dapat kick; tidak ada chat antar-pemain |
| **Retensi** | `room_players` + `answers` dihapus 12 bulan pasca-acara; rekap sekolah disimpan sebagai agregat |
| **Compliance** | UU No. 27/2022 (PDP); persetujuan via sekolah/panitia; host centang konfirmasi saat buat room |
