# 09 — Testing Plan
## Jejak Inderasakti: Jelajah Pulau Penyengat

> **Dokumen**: Test Plan · Test Case Matrix · Traceability Matrix (Requirement → Test)  
> **Versi**: 1.0 · **Tanggal**: 2026-09-19

---

## 1. Testing Strategy

### 1.1 Piramida Testing

```
              /--------\
             /  Manual  \   ← Uji pengguna nyata (Kamis 24/9)
            /   Testing  \
           /--------------\
          /   Playwright   \  ← E2E: daftar → 15 soal → podium
         /    (E2E Tests)   \
        /--------------------\
       /    k6 Load Testing   \  ← 5 room × 15 pemain
      /------------------------\
     /     Go Unit Tests        \  ← Scoring, selector, room state
    /  (go test -race ./...)     \
   /------------------------------\
  /   Static Analysis & Linting    \  ← golangci-lint, pnpm lint, tsc
 /------------------------------------\
```

### 1.2 Testing Scope per Layer

| Layer | Tool | Scope | Jadwal | Lulus Jika |
|---|---|---|---|---|
| **Static Analysis** | golangci-lint, pnpm lint, tsc | Semua kode | Per commit (CI) | 0 error |
| **Unit Test** | `go test -race` | Scoring, selector, room goroutine | Per commit (CI) | 0 failure, 0 race |
| **Integration Test** | `go test` dengan PG+Redis test container | REST API handlers | Per PR | 0 failure |
| **E2E Test** | Playwright | Full user flow, kedua bahasa | Selasa 22/9 + Rabu 23/9 | 0 failure |
| **Load Test** | k6 | 5 room × 15 pemain, 15 soal | Rabu 23/9 | p95 < 300ms, 0 error |
| **Manual / UAT** | Manusia | UI/UX, aksesibilitas, HP nyata | Selasa 22/9 + Kamis 24/9 | Sesuai acceptance criteria |

---

## 2. Unit Test Cases

### 2.1 Scoring (game/scoring_test.go)

| TC ID | Nama | Input | Expected | Status |
|---|---|---|---|---|
| UT-SC-01 | Soal sedang, dalam masa baca (t=1s) | L=2, t=1000ms, T=20000ms, g=2000ms | 750 | 🟡 Pending |
| UT-SC-02 | Soal sedang, cepat (t=4s) | L=2, t=4000ms, T=20000ms, g=2000ms | 717 | 🟡 Pending |
| UT-SC-03 | Soal sedang, sedang (t=12s) | L=2, t=12000ms, T=20000ms, g=2000ms | 583 | 🟡 Pending |
| UT-SC-04 | Soal sedang, detik terakhir (t=19s) | L=2, t=19000ms, T=20000ms, g=2000ms | 467 | 🟡 Pending |
| UT-SC-05 | Jawaban salah | L=2, Correct=false | 0 | 🟡 Pending |
| UT-SC-06 | Lewat deadline + toleransi | L=2, t=21100ms, T=20000ms | 0 | 🟡 Pending |
| UT-SC-07 | Soal mudah, dalam masa baca | L=1, t=500ms, T=15000ms, g=2000ms | 500 | 🟡 Pending |
| UT-SC-08 | Soal sulit, dalam masa baca | L=3, t=500ms, T=25000ms, g=2000ms | 1000 | 🟡 Pending |
| UT-SC-09 | Mode akurasi ON | L=2, AccuracyMode=true, t=10000ms | 750 | 🟡 Pending |
| UT-SC-10 | Bonus streak ke-2 | StreakBefore=1 | poin + 50 | 🟡 Pending |
| UT-SC-11 | Bonus streak ke-3 | StreakBefore=2 | poin + 100 | 🟡 Pending |
| UT-SC-12 | Bonus streak maks. (StreakBefore=5+) | StreakBefore=6 | poin + 250 (capped) | 🟡 Pending |
| UT-SC-13 | Timer SD dikali 1.25 | L=2, T=25000ms (20×1.25), g=3000ms | sesuai formula | 🟡 Pending |
| UT-SC-14 | Masa baca SD = 3 detik | L=2, t=2500ms, g=3000ms, T=25000ms | speed=1.0 (masih grace) | 🟡 Pending |

### 2.2 Question Selector (game/selector_test.go)

| TC ID | Nama | Kondisi | Expected |
|---|---|---|---|
| UT-QS-01 | SD Sesi Normal | jenjang=SD, short=false | 15 soal: stage 1=[M,M,S], ..., stage 5=[M,S,Su] |
| UT-QS-02 | SMP Sesi Normal | jenjang=SMP, short=false | 15 soal sesuai komposisi SMP |
| UT-QS-03 | SMA Sesi Normal | jenjang=SMA, short=false | 15 soal sesuai komposisi SMA |
| UT-QS-04 | SD Sesi Singkat | jenjang=SD, short=true | 10 soal (soal pertama+terakhir per sel) |
| UT-QS-05 | Soal berbeda antar room | Buat 2 room SD | question_ids berbeda (probabilistik) |
| UT-QS-06 | Urutan opsi diacak | Soal yang sama, 2 pemain | opsi A di pemain 1 bisa B di pemain 2 |
| UT-QS-07 | Tidak ada soal inactive | active=FALSE di beberapa soal | Selector tidak pernah pilih soal inactive |

### 2.3 Room Goroutine (game/room_test.go)

| TC ID | Nama | Skenario | Expected |
|---|---|---|---|
| UT-RM-01 | `q.next` sebelum host.start ditolak | Peserta kirim q.next saat status=lobby | Error: room belum mulai |
| UT-RM-02 | `q.answer` ganda ditolak | Peserta jawab soal yang sama 2×  | Kedua: tolak; jawaban pertama tersimpan |
| UT-RM-03 | `q.answer` setelah deadline ditolak | answered_at > served_at + limit_ms + 1000ms | Poin = 0 |
| UT-RM-04 | Soal tidak dijawab → 0 poin | Peserta kirim q.next tanpa jawab | Soal lama dicatat 0; soal baru dikirim |
| UT-RM-05 | Room auto-end setelah 12 menit | Timer 12 menit | `room.ended` dikirim ke semua |
| UT-RM-06 | Room end saat semua selesai | Semua peserta current_index = len(questions) | `room.ended` dikirim |
| UT-RM-07 | host.end paksa | Host kirim host.end | `room.ended` dikirim |
| UT-RM-08 | Race condition: 15 peserta jawab bersamaan | -race flag | Tidak ada data race |
| UT-RM-09 | Reconnect: token valid, room running | Peserta reconnect | `room.state` + soal aktif dikirim |
| UT-RM-10 | Reconnect: soal sudah lewat deadline | Reconnect setelah soal ke-7 timeout | Soal ke-7 dicatat 0; soal ke-8 disiapkan |

---

## 3. Integration Test Cases (REST API)

| TC ID | Endpoint | Kondisi | Expected Status | Expected Body |
|---|---|---|---|---|
| IT-01 | POST /api/auth/login | Email + password valid | 200 | `{token}` |
| IT-02 | POST /api/auth/login | Password salah | 401 | `{error: UNAUTHORIZED}` |
| IT-03 | POST /api/rooms | Host valid, < 5 room aktif | 201 | `{id, pin, qr_url}` |
| IT-04 | POST /api/rooms | Sudah ada 5 room aktif | 409 | `{error: MAX_ROOMS}` |
| IT-05 | GET /api/rooms/{pin} | PIN valid, room lobby | 200 | `{status: lobby, slots_left: 15}` |
| IT-06 | GET /api/rooms/{pin} | PIN tidak ada | 404 | `{error: ROOM_NOT_FOUND}` |
| IT-07 | POST /api/rooms/{pin}/join | Data valid, ada slot | 200 | `{player_token}` |
| IT-08 | POST /api/rooms/{pin}/join | Room penuh (15 peserta) | 409 | `{error: ROOM_FULL}` |
| IT-09 | POST /api/rooms/{pin}/join | Nama sudah dipakai | 409 | `{error: NICKNAME_TAKEN}` |
| IT-10 | POST /api/rooms/{pin}/join | Nama mengandung kata kasar | 400 | `{error: INVALID_NICKNAME}` |
| IT-11 | GET /api/schools?q=SDN | Query valid | 200 | Array sekolah |
| IT-12 | GET /api/schools?q=Umum | Query "Umum" | 200 | Termasuk "Umum / General visitor" |
| IT-13 | GET /api/rooms/{id}/results.csv | Host JWT valid, room ended | 200 | CSV dengan header dan data |
| IT-14 | GET /api/rooms/{id}/results.csv | Bukan host room | 403 | `{error: FORBIDDEN}` |
| IT-15 | GET /api/leaderboard/schools | — | 200 | Array sekolah dengan avg_score |
| IT-16 | GET /api/healthz | — | 200 | `{status: ok}` |
| IT-17 | Rate limit join | > 10 join/menit dari IP yang sama | 429 | `{error: RATE_LIMITED}` |

---

## 4. E2E Test Cases (Playwright)

### 4.1 Viewport: Mobile (375×812) dan iPhone SE (375×667)

| TC ID | Skenario | Langkah | Expected |
|---|---|---|---|
| E2E-01 | **Full flow ID**: Daftar → 15 soal → Podium | P1→P2→P3→P4→P5→P6→P7(×15)→P8(×15)→P9(×5)→P10 | Semua layar tampil; podium muncul dengan confetti |
| E2E-02 | **Full flow EN**: Same as E2E-01 | Pilih English di P1 | Semua teks dalam Bahasa Inggris; konten soal EN |
| E2E-03 | **PIN salah**: Masukkan PIN yang tidak ada | Masukkan PIN "000000" | Pesan error jelas: "PIN tidak ditemukan" |
| E2E-04 | **Room penuh**: Coba join saat 15 peserta sudah ada | 15 peserta join terlebih dahulu | Pesan: "Room sudah penuh" |
| E2E-05 | **Nama duplikat**: Nama yang sudah dipakai | Peserta 2 pakai nama Peserta 1 | Pesan: "Nama sudah dipakai di room ini" |
| E2E-06 | **Timer habis**: Tidak jawab sampai timer = 0 | Biarkan timer habis | Soal berikutnya muncul; poin tidak bertambah |
| E2E-07 | **Jawaban benar**: Animasi dan poin | Jawab dengan opsi benar | Tile hijau; "+poin" terbang; Sakti bersorak |
| E2E-08 | **Jawaban salah**: Feedback | Jawab dengan opsi salah | Tile merah + bergetar; opsi benar di-highlight |
| E2E-09 | **Reconnect**: Putus di soal ke-7 | Reload halaman saat soal ke-7 | Kembali ke soal ke-7 (jika belum timeout) |
| E2E-10 | **Host flow**: Login → Buat room → Mulai → Monitor → Akhiri → CSV | Viewport 1280×720 | Semua fitur host berfungsi; CSV ter-download |
| E2E-11 | **Aksesibilitas**: Tidak hanya warna | Jawab benar/salah | Ada ikon centang/silang + motif + huruf A-D |
| E2E-12 | **Kartu Info skip**: Lewati kartu info | Klik tombol Lanjut saat P6 | Langsung ke soal pertama stage |
| E2E-13 | **Language toggle**: Ganti bahasa di P2-P4 | Toggle EN di P2, lalu ID lagi | UI berubah bahasa; PIN input tetap berfungsi |

### 4.2 Konfigurasi Playwright

```typescript
// playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
    testDir: './e2e',
    timeout: 60_000,
    retries: 1,
    use: {
        baseURL: process.env.BASE_URL || 'http://localhost:5173',
        trace: 'on-first-retry',
        video: 'on-first-retry',
    },
    projects: [
        {
            name: 'Mobile Chrome (375x812)',
            use: { ...devices['iPhone 12'] },
        },
        {
            name: 'Mobile Android (360x800)',
            use: {
                viewport: { width: 360, height: 800 },
                userAgent: 'Mozilla/5.0 (Linux; Android 10; SM-A505F) ...',
            },
        },
        {
            name: 'Host Desktop (1280x720)',
            use: { viewport: { width: 1280, height: 720 } },
        },
    ],
});
```

---

## 5. Load Test (k6)

### 5.1 Script Utama

```js
// loadtest/room.js
import ws from 'k6/ws';
import http from 'k6/http';
import { sleep, check } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const ackLatency = new Trend('ack_latency_ms');
const errorRate = new Rate('errors');

export let options = {
    scenarios: {
        five_rooms: {
            executor: 'constant-vus',
            vus: 80,        // 75 peserta + 5 host
            duration: '15m', // lebih dari durasi sesi maksimum
        },
    },
    thresholds: {
        'ack_latency_ms': ['p(95)<300'],
        'errors': ['rate<0.001'],           // < 0.1% error
        'ws_connecting': ['p(95)<1000'],    // koneksi WS < 1 detik
    },
};

export default function() {
    // 1. Join room
    const joinRes = http.post(`${BASE_URL}/api/rooms/${PIN}/join`, JSON.stringify({
        nickname: `Player_${__VU}`,
        school_id: 1,
        jenjang: 'SMP',
        avatar: 1,
        lang: 'id',
    }), { headers: { 'Content-Type': 'application/json' } });

    check(joinRes, { 'join status 200': (r) => r.status === 200 });
    const { player_token } = joinRes.json();

    // 2. Connect WebSocket
    const url = `wss://${HOST}/ws?token=${player_token}`;
    ws.connect(url, {}, function(socket) {
        socket.on('message', (data) => {
            const msg = JSON.parse(data);

            if (msg.t === 'q.show') {
                const thinkTime = Math.random() * 15000 + 1000; // 1–16 detik
                sleep(thinkTime / 1000);

                const sentAt = Date.now();
                socket.send(JSON.stringify({
                    t: 'q.answer',
                    d: {
                        question_id: msg.d.question_id,
                        option_id: 'opt-a', // simulasi memilih opsi A
                    }
                }));
            }

            if (msg.t === 'q.result') {
                // Track ACK latency (server side, bukan round-trip)
                // Server menyertakan processing_ms di d untuk tracking
                ackLatency.add(msg.d.processing_ms || 0);
            }

            if (msg.t === 'room.ended') {
                socket.close();
            }
        });

        socket.on('error', () => {
            errorRate.add(1);
        });

        // Setelah join, kirim q.next untuk soal pertama
        socket.on('open', () => {
            sleep(1); // tunggu host.start
            socket.send(JSON.stringify({ t: 'q.next', d: {} }));
        });
    });
}
```

### 5.2 Threshold Kelulusan

| Metrik | Threshold | Status |
|---|---|---|
| ACK latency p95 | < 300ms | 🟡 Akan diuji Rab 23/9 |
| Error rate | < 0.1% | 🟡 Akan diuji Rab 23/9 |
| WS connect p95 | < 1000ms | 🟡 Akan diuji Rab 23/9 |
| 0 data race | `go test -race` | 🟡 Akan diuji Sab 19/9 |

---

## 6. Traceability Matrix (Requirement → Test)

| Requirement ID | Requirement (Ringkasan) | Unit Test | Integration Test | E2E Test | Load Test |
|---|---|---|---|---|---|
| FR-01.1 | Host login email + password | — | IT-01, IT-02 | E2E-10 | — |
| FR-02.1 | Host buat room | — | IT-03 | E2E-10 | — |
| FR-02.3 | Tolak room ke-6 | — | IT-04 | — | k6 (max rooms) |
| FR-03.1 | Peserta masuk via PIN | — | IT-05, IT-06 | E2E-01, E2E-03 | k6 (75 join) |
| FR-03.4 | Nama unik per room | — | IT-09 | E2E-05 | — |
| FR-03.5 | Tolak peserta ke-16 | — | IT-08 | E2E-04 | k6 (max players) |
| FR-04.4 | Server kirim soal tanpa kunci | UT-RM-01 | — | E2E-01 | — |
| FR-04.5 | Validasi deadline + toleransi | UT-RM-03, UT-SC-06 | — | E2E-06 | — |
| FR-04.6 | Soal tidak dijawab = 0 | UT-RM-04 | — | E2E-06 | — |
| FR-05.1–5.6 | Sistem penilaian | UT-SC-01 s/d UT-SC-14 | — | E2E-07, E2E-08 | — |
| FR-06.1 | Leaderboard real-time host | UT-RM-08 | — | E2E-10 | k6 (lb.update) |
| FR-07.1 | Reconnect dengan PIN sama | UT-RM-09, UT-RM-10 | — | E2E-09 | — |
| FR-08.1 | Bilingual per pemain | — | — | E2E-01, E2E-02, E2E-13 | — |
| FR-09.1 | CSV ekspor | — | IT-13, IT-14 | E2E-10 | — |
| NFR-Perf | ACK p95 < 300ms | — | — | — | k6 threshold |
| NFR-Security | Kunci jawaban tidak di client | UT-RM-01 | — | — | — |
| NFR-Security | Rate limit join | — | IT-17 | — | k6 |
| NFR-Reliability | Double submit dicegah | UT-RM-02 | — | — | k6 |
| NFR-Accessibility | Non-color indicators | — | — | E2E-11 | — |

---

## 7. Manual Testing Checklist

### 7.1 Visual & UX (Selasa 22/9 — Di HP Nyata)

- [ ] Palet warna terlihat jelas di HP Android murah di dalam ruangan
- [ ] Palet warna terbaca di bawah sinar matahari langsung
- [ ] Tombol jawaban 64px — bisa ditekan dengan jempol tanpa misfire
- [ ] Timer bar berubah merah dan berdenyut saat < 5 detik
- [ ] "+poin" terbang animasi terlihat di layar 360px
- [ ] Maskot Sakti terlihat dan ekspresinya terbaca di layar kecil
- [ ] Kartu Info dapat di-skip dengan mengetuk layar
- [ ] Podium confetti berjalan lancar (tidak lag) di Android murah
- [ ] Leaderboard row sendiri di-highlight kuning dengan jelas
- [ ] Teks soal 20px terbaca tanpa pinch-zoom
- [ ] Toggle bahasa EN → ID → EN berfungsi dari P2–P4

### 7.2 Edge Cases Manual

- [ ] Buka 2 tab browser dengan akun peserta yang sama → token yang lebih baru menggantikan yang lama
- [ ] Host me-kick peserta saat sedang menjawab soal → peserta melihat pesan kicked
- [ ] Matikan WiFi saat di P7 (soal), hidupkan kembali → reconnect berjalan
- [ ] Browser di-minimize saat timer berjalan → timer tetap berjalan di server; saat buka kembali soal sudah timeout
- [ ] Peserta join saat room sudah status `running` → error "Room sudah dimulai"

### 7.3 Bilingual Manual

- [ ] Semua label UI dalam Bahasa Inggris saat pilih EN (tidak ada teks Indonesia yang tertinggal)
- [ ] Konten soal (prompt, opsi, penjelasan) dalam Bahasa Inggris saat pemain EN
- [ ] Nama diri (Masjid Raya, Gurindam Dua Belas) tetap dalam Bahasa Melayu di versi EN
- [ ] Layout tidak pecah untuk teks Indonesia (biasanya lebih panjang dari EN)
- [ ] Satu room dengan campuran pemain ID dan EN berjalan normal

---

## 8. Bug Severity & Triage

| Severity | Definisi | Target Fix |
|---|---|---|
| **P0 Critical** | Game tidak bisa dimulai; skor salah hitung; data hilang | Fix sebelum deploy; hotfix langsung |
| **P1 High** | Fitur utama tidak berfungsi (leaderboard tidak update, reconnect gagal) | Fix sebelum acara Kamis |
| **P2 Medium** | Fitur non-kritikal bermasalah (animasi lag, CSV tidak tepat) | Fix jika waktu cukup; otherwise backlog |
| **P3 Low** | Visual minor, teks sedikit salah, layout kurang perfect | Backlog post-launch |

### Bug Reporting Template

```
## Bug Report

**ID**: BUG-XXX
**Severity**: P0/P1/P2/P3
**Tanggal**: YYYY-MM-DD
**Reporter**: Nama

### Deskripsi
[Apa yang terjadi]

### Langkah Reproduksi
1. 
2.

### Expected
[Apa yang seharusnya terjadi]

### Actual
[Apa yang benar-benar terjadi]

### Environment
- Browser:
- HP model:
- OS:
- Room PIN (jika relevan):

### Screenshot / Log
[Attach di sini]
```
