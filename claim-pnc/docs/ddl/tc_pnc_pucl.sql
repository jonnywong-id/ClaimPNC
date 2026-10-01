-- ============================================================================
-- TC_PNC_PUCL — tabel datar antrean RCL/PUCL (Oracle 19c)
-- ============================================================================
--
-- STATUS: DDL DITETAPKAN WORK OWNER (2026-09-30), sumbernya `Database/CREATE_TABLE_3.SQL`.
-- Berkas ini menyalin DDL itu apa adanya dan menambahkan pemetaannya ke Report Definition
-- Pega — bukan merancang ulang. Bila keduanya berselisih, `CREATE_TABLE_3.SQL` yang berlaku.
--
-- Pelaksanaan menempuh `D-63` — permintaan tertulis -> persetujuan Work Owner -> DBA -> uji
-- dengan menjalankan Pega dan Go bersamaan. Akun aplikasi tidak memiliki hak DDL. Portal ASM
-- lebih dulu.
--
-- ============================================================================
-- APA YANG DIGANTIKANNYA
-- ============================================================================
--
-- Ketiga tab layar Inbox RCL/PUCL (`MENU_ID 61`) hari ini membaca DUA tabel milik Pega dan
-- menggabungkannya:
--
--   DATAPEGA.PC_ASM_FW_GCNMFW_WORK   objek kerja klaim
--   DATAPEGA.PC_ASSIGN_WORKBASKET    penugasan antrean bersama 'RCLPUCL'
--
-- Sesudah tabel ini terisi, ketiganya membaca SATU tabel milik aplikasi ini, tanpa gabungan.
-- Polanya sama dengan `POOLDATA.T_CLAIMLIST_ADMIN` yang sudah dipakai Inbox Outstanding dan
-- Inbox RCL: satu baris datar per klaim.
--
-- Yang BERBEDA dari `T_CLAIMLIST_ADMIN`: tabel ini TIDAK menyalin nama kolom Pega. Nama
-- kolomnya ditetapkan Work Owner — akhiran `_1` dibuang, dan kata dipisah garis bawah.
--
-- Berkas yang berubah sesudah tabel ini ada:
--   claim-pnc/backend/internal/inboxrclpucl/repo/sqlstore/inboxrclpucl.sql
--     - list_cetak_surat · list_kelengkapan_dokumen · list_klaim_msig  -> tabel ini
--     - check_rclpucl · check_columns                                  -> tabel ini
--     - daily_report · detail                                          -> LIHAT §5
--
-- ============================================================================
-- 1. PEMETAAN — properti Pega -> kolom di sini
-- ============================================================================
--
-- Daftar di bawah adalah GABUNGAN `pyListFields` dan `pyFilters` ketiga Report Definition
-- yang memasok ketiga tab:
--
--   tab                   Report Definition                                isian  penyaring
--   --------------------- ------------------------------------------------ -----  -----------------------
--   Cetak Surat           Report Definition/InboxPUCL_RD-RD.xml              25   A AND B AND C AND D
--   Kelengkapan Dokumen   Report Definition/InboxPUCLCetakSurat_RD-RD.xml    25   A AND B AND C AND D AND E
--   Klaim MSIG            Report Definition/InboxMISG_RD-RD.xml              18   A AND B AND C AND D AND E
--
-- PERHATIKAN NAMA BERKASNYA TERTUKAR DENGAN NAMA TABNYA. `InboxPUCL_RD` memasok tab
-- **Cetak Surat**; `InboxPUCLCetakSurat_RD` — yang namanya justru menyebut Cetak Surat —
-- memasok tab **Kelengkapan Dokumen**. Itu keadaan di export, bukan salah tulis di sini.
--
-- Urutan baris mengikuti `pyListFields` `InboxPUCL_RD-RD.xml` apa adanya, supaya dapat
-- dicocokkan baris demi baris dengan XML-nya.
--
--    #  properti Pega                                  kolom Pega                  kolom di sini
--   --- --------------------------------------------- --------------------------- ----------------------
--    1  .Policy.PolicyNo                               POLICYNO                    POLICY_NO
--    2  .Policy.Quotation.BusinessName                 BUSINESSNAME                BUSINESS_NAME
--    3  .Policy.Quotation.BranchName                   BRANCHNAME                  BRANCH_NAME
--    4  .Policy.Quotation.SobName                      SOBNAME                     SOB_NAME
--    5  .pxCreateDateTime                              PXCREATEDATETIME            TGL_CREATE_PUCL
--    6  .pzInsKey                                      PZINSKEY                    — TIDAK DIBAWA, §3
--    7  .pyID                                          PYID                        CLAIMID          (PK)
--    8  .Policy.QQName                                 QQNAME                      QQ_NAME
--    9  .ClaimData.UserTeknis                          USERTEKNIS_1                USER_TEKNIS
--   10  .pyOrigUserID                                  PYORIGUSERID                OPERATOR_ID
--   11  .pxCreateDateTime                              = #5, KEMBAR di RD-nya sendiri
--   12  .pyStatusWork                                  PYSTATUSWORK                STATUS_WORK
--   13  .ClaimData.PNCStatus                           PNCSTATUS_1                 PNC_STATUS
--   14  .ClaimData.StatusClaim                         STATUSCLAIM_1               STATUS_CLAIM
--   15  .ClaimData.PUCLStatus.KomentarAnalisator       KOMENTARANALISATOR_1        KOMENTAR_ANALISATOR
--   16  AssignBasket.pxAssignedOperatorID              PXASSIGNEDOPERATORID        ASSIGNED_OPERATOR_ID
--   17  .ClaimData.PUCLStatus.KomentarPUCL             KOMENTARPUCL_1              KOMENTAR_PUCL
--   18  .ClaimData.PUCLStatus.RCL_PUCL                 RCL_PUCL_1                  RCL_PUCL
--   19  .ClaimData.PUCLStatus.LamaKlaim                LAMAKLAIM_1                 LAMA_KLAIM
--   20  .ClaimData.PUCLStatus.StatusKlaim              STATUSKLAIM_1               STATUS_KLAIM
--   21  .ClaimData.TanggalSelesaiRawatInap             TANGGALSELESAIRAWATINAP_1   TGL_SELESAI_RI
--   22  .ClaimData.DateOfLoss                          DATEOFLOSS_1                DATE_OF_LOSS
--   23  .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL  TANGGALCETAKDOKUMENPUCL_1   TGL_CETAK_DOKUMEN_PUCL
--   24  .ClaimData.PUCLStatus.TanggalKirimPUCL         TANGGALKIRIMPUCL_1          TGL_KIRIM_PUCL
--   25  .ClaimData.PUCLStatus.StatusCase               STATUSCASE_1                STATUS_CASE
--
-- DARI DUA RD LAIN — tidak ada di `InboxPUCL_RD`, tetapi WAJIB, karena keduanya MENYARING
-- tab Kelengkapan Dokumen dan Klaim MSIG:
--
--   .ClaimData.PUCLStatus.PUCLApprove              PUCLAPPROVE_1               PUCL_APPROVE
--   .ClaimData.PUCLStatus.MSIG                     MSIG_1                      MSIG
--
-- Tanpa keduanya, tab Kelengkapan Dokumen dan Klaim MSIG TIDAK DAPAT DIBEDAKAN satu sama
-- lain maupun dari klaim yang sudah disetujui PUCL.
--
-- DI LUAR ketiga RD — dipakai LAPORAN HARIAN di balik tombol ekspor tab Cetak Surat, yang
-- cabang keduanya mengambil seluruh klaim Personal Accident:
--
--   .ClaimData.GroupPanel                          GROUPPANEL_1                GROUPPANEL
--
-- Ia satu-satunya kolom yang ejaannya TIDAK memisahkan kata dengan garis bawah —
-- `GROUPPANEL`, bukan `GROUP_PANEL`. Ditulis begitu di `CREATE_TABLE_3.SQL` dan diikuti apa
-- adanya; kode Go harus menyebutnya persis begini.
--
-- ============================================================================
-- 2. SEMBILAN ISIAN RD YANG TIDAK DIGAMBAR — dibawa, dan itu disengaja
-- ============================================================================
--
-- `BUSINESS_NAME`, `BRANCH_NAME`, `SOB_NAME`, `PNC_STATUS`, `USER_TEKNIS`, `OPERATOR_ID`,
-- `STATUS_CLAIM`, `KOMENTAR_PUCL`, dan `TGL_SELESAI_RI` DIAMBIL ketiga Report Definition
-- tetapi TIDAK punya sel di section mana pun — grid-nya hanya menggambar sembilan kolom
-- (lihat `inboxrclpucl/tab.go`).
--
-- Kesembilannya tetap dibawa. Menambah kolom pada tabel yang sudah berisi menempuh `D-63`
-- sekali lagi — tiga pihak, dan uji Pega+Go bersamaan — sementara membawanya sekarang
-- berbiaya nol byte per baris selama kolomnya kosong.
--
-- YANG TIDAK BERUBAH: layar tetap menggambar sembilan kolom yang sama. Tabel ini memuat
-- lebih banyak daripada yang digambar, persis seperti Report Definition-nya.
--
-- ============================================================================
-- 3. TIGA KOLOM YANG TIDAK DIBAWA — dan akibatnya masing-masing
-- ============================================================================
--
-- KEDUA BUTIR PERTAMA DIKONFIRMASI WORK OWNER 2026-10-01: *"PXOBJCLASS memang dihapus, dan
-- PZINSKEY pakai CLAIMID saja."* Keduanya karena itu BUKAN pertanyaan terbuka — yang ditulis
-- di bawah adalah akibatnya, supaya tidak ditemukan belakangan sebagai kejutan.
--
-- a. `PZINSKEY` — KUNCI OBJEK KERJA PEGA.
--
--    Kuncinya `CLAIMID`, yaitu `PYID` (nomor case, "PNC-1865"). Dua tempat dulu memakai
--    PZINSKEY, dan keduanya SUDAH ditangani:
--
--      * Kueri `detail` kini mencari `p.CLAIMID = :1`, satu bind.
--      * Kedua subkueri turunannya mencapai anak klaim LEWAT `POOLDATA.T_CLAIM_PNC`, yang
--        memuat kedua bentuk: `CLAIMNO` sama dengan nomor case, `CLAIMID` sama dengan kunci
--        teknis.
--
--    PREFIX TIDAK DIRANGKAI SENDIRI, meski lebih pendek. `'ASM-FW-GCNMFW-WORK ' || …` benar
--    untuk klaim warisan dan GAGAL DIAM-DIAM untuk klaim ber-nomor `PNCN.YY.xxxx`, yang
--    `D-22` dan `D-71` bebaskan dari prefix itu. Membaca pasangannya dari tabel benar untuk
--    kedua bentuk, karena yang dibaca nilainya — bukan tebakan tentang bentuknya.
--
--    PERHATIKAN NAMA `CLAIMID` MEMUAT HAL YANG BERBEDA DI TIGA TEMPAT:
--
--      POOLDATA.TC_PNC_PUCL.CLAIMID          nomor case          "PNC-1865"
--      POOLDATA.T_CLAIM_OBJECTLIST.CLAIMID   kunci berprefix     "ASM-FW-GCNMFW-WORK PNC-1865"
--      POOLDATA.TC_PNC_KOMITE.CLAIMID        kunci berprefix     (= T_CLAIM_PNC.CLAIMID)
--
--    Menggabungkan keduanya langsung menghasilkan NOL BARIS tanpa satu pun galat. Itu
--    kegagalan senyap, dan tempat paling mungkin ia terjadi adalah kueri `detail`.
--
-- b. `PXOBJCLASS` — kelas objek kerja.
--
--    Ketiga kueri hari ini menyaringnya, dan itu bukan kehati-hatian berlebih: katalog
--    2026-09-28 menemukan `T_CLAIMLIST_ADMIN` BERCAMPUR — 872 baris `Work-PNC` dan 142 baris
--    `Work-ReceiveDocument`.
--
--    Tanpa kolom ini, pemisahannya berpindah menjadi TANGGUNG JAWAB PROSES PENGISI: ia hanya
--    boleh menyisipkan baris ber-`PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'`. Itu syarat yang
--    harus tertulis di proses pengisi, karena tabel ini tidak lagi dapat menegakkannya
--    sendiri.
--
--    INI SATU-SATUNYA AKIBAT PENGHAPUSAN ITU YANG MASIH MENUNTUT TINDAKAN. Kuerinya sudah
--    bersih; yang belum ada adalah proses pengisinya, dan syarat di atas wajib ikut
--    dituliskan saat proses itu dibuat. Bila ia lalai, berkas penerimaan dokumen akan muncul
--    di antrean klaim — tanpa satu pun galat, karena seluruh kolom yang digambar ada pada
--    keduanya.
--
-- c. `DIBUAT_PADA` / `DIUBAH_*` / `DIHAPUS_*` — jejak dan soft delete (`D-66`).
--
--    `D-66` melarang `DELETE` fisik pada data BERNILAI BISNIS. Tabel ini bukan itu: ia
--    SALINAN dari objek kerja Pega, yang tetap menjadi satu-satunya sistem pencatat selama
--    masa paralel. Baris yang hilang di sini dapat disalin ulang; tidak ada yang lenyap.
--
--    Itu berlaku SELAMA tabel ini hanya diisi salinan. Begitu ada satu isian yang lahir di
--    layar baru dan tidak ada padanannya di Pega, pembacaannya berubah dan keenam kolom itu
--    harus ditambahkan.
--
-- ============================================================================
-- 4. SATU BARIS PER KLAIM — DAN INI MENGUBAH PERILAKU. HARUS DISETUJUI.
-- ============================================================================
--
-- Kueri hari ini memakai `INNER JOIN` ke `PC_ASSIGN_WORKBASKET`, mengikuti gabungan dalam
-- Report Definition-nya. Akibatnya objek kerja yang punya DUA penugasan terbuka di antrean
-- yang sama MUNCUL DUA KALI di layar. Itu perilaku sistem lama apa adanya, dan
-- `inboxrclpucl.sql` sengaja tidak menggantinya dengan `EXISTS` justru karena mengubah
-- jumlah baris yang terlihat pengguna adalah selisih yang tidak diputuskan siapa pun.
--
-- `PRIMARY KEY (CLAIMID)` membuat satu klaim = satu baris. Duplikasi itu HILANG, dan
-- `ASSIGNED_OPERATOR_ID` hanya dapat menampung SATU antrean per klaim.
--
--   * Bila duplikasi tidak pernah terjadi di produksi, tidak ada yang berubah sama sekali.
--   * Bila pernah terjadi, jumlah baris pada tab yang bersangkutan BERKURANG, dan proses
--     pengisi harus memutuskan penugasan mana yang menang.
--
-- Ini selisih terhadap sistem lama, dan `D-54` menuntutnya dinyatakan di muka lalu disetujui
-- Work Owner secara tertulis — bukan ditemukan saat uji kesetaraan gerbang 1.
--
-- HITUNG DULU SEBELUM MEMUTUSKAN. Kueri ini menjawabnya dengan angka, bukan dugaan:
--
--   SELECT COUNT(*) AS KLAIM_BERPENUGASAN_GANDA
--     FROM (SELECT b.PXREFOBJECTKEY
--             FROM DATAPEGA.PC_ASSIGN_WORKBASKET b
--            WHERE b.PXASSIGNEDOPERATORID = 'RCLPUCL'
--              AND b.PXOBJCLASS = 'Assign-WorkBasket'
--            GROUP BY b.PXREFOBJECTKEY
--           HAVING COUNT(*) > 1);
--
-- Bila hasilnya 0, selisih ini hipotetis dan cukup dicatat. Bila lebih dari 0, ia nyata dan
-- proses pengisi butuh aturan pemenang yang tertulis.
--
-- ============================================================================
-- 5. YANG TIDAK DISELESAIKAN TABEL INI
-- ============================================================================
--
-- a. LAYAR KERJA (kueri `detail`) — lihat §3a. Ia dapat pindah, tetapi kuncinya harus
--    dibentuk kembali menjadi PZINSKEY lebih dulu. Kedua subkuerinya ke
--    `T_CLAIM_OBJECTLIST` dan `T_CLAIM_ADJUSTMENT` tetap di tempatnya — keduanya sudah
--    membaca tabel POOLDATA.
--
-- b. SEMBILAN ISIAN CLIPBOARD TETAP TIDAK TERBACA. NIK, Business Unit/Seksi, Perihal, ketiga
--    Keterangan, Email Tertanggung, Tanggal Kelengkapan Dokumen, dan daftar Tanggal terima
--    Dokumen adalah properti clipboard `unexposed` di Pega — tidak punya kolom di tabel mana
--    pun. Tabel ini tidak mengubah keadaan itu, dan kolomnya sengaja TIDAK dibuat: kolom
--    kosong yang tidak ada pengisinya lebih menyesatkan daripada ketiadaannya.
--
-- c. MENAMBAH TABEL TIDAK MENGISINYA. Ini pelajaran yang sudah dibayar sekali di
--    `docs/kolom-t-claimlist-admin.md` §H.2: tujuh tab Inbox tertahan bukan oleh DDL
--    melainkan oleh PROSES PENGISI tabel — kolomnya sudah ada, isinya tidak pernah ditulis.
--    Tabel ini akan KOSONG sampai ada proses yang menyalin baris dari
--    `PC_ASM_FW_GCNMFW_WORK` + `PC_ASSIGN_WORKBASKET`. Siapa yang menulis proses itu, dan
--    seberapa sering ia berjalan, BELUM DIPUTUSKAN.
--
--    Tiga syarat yang HARUS dipenuhi proses itu, seluruhnya berasal dari §3 dan §4:
--      1. hanya baris `PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'`   (§3b)
--      2. `CLAIMID` diisi `PYID`, BUKAN `PZINSKEY`              (§3a)
--      3. satu baris per klaim; aturan pemenang bila penugasan ganda (§4)
--
-- ============================================================================
-- 6. PANJANG KOLOM — sudah ditetapkan, tetapi belum diadu dengan katalog
-- ============================================================================
--
-- Panjang di bawah berasal dari `CREATE_TABLE_3.SQL` dan diikuti apa adanya. DDL
-- `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` sendiri tidak tersedia (`R-08`), sehingga belum ada yang
-- pernah membandingkan keduanya.
--
-- PANJANG YANG TERLALU PENDEK MEMOTONG DATA TANPA GALAT saat proses pengisi berjalan. Kueri
-- katalog di §8 menjawabnya dengan angka; jalankan sebelum tabel diisi, bukan sesudahnya.
-- ============================================================================


-- ============================================================================
-- 7. DDL — salinan `Database/CREATE_TABLE_3.SQL`
-- ============================================================================
--
-- Yang DIUBAH terhadap berkas aslinya, dan hanya ini:
--   * `;` ditambahkan sesudah `CREATE TABLE`. Aslinya tidak punya terminator sama sekali,
--     sehingga SQL*Plus membacanya menyatu dengan `CREATE INDEX` di bawahnya dan gagal.
--   * `/` sesudah `CREATE INDEX` diganti `;`, supaya kedua pernyataan diakhiri sama.
--   * keterangan kolom ditambahkan di kanan; tidak satu pun nama, tipe, atau panjang berubah.

CREATE TABLE POOLDATA.TC_PNC_PUCL
(
  CLAIMID                 VARCHAR2(50 CHAR)     NOT NULL,  -- .pyID — nomor case; BUKAN kunci berprefix, §3a
  TGL_CREATE_PUCL         TIMESTAMP(6)          NOT NULL,  -- .pxCreateDateTime; kunci urut pertama, DESC
  OPERATOR_ID             VARCHAR2(128 CHAR),              -- .pyOrigUserID — "Nama Admin"; tidak digambar grid
  STATUS_WORK             VARCHAR2(30 CHAR),               -- penyaring <> 'Resolved-Completed'
  ASSIGNED_OPERATOR_ID    VARCHAR2(128 CHAR),              -- nama ANTREAN BERSAMA, bukan nama orang; 'RCLPUCL'
  POLICY_NO               VARCHAR2(50 CHAR),               -- "No Polis"
  QQ_NAME                 VARCHAR2(255 CHAR),              -- "Nama Tertanggung"
  BUSINESS_NAME           VARCHAR2(100 CHAR),              -- tidak digambar grid ini
  BRANCH_NAME             VARCHAR2(100 CHAR),              -- tidak digambar grid ini
  SOB_NAME                VARCHAR2(100 CHAR),              -- tidak digambar grid ini
  GROUPPANEL              VARCHAR2(10 CHAR),               -- laporan harian saja; ejaan tanpa garis bawah, §1
  USER_TEKNIS             VARCHAR2(128 CHAR),              -- "Nama PIC Teknik"; tidak digambar grid ini
  PNC_STATUS              VARCHAR2(50 CHAR),               -- tidak digambar grid ini
  STATUS_CLAIM            VARCHAR2(10 CHAR),               -- kode status klaim 1134-1166; laporan harian
  DATE_OF_LOSS            TIMESTAMP(6),                    -- "Tanggal Kejadian" di layar kerja
  TGL_SELESAI_RI          TIMESTAMP(6),                    -- .ClaimData.TanggalSelesaiRawatInap; tidak digambar
  RCL_PUCL                VARCHAR2(5 CHAR),                -- jalur: 1 RCL, 2 PUCL, 3 Notification
  KOMENTAR_ANALISATOR     VARCHAR2(1500 CHAR),             -- "Deskripsi Analyst"
  KOMENTAR_PUCL           VARCHAR2(1500 CHAR),             -- layar kerja; tidak digambar grid
  TGL_KIRIM_PUCL          TIMESTAMP(6),                    -- "Tanggal Masuk Inbox"; penyaring laporan harian
  TGL_CETAK_DOKUMEN_PUCL  TIMESTAMP(6),                    -- "Tanggal Cetak Surat"; PENYARING ketiga tab
  LAMA_KLAIM              TIMESTAMP(6),                    -- "Lama Klaim" — BERISI TANGGAL, bukan lama klaim
  STATUS_CASE             VARCHAR2(10 CHAR),               -- penyaring tab Cetak Surat (= '0'); tidak digambar
  STATUS_KLAIM            VARCHAR2(10 CHAR),               -- "Status Kadaluarsa" di grid
  PUCL_APPROVE            VARCHAR2(5 CHAR),                -- penyaring tab 2 dan 3 (<> '1'); tidak digambar
  MSIG                    VARCHAR2(20 CHAR),               -- penanda jalur MSIG; nilainya literal 'MSIG'

  -- ---------------------------------------------------------------------------------
  -- EMPAT KOLOM BERIKUT SUDAH ADA DI TABEL YANG BERJALAN, TETAPI BELUM DI
  -- `Database/CREATE_TABLE_3.SQL`. Dibaca dari katalog pada 2026-10-01: berkas itu
  -- mendefinisikan 26 kolom, tabelnya punya 30.
  --
  -- Keempatnya menutup empat dari sembilan isian layar kerja yang selama ini bertanda
  -- "di clipboard Pega" — persis jalan keluar yang `keputusan-implementasi.md` §79.3
  -- sebut sebagai satu-satunya: propertinya diekspos menjadi kolom.
  --
  -- Selisih ini WAJIB ditutup sebelum portal lain dibuat dari berkas DDL itu: tanpa
  -- keempatnya, layar kerja gagal ORA-00904 pada klaim pertama yang dibuka.
  -- ---------------------------------------------------------------------------------
  PERIHAL                 VARCHAR2(255 CHAR),              -- "Perihal"; pilihan dari M_PERIHAL_RCLPUCL
  KETERANGAN1             VARCHAR2(4000 CHAR),             -- "Keterangan Pembuka"
  KETERANGAN2             VARCHAR2(4000 CHAR),             -- "Keterangan Isi"
  KETERANGAN3             VARCHAR2(4000 CHAR),             -- "Keterangan Penutup"

  CONSTRAINT TC_PNC_PUCL_PK PRIMARY KEY (CLAIMID)
);

-- Keempat kolom index adalah penyaring yang SAMA PERSIS pada ketiga tab, diurutkan dari yang
-- paling menyaring; `TGL_CREATE_PUCL DESC` ikut supaya pengurutan dan `OFFSET … FETCH NEXT`
-- dilayani index yang sama, tanpa mengurutkan ulang seluruh hasil.
--
-- `PXOBJCLASS` tidak ada di sini karena tidak ada di tabelnya — lihat §3b.

CREATE INDEX POOLDATA.TC_PNC_PUCL_IDX01 ON POOLDATA.TC_PNC_PUCL
(ASSIGNED_OPERATOR_ID, STATUS_WORK, TGL_CETAK_DOKUMEN_PUCL, "TGL_CREATE_PUCL" DESC);

-- Laporan harian menyaring rentang TGL_KIRIM_PUCL, dan cabang keduanya menyaring GROUPPANEL
-- TANPA gabungan antrean sama sekali. Index ini HANYA dibuat bila laporan harian ikut
-- dipindahkan — sebelum itu ia index tanpa pembaca.
--
--   CREATE INDEX POOLDATA.TC_PNC_PUCL_IDX02 ON POOLDATA.TC_PNC_PUCL
--   (TGL_KIRIM_PUCL, GROUPPANEL);


-- ---- Keterangan kolom yang NAMANYA MENYESATKAN ----------------------------
--
-- Hanya kolom yang namanya tidak menyatakan isinya. Memberi keterangan pada seluruh kolom
-- membuat yang penting tenggelam.

COMMENT ON TABLE POOLDATA.TC_PNC_PUCL IS
    'Tabel datar antrean RCL/PUCL (MENU_ID 61); pengganti gabungan PC_ASM_FW_GCNMFW_WORK + PC_ASSIGN_WORKBASKET untuk ketiga tabnya';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.CLAIMID IS
    'Nomor case (.pyID), misal PNC-1865. BUKAN kunci berprefix — T_CLAIM_OBJECTLIST.CLAIMID dan TC_PNC_KOMITE.CLAIMID memuat ASM-FW-GCNMFW-WORK + nomor. Menggabungkannya langsung menghasilkan nol baris tanpa galat';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.LAMA_KLAIM IS
    'BERISI TANGGAL kirim proses PUCL, bukan lamanya klaim. Judul "Lama Klaim" menyesatkan sejak di Pega dan dibawa apa adanya (D-13)';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.STATUS_KLAIM IS
    'Digambar sebagai "Status Kadaluarsa" di layar ini. Layar Inbox Manager Receive/PUCL memakai judul yang sama untuk STATUS_CASE — kolom yang BERBEDA';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.RCL_PUCL IS
    'Digambar sebagai "Status RCL/PUCL". Kode: 1 RCL, 2 PUCL, 3 Notification. Penerjemahannya di lapisan domain Go, bukan di SQL';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.ASSIGNED_OPERATOR_ID IS
    'Nama ANTREAN BERSAMA, bukan nama orang — begitulah Pega mengisi pxAssignedOperatorID pada penugasan workbasket. Nilainya RCLPUCL';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.PUCL_APPROVE IS
    'Penanda PUCL: 0 dikirim ke PUCL, 1 dikembalikan ke Analyst. Penyaring <> 1 TIDAK menangkap NULL — perilaku Pega apa adanya, jangan diperbaiki sepihak';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.MSIG IS
    'Penanda jalur MSIG; nilainya literal MSIG. Terisi sangat jarang — hitungan 2026-09-30: 1 baris dari 7.722';

COMMENT ON COLUMN POOLDATA.TC_PNC_PUCL.GROUPPANEL IS
    'Kode Group Panel; 002 Personal Accident. Tidak dipakai ketiga tab — hanya laporan harian';

-- Hak akun aplikasi mengikuti tabel TC_PNC_* lain: SELECT, INSERT, UPDATE. Nama akun tidak
-- ditulis di sini.


-- ============================================================================
-- 8. JALANKAN SEBELUM TABEL DIISI — mengadu panjang kolom dengan sumbernya (§6)
-- ============================================================================
--
-- Hasilnya dibandingkan dengan §7 kolom demi kolom. Setiap kolom sumber yang LEBIH PANJANG
-- daripada tujuannya akan terpotong diam-diam saat proses pengisi berjalan.
--
--   SELECT column_name, data_type, data_length, char_length, nullable
--     FROM all_tab_columns
--    WHERE owner = 'DATAPEGA'
--      AND table_name = 'PC_ASM_FW_GCNMFW_WORK'
--      AND column_name IN ('PYID','PXCREATEDATETIME','PYORIGUSERID','PYSTATUSWORK',
--                          'POLICYNO','QQNAME','BUSINESSNAME','BRANCHNAME','SOBNAME',
--                          'GROUPPANEL_1','USERTEKNIS_1','PNCSTATUS_1','STATUSCLAIM_1',
--                          'DATEOFLOSS_1','TANGGALSELESAIRAWATINAP_1','RCL_PUCL_1',
--                          'KOMENTARANALISATOR_1','KOMENTARPUCL_1','TANGGALKIRIMPUCL_1',
--                          'TANGGALCETAKDOKUMENPUCL_1','LAMAKLAIM_1','STATUSCASE_1',
--                          'STATUSKLAIM_1','PUCLAPPROVE_1','MSIG_1')
--    ORDER BY column_name;
--
--   SELECT column_name, data_type, data_length, char_length
--     FROM all_tab_columns
--    WHERE owner = 'DATAPEGA'
--      AND table_name = 'PC_ASSIGN_WORKBASKET'
--      AND column_name IN ('PXASSIGNEDOPERATORID','PXREFOBJECTKEY')
--    ORDER BY column_name;
--
-- DUA KEMUNGKINAN YANG HARUS DITANGANI, BUKAN DIABAIKAN:
--
--   * Sebuah kolom TIDAK MUNCUL di hasil. Berarti ia properti clipboard `unexposed`, bukan
--     kolom — sama seperti kesembilan isian di §5b. `TANGGALSELESAIRAWATINAP_1` dan
--     `PNCSTATUS_1` yang paling mungkin. Kolomnya tetap ada di tabel ini, tetapi proses
--     pengisi harus membacanya dari blob objek kerja, dan itu pekerjaan yang berbeda.
--
--   * Sebuah kolom bertipe DATE, bukan TIMESTAMP. Menyimpan DATE ke TIMESTAMP tidak gagal,
--     tetapi membuat kedua tabel tidak dapat dibandingkan langsung saat uji kesetaraan
--     gerbang 1.


-- ============================================================================
-- 9. ROLLBACK (P-4) — dijalankan DBA HANYA bila perubahan ini harus ditarik
-- ============================================================================
--
--   DROP INDEX POOLDATA.TC_PNC_PUCL_IDX01;
--   DROP TABLE POOLDATA.TC_PNC_PUCL;
--
-- AMAN SELAMA TABEL INI BELUM DIPAKAI. Ia tabel baru yang tidak dibaca satu pun rule Pega
-- dan tidak menyentuh tabel lama mana pun, sehingga menariknya tidak menghilangkan data
-- siapa pun — asalkan dua syarat berikut dipenuhi:
--
--   1. Kueri `inboxrclpucl.sql` sudah dikembalikan membaca `PC_ASM_FW_GCNMFW_WORK`. Bila
--      belum, ketiga tab gagal dengan ORA-00942 begitu tabelnya hilang.
--   2. Proses pengisi sudah dihentikan.
--
-- Selama tabel ini hanya berisi SALINAN dari objek kerja Pega (§3c), `DROP` tidak
-- menghilangkan apa pun yang tidak dapat disalin ulang. Begitu ada isian yang lahir di layar
-- baru, pembacaan itu tidak berlaku lagi.
