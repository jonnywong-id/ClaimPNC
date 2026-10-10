-- Kueri modal "Kirim ke RCL/PUCL" — local action `KomentarRCLPUCL`
-- (`Section/SectionPUCL-sect.xml`).
--
-- ============================================================================
-- KENAPA TIDAK ADA MIGRASI BARU DI BELAKANG BERKAS INI
-- ============================================================================
--
-- Karena penampungnya SUDAH ADA. `POOLDATA.TC_PNC_PUCL` (34 kolom, dibaca dari katalog
-- 2026-10-04) sudah menjadi sumber ketiga tab layar Inbox RCL/PUCL, dan kolomnya cukup
-- untuk seluruh isian modal ini:
--
--   isian modal                 properti Pega                                kolom
--   --------------------------  -------------------------------------------  -------------------
--   Pilih RCL / PUCL            .ClaimData.PUCLStatus.RCL_PUCL               RCL_PUCL
--   Catatan untuk RCL/PUCL      .ClaimData.PUCLStatus.KomentarAnalisator     KOMENTAR_ANALISATOR
--   Perihal                     .ClaimData.PUCLStatus.Perihal                PERIHAL
--   Keterangan Pembuka          .ClaimData.PUCLStatus.Keterangan1            KETERANGAN1
--   Keterangan Isi              .ClaimData.PUCLStatus.Keterangan2            KETERANGAN2
--   Keterangan Penutup          .ClaimData.PUCLStatus.Keterangan3            KETERANGAN3
--   Nama Dokter                 .ClaimData.NamaDokterRCL                     — (lihat di bawah)
--
-- DUA KOLOM PEMILIK, DAN KEDUANYA BERBEDA ARTI:
--
--   OPERATOR_ID           analis yang menekan Kirim — catatan "siapa mengirim"
--   ASSIGNED_OPERATOR_ID  PIC Teknik klaim          — penyaring "siapa menerima"
--
-- Yang kedua adalah satu-satunya penyaring kepemilikan layar Inbox RCL
-- (`inboxrcl/repo/sqlstore/inboxrcl.sql`, penyaring A). Mengisinya dengan analis — bentuk
-- yang berlaku sampai 2026-10-06 — menaruh klaim berjalur RCL di Inbox RCL analis sendiri.
-- Work Owner menetapkan user teknis sebagai pemiliknya (2026-10-07); aturan lengkapnya di
-- `registrasi.PUCLLetter.AssignedOperator`, termasuk kenapa analis tetap menjadi cadangan
-- ketika klaim belum punya PIC Teknik.
--
-- Sempat dikira keempat isian surat hidup hanya di blob objek kerja Pega. Modul
-- `inboxrclpucl` membuktikan sebaliknya pada 2026-10-01: kolomnya ada dan TERISI, dan
-- isinya cocok kata demi kata dengan layar Pega.
--
-- ============================================================================
-- NAMA DOKTER — SEJAK 2026-10-07 DITULIS KE TABEL INI JUGA
-- ============================================================================
--
-- Work Owner melaporkan `NAMA_DOKTER_RCL` belum terisi di `TC_PNC_PUCL` pada jalur RCL dan
-- Notification. Kolomnya kini ikut ditulis `surat_rclpucl_perbarui`/`_sisip` (`:25`).
--
-- Catatan di bawah ditulis ketika tabel ini BELUM punya kolomnya, dan dipertahankan karena
-- ia menjelaskan kenapa `T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1` masih ikut ditulis — bukan
-- karena masih dibaca, melainkan karena belum ada keputusan mencabutnya.
--
-- YANG BERUBAH DI ANTARANYA: Inbox RCL tidak lagi membaca `T_CLAIMLIST_ADMIN` sama sekali
-- (Work Owner 2026-10-05; modul itu kini membaca `TC_PNC_PUCL`). Sejak saat itu
-- `NAMADOKTERRCL_1` **tidak dibaca satu kueri pun** di aplikasi ini — dicek dengan
-- pencarian menyeluruh pada seluruh berkas `.sql`. Menulis ke sana saja karena itu berarti
-- menyimpan nama dokter ke tempat yang tidak pernah dilihat siapa pun, dan itulah gejala
-- yang dilaporkan.
--
-- ============================================================================
-- CATATAN LAMA: KETIKA TABEL INI BELUM PUNYA KOLOM NAMA DOKTER
-- ============================================================================
--
-- `TC_PNC_PUCL` tidak punya kolom untuknya. Penampungnya
-- `T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1` — ditambahkan migrasi 0012 justru untuk layar
-- Inbox RCL, yang menyaring dengan kolom itu.
--
-- Isian ini hanya tampil pada jalur RCL lini PA, dan jalur itu berakhir di tahap
-- RCLDokter, bukan di antrean RCL/PUCL. Menulisnya ke baris daftar kerja karena itu
-- BUKAN penyimpangan: ia memang milik layar yang membaca baris itu.
--
-- ============================================================================
-- KEPEMILIKAN TABEL (`P-1`)
-- ============================================================================
--
-- `TC_PNC_PUCL` dimiliki Pega selama masa paralel. Penyisipan dari sini DIBATASI pada
-- klaim `PNCN.*`, sama seperti pembatasan yang sudah berlaku pada `T_CLAIMLIST_ADMIN`
-- (lihat `inboxentry.sql`). Baris Pega — berkunci nomor `PNC-xxxx` — tidak pernah
-- tersentuh, dan pembatasan itu ditegakkan pemanggil, bukan oleh kueri ini.

-- name: surat_rclpucl_perbarui
-- Surat klaim yang sudah pernah dikirim ke RCL/PUCL.
--
-- KUNCINYA `CLAIMID` SAJA — SATU BARIS PER KLAIM, BUKAN RIWAYAT.
--
-- Dibaca dari katalog 2026-10-04: `TC_PNC_PUCL_PK` adalah primary key atas `CLAIMID`
-- tunggal, bukan atas (CLAIMID, TGL_CREATE_PUCL) seperti yang sempat dikira di sini.
-- Akibatnya klaim yang kembali dari RCL/PUCL lalu dikirim lagi TIDAK boleh menyisipkan
-- baris kedua: Oracle menolaknya dengan ORA-00001, dan penolakan itu akan muncul kepada
-- analis sebagai kegagalan tanpa sebab yang dapat ia perbaiki.
--
-- Karena itu UPDATE lebih dulu, INSERT hanya bila belum ada — pola yang sama dengan
-- `inboxentry.sql` (keputusan 2.9: MERGE bukan sintaks yang sama antara Oracle dan
-- PostgreSQL).
--
-- `TGL_CREATE_PUCL` IKUT diperbarui: bagi layar Inbox RCL/PUCL kolom itu adalah urutan
-- antrean ("Tanggal Masuk Inbox" bersama TGL_KIRIM_PUCL), bukan tanggal lahir barisnya.
-- Klaim yang dikirim ulang memang masuk antrean lagi hari itu.
--
-- Yang TIDAK disentuh saat memperbarui: `KOMENTAR_PUCL`, `MSIG`, dan
-- `TGL_TERIMA_DOKUMEN_PUCL`. Ketiganya milik petugas RCL/PUCL, dan menimpanya dari sini
-- akan menghapus pekerjaan orang lain.
--
-- ============================================================================
-- DUA KOLOM YANG JUSTRU WAJIB DIKOSONGKAN — DAN KENAPA ITU BUKAN PENGECUALIAN
-- ============================================================================
--
-- `PUCL_APPROVE` dan `TGL_CETAK_DOKUMEN_PUCL` sempat ikut dalam daftar "tidak disentuh"
-- di atas, dengan alasan yang sama. Alasan itu benar untuk pengiriman PERTAMA dan salah
-- untuk pengiriman ULANG, karena keduanya bukan catatan milik petugas melainkan **penanda
-- posisi klaim di dalam satu siklus** — dan pengiriman ulang memulai siklus baru:
--
--   PUCL_APPROVE = '1'            "PUCL sudah selesai, klaim dikembalikan ke Analyst"
--   TGL_CETAK_DOKUMEN_PUCL        "surat siklus ini sudah dicetak"
--
-- Membawa keduanya ke siklus baru menghasilkan baris yang berbunyi *sudah selesai dan
-- sudah bersurat* padahal suratnya baru saja ditulis ulang — `PERIHAL` dan ketiga
-- `KETERANGAN*` di atas menimpa isi siklus sebelumnya.
--
-- AKIBATNYA KLAIM TIDAK TERLIHAT DI SATU TAB PUN, tanpa satu pun galat:
--
--   tab 1 Cetak Surat          TGL_CETAK_DOKUMEN_PUCL IS NULL      -> terisi, gugur
--   tab 2 Kelengkapan Dokumen  PUCL_APPROVE <> '1'                 -> '1',    gugur
--   tab 3 Klaim MSIG           MSIG = 'MSIG'                       -> NULL,   gugur
--
-- Terukur pada `PNCN.26.31` (2026-10-05): "Kirim Ke Analyst" 06:36 menyetel
-- `PUCL_APPROVE = '1'` lewat `inboxrclpucl` `return_to_analyst`, lalu pengiriman ulang
-- 08:18 memperbarui suratnya tanpa mencabut penanda itu. Klaimnya hilang dari ketiga tab
-- sementara tugasnya berjalan normal di `rcl-dokter` — persis kelas kegagalan senyap yang
-- `usecase/list.go` sebut "terlihat dua kali dapat diperbaiki, hilang sama sekali tidak".
--
-- Dikosongkan menjadi `NULL`, bukan diisi nilai lain, supaya hasil pengiriman ulang SAMA
-- PERSIS dengan hasil pengiriman pertama: `surat_rclpucl_sisip` di bawah tidak menulis
-- kedua kolom ini sama sekali.
--
-- Keduanya ditulis sebagai LITERAL, bukan penanda bind, sehingga urutan `:1`–`:25` tidak
-- bergeser dan daftar argumen di `rclpucl.go` tetap satu untuk UPDATE dan INSERT.
UPDATE POOLDATA.TC_PNC_PUCL
   SET TGL_CREATE_PUCL      = :1,
       OPERATOR_ID          = :2,
       ASSIGNED_OPERATOR_ID = :3,
       STATUS_WORK          = :4,
       STATUS_CASE          = :5,
       RCL_PUCL             = :6,
       KOMENTAR_ANALISATOR  = :7,
       PERIHAL              = :8,
       KETERANGAN1          = :9,
       KETERANGAN2          = :10,
       KETERANGAN3          = :11,
       POLICY_NO            = :12,
       QQ_NAME              = :13,
       BUSINESS_NAME        = :14,
       BRANCH_NAME          = :15,
       SOB_NAME             = :16,
       GROUPPANEL           = :17,
       USER_TEKNIS          = :18,
       STATUS_KLAIM         = :19,
       DATE_OF_LOSS         = :20,
       TGL_KIRIM_PUCL       = :21,
       ID_OBJECT            = :22,
       ID_COVERAGE          = :23,
       ID_ADJUSTMENT        = :24,
       NAMA_DOKTER_RCL      = :25,
       PUCL_APPROVE           = NULL,
       TGL_CETAK_DOKUMEN_PUCL = NULL
 WHERE CLAIMID = :26

-- name: surat_rclpucl_sisip
-- Surat klaim yang belum pernah dikirim ke RCL/PUCL. Dijalankan hanya bila UPDATE di
-- atas tidak mengenai satu baris pun.
--
-- Tiga nilai yang menempatkan baris di tab "Cetak Surat":
--
--   STATUS_WORK             'New'       — bukan 'Resolved-Completed' yang disaring keluar
--   STATUS_CASE             '0'         — nilai yang diterima penyaring tab itu
--   TGL_CETAK_DOKUMEN_PUCL  TIDAK diisi — milik tombol "Download Dokumen" di layar itu
--
-- STATUS_CASE DIKIRIM KOSONG PADA JALUR RCL, DAN ITU DISENGAJA.
--
-- Klaim berjalur RCL singgah di layar Inbox RCL lebih dulu (keputusan Work Owner
-- 2026-10-05), sehingga ia tidak boleh serentak muncul di antrean RCL/PUCL. Suratnya tetap
-- ditulis utuh — Perihal dan ketiga keterangan yang diketik analis tidak hilang — tetapi
-- tanpa STATUS_CASE penyaring tab itu melewatkannya, dan kedua tab lain pun melewatkannya
-- karena TGL_CETAK_DOKUMEN_PUCL kosong.
--
-- Kolomnya VARCHAR2(10 CHAR) dan NULLABLE (docs/ddl/tc_pnc_pucl.sql), jadi mengirim NULL
-- bukan pelanggaran batasan. Tidak ada nilai baru yang dikarang, dan tidak satu pun kueri
-- modul inboxrclpucl perlu disunting.
--
-- `STATUS_WORK` memakai kosakata PEGA ('New'), bukan kosakata proses aplikasi ini
-- ('BERJALAN'). Kolomnya sepadan dengan `T_CLAIMLIST_ADMIN.PYSTATUSWORK`, dan kedua baris
-- yang ada di tabel ini pun berbunyi 'New'. Menulis 'BERJALAN' akan menjadikan baris kita
-- satu-satunya yang berkosakata lain — penyaring `<> 'Resolved-Completed'` memang tetap
-- meloloskannya, tetapi penyaring `= 'New'` mana pun akan melewatkannya diam-diam.
--
-- KETIGA KOLOM `ID_*` DIISI SAAT KIRIM, bukan ditinggalkan untuk langkah berikutnya, dan
-- memakai kunci yang sama persis dengan T_CLAIM_ADJUSTMENT (`settlement.sql`): ID objek,
-- nomor urut jaminan, nomor urut baris. Klaim yang belum punya adjustment — tombol ini ada
-- sejak tahap Choose Surveyor — tetap membawa objek dan jaminan terakhirnya; hanya
-- ID_ADJUSTMENT yang dibiarkan kosong, karena menulis angka di sana akan menunjuk baris
-- yang tidak ada.
--
-- Ia menunjuk adjustment yang menjadi pokok surat —
-- `PUCLPost` menerimanya sebagai `idObj`, `idCov`, `idAdj` (terbaca dari konfigurasi tombol
-- "Kirim Ke Analyst") dan memakainya mencari baris yang distempel. Asal nilainya dan satu
-- asimetri di Pega sendiri ditulis lengkap pada `registrasi.PUCLLetter`.
--
-- Tidak satu pun rule di export MENULIS ke tabel ini — satu-satunya penyebutnya
-- `Database/CREATE_TABLE_3.SQL`, dan berkas itu bahkan hanya mendefinisikan 26 dari 34
-- kolom yang berjalan. Ketiga `ID_*` termasuk yang tidak ada di sana. Jadi yang diikuti
-- adalah properti clipboard yang dibaca `PUCLPost`, bukan pernyataan INSERT mana pun.
--
-- `LAMA_KLAIM`, `PUCL_APPROVE`, `MSIG`, `TGL_SELESAI_RI`, `TGL_TERIMA_DOKUMEN_PUCL`,
-- `KOMENTAR_PUCL`, dan `PNC_STATUS` tetap TIDAK diisi: seluruhnya milik langkah sesudah
-- surat ini terbit, dan menebak nilainya akan membuat layar berikutnya menampilkan angka
-- yang tampak benar tetapi tidak berasal dari mana pun.
INSERT INTO POOLDATA.TC_PNC_PUCL (
       CLAIMID, TGL_CREATE_PUCL, OPERATOR_ID, ASSIGNED_OPERATOR_ID,
       STATUS_WORK, STATUS_CASE, RCL_PUCL,
       KOMENTAR_ANALISATOR, PERIHAL, KETERANGAN1, KETERANGAN2, KETERANGAN3,
       POLICY_NO, QQ_NAME, BUSINESS_NAME, BRANCH_NAME, SOB_NAME,
       GROUPPANEL, USER_TEKNIS, STATUS_KLAIM, DATE_OF_LOSS, TGL_KIRIM_PUCL,
       ID_OBJECT, ID_COVERAGE, ID_ADJUSTMENT, NAMA_DOKTER_RCL)
VALUES (:1, :2, :3, :4,
        :5, :6, :7,
        :8, :9, :10, :11, :12,
        :13, :14, :15, :16, :17,
        :18, :19, :20, :21, :22,
        :23, :24, :25, :26)

-- name: surat_rclpucl_dokter
-- Kedua penyaring layar Inbox RCL, beserta kolom "Deskripsi Analyst" yang digambar di sana.
--
-- Dijalankan terpisah dari penyisipan surat, dan HANYA pada jalur RCL — tetapi pada jalur
-- itu SELALU. Baris daftar kerja klaim ini sudah ada sejak klaim diregistrasi
-- (`inboxentry.sql`), jadi ini UPDATE, bukan upsert.
--
-- KETIGANYA DITULIS SEKALIGUS KARENA INBOX RCL MENUNTUT DUA DI ANTARANYA BERSAMAAN:
--
--   NAMADOKTERRCL_1          penyaring D — dicocokkan dengan identitas pemanggil
--   TANGGALANALYSTSENDRCL_1  penyaring C — wajib IS NOT NULL, sekaligus kolom
--                            "Tanggal Masuk Inbox"
--   KOMENTARANALISATOR_1     kolom "Deskripsi Analyst"
--
-- Menulis yang satu tanpa yang lain membuat klaim berjalur RCL tidak muncul di inbox mana
-- pun, tanpa satu pun galat — itulah sebabnya keduanya tidak lagi bergantung pada apakah
-- isian "Nama Dokter" kebetulan terisi. Isian itu hanya tampil pada lini PA; pada lini lain
-- nilainya datang dari pemilik tugas RCLDokter (lihat usecase.SendToRCLPUCL).
UPDATE POOLDATA.T_CLAIMLIST_ADMIN
   SET NAMADOKTERRCL_1         = :1,
       KOMENTARANALISATOR_1    = :2,
       TANGGALANALYSTSENDRCL_1 = :3
 WHERE PYID = :4

-- name: perihal_rclpucl
-- Pilihan "Perihal" — `POOLDATA.M_PERIHAL_RCLPUCL`, 12 baris.
--
-- Padanan `BrowsePerihalRCLPUCL_RD`, yang TIDAK ADA di export. Yang terbaca dari
-- `Activity/InputPerihalRCLPUCL_act-Act.xml` hanyalah nama RD-nya, kelas sumbernya
-- (`ASM-FW-GCNMFW-Int-M_PERIHAL_RCLPUCL`), dan bahwa yang disalin ke klaim adalah
-- `PERIHAL_NAME` — teksnya, bukan `ID_PERIHAL`.
--
-- PENYARING STS_STATUS ADALAH PENYIMPULAN, BUKAN BUKTI.
--
-- Isi masternya terbelah dua dan belahannya sejalan dengan arti kedua jalur:
--
--   STS_STATUS = 1   "Tolakan klaim polis …"                        -> RCL
--   STS_STATUS = 2   "Kelengkapan Data dan Dokumen …"               -> PUCL
--                    "Pemberitahuan Penundaan Proses Klaim …"
--
-- Menolak klaim versus memintanya melengkapi dokumen adalah tepat perbedaan RCL dan
-- PUCL, sehingga pembelahan itu dipakai menyaring. Bila ternyata keliru, yang berubah
-- hanya baris WHERE ini.
--
-- :1 adalah kode jalur; :2 nilai STS_STATUS yang sepadan dengannya.
SELECT ID_PERIHAL, PERIHAL_NAME
  FROM POOLDATA.M_PERIHAL_RCLPUCL
 WHERE (:1 IS NULL OR STS_STATUS = :2)
 ORDER BY ID_PERIHAL

-- name: alasan_reject
-- Grid alasan penolakan — `POOLDATA.M_REASON_REJECT_REPRO`, 2.116 baris.
--
-- Padanan `BrowseReasonReject_RD`, yang juga TIDAK ADA di export. Yang terbaca dari
-- section hanyalah ketiga kolom yang digambar: `.REASON_ID`, `.REASON_NAME`,
-- `.REASON_DESC`.
--
-- TABELNYA MENYIMPAN SELURUHNYA SEBAGAI SATU CLOB JSON.
--
-- Hanya `REASON_ID` yang berupa kolom; sisanya ada di `JSONDATA`, berkunci `REASON_ID`,
-- `REASON_NAME`, `REASON_DESC`, dan tujuh kunci lain yang tidak dipakai grid ini.
-- `JSON_VALUE` dipakai membacanya — fungsi SQL/JSON standar yang sama sintaksnya di
-- Oracle 12c+ dan PostgreSQL 17+ (`D-20`, `D-24`), sehingga kueri ini tetap portabel.
--
-- BATAS BARIS WAJIB. 2.116 baris tidak boleh dikirim seluruhnya ke layar; grid Pega-nya
-- pun berpaginasi (`pyGridPaginator`). Pencarian dijalankan di basis data, bukan di
-- layar, supaya yang melintas hanya yang benar-benar ditampilkan.
--
-- :1 penentu apakah pencarian aktif · :2 :3 pola pencarian · :4 jumlah baris
SELECT r.REASON_ID,
       JSON_VALUE(r.JSONDATA, '$.REASON_NAME') AS REASON_NAME,
       JSON_VALUE(r.JSONDATA, '$.REASON_DESC') AS REASON_DESC
  FROM POOLDATA.M_REASON_REJECT_REPRO r
 WHERE (:1 IS NULL
        OR UPPER(JSON_VALUE(r.JSONDATA, '$.REASON_NAME')) LIKE :2 ESCAPE '\'
        OR UPPER(r.REASON_ID) LIKE :3 ESCAPE '\')
 ORDER BY r.REASON_ID
 FETCH NEXT :4 ROWS ONLY

-- Kueri `dokter_rcl` DIHAPUS pada 2026-10-06, dan alasannya layak dibaca sebelum ada yang
-- menulisnya kembali.
--
-- Ia menarik identitas lama pada ketiga grup akses `POOLDATA.T_ACCESS_GROUP_PNC` sebagai
-- isi dropdown "Nama Dokter" (sel ke-7 `Section/SectionPUCL-sect.xml`). Sumber pilihan
-- sel itu tidak ada di export (`R-16`), sehingga isinya DITURUNKAN dari satu-satunya
-- tempat nilainya dipakai kembali: penyaring D `InboxRCLDokter_RD` mencocokkan
-- `T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1` dengan identitas lama pemanggil, jadi daftarnya
-- "pasti" himpunan identitas itu.
--
-- Property `NamaDokterRCL` yang diserahkan Work Owner membuktikan sebaliknya: dropdown-nya
-- `pyTableOption = PromptList` dengan `pyPromptTableList` berisi TEPAT DUA baris pada
-- property itu sendiri — tanpa kueri, tanpa tabel. Daftarnya kini ada di Go sebagai
-- `registrasi.RCLDoctorOptions`, bukan di sini.
--
-- Penyaring D Inbox RCL tidak ikut berubah; yang gugur hanya anggapan bahwa isi dropdown
-- seluas himpunan yang dapat dicocokkannya. Pega membatasinya pada dua orang.
