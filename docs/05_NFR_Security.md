# 05 — Non-Functional Requirements, Security & Reliability
## Jejak Inderasakti: Jelajah Pulau Penyengat

> **Dokumen**: Threat Model · Security Checklist · Capacity Planning · SLA/SLO/SLI · DR & Backup Plan  
> **Versi**: 1.0 · **Tanggal**: 2026-09-19

---

## 1. Non-Functional Requirements (NFR) Lengkap

### 1.1 Performance

| NFR | Target | Cara Ukur |
|---|---|---|
| ACK jawaban (q.answer → q.result, end-to-end) | p95 < 300ms | k6 histogram |
| WebSocket message processing (server only) | p99 < 50ms | slog timing |
| Halaman awal (FCP) | < 2.5 detik di 4G (10 Mbps sim) | Lighthouse |
| Bundle FE (gzip) | ≤ 5 MB total aset | `du -sh web/dist` |
| API response (REST, non-game) | p95 < 200ms | k6 |

### 1.2 Scalability

| NFR | Target | Keterangan |
|---|---|---|
| Concurrent WebSocket connections | Maks. 75 + 5 host = 80 | Hard limit via `rooms:active` |
| Concurrent rooms | Maks. 5 | Tolak room ke-6 dengan HTTP 409 |
| Concurrent users per room | Maks. 15 | Tolak peserta ke-16 |
| Throughput q.answer | ~75 req/detik (peak, semua jawab bersamaan) | Ditest dengan k6 |

*Catatan: VPS 2 vCPU / 4 GB jauh di atas kebutuhan; bottleneck lebih mungkin di koneksi internet peserta daripada server.*

### 1.3 Availability

| NFR | Target | Periode |
|---|---|---|
| Availability saat acara (Kamis 24/9, 08:00–17:00 WIB) | 99.9% | = < 26 detik downtime dalam 8 jam |
| Recovery time jika Go crash | < 30 detik | Docker restart policy: `always` |
| Recovery time jika DB crash | < 5 menit (manual restart) | Data tidak hilang karena PG durable |

### 1.4 Reliability

| NFR | Target |
|---|---|
| Reconnect berhasil (token valid, room masih running) | 100% |
| State konsisten setelah Go restart | State dibangun ulang dari PG; 0 data loss |
| Jawaban ganda (double submit) | Dicegah oleh PK (room_player_id, question_id) |
| Skor cheat via client-side tampering | Tidak mungkin: server yang hitung poin |

### 1.5 Security

| NFR | Target |
|---|---|
| Transport | HTTPS wajib (TLS 1.2+); redirect HTTP → HTTPS |
| Auth host | bcrypt cost 12; JWT RS256 atau HS256 dengan secret kuat |
| Data anak | Tidak ada PII (email/HP/tgl lahir/foto) |
| Input validation | Semua input divalidasi di server (bukan hanya client) |
| Rate limiting | 10 percobaan join/menit per IP |

### 1.6 Usability

| NFR | Target |
|---|---|
| Onboarding host (login + buat room + siap) | < 5 menit |
| Onboarding peserta (PIN + daftar + lobby) | < 3 menit |
| Layar peserta tanpa scroll | Semua layar kecuali P3 (daftar) |
| Aksesibilitas kontras | ≥ 4.5:1 (WCAG AA) untuk semua teks |

### 1.7 Compatibility

| NFR | Target |
|---|---|
| Browser | Chrome ≥ 90, Firefox ≥ 88, Safari ≥ 14 (iOS 14+) |
| OS | Android 8+, iOS 14+ |
| Layar | 360px – 414px lebar (mobile); 1280px (host landscape) |
| Koneksi minimum | 4G (≥ 10 Mbps download) |

---

## 2. Threat Model (STRIDE)

### 2.1 Data Flow Diagram (untuk Threat Modeling)

```
[Peserta Browser] --HTTPS--> [Caddy] --HTTP--> [Go API] --TCP--> [PostgreSQL]
                                                         --TCP--> [Redis]
[Host Browser]    --HTTPS--> [Caddy] --WS----> [Go API]
```

### 2.2 Analisis STRIDE

| Ancaman | Kategori | Deskripsi | Mitigasi |
|---|---|---|---|
| **T1** | **S**poofing | Pemain berpura-pura sebagai pemain lain dengan token curian | Token HMAC per-room; token hanya berlaku untuk satu `room_player_id` |
| **T2** | **T**ampering | Pemain memodifikasi request `q.answer` untuk kirim option_id "correct" | Server validasi option_id dari DB; server yang hitung correct/incorrect |
| **T3** | **T**ampering | Pemain replay request `q.answer` untuk jawab soal yang sama dua kali | PK (room_player_id, question_id) di tabel `answers`; server tolak duplicate |
| **T4** | **T**ampering | Pemain kirim `q.next` sebelum menjawab soal sebelumnya | Server cek `current_index` dan status jawaban sebelum melayani `q.next` |
| **T5** | **R**epudiation | Host menuduh server salah hitung skor | Semua jawaban tersimpan di `answers` dengan timestamp; dapat di-audit |
| **T6** | **I**nformation Disclosure | Pemain intercept `q.show` untuk mendapatkan kunci jawaban | `q.show` tidak mengandung field `correct`; validasi di server |
| **T7** | **I**nformation Disclosure | Attacker dump data anak via SQL injection | Semua query via sqlc (parameterized); tidak ada raw SQL concatenation |
| **T8** | **D**enial of Service | Brute force PIN untuk masuk room orang lain | Rate limit: 10 join/menit per IP; PIN hanya 6 digit tapi TTL 2 jam |
| **T9** | **D**enial of Service | Attacker membuat 5+ room untuk memblokir host lain | Hard limit via Redis `rooms:active`; hanya host terautentikasi yang bisa buat room |
| **T10** | **D**enial of Service | Flood WebSocket messages | Read pump mengabaikan unknown message types; goroutine room memproses 1 event per iterasi |
| **T11** | **E**levation of Privilege | Peserta mencoba akses endpoint host (buat room, CSV) | Endpoint host dilindungi middleware JWT; token peserta tidak valid untuk endpoint host |
| **T12** | **S**poofing | Attacker berpura-pura sebagai host dengan JWT palsu | JWT divalidasi signature; secret hanya di `.env` server |

### 2.3 Risiko yang Diterima (Acceptable Risk)

| Risiko | Alasan Diterima |
|---|---|
| Brute force PIN (4.096 kemungkinan, bukan 10^6 karena hanya digit 0-9) | Rate limit 10/menit; PIN hanya aktif saat room berlangsung; tidak ada data kritis di balik PIN |
| Pemain bisa logout dan join ulang dengan nama berbeda | Room tidak menyimpan "banned users"; panitia bisa kick dan buat PIN baru jika diperlukan |
| Host password di-seed (bukan self-register) | Lebih aman dari phishing; tidak ada halaman reset password yang bisa diserang |

---

## 3. Security Checklist

### 3.1 Transport & Network

- [ ] HTTPS dipaksa (Caddy auto TLS via Let's Encrypt)
- [ ] HTTP → HTTPS redirect aktif
- [ ] HSTS header dikonfigurasi
- [ ] WebSocket menggunakan `wss://` (bukan `ws://`)
- [ ] CORS: `same-origin` (SPA dan API di domain yang sama)

### 3.2 Authentication & Authorization

- [ ] Password host di-hash dengan bcrypt cost 12
- [ ] JWT memiliki expiry (8 jam); secret minimal 256-bit random
- [ ] Token peserta HMAC-signed dengan secret server; mengandung room_id + player_id
- [ ] Middleware JWT diaplikasikan ke semua endpoint host sebelum handler
- [ ] Endpoint peserta memvalidasi token sebelum aksi apapun

### 3.3 Input Validation

- [ ] Nama panggilan: 2–20 karakter; filter kata kasar; unique per room
- [ ] PIN: tepat 6 digit numerik
- [ ] option_id: harus ada di opsi soal yang di-serve
- [ ] question_id dalam `q.answer` harus sesuai soal yang sedang aktif untuk pemain
- [ ] Semua query parameterized (sqlc)

### 3.4 Data Privacy

- [ ] Tidak ada email/HP/tanggal lahir/foto yang dikumpulkan dari peserta
- [ ] Log tidak mengandung PII (nama panggilan tidak di-log, hanya room_player_id)
- [ ] `.env` tidak masuk ke Git (`.gitignore`)
- [ ] Database connection string, JWT secret, Redis password hanya di `.env`
- [ ] Host mencentang konfirmasi persetujuan izin sekolah saat buat room

### 3.5 Application Security

- [ ] Rate limit: 10 join/menit per IP (`rl:join:{ip}` di Redis)
- [ ] Batas room: maks. 5 aktif (`rooms:active` set di Redis)
- [ ] Batas peserta: maks. 15 per room (cek di DB saat join)
- [ ] Kunci jawaban tidak ada di payload `q.show`
- [ ] Semua penilaian dihitung di server
- [ ] PK (room_player_id, question_id) mencegah jawab ganda
- [ ] `go test -race` harus lulus untuk room goroutine

### 3.6 Infrastructure

- [ ] VPS dengan firewall: hanya port 80 dan 443 terbuka (Caddy)
- [ ] Port 5432 (PG) dan 6379 (Redis) tidak expose ke internet
- [ ] Docker network internal untuk komunikasi container
- [ ] SSH key-based login; password SSH dinonaktifkan
- [ ] `pg_dump` harian via cron; manual dump sebelum deploy produksi

---

## 4. Capacity Planning

### 4.1 Estimasi Beban

| Metrik | Kalkulasi | Nilai |
|---|---|---|
| Peak concurrent users | 5 room × 15 peserta + 5 host | 80 |
| WebSocket connections | 80 (1 per user) | 80 |
| Messages per detik (peak) | 80 users × ~1 msg/4 detik rata-rata | ~20 msg/s |
| Peak q.answer (semua jawab bersamaan) | 75 dalam ~1 detik | 75 req/s |
| DB writes per detik (peak) | 75 `answers` INSERT + 75 `room_players` UPDATE | 150 write/s |
| Redis ZINCRBY per detik (peak) | 75/s | 75 ops/s |

### 4.2 Estimasi Resource Usage

| Resource | Estimasi Usage | VPS Kapasitas | Headroom |
|---|---|---|---|
| CPU | < 5% (Go goroutine ringan; 80 WS conn) | 2 vCPU = 200% | 40× |
| RAM | Go ~50MB + PG ~256MB + Redis ~32MB + OS ~512MB | 4 GB | ~13× |
| Network (peak) | ~80 conn × 1KB/msg × 20 msg/s = 1.6 MB/s | 1 Gbps typical VPS | 600× |
| Disk I/O | 150 write/s × ~1KB = 150 KB/s | ~100 MB/s NVMe typical | 667× |
| Disk storage (60 hari data) | rooms: ~5KB × 100 room = 500KB; answers: ~200B × 100×15 = 300KB | 50+ GB typical | Sangat cukup |

### 4.3 Bottleneck Analysis

| Komponen | Risiko | Mitigasi |
|---|---|---|
| **Koneksi internet peserta** | 4G sinyal lemah di Pulau Penyengat | Desain tempo per peserta; reconnect otomatis; soal tidak "hilang" |
| **WebSocket goroutine leak** | Room goroutine tidak berhenti jika cleanup gagal | Context + timer 12 menit; panic recovery di goroutine |
| **Redis unavailable** | Leaderboard tidak update; rate limit bypass | Graceful degradation: leaderboard dari DB; rate limit dari in-memory counter sementara |
| **PostgreSQL deadlock** | Concurrent INSERT ke `answers` untuk soal sama | PK mencegah duplicate; pgx connection pool handles retry |

---

## 5. SLA / SLO / SLI Definition

### 5.1 SLA (Service Level Agreement)
> Berlaku untuk acara Kamis 24/9/2026, 08:00–17:00 WIB (9 jam)

| Komitmen | Target |
|---|---|
| Availability (uptime) | ≥ 99.9% (= max 26 detik downtime per 8 jam sesi) |
| Error rate (5xx responses) | < 0.1% dari total request |
| Incident response time | < 5 menit (tim on-call selama acara) |

### 5.2 SLO (Service Level Objectives)

| SLO | Target | Window |
|---|---|---|
| ACK latency (q.answer → q.result) p95 | < 300ms | Rolling 5 menit |
| REST API latency p95 | < 200ms | Rolling 5 menit |
| WebSocket connection success rate | ≥ 99% | Per acara |
| Reconnect success rate | ≥ 95% | Per acara |
| Data integrity (skor akhir akurat) | 100% | Per sesi |

### 5.3 SLI (Service Level Indicators)

| SLI | Cara Ukur | Tool |
|---|---|---|
| Latency | `answered_at - served_at` di tabel `answers`; histogram di k6 | k6 + slog |
| Availability | `/api/healthz` response time; `docker compose ps` | Uptime Kuma (opsional) |
| Error rate | Count HTTP 5xx / total request dari log | slog + grep |
| Throughput | Count `answers` INSERT per menit | PostgreSQL query |
| Reconnect success | Count `room.state` dikirim / count WS reconnect attempt | slog event |

---

## 6. Disaster Recovery & Backup Plan

### 6.1 Backup Strategy

| Data | Metode | Frekuensi | Retensi | Lokasi |
|---|---|---|---|---|
| PostgreSQL full dump | `pg_dump -Fc` via cron | Harian 02:00 | 7 hari | `/backups/pg/` di VPS |
| Pre-deploy manual dump | `make backup` sebelum setiap deploy | Per deploy | 30 hari | `/backups/pg/manual/` |
| `seed/questions.json` | Git repository | Per commit | Permanen | GitHub |
| `seed/schools.csv` | Git repository | Per commit | Permanen | GitHub |
| `.env` | Manual encrypted storage | Per perubahan | Permanen | Offline / password manager |

```bash
# Cron job untuk pg_dump harian (crontab di VPS)
0 2 * * * pg_dump -Fc $DATABASE_URL > /backups/pg/$(date +%Y%m%d).dump

# Cleanup dump > 7 hari
0 3 * * * find /backups/pg/ -name "*.dump" -mtime +7 -delete
```

### 6.2 Recovery Scenarios

| Skenario | Dampak | RTO | Prosedur |
|---|---|---|---|
| **Go process crash** | Sesi terinterupsi ~30 detik | < 1 menit | Docker restart otomatis (`restart: always`); state rebuild dari PG |
| **Redis crash** | Leaderboard tidak update; PIN lookup gagal | < 2 menit | `docker compose restart redis`; leaderboard rebuild dari PG |
| **PostgreSQL crash** | Semua sesi gagal | < 5 menit | `docker compose restart postgres`; data aman di volume PG |
| **VPS down** | Semua sesi gagal | < 30 menit | Jalankan `docker compose up` dari backup atau instance cadangan; restore pg_dump terbaru |
| **Deploy gagal (FE/BE corrupt)** | Sebagian fitur rusak | < 5 menit | `git revert` + `docker compose up --build`; atau manual rollback ke image sebelumnya |
| **Soal seed salah** | Konten soal tidak akurat | Sesi baru | `make seed` ulang dengan `questions.json` yang dikoreksi; room yang sudah berjalan tidak terpengaruh |

### 6.3 Pre-Event Checklist (Rabu 23/9 Malam)

- [ ] `pg_dump` manual setelah seeding final
- [ ] Verifikasi `/api/healthz` mengembalikan `{"status": "ok"}`
- [ ] Uji beban k6 lulus (75 pemain, p95 < 300ms, 0 error)
- [ ] Uji E2E Playwright lulus di kedua bahasa
- [ ] Screenshot leaderboard dan podium tersimpan sebagai bukti
- [ ] Semua akun host telah di-seed dan bisa login
- [ ] DNS A record mengarah ke IP VPS yang benar
- [ ] Sertifikat TLS valid (Caddy auto-renew; cek via browser)
- [ ] Poster QR room siap cetak

### 6.4 During-Event Runbook

```bash
# Cek status container
docker compose ps

# Lihat log real-time
docker compose logs -f api

# Restart Go API jika ada issue (room state rebuild dari PG)
docker compose restart api

# Cek PostgreSQL
docker compose exec postgres psql -U $POSTGRES_USER -c "\l"

# Cek Redis
docker compose exec redis redis-cli ping

# Lihat active rooms
docker compose exec redis redis-cli SMEMBERS rooms:active

# Lihat leaderboard room tertentu
docker compose exec redis redis-cli ZREVRANGE room:{id}:lb 0 9 WITHSCORES
```

### 6.5 Incident Response

| Severity | Definisi | Eskalasi |
|---|---|---|
| **P1** | Semua room tidak bisa diakses | Tim BE on-site; restart container + investigasi log |
| **P2** | Satu room bermasalah; yang lain normal | Host: akhiri room dan buat room baru |
| **P3** | Bug minor (visual, non-blocking) | Catat; hotfix setelah acara |

---

## 7. Monitoring Plan

| Tool | Apa yang Dipantau | Notifikasi |
|---|---|---|
| `docker compose logs -f api` | Error log, latency warning | Manual (tim on-site) |
| `/api/healthz` | Uptime | Uptime Kuma (opsional; ping tiap 30 detik) |
| `docker stats` | CPU, memory, network per container | Manual |
| PostgreSQL `pg_stat_activity` | Query yang lambat / deadlock | Manual jika ada lag |

> **Post-launch**: Pertimbangkan Prometheus + Grafana atau Datadog untuk monitoring production yang lebih baik.
