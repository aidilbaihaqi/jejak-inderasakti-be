# Jejak Inderasakti — Master Documentation Index
## Jelajah Pulau Penyengat: Game Kuis Edukasi Multiplayer Real-time

> **Dibuat dari**: 3 dokumen sumber (About Game · Architecture Plan · Design Document)  
> **Tanggal**: 2026-09-19 · **Target Launch**: Kamis, 24 September 2026

---

## Ringkasan Proyek

**Jejak Inderasakti** adalah game kuis web mobile-first multiplayer real-time tentang 5 situs cagar budaya Pulau Penyengat, Kepulauan Riau. Satu sesi berlangsung 5–10 menit (5 situs × 3 soal), maks. 15 peserta per room, maks. 5 room bersamaan (75 pemain), bilingual ID/EN.

**Target**: Deploy dan diuji dengan pengguna nyata pada **Kamis 24/9/2026**.

---

## Daftar Dokumen

| # | Dokumen | Isi Utama | Link |
|---|---|---|---|
| 01 | **PRD & BRD** | Problem statement, scope/MVP, stakeholder, user persona, use case, user story, FR/NFR, acceptance criteria | [01_PRD_BRD.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/01_PRD_BRD.md) |
| 02 | **UI/UX Design** | User flow, wireframe 15 layar, design system, komponen, motion, ilustrasi brief, usability testing | [02_UI_UX_Design.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/02_UI_UX_Design.md) |
| 03 | **Arsitektur Sistem** | C4 model, sequence diagram, tech stack, API contract, WebSocket spec, 6 ADR | [03_System_Architecture.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/03_System_Architecture.md) |
| 04 | **Data Modelling** | ERD, DDL schema 6 tabel + Redis, data dictionary, selector logic, migration plan, governance & retention | [04_Data_Modelling.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/04_Data_Modelling.md) |
| 05 | **NFR, Security & Reliability** | NFR lengkap, STRIDE threat model, security checklist, capacity planning, SLA/SLO/SLI, DR & backup | [05_NFR_Security.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/05_NFR_Security.md) |
| 06 | **Infrastructure & Deployment** | Topology, Docker Compose, Caddyfile, Dockerfile, CI/CD pipeline, environment matrix, pre-deploy checklist | [06_Infrastructure_Deployment.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/06_Infrastructure_Deployment.md) |
| 07 | **Risk & Trade-off** | Risk register 10 risiko, ADR summary, trade-off analysis 4 keputusan, 3 RFC | [07_Risk_Tradeoff.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/07_Risk_Tradeoff.md) |
| 08 | **PoC Report** | 5 spike report: scoring formula, 75 WS concurrent, PG concurrent write, WS client state, JSONB bilingual | [08_PoC_Report.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/08_PoC_Report.md) |
| 09 | **Testing Plan** | 14 unit TC, 17 integration TC, 13 E2E TC, k6 load test, traceability matrix, bug triage | [09_Testing_Plan.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/09_Testing_Plan.md) |
| 10 | **Team & Workflow** | RACI matrix, Definition of Done, coding standard, branching strategy, 6-day Gantt roadmap | [10_Team_Workflow.md](file:///C:/Users/MSI%20MODEREN%2015/.gemini/antigravity-ide/brain/82de2c25-3b82-4ebb-b6e0-96df095f5958/10_Team_Workflow.md) |

---

## Keputusan Arsitektur Kritis

| ADR | Keputusan | Alasan |
|---|---|---|
| ADR-001 | Go + `net/http` standar (no framework) | Minimal dependensi; mudah dibantu AI |
| ADR-002 | **Tempo per peserta** (bukan serentak) | Toleran sinyal lemah; hemat 1 hari BE |
| ADR-003 | Satu goroutine per room (channel) | Zero data race; `go test -race` wajib lulus |
| ADR-004 | PostgreSQL sumber kebenaran + Redis cache | Durable; leaderboard O(log n) |
| ADR-005 | Docker Compose (bukan Kubernetes) | Deploy satu perintah; cukup untuk 75 user |
| ADR-006 | Scoring di server; kunci jawaban tidak di client | Anti-cheat fundamental |

---

## Open Questions (Perlu Dijawab Segera)

> **Deadline**: Sabtu 19/9 agar tidak memblokir development

| # | Pertanyaan | Dampak Jika Tidak Dijawab |
|---|---|---|
| **OQ-01** | 🔴 Siapa yang memvalidasi bank soal sebelum Kamis? (juru pelihara / Yayasan Indrasakti / guru sejarah) | Soal belum tervalidasi → risiko konten salah saat acara |
| **OQ-02** | 🔴 Siapa yang mereview terjemahan Bahasa Inggris? | Versi EN tidak akurat → matikan toggle EN |
| **OQ-03** | 🟠 Berapa akun host yang dibutuhkan, dan untuk siapa? | Seed `host_users` tidak lengkap |
| **OQ-04** | 🟡 Apakah Mode Akurasi default ON untuk SD? | Default perlu dikonfirmasi sebelum implementasi |

---

## Jadwal Kritis

```
Sab 19/9  → Kontrak API/WS dikunci; scaffold repo; style guide; bank soal seed
Min 20/9  → WS hub + room goroutine; lobby + soal + feedback FE; aset situs v1
Sen 21/9  → Scoring lulus unit test; alur soal end-to-end; Lottie maskot
Sel 22/9  → Deploy staging VPS; integrasi WS nyata; QA visual HP nyata
Rab 23/9  → k6 lulus; Playwright lulus; FEATURE FREEZE 18.00; pg_dump produksi
Kam 24/9  → Deploy produksi; uji pengguna nyata; monitor log; hotfix
```

---

## Tech Stack Ringkas

| Layer | Stack |
|---|---|
| Backend | Go 1.23 + net/http + gorilla/websocket + pgx v5 + sqlc + goose + go-redis |
| Database | PostgreSQL 17 (sumber kebenaran) + Redis 7 (PIN, leaderboard, rate limit) |
| Frontend | Vite + React + TypeScript + Tailwind CSS + Framer Motion + lottie-react + Zustand + react-i18next |
| Infra | Docker Compose (4 service: Caddy + Go API + PostgreSQL + Redis) di VPS 2vCPU/4GB |
| CI | GitHub Actions: `go test -race`, golangci-lint, pnpm lint + tsc + build |
| Testing | `go test -race` + Playwright E2E + k6 load test |

---

## NFR Targets

| Metrik | Target |
|---|---|
| ACK latency (q.answer → q.result) p95 | **< 300ms** |
| Halaman awal (FCP) di 4G | **< 2.5 detik** |
| Concurrent users | **75 (5 room × 15)** |
| Uji beban k6 error rate | **0 error** |
| Availability saat acara | **≥ 99.9%** |

---

*Semua dokumen di atas dihasilkan dari 3 dokumen sumber: [About Game.md](file:///d:/Workspace/jejak-inderasakti-be/About%20Game.md) · [Architecture Plan.md](file:///d:/Workspace/jejak-inderasakti-be/Architecture%20Plan.md) · [Design Document.md](file:///d:/Workspace/jejak-inderasakti-be/Design%20Document.md)*
