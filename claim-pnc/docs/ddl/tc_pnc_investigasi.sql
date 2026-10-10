-- ============================================================================
-- TC_PNC_INVESTIGASI — hasil investigasi klaim (Oracle 19c)
-- ============================================================================
--
-- STATUS: DIAJUKAN 2026-10-05, menunggu persetujuan Work Owner.
--
-- Pelaksanaan menempuh `D-63` — permintaan tertulis -> persetujuan Work Owner -> DBA -> uji
-- dengan menjalankan Pega dan Go bersamaan. Akun aplikasi tidak memiliki hak DDL. Portal ASM
-- lebih dulu.
--
-- CARA MENJALANKANNYA: TIGA pernyataan, berurutan, TIDAK dapat digabung menjadi satu —
--
--   1. CREATE TABLE POOLDATA.TC_PNC_INVESTIGASI
--   2. CREATE INDEX POOLDATA.IX_TC_PNC_INVESTIGASI_KLAIM   (menuntut (1) sudah ada)
--   3. COMMENT ON TABLE POOLDATA.TC_PNC_INVESTIGASI        (menuntut (1) sudah ada)
--
-- TIDAK ADA BARIS KOSONG DI DALAM SATU PERNYATAAN, dan itu disengaja: SQL*Plus baku
-- (`SET SQLBLANKLINES OFF`) memperlakukan baris kosong di tengah pernyataan sebagai akhir
-- pernyataan, lalu membuang sisanya. Baris kosong hanya boleh berada DI ANTARA ketiganya,
-- sebab `;` sudah menutup pernyataan sebelumnya. Jangan merapikan berkas ini dengan
-- menyisipkan baris kosong ke dalam `CREATE TABLE`.
--
-- Hak akses TIDAK disertakan di sini: tabel ini memuat data medis, dan `FR-R2` menuntut
-- hak aksesnya ditetapkan bersamaan dengan pembuatannya — menempuh Compliance, bukan DBA
-- sendirian.
--
--
-- ============================================================================
-- APA YANG DIGANTIKANNYA
-- ============================================================================
--
-- Menekan **Nomor Case** di Inbox Investigator (`MENU_ID 48`) menjalankan Flow Action
-- `InputInvestigator` — bukan tampilan, melainkan FORMULIR KERJA yang menyimpan:
--
--   Flow Action/InputInvestigator-FA.xml:128   pra-proses  PresetInvestigation
--   Flow Action/InputInvestigator-FA.xml:49    formulir    InputClaimInvestigasiDetail
--   Flow Action/InputInvestigator-FA.xml:144   simpan      SetStatusInvestigator_Act
--
-- Hari ini isian formulir itu disimpan Pega ke dalam BLOB objek kerja, lalu disalin ke
-- `POOLDATA.JSON_KLAIM.DATA_JSONBLOB` oleh `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc`.
--
-- Aplikasi ini MEMBACA dokumen itu — fitur Export Data Investigation sudah berjalan di
-- atasnya. Yang tidak boleh adalah MENULIS ke sana: `JSON_KLAIM` dimiliki Pega selama masa
-- paralel (`P-1`, `D-21`).
--
-- Tiga tabel yang sudah ada sudah diperiksa dan tidak satu pun dapat dipakai:
--
--   POOLDATA.T_SURVEYORLIST         tidak punya satu pun kolom investigasi; isinya hasil
--                                   survei milik kasus Work-SurveyClaim
--   POOLDATA.INVESTIGATIONREPORT    NOL rujukan dari rule Pega mana pun di seluruh export —
--                                   ia milik sistem lain
--   POOLDATA.JSON_KLAIM             ditulis Pega; lihat di atas
--
--
-- ============================================================================
-- 1. PEMETAAN — properti Pega -> kolom di sini
-- ============================================================================
--
-- Daftar di bawah dibaca PER SEL dari `Section/InputClaimInvestigasiDetail-Section.xml`,
-- bukan dari kedekatan baris: setiap sel `Embed-Display-Table-Cell` diambil `pyValue`,
-- `pyLabelFieldValue`, dan `pyFormat`-nya sendiri. Metode kedekatan baris sudah dicoba dan
-- menghasilkan pasangan yang jelas mustahil ("Tanggal Keluar" <- `NoRekapMedis`).
--
-- Seluruh baris di bawah berada di bawah `SurveyResults(1).SurveyList(1)`.
--
--    #  properti Pega                 kolom di sini                 kendali   nilai terukur
--   --- ----------------------------- ----------------------------- --------- ---------------
--    1  TanggalInvestigasi            TANGGAL_INVESTIGASI           DateTime  diisi otomatis
--    2  IsInvestigated                IS_INVESTIGATED               Radio     '1' / '0'
--    3  SelectRS                      SELECT_RS                     Radio     '1' / '0'
--    4  RSKlinikYangDisurvey_1        RS_KLINIK_DISURVEI            AutoCompl bila SELECT_RS=1
--    5  RSKlinikYangDiSurvey          RS_KLINIK_DISURVEI_LAIN       Text      bila SELECT_RS=0
--    6  AlamatRSKlinik                ALAMAT_RS_KLINIK              TextArea  —
--    7  NoRekapMedis                  NO_REKAP_MEDIS                Text      DATA MEDIS
--    8  NamaPasien                    NAMA_PASIEN                   Text      DATA NASABAH
--    9  DateOfBirth                   TANGGAL_LAHIR                 DateTime  DATA NASABAH
--   10  FlagDOB                       FLAG_DOB                      Radio     —
--   11  RemarksDOB                    KETERANGAN_DOB                TextArea  —
--   12  PasienTerdaftar               PASIEN_TERDAFTAR              Radio     '1' / '0'
--   13  PTReg                         PT_REG                        Text      —
--   14  TanggalPerawatan              TANGGAL_PERAWATAN             DateTime  DATA MEDIS
--   15  TanggalSelesaiPerawatan       TANGGAL_SELESAI_PERAWATAN     DateTime  bila rawat inap
--   16  TotalTagihan                  TOTAL_TAGIHAN                 Text      TEKS, bukan angka
--   17  TagihanLunasBelumLunas        TAGIHAN_LUNAS                 Radio     —
--   18  CheckBoxPasien                BAYAR_PASIEN                  Checkbox  'true'/'false'
--   19  CheckBoxPerusahaan            BAYAR_PERUSAHAAN              Checkbox  'true'/'false'
--   20  CheckBoxAsuransiLain          BAYAR_ASURANSI_LAIN           Checkbox  'true'/'false'
--   21  CheckBoxTidakadapembayaran    BAYAR_TIDAK_ADA               Checkbox  'true'/'false'
--   22  OtherInsurance                ASURANSI_LAIN                 Text      bila (19) atau (20)
--   23  KonfirmasiModelKwitansi       KONFIRMASI_KWITANSI           Radio     '1' / '0'
--   24  NamaPCRS                      NAMA_PIC_RS                   Text      "Nama PIC Yang
--                                                                            Dapat Dihubungi"
--   25  NamaPenelepon                 NAMA_PENELEPON                Text      baca-saja di kepala
--   26  NamaKaryawan                  NAMA_KARYAWAN                 Text      "Nama Karyawan"
--   27  KodeAreaTelp                  KODE_AREA_TELP                Text      —
--   28  NoTelpDiHubungi               NO_TELP_DIHUBUNGI             Text      —
--   29  Ekst                          EKSTENSI                      Text      —
--   30  Remaks                        REMAKS                        TextArea  Hasil Investigasi
--
-- Tipe `VARCHAR2` dipakai pada kolom yang di Pega bernilai '1'/'0' dan 'true'/'false',
-- BUKAN NUMBER maupun CHAR(1): nilainya memang teks di dokumen JSON hari ini, dan berkas
-- ekspor menuliskannya apa adanya (`P-5`). Mengubahnya menjadi angka di sini akan membuat
-- berkas baru tidak dapat dibandingkan baris per baris dengan berkas lama.
--
--
-- ============================================================================
-- 2. ENAM ISIAN BERSYARAT, DAN DUA YANG MATI
-- ============================================================================
--
-- Syarat tampilnya dibaca dari `pyCondition` pembungkus tiap sel:
--
--   SELECT_RS = '1'                        -> Alamat RS/Klinik/Lokasi Kejadian
--   SELECT_RS = 1                          -> Nama Rumah Sakit
--   SELECT_RS = 0                          -> Nama Tempat Lainnya
--   ClaimData.SelectRawatInap = true       -> Tanggal Keluar, Tanggal Selesai Perawatan
--   BAYAR_PERUSAHAAN atau BAYAR_ASURANSI_LAIN bernilai 'true'
--                                          -> Nama Asuransi/Perusahaan Lain
--
-- DUA sel ber-`pyCondition = 1=2`, yaitu SELALU TERSEMBUNYI. Keduanya isian
-- **"Akan Dikirim ke Analyst"** (`.ClaimData.UserTeknis`). Ia TIDAK dibangun — bukan karena
-- lingkup, melainkan karena di layar lama pun ia tidak pernah tampil.
--
--
-- ============================================================================
-- 3. YANG TIDAK DISIMPAN DI SINI
-- ============================================================================
--
-- Empat isian formulir TIDAK masuk tabel ini karena ia milik KLAIM, bukan milik baris
-- investigasi, dan seluruhnya sudah punya rumah:
--
--   .Policy.QQName                                      Nama Tertanggung  — T_CLAIM_PNC.QQNAME
--   .ClaimData.ObjectList(1).ObjectName                 Nama Peserta      — T_CLAIM_OBJECTLIST
--   .ClaimData.DateOfLoss                               Tanggal Masuk     — T_CLAIM_PNC.DATEOFLOSS
--   .ClaimData.ObjectList(1)...AdjustmentList(1).EstimationValue
--                                                       Total Pengajuan   — tabel adjustment
--
-- Keempatnya BACA-SAJA di formulir: ia menampilkan klaim yang sedang diinvestigasi.
--
-- TIGA isian lain milik klaim, dan KETIGANYA TIDAK PUNYA KOLOM DI MANA PUN. Diperiksa
-- langsung ke `ALL_TAB_COLUMNS` pada 2026-10-06 — bukan lagi dugaan:
--
--   .ClaimData.AnalystRemaksInvestigator   Pertanyaan/Alasan dari Analyst   TIDAK ADA
--   .ClaimData.TanggalSelesaiRawatInap     Tanggal Keluar (tingkat klaim)   TIDAK ADA *
--   .ClaimData.SelectRawatInap             penentu isian rawat inap         TIDAK ADA
--
--   * satu-satunya kemunculan namanya di seluruh skema adalah
--     LOG_PC_ASM_FW_GCNMFW_WORK.TANGGALSELESAIRAWATINAP_1 — kolom berakhiran `_1` pada
--     tabel LOG milik engine Pega, yang pada modul ini sudah terbukti selalu kosong.
--
-- Catatan pendukung: `T_CLAIM_PNC` kini punya 106 kolom, bukan 82 seperti yang tercatat
-- setelah pemangkasan 2026-09-26. Ia bertambah sejak itu; ketiga nama di atas tetap tidak
-- ada di antaranya.
--
-- AKIBATNYA: "Pertanyaan Dari Analyst" — yang terlihat di layar lama — BELUM DAPAT DIBANGUN.
-- Ia butuh penambahan kolom pada `T_CLAIM_PNC`, diminta terpisah lewat `D-63`, dan BUKAN
-- ditumpangkan ke tabel ini: ia milik klaim, bukan milik satu baris investigasi.
--
-- Selama `SelectRawatInap` belum ada, kedua isian rawat inap diperlakukan SELALU TAMPIL.
-- Menyembunyikan isian yang mungkin seharusnya terisi lebih merugikan daripada menampilkan
-- isian yang mungkin tidak perlu.
--
--
-- ============================================================================
-- 4. PERPINDAHAN STATUS — TIDAK memerlukan tabel baru
-- ============================================================================
--
-- `SetStatusInvestigator_Act` menulis:
--
--   SurveyResults(1).SurveyStatus := 5
--   PNCStatus                     := 5
--   ClaimData.StatusClaim         := '1151'   -> Analyst
--   ClaimData.InvestTfDate        := waktu kini
--   ClaimData.AnalystTransferDate := waktu kini
--
-- Seluruhnya bermuara ke `POOLDATA.T_CLAIM_PNC`, yang SUDAH ditulis aplikasi ini lewat
-- `klaim_perbarui`. Tidak ada kepemilikan baru yang perlu dinegosiasikan untuk bagian itu.
--
--
-- ============================================================================
-- 5. DATA SENSITIF
-- ============================================================================
--
-- Tabel ini memuat DATA MEDIS — nomor rekam medis, tanggal perawatan, nama pasien, tanggal
-- lahir. `FR-R2` membatasi aksesnya, dan `D-64` menetapkan salinan produksi di staging
-- TIDAK disamarkan. Hak akses tabel ini karena itu ditetapkan BERSAMAAN dengan
-- pembuatannya, bukan sesudahnya.
--
-- ============================================================================


CREATE TABLE POOLDATA.TC_PNC_INVESTIGASI (
    -- Kunci. KLAIM_ID adalah pzInsKey klaim, sama dengan T_CLAIM_PNC.CLAIMID.
    -- URUTAN_SURVEI dan URUTAN adalah indeks page list SurveyResults(n).SurveyList(m);
    -- keduanya ikut supaya satu klaim dapat punya lebih dari satu baris investigasi,
    -- persis seperti di Pega. Hari ini keduanya selalu 1.
    KLAIM_ID                  VARCHAR2(100)   NOT NULL,
    URUTAN_SURVEI             NUMBER(5)       DEFAULT 1 NOT NULL,
    URUTAN                    NUMBER(5)       DEFAULT 1 NOT NULL,
    -- Isian formulir, urut sesuai §1.
    TANGGAL_INVESTIGASI       DATE,
    IS_INVESTIGATED           VARCHAR2(1),
    SELECT_RS                 VARCHAR2(1),
    RS_KLINIK_DISURVEI        VARCHAR2(200),
    RS_KLINIK_DISURVEI_LAIN   VARCHAR2(200),
    ALAMAT_RS_KLINIK          VARCHAR2(500),
    NO_REKAP_MEDIS            VARCHAR2(50),
    NAMA_PASIEN               VARCHAR2(200),
    TANGGAL_LAHIR             DATE,
    FLAG_DOB                  VARCHAR2(1),
    KETERANGAN_DOB            VARCHAR2(500),
    PASIEN_TERDAFTAR          VARCHAR2(1),
    PT_REG                    VARCHAR2(200),
    TANGGAL_PERAWATAN         DATE,
    TANGGAL_SELESAI_PERAWATAN DATE,
    -- VARCHAR2, BUKAN NUMBER. Isiannya `pxTextInput` di layar lama dan tersimpan sebagai
    -- TEKS di dokumen JSON hari ini — nilai seperti "1.500.000" akan menggagalkan konversi
    -- diam-diam bila kolomnya numerik. Perhitungan uang tidak dilakukan atas kolom ini.
    TOTAL_TAGIHAN             VARCHAR2(50),
    TAGIHAN_LUNAS             VARCHAR2(1),
    BAYAR_PASIEN              VARCHAR2(5),
    BAYAR_PERUSAHAAN          VARCHAR2(5),
    BAYAR_ASURANSI_LAIN       VARCHAR2(5),
    BAYAR_TIDAK_ADA           VARCHAR2(5),
    ASURANSI_LAIN             VARCHAR2(200),
    KONFIRMASI_KWITANSI       VARCHAR2(1),
    NAMA_PIC_RS               VARCHAR2(200),
    NAMA_PENELEPON            VARCHAR2(200),
    NAMA_KARYAWAN             VARCHAR2(200),
    KODE_AREA_TELP            VARCHAR2(10),
    NO_TELP_DIHUBUNGI         VARCHAR2(50),
    EKSTENSI                  VARCHAR2(10),
    REMAKS                    VARCHAR2(2000),
    -- Jejak. DIHAPUS_PADA adalah soft delete: `D-66` melarang penghapusan fisik data
    -- bernilai bisnis, dan setiap kueri pembaca WAJIB menyaringnya.
    DIBUAT_OLEH               VARCHAR2(100),
    DIBUAT_PADA               TIMESTAMP,
    DIUBAH_OLEH               VARCHAR2(100),
    DIUBAH_PADA               TIMESTAMP,
    DIHAPUS_PADA              TIMESTAMP,
    CONSTRAINT PK_TC_PNC_INVESTIGASI
        PRIMARY KEY (KLAIM_ID, URUTAN_SURVEI, URUTAN)
);

-- Pembacaan selalu per klaim dan selalu menyaring baris terhapus. Index ini yang
-- melayaninya tanpa memindai tabel.
CREATE INDEX POOLDATA.IX_TC_PNC_INVESTIGASI_KLAIM
    ON POOLDATA.TC_PNC_INVESTIGASI (KLAIM_ID, DIHAPUS_PADA);

COMMENT ON TABLE POOLDATA.TC_PNC_INVESTIGASI IS
    'Hasil investigasi klaim PNC. MEMUAT DATA MEDIS (FR-R2). Menggantikan SurveyResults(1).SurveyList(1).* pada BLOB objek kerja Pega.';
