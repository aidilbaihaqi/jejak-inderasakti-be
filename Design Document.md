Design Document — Jejak Inderasakti 

# Design Document — Jejak Inderasakti 

2026-09-19 · @Someone 

## Prinsip Desain 

Desain mengejar tiga hal: jelas untuk anak SD kelas 4, cepat di HP Android murah, dan terasa Melayu tanpa terlihat kuno. Semua keputusan di bawah diuji terhadap sesi 5–10 menit dengan 15 soal. 

1. **Satu layar, satu tugas.** Tiap layar hanya punya satu aksi utama; tidak ada menu tersembunyi. 

2. **Besar dan berwarna.** Teks minimal 16 px, tombol jawaban minimal 64 px tinggi, dan tiap situs punya warna sendiri agar anak tahu sedang di stage mana. 

3. **Umpan balik instan.** Benar/salah terlihat dalam < 300 ms lewat warna, ikon, suara, dan ekspresi maskot. 

4. **Kolase kertas bernuansa Melayu.** UI tampil seperti buku perjalanan dari kertas: memo, selotip, struk, prangko, dan peta lipat; motif pucuk rebung dan awan larat dipakai sebagai aksen tipis. 

5. **Bilingual sejak awal.** Tidak ada teks di dalam gambar; layout diuji dengan teks Indonesia yang umumnya lebih panjang. 

**Yang diambil dari referensi.** Duolingo: maskot ekspresif, tombol tebal, peta jalur. Quizizz: kotak jawaban besar, leaderboard dan podium. KŌSA karya studio Hello-World: hanya gaya cerianya — kolase stiker, kertas bertempel, label berkurung, stabilo, lencana bernomor, dan metafora persimpangan jalan. Palet KŌSA tidak dipakai; warnanya diganti kuning Melayu, emas, dan coklat dalam tema terang. Ilustrasi, karakter, logo, dan foto mereka juga tidak ditiru. 

## Identitas Visual 

Tema terang dengan warna hangat: kertas krem sebagai dasar, kuning Melayu sebagai warna utama, emas untuk hiasan, dan coklat tua menggantikan hitam untuk teks. Warna kertas dibuat sedikit lebih hangat dari latar KŌSA (#F8F5EE, diambil dari screenshot), dan semua pasangan teks di bawah sudah dicek rasio kontrasnya. 

Page 1 of 14 

Design Document — Jejak Inderasakti 

|Token|Hex|Peran|Kontras teks|
|---|---|---|---|
|`kertas`|#FBF5E6|Latar semua layar|`tinta`13,4:1|
|`kertas-putih`|#FFFDF7|Kartu soal, memo, struk|`tinta`14,3:1|
|`kraft`|#EAD7B0|Kertas kraft, grid, selotip,<br>papan lobby|`tinta`10,3:1|
|`kuning`(kuning<br>Melayu)|#FFC629|Warna utama: tombol utama,<br>pita, sorotan stage|`tinta`9,3:1|
|`stabilo`|#FFE58A|Stabilo kata kunci, baris<br>sendiri di leaderboard|`tinta`11,7:1|
|`emas`|#C8922A|Lencana starburst, bingkai<br>prangko, bintang, ikon|`tinta`5,3:1;**jangan jadi**<br>**warna teks**di atas kertas<br>(2,5:1)|
|`coklat`|#8B5A2B|Tombol sekunder, outline<br>stiker, teks sekunder|Putih 5,8:1; sebagai teks di<br>atas kertas 5,4:1|
|`tinta`|#3A2412|Teks utama (coklat sangat<br>tua)|—|
|`benar`|#1B7F4A|Umpan balik benar|Putih 5,0:1|
|`salah`|#C53A3A|Umpan balik salah|Putih 5,2:1|



**Proporsi.** Sekitar 70% kertas dan kraft, 20% kuning/emas/coklat, 10% aksen situs dan umpan balik. Dengan proporsi ini layar tetap terasa terang dan ceria, bukan kusam. 

**Ilustrasi vs UI.** Ilustrasi situs tetap memakai warna asli bangunan (masjid tetap kuning– hijau) karena ada soal tentang warna. Palet hangat di atas dipakai untuk UI, kertas, dan hiasan. 

**Tipografi.** Judul dan angka skor: **Baloo 2** (700–800). Teks: **Nunito** (400/700). Label berkurung, PIN, dan angka di struk: **Space Mono** (400/700). Ketiganya Google Fonts. Coretan tulisan tangan dibuat sebagai bagian aset SVG, bukan font keempat. Skala 32 / 24 / 20 / 18 / 16 / 14 px, tinggi baris 1,4; teks soal 20 px (room SD 22 px). 

**Maskot Sakti (disetujui).** Lebah penyengat bergaya chibi: badan bulat kuning–hitam, sayap biru muda transparan, tanjak (ikat kepala Melayu) warna `coklat` dengan motif emas, sengat kecil membulat agar tidak menakutkan. 

Page 2 of 14 

Design Document — Jejak Inderasakti 

|Pose|Dipakai di|
|---|---|
|Melambai|Layar bahasa, lobby|
|Berpikir|Menunggu host, soal sulit|
|Menunjuk|Kartu info|
|Bersorak|Jawaban benar|
|Menyemangati (bukan sedih)|Jawaban salah|
|Memegang piala|Podium|



## Gaya Ceria: Kertas, Stiker & Peta Jalan 

Gaya ceria KŌSA diambil dari struktur visualnya, lalu diisi ulang dengan benda-benda Penyengat dan palet hangat. Setiap situs menjadi persimpangan di peta kertas, dan setiap komponen punya padanan benda kertas. 

|Elemen ceria di KŌSA (screenshot)|Padanan di Jejak Inderasakti|
|---|---|
|Hero penuh stiker bertebaran: lampu lalu<br>lintas, pohon, makanan, kamera, bintang|Layar bahasa dan lobby: stiker Penyengat di<br>tepi layar — perahu pompong, kubah, tanjak,<br>daun sirih, bintang emas|
|Catatan kuning ditempel dengan teks mesin<br>tik|Kartu info: memo kraft dengan teks Space<br>Mono|
|Coretan tulisan tangan ("Do more of what<br>makes you happy")|Coretan tangan dalam aset SVG: panah,<br>lingkaran, "Ayo!" / "Let's go!"|
|Label hitam berkurung<br>`[`交差`KOSA ]`|Label coklat tua berkurung<br>`[ Stage 1/5 ]`,<br>`[ PIN 482913 ]`|
|Stabilo pink pada kata kunci|Stabilo kuning pada kata kunci di kartu info<br>dan pembahasan|
|Foto dipotong bentuk blob dan segi tak<br>beraturan|Ilustrasi situs dalam bingkai blob atau<br>prangko bergerigi|
|Section berlatar kertas grid|Kertas grid kraft untuk peta dan ringkasan<br>stage|



Page 3 of 14 

Design Document — Jejak Inderasakti 

|Elemen ceria di KŌSA (screenshot)|Padanan di Jejak Inderasakti|
|---|---|
|Kartu warna-warni bernomor dengan<br>lencana starburst 01–04|Kartu stage dengan lencana starburst emas<br>bernomor 1–5|
|Foto orang cut-out berbingkai putih + kartu<br>kertas miring|Avatar peserta sebagai stiker cut-out<br>berbingkai putih; kartu pembahasan sedikit<br>miring|
|Teks berjalan di bawah hero|Marquee lobby: "Jejak Inderasakti✦Pulau<br>Penyengat✦Gurindam 12"|
|Doodle kecil (spiral, pesawat, bintang)|Doodle emas kecil: spiral, bintang, jejak kaki<br>di sepanjang rute|
|Headline besar dengan latar putih per baris|Judul hasil dan podium dengan latar kuning<br>per baris|
|Footer gelap|**Tidak dipakai**: seluruh aplikasi bertema<br>terang|
|Emoji di tombol|**Tidak dipakai**: emoji tampil berbeda di tiap<br>merek HP Android; pakai ikon SVG sendiri|



### **Perlakuan per komponen.** 

|Komponen|Elemen kertas / stiker / peta|Implementasi|
|---|---|---|
|Latar aplikasi|Kertas krem bertekstur halus; grid<br>kraft untuk peta dan ringkasan|Tile WebP 512 px ≤ 40 KB di<br>atas<br>`kertas`|
|Tombol utama|Stiker<br>`kuning`dengan teks<br>`tinta`,<br>outline + bayangan offset<br>`coklat`|CSS|
|Tombol sekunder|`kertas-putih`dengan outline<br>`coklat`|CSS|
|`StageHeader`|Label berkurung<br>`tinta`+ selotip<br>kuning + lencana starburst emas<br>bernomor stage|Space Mono + SVG|
|`TimerBar`|Jalan putus-putus coklat; Sakti kecil<br>berjalan; jalur terlewat terisi kuning;<br>merah saat < 5 detik|SVG<br>`stroke-dasharray`|



Page 4 of 14 

Design Document — Jejak Inderasakti 

|Komponen|Elemen kertas / stiker / peta|Implementasi|
|---|---|---|
|`QuestionCard`|Memo<br>`kertas-putih`, selotip kraft,<br>tepi bawah sobek|SVG mask|
|`AnswerTile`|Stiker kertas berwarna (lihat tabel<br>opsi), outline<br>`tinta`2 px, bayangan<br>offset<br>`coklat`4 px, tidak dimiringkan|CSS|
|Umpan balik|Cap stempel "BENAR!" / "COBA LAGI"<br>miring −8° + taburan bintang emas<br>saat benar|SVG + animasi skala|
|Peta pulau (P5)|Peta kertas lipat di atas grid kraft, rute<br>putus-putus coklat, pin starburst<br>bernomor, jejak kaki doodle|SVG|
|Progres 5 situs|Paspor dengan 5 kotak prangko; terisi<br>cap berwarna aksen situs|SVG|
|Kartu Warisan|Kartu pos + prangko bergerigi<br>berbingkai emas; dibalik untuk melihat<br>fakta|CSS 3D flip|
|`LeaderboardRow`|Struk<br>`kertas-putih`bertepi zigzag;<br>angka Space Mono; baris sendiri<br>distabilo|CSS mask|
|Lobby (P4)|Papan kraft, avatar stiker cut-out,<br>marquee, stiker Penyengat di tepi|CSS animation|
|`PinInput`|Karcis kuning dengan garis perforasi|CSS|
|Podium (P10)|Pita juara kuning–emas, confetti kertas<br>kuning/emas/coklat + aksen situs|canvas-confetti|



**Perlakuan per situs.** Aksen situs memakai warna tanah yang serasi dengan emas dan coklat; semuanya lolos kontras dengan teks putih. 

|Situs|Aksen|Hex|Kontras teks<br>putih|Kertas & elemen|Cap / prangko|
|---|---|---|---|---|---|
|1 Masjid Raya|Hijau|#2F7D4F|5,0:1|Kertas krem, selotip kuning,|Kubah bawang|
||Masjid|||rute berawal dari dermaga||



Page 5 of 14 

Design Document — Jejak Inderasakti 

|Situs|Aksen|Hex|Kontras teks<br>putih|Kertas & elemen|Cap / prangko|
|---|---|---|---|---|---|
|2 Makam Engku<br>Putri & RAH|Emas<br>Tua|#9A6B12|4,7:1|Kertas kraft seperti naskah<br>lama, garis tulisan samar|Gulungan<br>naskah & pena|
|3 Istana Kantor|Bata|#B4533A|5,0:1|Kertas krem dengan pola bata<br>tipis|Gapura|
|4 Gedung Tabib|Zaitun|#5F7F2E|4,6:1|Kertas gaya herbarium, daun<br>ditempel selotip|Daun & lesung|
|5 Perigi Puteri|Biru<br>Perigi|#2E7EA0|4,6:1|Kertas krem dengan riak cat<br>air|Tetes air|



### **Aturan pakai.** 

1. Teks soal selalu di area kertas polos; semua pasangan teks–latar di dokumen ini ≥ 4,5:1. 

2. `emas` hanya untuk isian, bingkai, dan ikon — tidak pernah sebagai warna teks di atas kertas. 

3. Layar soal: maksimal 3 elemen hias. Layar bahasa, lobby, dan podium boleh lebih ramai (5–7 stiker) karena tidak ada soal yang harus dibaca. 

4. Miring ±2–4° hanya untuk hiasan; tombol dan teks soal tetap lurus. 

5. Total aset kertas dan stiker tambahan ≤ 300 KB; tepi sobek, zigzag, dan perforasi dibuat dengan SVG mask. 

6. Maskot Sakti tampil sebagai stiker berbingkai putih agar menyatu dengan kolase. 

## Layout Mobile-First 

Frame desain utama 360 × 800 px (Android kecil); semua layar peserta harus muat tanpa scroll kecuali layar daftar. 

|Aturan|Nilai|
|---|---|
|Frame uji|360×800, 390×844, 412×915; layar host 1280×720<br>landscape|
|Lebar konten|Maks. 480 px, di tengah pada tablet/desktop|
|Grid|4 kolom, margin 16 px, gutter 12 px|
|Spasi|Skala 4 px: 4, 8, 12, 16, 24, 32|



Page 6 of 14 

Design Document — Jejak Inderasakti 

|Aturan|Nilai|
|---|---|
|Target sentuh|Min. 48×48 px; tombol jawaban min. 64 px tinggi,<br>jarak 12 px|
|Zona jempol|Tombol jawaban di setengah bawah layar; timer dan<br>progres di atas|
|Breakpoint|< 480 px: opsi 1 kolom · 480–1024 px: opsi 2 kolom ·<br>≥ 1024 px: layout host|
|Keyboard|Layar daftar tetap bisa dipakai saat keyboard<br>terbuka; tombol "Siap!" menempel di atas keyboard|



## User Flow & Daftar Layar 

Versi Kamis butuh 10 layar peserta dan 5 layar host; urutan pengerjaan desain mengikuti nomor di tabel. 

```
flowchart LR
  P1[Bahasa] --> P2[PIN]
  P2 --> P3[Daftar]
  P3 --> P4[Lobby]
  P4 --> P5[Peta]
  P5 --> P6[Kartu info]
  P6 --> P7[Soal]
  P7 --> P8[Umpan balik]
  P8 -->|soal berikut| P7
  P8 -->|stage selesai| P9[Ringkasan stage]
  P9 -->|stage berikut| P5
  P9 -->|stage 5| P10[Hasil & podium]
```

|#|Layar|Elemen utama|Catatan|
|---|---|---|---|
|P1|Pilih bahasa|Logo, Sakti melambai, 2 tombol|Toggle bahasa juga ada di pojok|
|||besar "Bahasa Indonesia" /|kanan atas P2–P4|
|||"English"||
|P2|Masukkan|6 kotak digit, keyboard angka,|Pesan jelas bila PIN salah atau|
||PIN|tombol scan QR|room penuh (15/15)|



Page 7 of 14 

Design Document — Jejak Inderasakti 

|#|Layar|Elemen utama|Catatan|
|---|---|---|---|
|P3|Daftar|Nama, autocomplete sekolah,<br>chip jenjang, grid 12 avatar|Satu layar scroll, tombol "Siap!"<br>menempel di bawah|
|P4|Lobby|Avatar peserta muncul satu per<br>satu, hitungan x/15, Sakti berpikir<br>"Menunggu host…"|Tanpa tombol aksi|
|P5|Peta pulau|Ilustrasi pulau, 5 pin situs, jalur<br>putus-putus, pin aktif berdenyut|Otomatis lanjut 1,5 detik|
|P6|Kartu info|Ilustrasi situs, nama situs, 2 fakta,<br>tombol Lanjut, hitung mundur 10<br>detik|Warna situs|
|P7|Soal|Stage x/5, soal x/3, bar timer, teks<br>soal, 2–4 tombol jawaban|Lihat wireframe di bawah|
|P8|Umpan balik|Tombol berubah hijau/merah +<br>ikon, "+717" terbang ke skor, Sakti,<br>1 kalimat pembahasan|3 detik atau ketuk untuk lanjut|
|P9|Ringkasan<br>stage|Kartu Warisan situs, skor stage,<br>peringkat sementara|3 detik|
|P10|Hasil &<br>podium|Podium 3 besar, peringkat sendiri,<br>jumlah benar, skor|Confetti|
|H1–<br>H5|Host|Login · Buat room (jenjang, Sesi<br>Normal/Singkat, Mode Akurasi) ·<br>Lobby host (PIN + QR besar,<br>daftar peserta, Mulai) · Monitor<br>(progres per peserta,<br>leaderboard) · Hasil (podium,<br>unduh CSV)|Landscape 1280×720 untuk<br>proyektor|



### **Wireframe P7 (Soal), 360 px.** 

Page 8 of 14 

Design Document — Jejak Inderasakti 



<!-- Start of picture text -->
+----------------------------------+<br>| [Masjid Raya]  Stage 1/5  1/3 ***|  <- warna situs, skor<br>| [=================-----]  12s    |  <- bar timer<br>|                                  |<br>|  Berapa jumlah kubah masjid ini? |  <- 20 px, maks. 3 baris<br>|                                  |<br>|      [ ilustrasi opsional ]      |<br>|                                  |<br>| +------------------------------+ |<br>| | /\  A  4                     | |  <- 64 px, motif + huruf<br>| +------------------------------+ |<br>| | <>  B  9                     | |<br>| +------------------------------+ |<br>| | *   C  13                    | |<br>| +------------------------------+ |<br>| | ()  D  17                    | |<br>| +------------------------------+ |<br>+----------------------------------+<br><!-- End of picture text -->

## Komponen UI & Design Tokens 

Empat belas komponen (termasuk `QuestionCard` di bagian Bahasa Visual) cukup untuk seluruh layar versi Kamis; FE membangunnya sebagai komponen React dengan token di bawah sebagai konfigurasi Tailwind. 

|Komponen|Varian / state|Catatan|
|---|---|---|
|`Button`|primary, secondary, ghost ·<br>default, pressed, disabled|Gaya stiker kertas: outline tinta<br>2 px + bayangan offset 4 px;<br>saat ditekan bergeser 2 px|
|`AnswerTile`|A–D · idle, selected, correct,<br>wrong, reveal|Motif + huruf + teks; lihat tabel<br>opsi|
|`TimerBar`|normal, < 50%, < 20%|Jalan putus-putus terisi kuning;<br>merah berdenyut saat < 5 detik|
|`StageHeader`|5 warna situs|Nama situs, stage x/5, soal x/3|
|`ScoreChip`|default, naik|Angka memantul saat<br>bertambah|



Page 9 of 14 

Design Document — Jejak Inderasakti 

|Komponen|Varian / state|Catatan|
|---|---|---|
|`StreakBadge`|×3, ×5|Muncul mulai 3 benar beruntun|
|`Avatar`|12 gambar · S/M/L|Lingkaran dengan cincin warna<br>situs|
|`PinInput`|kosong, terisi, error|6 kotak, keyboard angka|
|`SchoolAutocomplete`|ketik, hasil, kosong|Opsi "Sekolah lain" dan "Umum"<br>selalu di bawah|
|`LeaderboardRow`|biasa, diri sendiri, 1–3|Baris sendiri disorot kuning|
|`SiteCard`|kartu info, Kartu Warisan|Ilustrasi + teks|
|`MascotSakti`|6 pose statis + 4 Lottie|Fallback gambar statis bila<br>Lottie gagal|
|`LanguageToggle`|ID, EN|Pojok kanan atas|



|Opsi|Motif|Warna|Warna teks|Kontras|
|---|---|---|---|---|
|A|Pucuk rebung (segitiga)|`kuning`#FFC629|`tinta`|9,3:1|
|B|Wajik (belah ketupat)|Bata #B4533A|Putih|5,0:1|
|C|Bunga cengkih (4 kelopak)|Zaitun #5F7F2E|Putih|4,6:1|
|D|Tetes air (lingkaran)|Biru Perigi #2E7EA0|Putih|4,6:1|



Soal Benar/Salah memakai opsi A dan B saja dengan label "Benar"/"Salah" ("True"/"False"). 

```
// tailwind.config.js (potongan) — tema terang hangat
theme: {
  extend: {
    colors: {
      kertas: { DEFAULT: '#FBF5E6', putih: '#FFFDF7', kraft: '#EAD7B0' },
      kuning: '#FFC629', stabilo: '#FFE58A', emas: '#C8922A',
      coklat: '#8B5A2B', tinta: '#3A2412',
      benar: '#1B7F4A', salah: '#C53A3A',
      situs: { masjid: '#2F7D4F', makam: '#9A6B12', istana: '#B4533A',
               tabib: '#5F7F2E', perigi: '#2E7EA0' },
    },
    fontFamily: {
```

Page 10 of 14 

Design Document — Jejak Inderasakti 

```
      display: ['"Baloo 2"', 'sans-serif'],
      body: ['Nunito', 'sans-serif'],
      label: ['"Space Mono"', 'monospace'],
    },
    backgroundImage: { kertas: "url('/assets/paper.webp')", grid:
"url('/assets/grid-kraft.svg')" },
    borderRadius: { tile: '16px', card: '24px' },
    boxShadow: { stiker: '4px 4px 0 #8B5A2B' },
    transitionDuration: { fast: '150ms', base: '250ms', slow: '600ms' },
  },
}
```

## Motion, Suara & Umpan Balik 

Total animasi per soal dibatasi 1 detik supaya sesi tetap 5–10 menit; semua animasi diganti fade bila perangkat mengaktifkan `prefers-reduced-motion` . 

|Momen|Animasi|Durasi|Suara|
|---|---|---|---|
|Peserta masuk<br>lobby|Avatar pop (skala 0 → 1,1 → 1)|300 ms|Klik ringan|
|Soal muncul|Teks naik, opsi muncul berurutan<br>(jeda 60 ms)|250 ms|—|
|Timer < 5 detik|Bar merah berdenyut|Sampai<br>habis|Tik tiap detik (bisa<br>dimatikan)|
|Jawaban benar|Tile hijau + centang, "+poin"<br>terbang ke skor, Sakti bersorak|600 ms|"Ding" cerah|
|Jawaban salah|Tile bergetar, tile benar disorot,<br>Sakti menyemangati|300 ms|Nada lembut, bukan<br>buzzer|
|Streak ×3|StreakBadge memantul|400 ms|—|
|Pindah stage|Peta zoom ke pin berikutnya|1,2 detik|Whoosh|
|Podium|Confetti + Sakti memegang piala|2 detik|Fanfare pendek|



Audio dalam satu sprite MP3 + OGG, total ≤ 150 KB, volume awal rendah, tombol mute di header. Lottie maskot masing-masing ≤ 60 KB. 

Page 11 of 14 

Design Document — Jejak Inderasakti 

## Brief Ilustrasi Vektor per Situs 

Semua ilustrasi bergaya kolase kertas: bentuk flat vector cerah seolah digunting dari kertas berwarna, dengan serat halus, tepi sedikit tidak rata, dan bayangan tempel tipis. Tidak ada teks di dalam gambar, dan detail bangunan tetap akurat sesuai riset karena beberapa soal menanyakan hal yang terlihat di gambar. 

|Situs|Wajib ada|Hindari|
|---|---|---|
|1 Masjid Raya|Bangunan kuning aksen hijau, 4<br>menara di sudut, kubah bawang,<br>Rumah Sotoh di kiri-kanan<br>depan, laut dan perahu<br>pompong|Jumlah kubah/menara yang salah:<br>jika semua terlihat, harus 13 kubah<br>dan 4 menara|
|2 Makam Engku<br>Putri & RAH|Kompleks makam berpagar,<br>dinding bertulisan (prasasti<br>Gurindam 12, cukup garis-garis),<br>gulungan naskah, pena|Kesan seram; menggambar wajah<br>tokoh sejarah|
|3 Istana Kantor|Bangunan induk dua lantai,<br>tembok keliling seperti benteng,<br>gapura barat dengan pos<br>pengintai|Istana megah berkubah emas yang<br>tidak sesuai aslinya|
|4 Gedung Tabib|Dinding bata tanpa atap, akar<br>pohon besar melilit dinding,<br>kusen jendela kayu, jahe/daun<br>herbal, lesung, botol ramuan|Atap utuh; suasana horor<br>reruntuhan|
|5 Perigi Puteri|Bangunan segi empat berkubah<br>setengah silinder, pintu relung<br>dengan pilar semu, sumur<br>dengan air jernih|Figur orang mandi; kubah<br>berbentuk bawang|
|Peta pulau|Pulau Penyengat bergaya, 5 pin<br>sesuai urutan stage, dermaga,<br>laut|Skala dan posisi yang terlalu detail<br>seperti peta resmi|



|Aset per situs|Ukuran|Dipakai di|
|---|---|---|
|Hero|1080 × 1350 (4:5)|Kartu info (P6)|



Page 12 of 14 

Design Document — Jejak Inderasakti 

|Aset per situs|Ukuran|Dipakai di|
|---|---|---|
|Thumbnail|512 × 512|Pin peta, Kartu Warisan (P9)|
|Ikon|96 × 96|StageHeader|



Format SVG dioptimalkan dengan SVGO (≤ 150 KB per file) + WebP cadangan. Warna mengikuti palet situs di bagian Identitas Visual. 

## Aksesibilitas & Checklist Aset 

Aset kunci untuk integrasi harus sampai ke FE paling lambat Minggu 20/9 (versi 1) dan Senin 21/9 (final); sisanya menyusul tanpa memblokir uji Kamis. 

### **Aksesibilitas.** 

1. Kontras teks minimal 4,5:1 (teks ≥ 18 px tebal minimal 3:1); teks di atas kuning selalu warna `tinta` . 

2. Benar/salah tidak hanya lewat warna: selalu ada ikon centang/silang; opsi jawaban punya motif + huruf A–D. 

3. Ukuran font memakai rem sehingga ikut pengaturan HP; tidak ada teks di dalam gambar. 

4. Hasil jawaban diumumkan lewat `aria-live` ; layar host bisa dioperasikan dengan keyboard. 

5. Hormati `prefers-reduced-motion` dan sediakan tombol mute. 

|Aset|Jumlah|Format|Ukuran|Tenggat|
|---|---|---|---|---|
|Uji palet hangat di 3 HP murah,<br>termasuk di bawah sinar<br>matahari|9|Hex|—|Sab 19/9|
|Logo Jejak Inderasakti (+ versi<br>horizontal)|2|SVG|—|Sab 19/9|
|Ikon motif opsi jawaban|4|SVG|64×64|Sab 19/9|
|Tekstur kertas (tile)|1|WebP|512×512, ≤ 40<br>KB|Sab 19/9|



Page 13 of 14 

Design Document — Jejak Inderasakti 

|Aset|Jumlah|Format|Ukuran|Tenggat|
|---|---|---|---|---|
|Mask tepi sobek + zigzag struk<br>+ perforasi karcis|3|SVG|—|Min 20/9|
|Selotip washi (5 warna situs)|5|SVG|160×48|Min 20/9|
|Maskot Sakti pose statis (gaya<br>stiker)|6|SVG|512×512|Min 20/9|
|Ilustrasi hero situs (kolase<br>kertas)|5|SVG +<br>WebP|1080×1350|Min 20/9 (v1), Sen<br>21/9 (final)|
|Peta kertas lipat Pulau<br>Penyengat|1|SVG|1080×1920|Min 20/9|
|Avatar|12|SVG|256×256|Min 20/9|
|Ikon UI (centang, silang, timer,<br>mute, bahasa, QR)|±12|SVG|24×24|Min 20/9|
|Cap/prangko situs + cap<br>BENAR/COBA LAGI|7|SVG|256×256|Sen 21/9|
|Thumbnail + ikon situs|10|SVG|512×512,<br>96×96|Sen 21/9|
|Lottie maskot (idle, benar, salah,<br>menang)|4|JSON|≤ 60 KB|Sen 21/9|
|Efek suara (klik, benar, salah,<br>tik, pindah stage, fanfare)|6|MP3 +<br>OGG|Total ≤ 150<br>KB|Sen 21/9|
|Favicon & ikon aplikasi|1 set|PNG|192, 512|Sel 22/9|
|Poster QR room|1|PDF|A4|Rab 23/9|



Page 14 of 14 

