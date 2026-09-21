# 10 — Team & Workflow
## Jejak Inderasakti: Jelajah Pulau Penyengat

> **Dokumen**: RACI Matrix · Definition of Done · Coding Standard · Branching Strategy · Sprint Roadmap  
> **Versi**: 1.0 · **Tanggal**: 2026-09-19

---

## 1. RACI Matrix

**R** = Responsible (pelaksana) · **A** = Accountable (penanggung jawab) · **C** = Consulted · **I** = Informed

| Aktivitas | Panitia | BE Dev | FE Dev | Desainer |
|---|---|---|---|---|
| **Validasi bank soal & terjemahan EN** | **A** | I | I | I |
| **Desain arsitektur & ADR** | I | **A/R** | C | I |
| **Scaffold repo + kontrak API/WS** | I | **A/R** | C | I |
| **Implementasi BE (REST + WS + Game)** | I | **A/R** | I | I |
| **Implementasi FE (React + i18n)** | I | C | **A/R** | C |
| **Desain UI: style guide, mockup hi-fi** | I | I | C | **A/R** |
| **Produksi aset: maskot, ilustrasi, ikon** | I | I | I | **A/R** |
| **Animasi Lottie maskot** | I | I | C | **A/R** |
| **Seed bank soal (questions.json)** | **A** | **R** | I | I |
| **Review terjemahan EN** | **A** | I | C | I |
| **Setup VPS: domain, DNS, SSH** | **A/R** | C | I | I |
| **Deploy staging + Docker Compose** | I | **A/R** | I | I |
| **Uji beban k6** | I | **A/R** | I | I |
| **Uji E2E Playwright** | I | C | **A/R** | I |
| **QA visual di HP nyata** | I | I | **A/R** | **R** |
| **Uji pengguna Kamis 24/9** | **A** | **R** | **R** | **R** |
| **Hotfix saat acara** | I | **A/R** | **R** | I |
| **Ekspor CSV pasca-acara** | **A/R** | I | I | I |

---

## 2. Definition of Done (DoD)

### 2.1 DoD per Tiket

Sebuah tiket dinyatakan **selesai** hanya jika **semua** kriteria berikut terpenuhi:

- [ ] Kode di-push ke branch fitur sesuai konvensi nama
- [ ] Tidak ada linting error (`golangci-lint` / `pnpm lint`)
- [ ] Unit test untuk logika baru ditulis dan lulus
- [ ] `go test -race ./...` lulus (untuk BE)
- [ ] `pnpm tsc --noEmit` lulus (untuk FE)
- [ ] `contracts/openapi.yaml` atau `contracts/ws.md` **tidak diubah** kecuali tiket eksplisit tentang kontrak
- [ ] PR dibuat dengan deskripsi: konteks, perubahan, cara test
- [ ] PR di-review (setidaknya sekilas) oleh anggota tim lain
- [ ] Merge ke `main` setelah CI lulus

### 2.2 DoD per Fitur (Tambahan)

Untuk fitur yang terlihat di UI:
- [ ] Dicek di viewport 360×800px dan 390×844px
- [ ] Dicek di kedua bahasa (ID dan EN)
- [ ] Dicek dengan `prefers-reduced-motion: reduce`
- [ ] Kontras warna teks ≥ 4.5:1

Untuk fitur backend yang mengubah DB:
- [ ] Migration file goose ada dengan `-- +goose Down`
- [ ] Query diuji dengan `EXPLAIN ANALYZE`

### 2.3 DoD untuk Release (Versi Kamis)

- [ ] Semua tiket P0 selesai
- [ ] `go test -race ./...` lulus 100%
- [ ] k6 load test: p95 < 300ms, 0 error, 5 room × 15 pemain, 15 soal
- [ ] Playwright E2E: semua skenario P0/P1 lulus, kedua bahasa
- [ ] QA visual di HP nyata (minimal 3 model HP berbeda)
- [ ] Deploy staging berhasil; domain HTTPS aktif
- [ ] Bank soal divalidasi kontennya
- [ ] Terjemahan EN direview
- [ ] `pg_dump` manual produksi tersimpan
- [ ] Semua akun host di-seed

---

## 3. Coding Standards

### 3.1 Go Backend

```
Konvensi:
- Package: lowercase, satu kata (game, store, http)
- Exported types: PascalCase (ScoreInput, RoomState)
- Error handling: selalu cek error; jangan _ = err
- Context: setiap fungsi yang melakukan I/O harus menerima ctx context.Context
- Log: pakai log/slog; level info/warn/error; format JSON

Contoh handler:
func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    hostID := mustHostID(ctx)  // dari JWT middleware

    var req CreateRoomRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_BODY", err)
        return
    }

    room, err := h.rooms.Create(ctx, hostID, req)
    if err != nil {
        // handle specific errors
        var maxRoomsErr *MaxRoomsError
        if errors.As(err, &maxRoomsErr) {
            writeError(w, http.StatusConflict, "MAX_ROOMS", nil)
            return
        }
        slog.ErrorContext(ctx, "create room failed", "error", err)
        writeError(w, http.StatusInternalServerError, "INTERNAL", nil)
        return
    }

    writeJSON(w, http.StatusCreated, room)
}
```

**Lint rules** (`.golangci.yml`):
- `errcheck`: semua error harus dicek
- `govet`: semua vet checks
- `staticcheck`: semua static checks
- `gofmt`: kode harus diformat dengan gofmt
- `gosec`: security checks (no SQL injection, no weak crypto)

### 3.2 React / TypeScript Frontend

```typescript
// Konvensi:
// - Komponen: PascalCase (AnswerTile.tsx, TimerBar.tsx)
// - Hooks: camelCase, prefix 'use' (useWsStore, useTimer)
// - File: komponen di src/components/, halaman di src/pages/
// - Props: selalu define interface; jangan pakai 'any'
// - i18n: selalu pakai t('key') dari react-i18next; TIDAK ada hardcode teks UI

// Contoh komponen:
interface AnswerTileProps {
    option: Option;
    state: 'idle' | 'selected' | 'correct' | 'wrong' | 'reveal';
    onClick: (optionId: string) => void;
    disabled: boolean;
}

export function AnswerTile({ option, state, onClick, disabled }: AnswerTileProps) {
    const { t } = useTranslation();
    // ...
}
```

**Lint rules** (`eslint.config.js`):
- `@typescript-eslint/no-explicit-any`: error
- `@typescript-eslint/no-unused-vars`: error
- `react-hooks/exhaustive-deps`: warn
- `no-console`: warn (gunakan slog pattern di sisi BE; di FE hanya untuk dev)

### 3.3 SQL Queries

```sql
-- Gunakan sqlc: tulis SQL di db/queries/*.sql, generate Go code
-- Jangan pernah string concatenation untuk SQL

-- BENAR (sqlc parameterized):
-- name: GetRoomByPIN :one
SELECT * FROM rooms WHERE pin = $1 AND status != 'ended';

-- SALAH (jangan lakukan ini):
-- query := "SELECT * FROM rooms WHERE pin = " + pin
```

### 3.4 Konvensi Commit Message

```
Format: <type>(<scope>): <subject>

Types:
  feat:     Fitur baru
  fix:      Bug fix
  test:     Tambah/perbaiki test
  refactor: Refactoring tanpa perubahan fungsional
  docs:     Dokumentasi
  chore:    Build, CI, dependency update

Contoh:
  feat(game): implement q.answer handler with deadline validation
  fix(scoring): use math.Round instead of int cast for accuracy
  test(room): add race condition tests for concurrent answers
  docs: add ADR-003 goroutine per room decision
```

---

## 4. Branching Strategy

### 4.1 Branch Model (GitHub Flow)

```
main (production-ready)
  |
  +-- feature/BE-01-auth-host
  |       (PR → review → CI → merge)
  |
  +-- feature/FE-03-answer-tile
  |       (PR → review → CI → merge)
  |
  +-- fix/BUG-12-scoring-grace-period
          (PR → review → CI → merge)
```

**Aturan**:
- `main` selalu deployable
- Tidak ada direct push ke `main` (kecuali hotfix darurat saat acara)
- Satu branch per tiket; nama branch: `feature/BE-{num}-{slug}` atau `fix/BUG-{num}-{slug}`
- Branch hidup maksimal 2 jam (sesuai ukuran tiket)
- **AI agent berjalan di branch atau worktree terpisah; tidak pernah di `main`**

### 4.2 Worktree untuk AI Agent

```bash
# Setup worktree untuk agent BE dan FE berjalan paralel
git worktree add ../jejak-be-work feature/BE-07-q-answer
git worktree add ../jejak-fe-work feature/FE-05-question-screen

# Agent BE bekerja di ../jejak-be-work
# Agent FE bekerja di ../jejak-fe-work
# Tidak ada conflict di working tree utama
```

### 4.3 Aturan AI Agent

- Agent tidak diberi akses SSH ke produksi
- Agent tidak boleh mengubah `contracts/openapi.yaml` atau `contracts/ws.md` tanpa tiket eksplisit
- Agent harus membaca `AGENTS.md` dan `docs/` sebelum mulai kode
- Setiap tiket agent: maks. 2 jam, kriteria selesai jelas, 1 branch per tiket

---

## 5. Template Tiket

```markdown
## BE-07: Implementasi q.answer

**Konteks**: 
Baca docs/03_System_Architecture.md (bagian API & WebSocket) dan contracts/ws.md

**Kerjakan**: 
Handler q.answer di api/internal/game/room.go

**Aturan**:
- Tolak bila soal belum di-serve, sudah dijawab, atau lewat deadline (+1000ms)
- Poin memakai game.Score(ScoreInput)
- Simpan ke tabel answers
- ZINCRBY room:{id}:lb di Redis
- Kirim q.result ke peserta
- Kirim lb.update ke host (throttled 1x/detik via debounce)

**Selesai bila**:
- Test tabel scoring_test.go lulus (go test -race)
- room_test.go TC UT-RM-02 dan UT-RM-03 lulus
- contracts/ tidak berubah

**Estimasi**: 1.5 jam
```

---

## 6. Roadmap 6 Hari (Gantt)

```
           Sab 19/9  Min 20/9  Sen 21/9  Sel 22/9  Rab 23/9  Kam 24/9
           [=======] [=======] [=======] [=======] [=======] [=======]

BACKEND:
  Scaffold + migr.  [===]
  Bank soal seed    [===]
  Kontrak API/WS    [===]
  Auth host              [===]
  rooms/join/schools     [===]
  WS hub + goroutine     [===]
  Pemilihan soal              [===]
  Score + tests               [===]
  q.next/q.answer             [===]
  Leaderboard Redis           [===]
  Reconnect                        [=]
  Akhiri room + CSV                [=]
  Deploy staging                        [=]
  Rate limit                            [=]
  Uji beban k6                              [=]
  Bug fix                                   [=]
  Deploy produksi                                [=]
  Monitor + hotfix                               [=]

FRONTEND:
  Scaffold Vite                [===]
  Routing + i18n               [===]
  P1-P3 + mock WS              [===]
  P4 (Lobby)                         [===]
  P7 (Soal)                          [===]
  P8 (Umpan balik)                   [===]
  P5/P6 (Peta + Kartu Info)               [===]
  P9/P10 (Ringkasan + Podium)             [===]
  H1-H5 (Host)                            [===]
  WS integration real                          [=]
  Reconnect                                    [=]
  Toggle EN                                    [=]
  Polish + a11y                                    [=]
  Bug fix                                          [=]
  Hotfix                                                [=]

DESAINER:
  Style guide final    [=]
  Maskot pose utama    [=]
  Hi-fi 6 layar kunci  [=]
  Ikon motif opsi      [=]
  Ilustrasi situs v1       [===]
  Peta pulau               [===]
  12 avatar                [===]
  Lottie maskot                 [===]
  Aset final                    [===]
  QA visual HP nyata                 [=]
  Poster QR room                         [=]
  Aset cadangan                          [=]
  Dampingi uji pengguna                       [=]

CUTOFF: Feature freeze Rab 23/9 18:00 ← TIDAK ADA FITUR BARU SETELAH INI
```

---

## 7. Sprint Breakdown (Daily)

### Sabtu 19/9 — Fondasi

> **Status: ✅ Selesai (2026-09-20)** — scaffold, migrasi 001–008, seed 66 soal ID+EN, `contracts/`. Sisa: `contracts/` belum di-PR review.

| Tim | Goal | Kriteria Selesai |
|---|---|---|
| BE | Scaffold repo, migrasi 6 tabel, seed bank soal 66 soal (ID+EN), kunci `contracts/openapi.yaml` + `contracts/ws.md` | `make seed` berhasil; `go test ./...` lulus; contracts dikunci di PR |
| FE | Scaffold Vite+React, routing 10 halaman + 5 host, setup i18n (id.json + en.json), layar P1-P3 dengan mock data | `pnpm dev` jalan; P1→P2→P3 navigasi manual berfungsi |
| Desainer | Style guide final (token warna, tipografi, spacing), maskot Sakti pose utama (6 pose statis SVG), hi-fi 6 layar kunci, ikon motif opsi | Aset dikirim ke FE: `paper.webp`, 4 SVG motif, 6 SVG Sakti |

### Minggu 20/9 — Core Game

> **Status: ✅ Selesai (2026-09-20, commit `fc53aad`)** — login host (JWT), `rooms`/`join`/`schools`, WS hub + room goroutine, pemilihan soal. Diverifikasi: `go test -race` (via `make test-docker`), Newman, dan alur WS nyata di Docker.

| Tim | Goal | Kriteria Selesai |
|---|---|---|
| BE | Auth host (JWT), rooms/join/schools API, WS hub + room goroutine, pemilihan soal | `POST /api/rooms` berfungsi; WS connect; `go test -race ./...` lulus |
| FE | P4 (Lobby), P7 (Soal), P8 (Umpan Balik), timer, dengan mock WS | Komponen render di 360px; timer berjalan; animasi feedback |
| Desainer | 5 ilustrasi situs v1, peta pulau, 12 avatar | Aset dikirim ke FE untuk integrasi |

### Senin 21/9 — Integrasi & Scoring

> **Status: ✅ Selesai (2026-09-21, commit `83bf01d`)** — `Score()` (UT-SC-01..14), `q.next`/`q.answer`, leaderboard Redis + `lb.update` 1×/detik, reconnect, room ended saat semua selesai. Diverifikasi: `make test-docker` (`-race`) dan satu sesi SD 15 soal end-to-end di Docker (skor Ani 12.750 = maksimum SD di dokumen desain).

| Tim | Goal | Kriteria Selesai |
|---|---|---|
| BE | Score function (UT-SC-01 s/d 14 lulus), q.next/q.answer, leaderboard Redis, reconnect, room ended | Semua unit test lulus termasuk -race; alur game end-to-end di localhost |
| FE | P5 (Peta), P6 (Kartu Info), P9 (Ringkasan), P10 (Podium), H1-H5 (Host), Lottie maskot | Seluruh flow peserta berjalan di localhost; layout host di 1280px |
| Desainer | Lottie maskot 4 animasi (≤60KB each), efek suara 6 file (≤150KB total), aset final | Lottie berjalan di browser; sound tidak distorsi di HP murah |

### Selasa 22/9 — Deploy & Integrasi Nyata

> **Status: 🟡 Kode selesai, deploy VPS menunggu server + domain (2026-09-22).** Selesai: rate limit join 10/menit/IP (Redis + fallback memori), `results.csv`, leaderboard sekolah, `deploy/docker-compose.yml` + `Caddyfile` + panduan `deploy/README.md`, CI GitHub Actions, lint bersih. Diverifikasi: stack staging penuh berjalan lokal (Caddy TLS 1.3, redirect, WSS, rate limit, CSV, `make test-docker`, Newman 41/41). Belum bisa: `https://staging.jejak.example.id` dan sertifikat Let's Encrypt (butuh VPS, DNS, dan akses SSH).

| Tim | Goal | Kriteria Selesai |
|---|---|---|
| BE | Deploy staging di VPS (Compose+Caddy+domain), rate limit, CSV ekspor | https://staging.jejak.example.id accessible; /api/healthz 200; TLS valid |
| FE | Integrasi WS nyata (ganti mock), reconnect, toggle EN, uji di HP nyata | Full flow peserta berjalan di staging dengan HP Android; kedua bahasa |
| Desainer | QA visual di 3 HP nyata + revisi aset | Tidak ada visual bug P1+ di HP uji |

### Rabu 23/9 — QA & Feature Freeze

| Tim | Goal | Kriteria Selesai |
|---|---|---|
| BE | Uji beban k6 lulus (p95<300ms, 0 error), perbaiki bug, backup DB | k6 report pass; `pg_dump` manual tersimpan |
| FE | Polish, aksesibilitas, bug fix; **feature freeze 18.00** | SUS score ≥ 75 dari 2 guru; semua E2E Playwright lulus |
| Desainer | Poster QR room (PDF A4), aset cadangan | Poster siap cetak; asset fallback tersedia |

### Kamis 24/9 — Hari H

| Tim | Tugas |
|---|---|
| BE | Deploy produksi pagi hari; pantau log; hotfix jika ada P0/P1 bug |
| FE | Hotfix jika ada P0/P1 bug; support teknis host |
| Desainer | Dampingi uji pengguna; catat feedback visual |
| Panitia | Koordinasi sesi; distribusi PIN ke host; komunikasi peserta |

---

## 8. Komunikasi Tim

| Channel | Tujuan | Frekuensi |
|---|---|---|
| **Group chat (WhatsApp/Telegram)** | Update harian, blocker, quick questions | Setiap hari |
| **PR comments di GitHub** | Code review, feedback teknis | Per PR |
| **`AGENTS.md`** | Konteks untuk AI agent: stack, perintah, konvensi | Update jika ada perubahan besar |
| **Daily standup (async)** | 3 poin: kemarin / hari ini / blocker | Pagi hari, tiap hari |

### Format Daily Standup

```
[Sabtu 20/9 - BE Dev]
✅ Kemarin: Scaffold repo, migrasi 6 tabel, kunci openapi.yaml
🔨 Hari ini: Auth host + POST /api/rooms
🚧 Blocker: Butuh konfirmasi jumlah akun host dari panitia (OQ-03)
```

---

## 9. Aturan Scope Freeze

> **Setelah Rabu 23/9 pukul 18.00**: TIDAK ADA fitur baru. Yang boleh:
> - Hotfix P0/P1 bug
> - Perubahan konten (teks, soal) tanpa deploy ulang
> - Rollback ke versi sebelumnya

**Prinsip**: Lebih baik game dengan fitur terbatas tapi stabil, daripada banyak fitur tapi crash saat acara.

---

## 10. Post-Event Tasks

| Tugas | PIC | Deadline |
|---|---|---|
| Review soal yang di-flag `review_flag` | Panitia + validator | 1 minggu pasca-acara |
| Update bank soal dari feedback | BE Dev + Panitia | 2 minggu |
| Analisis log server: latency, error rate | BE Dev | 3 hari |
| Kumpulkan feedback guru/siswa | Panitia | 1 minggu |
| Buat backlog fitur post-launch | Semua | 2 minggu |
| Buat Knowledge Item (KI) tentang lesson learned | Tim Dev | 2 minggu |
