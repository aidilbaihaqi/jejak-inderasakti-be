About Game 

# About Game — Jejak Inderasakti: Jelajah Pulau Penyengat 

2026-09-19 · @Someone 

## Ringkasan Game 

Jejak Inderasakti adalah game kuis web mobile-first yang dimainkan online tentang 5 situs cagar budaya Pulau Penyengat. Satu sesi berlangsung 5–10 menit (5 situs × 3 soal), maksimal 15 peserta per room, dalam Bahasa Indonesia atau Inggris. 

|Aspek|Keputusan|
|---|---|
|Nama resmi|Jejak Inderasakti (disetujui)|
|Maskot|Sakti, lebah penyengat bertanjak (disetujui)|
|Genre|Kuis edukasi + petualangan peta|
|Platform|Web app online di browser HP; tanpa mode offline|
|Bahasa|Indonesia & Inggris, dipilih tiap pemain; satu room<br>boleh campur bahasa|
|Audiens|SD kelas 4 sampai SMA/SMK, pengunjung umum;<br>guru/panitia sebagai host|
|Room|Maks. 15 peserta; maks. 5 room berjalan bersamaan<br>(75 peserta)|
|Durasi 1 sesi|5–10 menit: 15 soal (Sesi Singkat: 10 soal)|
|Bank soal|Bank soal hasil riset dipakai langsung: 60 soal situs<br>+ 6 soal cadangan|



**Tujuan pembelajaran.** Setelah satu sesi, pemain mampu: 

1. Menyebutkan 5 situs utama Pulau Penyengat dan fungsi historisnya. 

2. Mengenali tokoh kunci: Engku Putri Hamidah, Raja Abdurrahman, Raja Ali, Raja Ali Haji, dan Raja Daud. 

Page 1 of 10 

About Game 

3. Mengenali peristiwa penting: mahar awal abad ke-19, renovasi masjid 1832, Gurindam 12 tahun 1847. 

4. Mengaitkan nilai Gurindam 12 dengan sikap sehari-hari. 

**Apa yang diambil dari referensi.** Dari Quizizz: room ber-PIN, tiap peserta menjawab dengan temponya sendiri, dan leaderboard real-time. Dari Duolingo: peta jalur stage, soal pendek, dan maskot yang memberi semangat. 

## Target Pengguna & Pendaftaran 

Satu bank soal melayani semua jenjang; jenjang room yang dipilih host menentukan campuran tingkat kesulitan dan panjang timer. 

|Pengguna|Usia|Kebutuhan utama|Penyesuaian di game|
|---|---|---|---|
|SD kelas 4–6|9–12<br>tahun|Bahasa sederhana,<br>visual besar|Lebih banyak soal mudah, timer<br>×1,25, masa baca 3 detik|
|SMP|12–15<br>tahun|Tantangan dan<br>kompetisi|Campuran mudah–sedang–sulit<br>seimbang|
|SMA/SMK|15–18<br>tahun|Soal analitis, adu<br>antarsekolah|Lebih banyak soal sedang dan<br>sulit|
|Pengunjung<br>umum|Semua<br>usia|Versi Inggris|Pilih "Umum / General visitor"<br>sebagai asal sekolah|
|Guru/panitia<br>(host)|Dewasa|Buat room, pantau,<br>ekspor hasil|Akun host dibuatkan panitia|



**Data yang diminta dari peserta.** Tanpa email, nomor HP, tanggal lahir, atau foto. 

|Field|Wajib|Aturan|
|---|---|---|
|Bahasa|Ya|Indonesia / English, dipilih di layar pertama|
|Nama panggilan|Ya|2–20 karakter, filter kata kasar, unik di dalam room|
|Asal sekolah|Ya|Autocomplete dari daftar sekolah, plus "Sekolah<br>lain" dan "Umum"|
|Jenjang & kelas|Ya|Untuk rekap; campuran soal mengikuti jenjang<br>room|



Page 2 of 10 

About Game 

|Field|Wajib|Aturan|
|---|---|---|
|Avatar|Ya|Pilih 1 dari 12 avatar ilustrasi|



```
flowchart LR
```

```
  A[Buka link / scan QR] --> B[Pilih bahasa]
  B --> C[Masukkan PIN]
  C --> D[Nama panggilan]
  D --> E[Sekolah & jenjang]
  E --> F[Avatar]
  F --> G[Lobby room]
```

Data pemain hanya berlaku di room itu, tanpa akun. Jika HP terputus atau browser tertutup, pemain masuk lagi dengan PIN yang sama dan melanjutkan dari soal terakhir. 

## Struktur Satu Sesi 

Satu sesi = 5 stage × 3 soal = 15 soal dalam 5–10 menit, tanpa babak final. Host dapat memilih Sesi Singkat (2 soal per situs, 10 soal) bila waktu sangat terbatas. 

```
flowchart LR
```

```
  L[Lobby] --> S1[Masjid Raya<br/>3 soal]
  S1 --> S2[Makam Engku Putri<br/>& RAH, 3 soal]
  S2 --> S3[Istana Kantor<br/>3 soal]
  S3 --> S4[Gedung Tabib<br/>3 soal]
  S4 --> S5[Perigi Puteri<br/>3 soal]
  S5 --> R[Hasil & podium]
```

### **Isi satu stage.** 

1. **Kartu "Tahukah Kamu?"** — 1 ilustrasi + 2 fakta singkat, tampil 10 detik dan bisa dilewati. 

2. **3 soal** , urut dari mudah ke sulit. 

3. **Pembahasan** 1 kalimat selama 3 detik setelah tiap soal. 

4. **Transisi peta** 1–2 detik; Kartu Warisan situs masuk koleksi. 

|Bagian|Cepat|Paling lama|
|---|---|---|
|Lobby & sambutan maskot|0,5 menit|1 menit|



Page 3 of 10 

About Game 

|Bagian|Cepat|Paling lama|
|---|---|---|
|5 kartu info|0,3 menit|0,8 menit|
|15 soal + pembahasan|3 menit|6,5 menit|
|Hasil & podium|0,5 menit|1 menit|
|**Total**|**± 4,5 menit**|**± 9,5 menit**|



## Mode Bermain 

Versi Kamis hanya punya satu mode: Room Live online dengan tempo per peserta seperti Quizizz. Mode Mandiri (gaya Duolingo) dan Jelajah Lapangan ditunda sampai setelah launch. 

|Aturan|Nilai|
|---|---|
|Peserta per room|Maks. 15|
|Room bersamaan|Maks. 5 (75 peserta)|
|Masuk|PIN 6 digit, scan QR, atau link|
|Tempo|Host menekan Mulai; setelah itu tiap peserta<br>menjawab dengan temponya sendiri, timer per soal<br>tetap|
|Set soal|Sama untuk semua peserta di room sesuai jenjang<br>room; urutan opsi diacak per peserta|
|Leaderboard|Real-time di layar host; peserta melihat<br>peringkatnya di akhir tiap stage|
|Selesai|Semua peserta tamat, host menekan Akhiri, atau<br>batas 12 menit|
|Koneksi putus|Masuk lagi dengan PIN yang sama; soal yang timer-<br>nya habis dihitung 0|



**Mengapa tempo per peserta, bukan serentak seperti Kahoot?** Peserta dengan sinyal lemah tidak tertinggal satu soal penuh, dan server tidak perlu menyinkronkan jam antarHP. Ini menghemat sekitar satu hari kerja backend, penting untuk tenggat Kamis. 

Page 4 of 10 

About Game 

**Layar host.** Buat room (jenjang, Sesi Normal/Singkat, Mode Akurasi), tampilkan PIN + QR besar, lihat peserta masuk (x/15), tekan Mulai, pantau progres dan leaderboard, lalu Akhiri dan unduh CSV. 

## Aturan Tingkat Kesulitan 

Di setiap stage soal naik dari mudah ke sulit, dan porsi soal sulit bertambah dari stage 1 ke stage 5. 

|Level|Kemampuan yang diuji|Tipe soal|Timer|
|---|---|---|---|
|Mudah|Mengingat fakta di kartu info atau|Pilihan ganda,|15|
|(M)|gambar|Benar/Salah|detik|
|Sedang|Memahami fungsi, alasan, dan makna|Pilihan ganda, lengkapi|20|
|(S)||kalimat|detik|
|Sulit (Su)|Menghubungkan fakta, mencari<br>pernyataan salah|Pilihan ganda|25<br>detik|



Timer untuk room SD dikali 1,25. Dua detik pertama tiap soal (tiga detik untuk SD) adalah masa baca: tombol sudah aktif, tetapi poin belum berkurang. 

**Komposisi 3 soal per stage** (urut kiri ke kanan). 

|Stage|SD|SMP|SMA/SMK|
|---|---|---|---|
|1 Masjid Raya|M, M, S|M, M, S|M, S, S|
|2 Makam Engku Putri & RAH|M, M, S|M, S, S|M, S, Su|
|3 Istana Kantor|M, S, S|M, S, Su|M, S, Su|
|4 Gedung Tabib|M, S, S|M, S, Su|S, S, Su|
|5 Perigi Puteri|M, S, Su|M, S, Su|S, Su, Su|



Sesi Singkat mengambil soal pertama dan terakhir dari tiap sel. 

### **Aturan pengambilan soal.** 

1. Server memilih soal acak sesuai komposisi saat room dibuat; semua peserta di room mendapat set dan urutan soal yang sama. 

Page 5 of 10 

About Game 

2. Urutan opsi jawaban diacak per peserta, kecuali Benar/Salah. 

3. Tiap room memakai acakan baru, sehingga 5 room serentak tidak mendapat set yang persis sama. 

4. Soal yang dijawab salah oleh lebih dari 80% peserta ditandai untuk ditinjau panitia. 

## Sistem Poin: Ketepatan dan Kecepatan 

Jawaban benar selalu bernilai minimal 60% poin dasar; kecepatan hanya menambah hingga 40% sisanya. Jawaban salah bernilai 0, tanpa pengurangan. Desain ini sengaja lebih condong ke ketepatan dibanding Kahoot, yang memberi minimal 50% (Kahoot Help), karena guru di forum Kahoot sendiri mengeluhkan siswa yang asal cepat menjawab. 

```
P = \operatorname{round}\left( B \cdot \left( 0{,}6 + 0{,}4 \cdot \left( 1 -
\frac{\max(0,\, t - g)}{T - g} \right) \right) \right) + S
```

Keterangan: B = poin dasar (mudah 500, sedang 750, sulit 1.000), t = waktu jawab dalam detik yang diukur server, T = timer soal, g = masa baca (2 detik, SD 3 detik), S = bonus beruntun. 

**Bonus beruntun.** Jawaban benar ke-2 berturut-turut +50, ke-3 +100, dan seterusnya naik 50 hingga maksimal +250. Satu jawaban salah mengembalikan hitungan ke nol. 

**Skor maksimum per sesi** sekitar 12.750 untuk room SD dan 14.750 untuk room SMA (semua benar di masa baca, termasuk bonus beruntun). Skor antarjenjang tidak dibandingkan di leaderboard. 

|Kasus (soal sedang, T = 20 detik, g = 2 detik)|Waktu jawab|Poin (tanpa bonus)|
|---|---|---|
|Benar, masih masa baca|1 detik|750|
|Benar, cepat|4 detik|717|
|Benar, sedang|12 detik|583|
|Benar, detik terakhir|19 detik|467|
|Salah atau habis waktu|—|0|



**Penentu peringkat bila skor sama:** jumlah jawaban benar lebih banyak, lalu total waktu jawab lebih kecil. 

Page 6 of 10 

About Game 

**Leaderboard sekolah:** rata-rata 5 skor tertinggi dari tiap sekolah, minimal 3 pemain. Cara ini mencegah sekolah dengan peserta paling banyak otomatis menang. 

**Mode Akurasi (opsional, untuk SD).** Host dapat mematikan faktor kecepatan; setiap jawaban benar langsung bernilai B. 

## Elemen Gamifikasi 

Versi Kamis hanya memakai elemen yang terasa dalam satu sesi 5–10 menit; elemen jangka panjang ditunda karena belum ada akun pemain permanen. 

|Elemen|Aturan|Versi|
|---|---|---|
|Maskot Sakti|Lebah penyengat bertanjak; nama pulau<br>berasal dari cerita pelaut yang disengat<br>serangga saat mengambil air tawar. Muncul<br>saat sambutan, benar, salah, dan podium|Kamis|
|Poin + bonus beruntun|Lihat Sistem Poin|Kamis|
|Leaderboard room|Real-time di layar host; peserta melihat<br>peringkat di akhir tiap stage|Kamis|
|Podium 3 besar|Confetti + maskot di akhir sesi|Kamis|
|Kartu Warisan|Ilustrasi situs + 1 fakta kunci di akhir tiap<br>stage|Kamis|
|Rekap sekolah|Rata-rata 5 skor tertinggi tiap sekolah selama<br>acara|Kamis|
|Lencana situs & prestasi|Penjaga Masjid Kuning, Sahabat Gurindam,<br>Arsitek Istana, Peracik Ramuan, Penjaga<br>Perigi|Setelah<br>launch|
|XP, gelar, streak harian|Butuh akun pemain|Setelah<br>launch|
|Bantuan 50:50 & Bisikan|1× per sesi|Setelah|
|Sakti||launch|
|Sertifikat PDF|Nama + sekolah + skor|Setelah<br>launch|



Page 7 of 10 

About Game 

## Hasil Riset & Koreksi Kunci Jawaban 

Sebagian besar kunci jawaban LKP sesuai sumber, tetapi 7 hal perlu diperbaiki atau diperhalus sebelum masuk game. Soal di bank sudah memakai versi terkoreksi; tahun yang sumbernya berbeda-beda tidak dijadikan soal tebak angka. 

|Topik|Di LKP|Temuan riset|Keputusan untuk<br>game|
|---|---|---|---|
|Tahun mahar Pulau<br>Penyengat|1803|Sumber berbeda: 1801<br>(<br>Kemdikbud 2018),<br>1803 (<br>JOM Unri), 1805<br>(<br>GNFI)|Tulis "awal abad ke-<br>19"; jangan jadikan<br>soal tahun|
|Pembangunan<br>masjid|Dibangun 1832 oleh<br>Raja Abdurrahman|Awal dibangun Sultan<br>Mahmud 1803,<br>direnovasi ke bentuk<br>sekarang 1832 masa<br>YDM VII Raja<br>Abdurrahman<br>(<br>Kemdikbud 2018)|Pakai kata<br>"direnovasi ke<br>bentuk sekarang<br>pada 1832"|
|Tahun Istana<br>Kantor|Selesai sekitar 1855|Sumber yang dibuka<br>hanya menyebut Raja<br>Ali tinggal/menjabat<br>1844–1857 (<br>UMRAH);<br>angka 1855 tidak<br>ditemukan|Pakai rentang 1844–<br>1857|
|Gaya arsitektur<br>Istana Kantor|"Einfach Klasik" /<br>Eropa|Istilah "Einfach Klasik"<br>tidak ditemukan; kajian<br>arsitektur menyebut<br>akulturasi Melayu<br>dengan Eropa, Timur<br>Tengah, dan India(<br>SIAR<br>UMS 2022)|Jawaban: pengaruh<br>Eropa (kolonial)<br>berpadu Melayu|



Page 8 of 10 

About Game 

|Topik|Di LKP|Temuan riset|Keputusan untuk<br>game|
|---|---|---|---|
|Kitab di Gedung<br>Tabib|"Risalah Ilmu Tabib<br>dan Rumah Obat"<br>(satu judul)|Itu dua naskah<br>berbeda:_Ilmu Tabib_<br>ditulis Raja Haji Daud;<br>_Rumah Obat di Pulau_<br>_Penyengat_dikarang Ali<br>Azimat; keduanya<br>koleksi Yayasan<br>Indrasakti (<br>Jumantara<br>2020)|Jawaban: naskah<br>_Ilmu Tabib_|
|Tokoh Gedung<br>Tabib|Raja Daud bin Raja<br>Ahmad bin Raja Haji<br>Fisabilillah|Kediaman Tabib Raja<br>Daud dikonfirmasi<br>Disbudpar<br>Tanjungpinang;<br>silsilahnya tidak<br>ditemukan.<br>Kompas<br>2024 mencampurnya<br>dengan Raja Ahmad<br>Tabib, tokoh berbeda|Jawaban cukup<br>"Raja Daud"; silsilah<br>tidak ditanyakan|
|Perigi Puteri|"Dinding pembatas<br>tinggi"; air tak<br>pernah kering|Bangunan berdenah<br>segi empat beratap<br>kubah setengah<br>silinder, satu-satunya<br>perigi berkubah di<br>pulau; dipugar pertama<br>1982(<br>Wikipedia,<br>Registrasi Cagar<br>Budaya)|"Tak pernah kering"<br>ditulis sebagai<br>kepercayaan warga,<br>bukan fakta|
|Gelar Raja Ali pada<br>sumber<br>Kemdikbud|—|Kemdikbud 2018<br>menyebut Raja Ali<br>"YDM VII"; kajian<br>Unimed dan LKP<br>menyebut YDM VIII<br>(<br>Unimed)|Pakai YDM VIII|



Page 9 of 10 

About Game 

**Fakta tambahan yang dipakai di bank soal:** Raja Ali Haji lahir 1809, cucu Raja Haji Fisabilillah (Kembara UMM); Pahlawan Nasional lewat Keppres 089/TK/2004 dan _Kitab Pengetahuan Bahasa_ 1858 sebagai pelopor kamus ekabahasa Melayu (GNFI 2022); RSUD Provinsi Kepri dinamai dari Raja Ahmad Tabib, cucu Raja Ali Haji, tabib kerajaan sejak 1901 (PPID Kepri). 

**Catatan tugas lapangan.** Perigi Putri masih dipakai warga untuk mandi dan mencuci, jadi tugas "basuh tangan" wajib dengan izin penjaga dan peserta dilarang meminum airnya. 

## Bank Soal 

Bank soal hasil riset dipakai langsung: 60 soal situs (12 per situs) untuk sesi, plus 6 soal lintas situs sebagai cadangan, di tab Bank Soal . Terjemahan Inggris ada di tab Question Bank (EN) . Satu room hanya memakai 15 soal, jadi peserta yang main ulang akan mendapat banyak soal baru. 

## Pertanyaan Terbuka & Sumber 

- ~~Peserta per room maks. 15, maks. 5 room serentak.~~ 

- ~~Game dimainkan online.~~ 

- ~~Nama "Jejak Inderasakti" dan maskot Sakti disetujui.~~ 

- ~~Perlu versi Bahasa Inggris.~~ 

- Siapa yang memvalidasi bank soal sebelum uji Kamis: juru pelihara situs, Yayasan Indrasakti, atau guru sejarah? 

- Siapa yang mereview terjemahan Inggris? 

- Berapa akun host yang perlu dibuat, dan untuk siapa? 

**Sumber utama riset situs:** <u>Kemdikbud — Digitalisasi Data Keraton Penyengat (2018)</u> · <u>Mu'jizah, Jumantara Vol. 11 No. 1 (2020)</u> · Disbudpar Kota Tanjungpinang — Gedung Tabib · <u>Registrasi Nasional Cagar Budaya — Perigi Putri</u> · Unimed — Situs Penyengat, Bab V · SOJ <u>UMRAH</u> · Kahoot — How points work. Sumber tambahan per soal ada di tab Bank Soal. 

Page 10 of 10 

