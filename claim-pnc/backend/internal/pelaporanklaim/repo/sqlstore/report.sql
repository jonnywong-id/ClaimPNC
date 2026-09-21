-- Kueri modul Pelaporan Klaim: POOLDATA.CPNC_LAPORAN_KLAIM.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap berbahasa Indonesia karena keduanya termasuk pengecualian `D-80` — ia milik basis
-- data, dan perubahannya menempuh `D-63`.
--
-- ============================================================================
-- PEMETAAN TIGA ARAH — properti Pega -> kolom lama -> kolom di sini
-- ============================================================================
--
-- Ini satu-satunya tempat ketiganya dapat dibandingkan berdampingan, dan ia ada karena
-- alias sistem lama MENYESATKAN SECARA AKTIF. Sumbernya
-- `RDB List/Rcv_ProcInsertRecivedDocument-SQL.xml:99-126` (pemetaan properti ke parameter)
-- dan `Database/PROCINSERTDATARECIVEDKLAIM.prc:2-28` (parameter ke kolom).
--
-- Properti klipboard        Kolom T_CLAIM_RECIVEDCLAIM  Kolom di sini
-- ------------------------- --------------------------- --------------------------
-- ReceiveDocument.Sender    NAMAPELAPOR                 NAMA_PELAPOR
-- .EmailPengirim            EMAILPENGIRIM               EMAIL_PENGIRIM
-- .TelpPengirim             TLPPENGIRIM                 TELEPON_PENGIRIM
-- .Kurir                    NAMAKURIRASM                NAMA_KURIR          (!)
-- .SubjectEmail             SUBJECTEMAIL                SUBJEK_EMAIL
-- .PolicyNo                 NOPOLIS                     NOMOR_POLIS
-- .QQName                   NAMATERTANGGUNG             NAMA_TERTANGGUNG    (!)
-- .EmailLOD                 EMAILTERTANGGUNG            EMAIL_TERTANGGUNG   (!)
-- Policy.Quotation.BusinessCode BUSINESSCODE            KODE_BISNIS
-- Policy.Quotation.GroupPanel   GROUPPANEL              GROUP_PANEL
-- Policy.BookNo             NOREFERENSI                 NOMOR_REFERENSI     (!)
-- .TglKejadian              DOL                         TANGGAL_KEJADIAN
-- .LokasiKejadian           LOKASIKEJADIAN              LOKASI_KEJADIAN
-- .KronologisKejadian       KRONOLOGIKEJADIAN           KRONOLOGI
-- .RincianKerusakan         RINCIANKERUSAKAN            RINCIAN_KERUSAKAN
-- .SIM                      SIMPENGENDARA               SIM_PENGENDARA
-- .Keterangan               ALASANBLMTRANSFER           ALASAN_BELUM_TRANSFER   (!)
-- .NotRegistNote            KETERANGANBLMREGIST         CATATAN_BELUM_REGISTRASI
-- .ReceivedDate             TANGGALTERIMADOKUMEN        TANGGAL_TERIMA_DOKUMEN  (!)
-- .PNCCaseID                NOKLAIM                     NOMOR_KLAIM
-- .DateOfSendASM            TRANSFERASM                 TANGGAL_TRANSFER
-- .DateOfRegistrcv          REGISTDATE                  TANGGAL_REGISTRASI      (!)
-- .KodeCabang               KODECABANG                  KODE_CABANG
-- pxCreateOperator          USERINPUT                   DIINPUT_OLEH            (!)
-- pxCreateDateTime          TANGGALINPUTDOKUMEN         DIINPUT_PADA
-- .StatusLock               — (hanya di blob Pega)      DITRANSFER              (!)
-- .Estimasi                 — (hanya di blob Pega)      NILAI_ESTIMASI
-- .TypeOfClaim              — (hanya di blob Pega)      TIPE_KLAIM
-- .NumberOfDocument         — (hanya di blob Pega)      JUMLAH_DOKUMEN
--
-- Tanda (!) menandai pemetaan yang namanya berbohong. Enam yang paling berbahaya, karena
-- pembacanya akan yakin sudah mengerti padahal salah:
--
--   TampunganPages.TelpTertanggung          -> USERINPUT     : petugas penginput,
--                                                              BUKAN telepon tertanggung
--   TampunganPages.TanggalSelesaiRawatInap  -> REGISTDATE    : tanggal registrasi,
--                                                              BUKAN tanggal pulang rawat
--   TampunganPages.NamaSurveyor             -> NAMAKURIRASM  : nama kurir, BUKAN surveyor
--   TampunganPages.UserTeknis               -> NAMATERTANGGUNG : nama tertanggung,
--                                                              BUKAN petugas teknis
--   TampunganPages.LokasiSurveyor           -> LOKASIKEJADIAN : lokasi kejadian
--   TampunganPages.ReferenceId              -> TANGGALTERIMADOKUMEN : sebuah TANGGAL,
--                                                              bukan nomor referensi
--
-- `StatusLock` juga bukan seperti namanya: ia BUKAN kunci baris melainkan penanda bahwa
-- laporan sudah dikirim ke ASM pusat. Yang benar-benar mengunci layar adalah
-- `StatusLockFile`, yang diturunkan darinya oleh `Activity/Pre_ActReceiveDocument-Act.xml`.
--
-- Penamaan ulang ini melaksanakan `03-CURRENT-ARCHITECTURE.md` §4.2 dan `D-19`.
--
--
-- ============================================================================
-- KENAPA TABEL BARU, DAN BUKAN T_CLAIM_RECIVEDCLAIM
-- ============================================================================
--
-- Keputusan Work Owner 2026-09-18: `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` tidak dipakai lagi,
-- dan tabel baru dibuatkan.
--
-- Itu memang satu-satunya jalan yang tersisa. Data laporan di sistem lama hidup di DUA
-- tempat, dan keduanya tidak dapat dipakai:
--
--   1. Header case-nya — nomor laporan, penanda transfer, nomor klaim, aging — ada di
--      `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, tabel milik ENGINE Pega yang dibaca 116 rule.
--      `P-1` melarang dua sistem menulis satu tabel, dan tabel itu tetap ditulis Pega
--      untuk seluruh case type lain yang belum bermigrasi.
--   2. Rinciannya ada di `POOLDATA.T_CLAIM_RECIVEDCLAIM`, yang penulis tunggalnya memang
--      terbukti — hanya `PROCINSERTDATARECIVEDKLAIM` yang menyentuhnya, dan hanya
--      `RDB List/Rcv_ProcInsertRecivedDocument-SQL.xml` yang memanggilnya. Tetapi ia
--      HANYA MEMUAT SEBAGIAN: penanda transfer, estimasi, tipe klaim, dan jumlah dokumen
--      tidak punya kolom di sana. Memakainya berarti tetap kehilangan empat field.
--
-- Satu catatan yang perlu diketahui sebelum tabel lama ditinggalkan: penelusuran seluruh
-- export TIDAK menemukan satu pun rule yang MEMBACA `T_CLAIM_RECIVEDCLAIM`. Ia
-- write-only sejauh yang terlihat. Bila ada pembaca di luar export — laporan BI,
-- perkakas cabang — pembaca itu akan berhenti menerima baris baru begitu modul ini
-- menyala. Itu pertanyaan untuk Work Owner dan DBA, bukan sesuatu yang dapat dijawab
-- dari export.
--
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding —
-- tidak ada satu pun perangkaian teks SQL, dan itu yang menutup pola `{ASIS:...}` yang
-- muncul TIGA KALI pada setiap kueri inbox lama
-- (`RDB List/ViewTableBrowseRCVInProcess-SQL.xml` dan dua saudaranya).


-- name: report_list
--
-- # Penyaring tanpa merangkai SQL
--
-- Setiap penyaring memakai pola `:n IS NULL OR ...`. Penyaring yang tidak diisi menjadi
-- NULL dan tidak mempersempit apa pun, sehingga SATU kueri melayani seluruh gabungan
-- penyaring tanpa satu pun potongan teks yang dirangkai.
--
-- # Tahap dihitung, tidak disimpan
--
-- Ekspresi CASE di bawah adalah SATU-SATUNYA definisi tahap di sisi SQL, dan ia harus
-- sama persis dengan ClaimReport.Stage di Go. Menyimpan tahap sebagai kolom akan membuat
-- dua sumber kebenaran yang dapat berselisih tanpa ada yang menyadarinya.
--
-- Urutan pemeriksaannya penting: hasil klaim diperiksa LEBIH DULU, karena laporan yang
-- klaimnya sudah selesai tetap punya NOMOR_KLAIM.
--
-- # Pencarian
--
-- Cocokkan tanpa peduli besar-kecil huruf ke lima field yang dipakai orang mencari.
-- UPPER dipakai, bukan LOWER, karena keduanya portabel dan UPPER yang dipakai indeks
-- fungsional pada migrasi 0003. Tanda ESCAPE membuat `%` dan `_` yang diketik pengguna
-- dicari apa adanya alih-alih menjadi wildcard.
--
-- # Paginasi
--
-- OFFSET ... FETCH NEXT, bukan ROWNUM. `09-DATABASE-STRATEGY.md` §3.3 menetapkannya, dan
-- ketiga kueri inbox lama memakai ROWNUM yang tidak portabel ke PostgreSQL.
SELECT NOMOR,
       NAMA_PELAPOR,
       EMAIL_PENGIRIM,
       TELEPON_PENGIRIM,
       NAMA_KURIR,
       SUBJEK_EMAIL,
       NOMOR_POLIS,
       NAMA_TERTANGGUNG,
       EMAIL_TERTANGGUNG,
       KODE_BISNIS,
       GROUP_PANEL,
       NOMOR_REFERENSI,
       TANGGAL_KEJADIAN,
       LOKASI_KEJADIAN,
       KRONOLOGI,
       RINCIAN_KERUSAKAN,
       SIM_PENGENDARA,
       NILAI_ESTIMASI,
       TIPE_KLAIM,
       JUMLAH_DOKUMEN,
       TANGGAL_TERIMA_DOKUMEN,
       NOMOR_KLAIM,
       DITRANSFER,
       TANGGAL_TRANSFER,
       TANGGAL_REGISTRASI,
       ALASAN_BELUM_TRANSFER,
       CATATAN_BELUM_REGISTRASI,
       HASIL_KLAIM,
       KODE_CABANG,
       DIINPUT_OLEH,
       DIINPUT_PADA,
       DIUBAH_PADA
  FROM POOLDATA.CPNC_LAPORAN_KLAIM
 WHERE (:1 IS NULL
        OR :2 = CASE
                  WHEN HASIL_KLAIM = 'DIAKSEPTASI' THEN 'SUDAH_AKSEPTASI'
                  WHEN HASIL_KLAIM = 'DITOLAK'     THEN 'DITOLAK'
                  WHEN NOMOR_KLAIM IS NOT NULL     THEN 'SUDAH_REGISTRASI'
                  WHEN DITRANSFER = 1              THEN 'BELUM_REGISTRASI'
                  ELSE 'BELUM_TRANSFER'
                END)
   AND (:3 IS NULL OR UPPER(KODE_CABANG) = UPPER(:4))
   AND (:5 IS NULL
        OR UPPER(NOMOR)            LIKE '%' || UPPER(:6)  || '%' ESCAPE '\'
        OR UPPER(NOMOR_KLAIM)      LIKE '%' || UPPER(:7)  || '%' ESCAPE '\'
        OR UPPER(NOMOR_POLIS)      LIKE '%' || UPPER(:8)  || '%' ESCAPE '\'
        OR UPPER(NAMA_TERTANGGUNG) LIKE '%' || UPPER(:9)  || '%' ESCAPE '\'
        OR UPPER(NAMA_PELAPOR)     LIKE '%' || UPPER(:10) || '%' ESCAPE '\')
 ORDER BY DIINPUT_PADA DESC, NOMOR DESC
OFFSET :11 ROWS FETCH NEXT :12 ROWS ONLY


-- name: report_count
--
-- Banyaknya baris yang cocok SEBELUM dipotong paginasi. Penyaringnya sama persis dengan
-- report_list — bila keduanya berbeda, layar akan menampilkan jumlah halaman yang tidak
-- pernah ada isinya.
SELECT COUNT(1)
  FROM POOLDATA.CPNC_LAPORAN_KLAIM
 WHERE (:1 IS NULL
        OR :2 = CASE
                  WHEN HASIL_KLAIM = 'DIAKSEPTASI' THEN 'SUDAH_AKSEPTASI'
                  WHEN HASIL_KLAIM = 'DITOLAK'     THEN 'DITOLAK'
                  WHEN NOMOR_KLAIM IS NOT NULL     THEN 'SUDAH_REGISTRASI'
                  WHEN DITRANSFER = 1              THEN 'BELUM_REGISTRASI'
                  ELSE 'BELUM_TRANSFER'
                END)
   AND (:3 IS NULL OR UPPER(KODE_CABANG) = UPPER(:4))
   AND (:5 IS NULL
        OR UPPER(NOMOR)            LIKE '%' || UPPER(:6)  || '%' ESCAPE '\'
        OR UPPER(NOMOR_KLAIM)      LIKE '%' || UPPER(:7)  || '%' ESCAPE '\'
        OR UPPER(NOMOR_POLIS)      LIKE '%' || UPPER(:8)  || '%' ESCAPE '\'
        OR UPPER(NAMA_TERTANGGUNG) LIKE '%' || UPPER(:9)  || '%' ESCAPE '\'
        OR UPPER(NAMA_PELAPOR)     LIKE '%' || UPPER(:10) || '%' ESCAPE '\')


-- name: report_summary
--
-- Jumlah laporan per tahap dalam SATU perjalanan ke basis data.
--
-- Bentuk ini meniru `RDB List/BrowseClaimRCV_Aksep-SQL.xml`, yang juga menghitung seluruh
-- lencana sekaligus — dan alasannya tetap berlaku: angka tiap tab harus konsisten satu
-- sama lain. Enam kueri terpisah dapat tiba di antara dua perubahan dan menghasilkan
-- lencana yang jumlahnya tidak cocok dengan isi tabelnya.
--
-- Penyaring tahap SENGAJA tidak ada di sini; angkanya dipakai seluruh tab.
SELECT CASE
         WHEN HASIL_KLAIM = 'DIAKSEPTASI' THEN 'SUDAH_AKSEPTASI'
         WHEN HASIL_KLAIM = 'DITOLAK'     THEN 'DITOLAK'
         WHEN NOMOR_KLAIM IS NOT NULL     THEN 'SUDAH_REGISTRASI'
         WHEN DITRANSFER = 1              THEN 'BELUM_REGISTRASI'
         ELSE 'BELUM_TRANSFER'
       END AS TAHAP,
       COUNT(1) AS JUMLAH
  FROM POOLDATA.CPNC_LAPORAN_KLAIM
 WHERE (:1 IS NULL OR UPPER(KODE_CABANG) = UPPER(:2))
   AND (:3 IS NULL
        OR UPPER(NOMOR)            LIKE '%' || UPPER(:4) || '%' ESCAPE '\'
        OR UPPER(NOMOR_KLAIM)      LIKE '%' || UPPER(:5) || '%' ESCAPE '\'
        OR UPPER(NOMOR_POLIS)      LIKE '%' || UPPER(:6) || '%' ESCAPE '\'
        OR UPPER(NAMA_TERTANGGUNG) LIKE '%' || UPPER(:7) || '%' ESCAPE '\'
        OR UPPER(NAMA_PELAPOR)     LIKE '%' || UPPER(:8) || '%' ESCAPE '\')
 GROUP BY CASE
            WHEN HASIL_KLAIM = 'DIAKSEPTASI' THEN 'SUDAH_AKSEPTASI'
            WHEN HASIL_KLAIM = 'DITOLAK'     THEN 'DITOLAK'
            WHEN NOMOR_KLAIM IS NOT NULL     THEN 'SUDAH_REGISTRASI'
            WHEN DITRANSFER = 1              THEN 'BELUM_REGISTRASI'
            ELSE 'BELUM_TRANSFER'
          END


-- name: report_get
SELECT NOMOR,
       NAMA_PELAPOR,
       EMAIL_PENGIRIM,
       TELEPON_PENGIRIM,
       NAMA_KURIR,
       SUBJEK_EMAIL,
       NOMOR_POLIS,
       NAMA_TERTANGGUNG,
       EMAIL_TERTANGGUNG,
       KODE_BISNIS,
       GROUP_PANEL,
       NOMOR_REFERENSI,
       TANGGAL_KEJADIAN,
       LOKASI_KEJADIAN,
       KRONOLOGI,
       RINCIAN_KERUSAKAN,
       SIM_PENGENDARA,
       NILAI_ESTIMASI,
       TIPE_KLAIM,
       JUMLAH_DOKUMEN,
       TANGGAL_TERIMA_DOKUMEN,
       NOMOR_KLAIM,
       DITRANSFER,
       TANGGAL_TRANSFER,
       TANGGAL_REGISTRASI,
       ALASAN_BELUM_TRANSFER,
       CATATAN_BELUM_REGISTRASI,
       HASIL_KLAIM,
       KODE_CABANG,
       DIINPUT_OLEH,
       DIINPUT_PADA,
       DIUBAH_PADA
  FROM POOLDATA.CPNC_LAPORAN_KLAIM
 WHERE NOMOR = :1


-- name: report_next_sequence
--
-- Nomor urut laporan berikutnya.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri, dengan
-- perlakuan yang sama seperti generator nomor klaim pada `ADR-0005` dan urutan master
-- status — satu-satunya tempat lain yang dibenarkan memuat percabangan dialek.
--
-- Urutan ini TIDAK direset tiap tahun. Akibatnya nomor urut menembus pergantian tahun,
-- dan segmen tahun pada nomor menjadi PENANDA, bukan penghitung per tahun:
-- `LPK.26.8125` diikuti `LPK.27.8126`. Itu perilaku yang sama dengan nomor klaim
-- (`D-71` pertanyaan terbuka nomor 1) — dipertahankan sengaja supaya keduanya tidak
-- berbeda aturan tanpa alasan.
SELECT POOLDATA.CPNC_LAPORAN_KLAIM_SEQ.NEXTVAL
  FROM DUAL


-- name: report_insert
INSERT INTO POOLDATA.CPNC_LAPORAN_KLAIM
       (NOMOR, NAMA_PELAPOR, EMAIL_PENGIRIM, TELEPON_PENGIRIM, NAMA_KURIR, SUBJEK_EMAIL,
        NOMOR_POLIS, NAMA_TERTANGGUNG, EMAIL_TERTANGGUNG, KODE_BISNIS, GROUP_PANEL,
        NOMOR_REFERENSI, TANGGAL_KEJADIAN, LOKASI_KEJADIAN, KRONOLOGI, RINCIAN_KERUSAKAN,
        SIM_PENGENDARA, NILAI_ESTIMASI, TIPE_KLAIM, JUMLAH_DOKUMEN,
        TANGGAL_TERIMA_DOKUMEN, NOMOR_KLAIM, DITRANSFER, TANGGAL_TRANSFER,
        TANGGAL_REGISTRASI, ALASAN_BELUM_TRANSFER, CATATAN_BELUM_REGISTRASI, HASIL_KLAIM,
        KODE_CABANG, DIINPUT_OLEH, DIINPUT_PADA, DIUBAH_PADA)
VALUES (:1, :2, :3, :4, :5, :6,
        :7, :8, :9, :10, :11,
        :12, :13, :14, :15, :16,
        :17, :18, :19, :20,
        :21, :22, :23, :24,
        :25, :26, :27, :28,
        :29, :30, :31, :32)


-- name: report_update
--
-- NOMOR, DIINPUT_OLEH, dan DIINPUT_PADA sengaja TIDAK ada di daftar SET. Ketiganya jejak
-- pencatatan yang tidak boleh berubah setelah laporan tercatat — dan yang tidak
-- disebutkan di sini tidak dapat diubah kode yang ditulis kemudian tanpa menyunting
-- berkas ini lebih dulu.
UPDATE POOLDATA.CPNC_LAPORAN_KLAIM
   SET NAMA_PELAPOR             = :1,
       EMAIL_PENGIRIM           = :2,
       TELEPON_PENGIRIM         = :3,
       NAMA_KURIR               = :4,
       SUBJEK_EMAIL             = :5,
       NOMOR_POLIS              = :6,
       NAMA_TERTANGGUNG         = :7,
       EMAIL_TERTANGGUNG        = :8,
       KODE_BISNIS              = :9,
       GROUP_PANEL              = :10,
       NOMOR_REFERENSI          = :11,
       TANGGAL_KEJADIAN         = :12,
       LOKASI_KEJADIAN          = :13,
       KRONOLOGI                = :14,
       RINCIAN_KERUSAKAN        = :15,
       SIM_PENGENDARA           = :16,
       NILAI_ESTIMASI           = :17,
       TIPE_KLAIM               = :18,
       JUMLAH_DOKUMEN           = :19,
       TANGGAL_TERIMA_DOKUMEN   = :20,
       NOMOR_KLAIM              = :21,
       DITRANSFER               = :22,
       TANGGAL_TRANSFER         = :23,
       TANGGAL_REGISTRASI       = :24,
       ALASAN_BELUM_TRANSFER    = :25,
       CATATAN_BELUM_REGISTRASI = :26,
       HASIL_KLAIM              = :27,
       KODE_CABANG              = :28,
       DIUBAH_PADA              = :29
 WHERE NOMOR = :30


-- name: report_check_table
--
-- Memastikan tabel migrasi 0003 sudah ada dan dapat dibaca akun aplikasi, tanpa mengambil
-- satu baris pun. Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak
-- mirip tetapi perbaikannya berbeda jauh: migrasi belum dijalankan DBA, versus akun
-- aplikasi tidak punya hak baca.
SELECT NOMOR,
       NAMA_PELAPOR,
       DITRANSFER,
       HASIL_KLAIM,
       DIINPUT_PADA
  FROM POOLDATA.CPNC_LAPORAN_KLAIM
 WHERE 1 = 0
