# 07 — Risk & Trade-off
## Jejak Inderasakti: Jelajah Pulau Penyengat

> **Dokumen**: Risk Register · ADR Summary · RFC  
> **Versi**: 1.0 · **Tanggal**: 2026-09-19

---

## 1. Risk Register

### Matriks Probabilitas × Dampak

```
Dampak →    Rendah        Sedang          Tinggi         Kritis
Prob ↓
Tinggi    |           | R5: Sinyal    | R1: Konten  |              |
          |           | lemah peserta | soal salah  |              |
Sedang    | R8: Layout| R6: Browser   | R2: Deploy  | R9: Data     |
          | HP lama   | lama          | gagal       | anak bocor   |
Rendah    | R7: Fonts | R3: EN review | R4: VPS     |              |
          | gagal load| belum selesai | crash       |              |
```

### Detail Risk Register

| ID | Risiko | Prob | Dampak | Skor | Pemilik | Mitigasi | Contingency |
|---|---|---|---|---|---|---|---|
| **R1** | Bank soal mengandung konten historis yang salah (belum divalidasi juru pelihara/Yayasan) | Tinggi | Tinggi | 🔴 9 | Panitia | Tanyakan validator sebelum Selasa; flag soal tidak pasti sebagai `review_flag` | Nonaktifkan soal bermasalah via `active = FALSE` tanpa deploy ulang |
| **R2** | Deploy ke VPS gagal atau tidak selesai sebelum Kamis | Sedang | Tinggi | 🟠 6 | BE Dev | Staging di VPS Selasa; CI/CD pipeline; fallback: Sesi Singkat (10 soal), animasi statis | Buat room di staging sebagai backup produksi dadakan |
| **R3** | Review terjemahan Inggris tidak selesai sebelum Rabu | Rendah | Sedang | 🟡 3 | Panitia | Tetapkan reviewer terjemahan Selasa; 60 soal EN sudah ada di tab Question Bank | Disable language toggle EN sementara; hanya mode ID |
| **R4** | VPS down saat acara (hardware failure, DDoS, dll.) | Rendah | Tinggi | 🟠 4 | BE Dev | Provider SLA ≥ 99.9%; Uptime Kuma monitor; tim on-call | Gunakan hotspot HP sebagai akses SSH; restart container dari backup |
| **R5** | Sinyal internet lemah di Pulau Penyengat atau sekolah | Tinggi | Sedang | 🟠 6 | Panitia | Desain tempo per peserta; reconnect otomatis; minimal data per WS message | Sediakan WiFi lokal; minta panitia sewa mifi |
| **R6** | Browser lama HP peserta tidak support WebSocket / ES2020 | Sedang | Sedang | 🟡 4 | FE Dev | Target Chrome ≥ 90, Safari ≥ 14; polyfill minimal; test di HP Android murah Minggu | Instruksikan peserta update Chrome sebelum acara |
| **R7** | Google Fonts gagal load karena koneksi lambat (Baloo 2, Nunito) | Rendah | Rendah | 🟢 1 | FE Dev | Self-host font di `/assets/fonts/`; font-display: swap | Fallback ke system sans-serif (teks tetap terbaca) |
| **R8** | Layout pecah di HP kecil (< 360px) atau HP lama resolusi rendah | Sedang | Rendah | 🟡 2 | FE Dev | Test di 3 HP murah Sabtu (min. 1 HP 360×640); breakpoint 360px | CSS min-height + overflow-y scroll sebagai fallback |
| **R9** | Data anak bocor akibat SQL injection atau endpoint tidak terproteksi | Rendah | Kritis | 🟠 5 | BE Dev | sqlc (parameterized query); middleware JWT; validasi input semua field | Incident response: isolate container, pg_dump forensic, notifikasi panitia |
| **R10** | Scope creep: stakeholder minta fitur baru H-1 atau H-0 | Sedang | Tinggi | 🟠 6 | PM/Panitia | Scope freeze resmi setelah dokumen ini disetujui; semua request masuk backlog post-launch | Prioritas: 0 fitur baru setelah Rab 23/9 18:00 |

### Prioritas Aksi Sebelum Kamis

| Prioritas | Aksi | Deadline | PIC |
|---|---|---|---|
| 🔴 **P0** | Konfirmasi validator bank soal | Sab 19/9 | Panitia |
| 🔴 **P0** | Konfirmasi reviewer terjemahan EN | Sab 19/9 | Panitia |
| 🟠 **P1** | Test di 3 HP Android murah + uji koneksi lemah | Min 20/9 | FE Dev |
| 🟠 **P1** | Deploy staging di VPS dan uji E2E | Sel 22/9 | BE Dev |
| 🟠 **P1** | k6 load test lulus (75 pemain, 0 error) | Rab 23/9 | BE Dev |
| 🟡 **P2** | Self-host Google Fonts | Min 20/9 | FE Dev |
| 🟡 **P2** | Scope freeze: tidak ada fitur baru setelah Rab 18:00 | Rab 23/9 | Semua |

---

## 2. ADR Summary

*(Detail lengkap di dokumen 03 — Arsitektur Sistem)*

| ADR | Keputusan | Alasan Utama | Trade-off |
|---|---|---|---|
| **ADR-001** | Go + net/http standar (tanpa framework) | Sedikit dependensi, mudah dibantu AI | Butuh helper manual untuk routing dan error |
| **ADR-002** | Tempo per peserta (bukan serentak seperti Kahoot) | Toleran sinyal lemah; tidak perlu sinkronisasi jam | Tidak bisa tampilkan "siapa yang paling cepat saat ini" |
| **ADR-003** | Satu goroutine per room (channel-based) | Tidak ada data race; mudah ditest -race | Goroutine leak jika cleanup gagal |
| **ADR-004** | PostgreSQL sumber kebenaran + Redis cache | Durable; state rebuild otomatis setelah crash | Setiap jawaban butuh write ke PG + Redis |
| **ADR-005** | Docker Compose (bukan Kubernetes) | Deploy 1 perintah; mudah dipahami semua tim | Tidak ada auto-scaling, rolling update |
| **ADR-006** | Scoring di server; kunci jawaban tidak dikirim | Tidak bisa cheat via inspect | Latency 1 round-trip sebelum feedback |
| **ADR-007** | Bilingual via JSONB (bukan tabel terpisah) | Query sederhana; satu row per soal | JSONB validasi lebih lemah dari relasi |

---

## 3. Trade-off Analysis

### 3.1 Tempo Per Peserta vs. Serentak (Kahoot Mode)

| Dimensi | Tempo Per Peserta (dipilih) | Serentak |
|---|---|---|
| **Fairness koneksi lemah** | ✅ Sinyal lemah tidak langsung tertinggal | ❌ Sinyal 0.5 detik lebih lambat = kerugian besar |
| **Ketegangan drama** | ⚠️ Lebih rendah (tidak ada "semua jawab sekarang!") | ✅ Tinggi; drama real-time |
| **Kompleksitas backend** | ✅ Server tidak perlu sinkronisasi jam | ❌ Butuh NTP atau timestamp terpusat |
| **Estimasi waktu implementasi** | ✅ Hemat ~1 hari BE | — |
| **Cocok untuk 4G** | ✅ | ❌ |

**Keputusan**: Tempo per peserta dipilih karena mayoritas peserta menggunakan HP di 4G/sinyal lemah. Drama leaderboard tetap ada via update real-time di akhir setiap stage.

### 3.2 Satu Binary Go vs. Microservice

| Dimensi | Monolith Go (dipilih) | Microservice |
|---|---|---|
| **Kompleksitas deploy** | ✅ 1 container | ❌ Butuh orchestrator |
| **Latency internal** | ✅ In-process function call | ❌ Network hop |
| **Debugging** | ✅ Satu log, satu trace | ❌ Distributed tracing |
| **Skalabilitas komponen** | ⚠️ Skala semua atau tidak ada | ✅ Skala per komponen |
| **Cocok untuk 75 user** | ✅ | ❌ Overkill |

**Keputusan**: Monolith untuk versi Kamis. Pisahkan hanya jika ada bottleneck terbukti post-launch.

### 3.3 PostgreSQL vs. Firestore / Supabase

| Dimensi | PostgreSQL + Docker (dipilih) | Firestore/Supabase |
|---|---|---|
| **Setup** | ⚠️ Perlu konfigurasi | ✅ Managed |
| **Latensi dari VPS Singapura** | ✅ Lokal (sub-ms) | ⚠️ Jaringan (10–50ms) |
| **Kontrol penuh** | ✅ | ⚠️ Bergantung vendor |
| **Biaya** | ✅ Termasuk dalam VPS | ⚠️ Bergantung usage |
| **Keamanan data anak** | ✅ Data di VPS sendiri | ⚠️ Data di cloud vendor asing |

**Keputusan**: PostgreSQL self-hosted karena data anak tidak boleh di vendor cloud tanpa audit kepatuhan PDP.

### 3.4 Leaderboard Redis Sorted Set vs. PostgreSQL Query

| Dimensi | Redis Sorted Set (dipilih) | PostgreSQL Real-time Query |
|---|---|---|
| **Performa update** | ✅ O(log n) ZINCRBY | ⚠️ UPDATE + SELECT per jawaban |
| **Performa baca** | ✅ ZREVRANGE O(n) | ⚠️ ORDER BY skor; butuh index |
| **Durabilitas** | ⚠️ Volatile (rebuild dari PG jika crash) | ✅ Durable |
| **Kompleksitas** | ⚠️ 2 sistem | ✅ 1 sistem |

**Keputusan**: Redis untuk leaderboard real-time; PG sebagai source of truth. Trade-off diterima karena rebuild cepat (query sederhana) dan Redis jarang crash.

---

## 4. RFC (Request for Comments)

### RFC-001: Apakah Perlu WebSocket untuk Leaderboard Host atau Cukup Polling?

**Konteks**: Layar host menampilkan leaderboard real-time. Opsi:
1. **WebSocket** (diusulkan): Server push `lb.update` ke host via WS yang sudah terbuka.
2. **SSE (Server-Sent Events)**: Stream satu arah dari server.
3. **Polling** setiap 2 detik: Sederhana, tidak perlu WS.

**Analisis**:
| Opsi | Pro | Kontra |
|---|---|---|
| WebSocket | Satu koneksi untuk semua event | Host juga butuh kirim host.start, host.end |
| SSE | Sederhana untuk push saja | Butuh koneksi terpisah untuk aksi host |
| Polling | Paling sederhana | Update 2 detik (lag); overhead per request |

**Rekomendasi**: WebSocket — karena host juga butuh kirim perintah (start, end, kick), wajar pakai satu WS bidirectional. SSE tidak cocok karena tidak bidirectional.

**Status**: ✅ Disetujui — WebSocket untuk host dan peserta.

---

### RFC-002: Apakah Mode Akurasi Default ON untuk Room SD?

**Konteks**: Mode Akurasi mematikan faktor kecepatan; setiap jawaban benar = poin dasar B. Saat ini mode ini opsional dan di-toggle host.

**Pertanyaan**: Apakah mode akurasi harus default ON untuk jenjang SD, atau tetap opsional?

**Opsi**:
1. Default ON untuk SD, default OFF untuk SMP/SMA
2. Tetap opsional (host yang menentukan)
3. Wajib ON untuk SD (tidak bisa diubah)

**Implikasi**:
- Opsi 1: Lebih ramah SD secara default; host yang tahu bisa matikan jika mau kompetisi kecepatan
- Opsi 2: Fleksibel tapi host perlu tahu fitur ini
- Opsi 3: Tidak ada kompetisi kecepatan sama sekali untuk SD

**Rekomendasi**: Opsi 1 — default ON untuk SD, tetapi host bisa override. Dokumentasikan di tooltip Buat Room.

**Status**: 🟡 Menunggu keputusan panitia.

---

### RFC-003: Kapan `review_flag` di Bank Soal Ditinjau?

**Konteks**: Server akan otomatis menandai soal yang dijawab salah oleh >80% peserta dengan `review_flag = TRUE`. Pertanyaan: siapa yang melakukan review, dan kapan?

**Skenario**:
- Jika 12+ dari 15 peserta di room pertama menjawab salah soal tertentu, soal itu di-flag
- Panitia bisa lihat via query DB atau halaman admin (post-launch)
- Soal tetap aktif selama acara; tidak ada auto-deaktivasi

**Rekomendasi**:
1. Selama acara Kamis: biarkan soal berjalan, catat flag-nya
2. Setelah acara: panitia/validator review soal yang di-flag dalam 1 minggu
3. Perbarui `questions.json` dan re-seed untuk acara berikutnya

**Status**: ✅ Disepakati. Query untuk melihat soal flag:
```sql
SELECT id, site, level, prompt->>'id' AS pertanyaan
FROM questions
WHERE review_flag = TRUE
ORDER BY site, level;
```
