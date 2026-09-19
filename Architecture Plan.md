Architecture Plan — Jejak Inderasakti 

# Architecture Plan — Jejak Inderasakti 

2026-09-19 · @Someone 

## Ringkasan & Asumsi 

Rekomendasi: satu binary Go (REST + WebSocket + mesin game) di belakang Caddy, PostgreSQL sebagai sumber kebenaran, dan Redis untuk PIN, leaderboard, serta rate limit — empat kontainer Docker Compose di satu VPS. Arsitektur sengaja dibuat sekecil mungkin agar live dan diuji di VPS pada 2026-09-24 . 

|Parameter|Nilai|Dampak ke desain|
|---|---|---|
|Peserta per<br>room|Maks. 15|Satu goroutine per room sudah cukup|
|Room<br>bersamaan|Maks. 5|Maks. 75 koneksi WebSocket; room ke-6<br>ditolak|
|Durasi sesi|5–10 menit, 15 soal|Batas keras room 12 menit|
|Koneksi|Online saja|Tanpa mode offline; cukup reconnect<br>otomatis|
|Bahasa|Indonesia + Inggris|Kolom konten bilingual + file i18n UI|
|Tim|1 FE, 1 BE, 1 desainer + AI<br>agent|Kontrak API/WS dikunci hari pertama agar<br>FE & BE jalan paralel|
|Server|VPS 2 vCPU / 4 GB RAM|Jauh di atas kebutuhan 75 pemain|



Target kualitas: ack jawaban p95 < 300 ms, halaman awal < 2,5 detik di 4G, dan 0 error pada uji beban 75 pemain. 

## Tech Stack 

Setiap lapisan memakai pilihan paling umum dan paling sedikit dependensi, supaya cepat dikerjakan dan mudah dibantu AI agent. 

Page 1 of 14 

Architecture Plan — Jejak Inderasakti 

|Lapisan|Pilihan|Alasan|
|---|---|---|
|Backend|Go +<br>`net/http`standar (routing<br>method + path param)|Tanpa framework, lebih sedikit yang<br>bisa salah|
|WebSocket|gorilla/websocket|Contoh dan dokumentasi paling<br>banyak; pola read/write pump sudah<br>baku|
|Database|PostgreSQL + pgx v5|Driver Postgres paling matang di Go|
|Query|sqlc|Tulis SQL biasa, kode Go type-safe<br>dihasilkan otomatis|
|Migrasi|goose|File SQL berurutan, dijalankan saat<br>startup|
|Cache|Redis + go-redis v9|PIN → room, leaderboard (sorted<br>set), rate limit|
|Log|`log/slog`(JSON)|Standar library|
|Frontend|Vite + React + TypeScript|SPA statis tanpa SSR; hasil build<br>dilayani langsung oleh Caddy|
|UI|Tailwind CSS + Framer Motion + lottie-<br>react|Cepat untuk UI berwarna dan<br>animasi maskot|
|State & i18n|Zustand + react-i18next|Ringan; file<br>`id.json`dan<br>`en.json`|
|Proxy &<br>TLS|Caddy|HTTPS otomatis + file server FE|
|Deploy|Docker Compose|Satu perintah untuk naik/turun|



Tidak dipakai di versi Kamis: Kubernetes, message queue, CDN, object storage, dan service worker. Aset FE dibundel ke hasil build dengan total ≤ 5 MB. Jika developer FE lebih nyaman dengan Next.js, pakai `output: 'export'` agar hasilnya tetap file statis. 

## Diagram Arsitektur 

Semua permintaan masuk lewat Caddy; hanya `/api` dan `/ws` yang diteruskan ke Go, sisanya file statis FE. 

Page 2 of 14 

Architecture Plan — Jejak Inderasakti 

```
flowchart LR
  P[HP Peserta<br/>SPA] --> C[Caddy<br/>HTTPS + file FE]
  H[Layar Host] --> C
  C -->|/api, /ws| G[Go API<br/>REST + WebSocket]
  G --> R1[Room goroutine<br/>maks. 5]
  G --> PG[(PostgreSQL)]
  G --> RD[(Redis)]
```

### **Alur satu room (tempo per peserta, gaya Quizizz).** 

```
sequenceDiagram
  participant Host
  participant Go as Go API
  participant DB as Postgres/Redis
  participant P as Peserta
  Host->>Go: POST /api/rooms
  Go->>DB: simpan room + 15 question_ids, SET pin
  P->>Go: POST /api/rooms/{pin}/join
  Go-->>P: token pemain
  P->>Go: WS connect
  Host->>Go: host.start
  Go-->>P: room.started
  loop 15 soal
    P->>Go: q.next
    Go->>DB: catat served_at
    Go-->>P: q.show (tanpa kunci)
    P->>Go: q.answer
    Go->>DB: simpan jawaban, ZINCRBY
    Go-->>P: q.result (kunci, poin)
    Go-->>Host: lb.update
  end
  Go-->>P: room.ended (podium)
```

Setelah host menekan Mulai, tiap peserta berjalan dengan temponya sendiri. Server cukup mencatat waktu soal dikirim dan jawaban diterima, jadi tidak perlu sinkronisasi jam antar-HP. 

## Scope Versi Kamis & Struktur Repo 

Dengan 1 FE + 1 BE dalam 6 hari, semua yang tidak ada di kolom "Kamis" harus ditunda; menambah satu fitur besar (misalnya CMS) hampir pasti menggeser uji VPS. 

Page 3 of 14 

Architecture Plan — Jejak Inderasakti 

|Fitur|Kamis|Setelah launch|
|---|---|---|
|Daftar per room: nama, sekolah,<br>jenjang, avatar, bahasa|Ya|—|
|Room PIN + QR, maks. 15 peserta,<br>maks. 5 room aktif|Ya|—|
|15 soal (Sesi Singkat 10), timer, poin,<br>bonus beruntun|Ya|—|
|Leaderboard room real-time +<br>podium|Ya|—|
|Rekap leaderboard sekolah|Ya (halaman sederhana)|Grafik & filter|
|Layar host: buat room, lobby, pantau,<br>akhiri|Ya|—|
|Ekspor hasil CSV|Ya|—|
|Bilingual ID/EN|Ya|—|
|Akun host|Di-seed panitia, tanpa halaman<br>daftar|Manajemen<br>akun|
|Bank soal|`seed/questions.json`+<br>perintah seed|CMS web|
|Mode Mandiri, XP, lencana, sertifikat,<br>analitik butir soal|—|Ya|



Page 4 of 14 

Architecture Plan — Jejak Inderasakti 

```
jejak-inderasakti/
├── AGENTS.md            # konteks untuk AI agent
├── docs/                # 3 dokumen ini, diekspor ke markdown
├── contracts/
│   ├── openapi.yaml     # REST
│   └── ws.md            # pesan WebSocket + contoh JSON
├── api/                 # Go
│   ├── cmd/server/  cmd/seed/
│   ├── internal/http/   # handler REST, middleware
│   ├── internal/ws/     # koneksi, read/write pump
│   ├── internal/game/   # room, pemilihan soal, scoring
│   ├── internal/store/  # kode hasil sqlc
│   └── db/migrations/  db/queries/
├── web/                 # Vite + React
│   └── src/pages/  src/components/  src/game/  src/i18n/
├── seed/questions.json  seed/schools.csv
├── deploy/docker-compose.yml  deploy/Caddyfile
└── loadtest/room.js     # k6
```

## Data Model 

Enam tabel Postgres dan empat key Redis. Tidak ada akun pemain permanen: identitas pemain hanya berlaku di satu room, sehingga data anak yang disimpan tetap minimal. 

```
erDiagram
```

```
  SCHOOLS ||--o{ ROOM_PLAYERS : asal
  HOST_USERS ||--o{ ROOMS : membuat
  ROOMS ||--o{ ROOM_PLAYERS : berisi
  ROOM_PLAYERS ||--o{ ANSWERS : menjawab
  QUESTIONS ||--o{ ANSWERS : dijawab
```

|Tabel|Kolom|Catatan|
|---|---|---|
|`schools`|id, name, jenjang, city, verified|Seed dari CSV, plus "Sekolah lain"<br>dan "Umum / General visitor"|
|`host_users`|id, email, password_hash (bcrypt),<br>name|Di-seed panitia|



Page 5 of 14 

Architecture Plan — Jejak Inderasakti 

|Tabel|Kolom|Catatan|
|---|---|---|
|`questions`|id (<br>`M1-01`), site (1–5), level (1–3),<br>type (<br>`mc`,<br>`tf`,<br>`fill`), prompt<br>JSONB<br>`{id, en}`, options JSONB<br>`[{id, label:{id,en}, correct}]`,<br>explanation JSONB<br>`{id, en}`,<br>active|Diisi dari<br>`seed/questions.json`|
|`rooms`|id, pin, host_id, jenjang,<br>short_session, accuracy_mode,<br>status (<br>`lobby`/<br>`running`/<br>`ended`),<br>question_ids TEXT[], created_at,<br>started_at, ended_at|Set soal dipilih sekali saat room<br>dibuat|
|`room_players`|id, room_id, nickname, school_id,<br>jenjang, avatar, lang, score,<br>correct_count, total_ms,<br>current_index, streak, finished_at|Unique (room_id, nickname)|
|`answers`|room_player_id, question_id,<br>served_at, answered_at, option_id,<br>correct, points|PK (room_player_id, question_id)<br>mencegah jawab ganda|



|Key Redis|Tipe|Isi|TTL|
|---|---|---|---|
|`pin:{pin}`|string|room_id|2 jam|
|`room:{id}:lb`|sorted set|room_player_id → skor|2 jam|
|`rooms:active`|set|room_id yang belum berakhir (maks. 5)|—|
|`rl:join:{ip}`|counter|Percobaan join|1 menit|



Postgres adalah sumber kebenaran; jika proses Go restart di tengah sesi, state room dibangun ulang dari `rooms` , `room_players` , dan `answers` . 

## API, WebSocket & Penilaian 

Klien tidak pernah menerima kunci jawaban sebelum menjawab dan tidak pernah menghitung poin sendiri. Kontrak di bagian ini disalin ke `contracts/` pada hari pertama dan dikunci. 

Page 6 of 14 

Architecture Plan — Jejak Inderasakti 

|Method|Path|Pengguna|Fungsi|
|---|---|---|---|
|POST|`/api/auth/login`|Host|Login, mengembalikan JWT|
|POST|`/api/rooms`|Host|Buat room; 409 bila sudah ada 5<br>room aktif|
|GET|`/api/rooms/{pin}`|Publik|Cek room ada dan sisa slot|
|POST|`/api/rooms/{pin}/join`|Peserta|Masuk room; 409 bila penuh (15)<br>atau nama dipakai|
|GET|`/api/schools?q=`|Publik|Autocomplete sekolah|
|GET|`/api/rooms/{id}/results.csv`|Host|Ekspor hasil|
|GET|`/api/leaderboard/schools`|Publik|Rekap sekolah|
|GET|`/api/healthz`|Ops|Health check|



### **Pesan WebSocket** ( `/ws?token=` ), format `{"t": "q.answer", "d": {...}}` . 

|Arah|Tipe|Data|
|---|---|---|
|Peserta → server|`q.next`|—|
|Peserta → server|`q.answer`|question_id, option_id (null bila waktu<br>habis)|
|Host → server|`host.start`,<br>`host.end`,<br>`host.kick`|player_id untuk kick|
|Server → peserta|`room.state`|status, daftar peserta, index soal<br>pemain|
|Server → peserta|`q.show`|index, site, level, prompt, options tanpa<br>`correct`, limit_ms|
|Server → peserta|`q.result`|correct, correct_option_id, explanation,<br>points, score, streak|
|Server → semua di<br>room|`lb.update`|Peringkat (maks. 1 kali/detik)|
|Server → semua di<br>room|`room.ended`|Podium dan skor akhir|



Page 7 of 14 

Architecture Plan — Jejak Inderasakti 

|Arah|Tipe|Data|
|---|---|---|
|Dua arah|`ping`/<br>`pong`|Keepalive tiap 20 detik|



### **Fungsi penilaian (Go).** 

```
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

Tes tabel wajib memakai contoh di dokumen About Game: soal sedang, timer 20 detik, dijawab di detik 1 / 4 / 12 / 19 → 750 / 717 / 583 / 467. 

### **Aturan server.** 

Page 8 of 14 

Architecture Plan — Jejak Inderasakti 

1. `q.next` hanya dilayani bila soal sebelumnya sudah dijawab atau melewati deadline; soal yang terlewat dicatat 0 poin. 

2. Deadline = served_at + limit_ms + 1.000 ms. 

3. Reconnect: token masih valid → server mengirim `room.state` dan soal aktif bila belum lewat deadline. 

4. Room otomatis berakhir saat semua peserta selesai, host menekan Akhiri, atau 12 menit setelah dimulai. 

5. Satu room = satu goroutine yang memproses event dari channel, sehingga state room tidak perlu mutex. Wajib lulus `go test -race` . 

6. Rate limit join 10 kali/menit per IP; PIN 6 digit dihapus dari Redis saat room berakhir. 

## Bilingual (ID/EN) 

Bahasa dipilih di layar pertama dan berlaku per pemain, jadi satu room boleh berisi peserta berbahasa Indonesia dan Inggris sekaligus. 

|Bagian|Cara|
|---|---|
|Teks UI|react-i18next, file<br>`web/src/i18n/id.json`dan<br>`en.json`, kunci datar seperti<br>`lobby.waiting`|
|Konten soal|Kolom JSONB<br>`{id, en}`di<br>`questions`; server<br>hanya mengirim bahasa pemain dalam<br>`q.show`dan<br>`q.result`|
|Nama diri|Tetap dalam bahasa Melayu (Rumah Sotoh,<br>Gurindam Dua Belas, tabib, perigi) dengan penjelas<br>singkat di versi EN|
|Ilustrasi|Tanpa teks di dalam gambar, supaya satu aset<br>dipakai untuk dua bahasa|
|Sumber terjemahan|Draf 60 soal sudah ada di tab Question Bank (EN)<br>dokumen About Game; perlu direview sebelum<br>Rabu|
|Pengujian|Tiap layar dicek di kedua bahasa; layout diuji<br>dengan teks Indonesia yang umumnya lebih<br>panjang|



Page 9 of 14 

Architecture Plan — Jejak Inderasakti 

## Deployment & Pengujian 

Deploy versi Kamis cukup `git pull` lalu `docker compose up -d --build` di VPS; registry image dan pipeline CD bisa menyusul setelah launch. 

```
# deploy/docker-compose.yml
services:
  caddy:
    image: caddy:2-alpine
    ports: ["80:80", "443:443"]
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - ../web/dist:/srv/web:ro
      - caddy_data:/data
  api:
    build: ../api
    env_file: .env   # DATABASE_URL, REDIS_URL, JWT_SECRET
    depends_on: [postgres, redis]
  postgres:
    image: postgres:17-alpine
    env_file: .env
    volumes: [pg_data:/var/lib/postgresql/data]
  redis:
    image: redis:7-alpine
volumes: { caddy_data: {}, pg_data: {} }
# deploy/Caddyfile
jejak.example.id {
  handle /api/* { reverse_proxy api:8080 }
  handle /ws    { reverse_proxy api:8080 }
  handle {
    root * /srv/web
    try_files {path} /index.html
    file_server
  }
}
```

Page 10 of 14 

Architecture Plan — Jejak Inderasakti 

|Area|Keputusan|
|---|---|
|VPS|2 vCPU / 4 GB, Ubuntu LTS, region Singapura atau<br>Jakarta; domain + A record siap paling lambat<br>Selasa|
|CI|GitHub Actions di setiap PR:<br>`go test -race`<br>`./...`, golangci-lint,<br>`pnpm lint && pnpm build`|
|Uji beban|k6: 5 room × 15 pemain virtual memainkan 15 soal;<br>lulus bila ack p95 < 300 ms dan 0 error|
|Uji end-to-end|Playwright di viewport HP: daftar → main 15 soal →<br>podium, dalam kedua bahasa|
|Uji pengguna|Kamis: 1–2 kelas memainkan 5 room serentak di HP<br>masing-masing|
|Backup|`pg_dump`harian via cron, plus manual sebelum uji<br>Kamis|
|Pemantauan|Log JSON (<br>`docker compose logs -f api`),<br>`/api/healthz`, Uptime Kuma opsional|



## Keamanan & Privasi Data Anak 

Hampir semua pemain di bawah 18 tahun, jadi sistem hanya menyimpan data minimal per room dan persetujuan diambil lewat sekolah/panitia. UU No. 27 Tahun 2022 (PDP) berlaku penuh sejak 17 Oktober 2024 dan mengatur data anak secara khusus, termasuk persetujuan orang tua/wali (Pasal 25). Ini bukan nasihat hukum; panitia sebaiknya memastikannya dengan pihak sekolah. 

|Area|Kontrol versi Kamis|
|---|---|
|Minimasi data|Nama panggilan, sekolah, jenjang, avatar, bahasa;<br>tanpa email, HP, tanggal lahir, atau foto|
|Persetujuan|Izin kegiatan dari sekolah; host mencentang<br>konfirmasi saat membuat room|



Page 11 of 14 

Architecture Plan — Jejak Inderasakti 

|Area|Kontrol versi Kamis|
|---|---|
|Retensi|`room_players`dan<br>`answers`dihapus 12 bulan<br>setelah acara; rekap sekolah disimpan sebagai<br>agregat|
|Moderasi|Filter nama kasar (Indonesia + Melayu); host bisa<br>kick; tidak ada chat antarpemain|
|Autentikasi|Host: bcrypt + JWT; peserta: token bertanda<br>tangan yang hanya berlaku di satu room|
|Aplikasi|HTTPS wajib, validasi input di server, rate limit join,<br>CORS same-origin, rahasia hanya di<br>`.env`|



## Rencana 6 Hari 

Titik kritis ada di Selasa malam: satu room penuh harus sudah berjalan end-to-end di VPS staging. Jika belum, potong Sesi Normal (pakai Sesi Singkat), ganti animasi Lottie dengan gambar statis, dan tunda rekap sekolah. 

|Hari|BE (Go)|FE (React)|Desainer|
|---|---|---|---|
|Sab<br>19/9|Scaffold repo, migrasi, seed<br>bank soal ID+EN, kunci<br>`contracts/`|Scaffold Vite, routing, i18n,<br>token desain sementara,<br>layar bahasa/PIN/daftar<br>dengan mock|Style guide final,<br>maskot Sakti pose<br>utama, hi-fi 6 layar<br>kunci, ikon opsi<br>jawaban|
|Min<br>20/9|Auth host,<br>rooms/join/schools, WS hub<br>+ room goroutine, pemilihan<br>soal|Lobby, layar soal<br>(PG/BS/LK), timer, umpan<br>balik, pakai mock WS|5 ilustrasi situs v1,<br>peta pulau, 12 avatar|
|Sen<br>21/9|Score + tes tabel, alur<br>q.next/q.answer,<br>leaderboard Redis,<br>reconnect, akhiri room|Peta & transisi stage, kartu<br>info, leaderboard, podium,<br>layar host|Lottie maskot (idle,<br>benar, salah,<br>menang), efek suara,<br>aset final|
|Sel<br>22/9|Deploy staging di VPS<br>(Compose + Caddy +<br>domain), rate limit, CSV|Integrasi WS nyata,<br>reconnect, toggle EN, cek<br>di HP asli|QA visual di HP asli,<br>revisi aset|



Page 12 of 14 

Architecture Plan — Jejak Inderasakti 

|Hari|BE (Go)|FE (React)|Desainer|
|---|---|---|---|
|Rab|Uji beban k6 75 pemain,|Polish, aksesibilitas, bug;|Poster QR room, aset|
|23/9|perbaikan bug, backup|feature freeze pukul 18.00|cadangan|
|Kam|Deploy produksi, pantau log,|Hotfix|Dampingi uji|
|24/9|hotfix||pengguna|



## Pengembangan dengan AI Agent 

AI agent (misalnya Claude Code) dipakai untuk kode yang polanya jelas; bagian yang menentukan kebenaran game — konkurensi room, scoring, keamanan — tetap di-review manusia baris per baris. 

### **Siapkan di jam pertama.** 

1. `AGENTS.md` di root: stack, perintah ( `make dev` , `make test` , `make seed` ), konvensi, dan definisi selesai. 

2. `contracts/openapi.yaml` + `contracts/ws.md` : satu-satunya sumber kebenaran FE– BE; agent tidak boleh mengubahnya tanpa tiket. 

3. Ekspor 3 dokumen ini ke `docs/` agar agent membaca aturan game langsung. 

|Tugas|Dikerjakan agent|Review manusia|
|---|---|---|
|Migrasi SQL, query sqlc, handler<br>REST|Penuh|Sekilas|
|Konversi bank soal dari dokumen ke<br>`seed/questions.json`(ID+EN)|Penuh|Cek 10 soal acak|
|Komponen UI, file i18n, layar|Penuh|Visual di HP asli|
|Klien WS dengan reconnect, store<br>Zustand|Penuh|Uji putus sinyal|
|Dockerfile, Compose, Caddyfile,<br>skrip deploy, skrip k6|Penuh|Sekali jalan di<br>staging|
|Room goroutine, alur|Draf|Wajib penuh +<br>`go`|
|`q.next`/<br>`q.answer`||`test -race`|



Page 13 of 14 

Architecture Plan — Jejak Inderasakti 

|Tugas|Dikerjakan agent|Review manusia|
|---|---|---|
|Fungsi Score|Tulis tes dulu dari contoh<br>dokumen, baru kode|Wajib penuh|
|Auth, rate limit, validasi input|Draf|Wajib penuh|



**Aturan kerja.** Tiket maksimal 2 jam dengan kriteria selesai yang jelas; satu branch per tiket; agent FE dan BE berjalan paralel di branch atau worktree terpisah; agent tidak diberi akses SSH produksi. 

```
Tiket BE-07: Implementasi q.answer
```

```
Konteks: docs/architecture.md (API, WebSocket & Penilaian), contracts/ws.md
Kerjakan: handler q.answer di api/internal/game/room.go
Aturan: tolak bila soal belum di-serve, sudah dijawab, atau lewat deadline
(+1000 ms).
```

```
        Poin memakai game.Score; simpan ke answers; ZINCRBY room:{id}:lb;
kirim q.result.
```

```
Selesai bila: tes tabel di room_test.go lulus dengan -race; contracts/ tidak
berubah.
```

Page 14 of 14 

