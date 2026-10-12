-- Kueri modul Input Req Protection: permintaan pembukaan proteksi.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- ============================================================================
-- PEMETAAN KOLOM — properti Pega -> kolom sebenarnya -> kolom layar
-- ============================================================================
--
-- Kolom di bawah DIBACA DARI KATALOG Oracle, bukan disalin dari usulan. Tabelnya sudah
-- empat kali dibentuk ulang Work Owner, dan yang berlaku selalu bentuk terakhirnya.
-- Bentuk yang berlaku sejak 2026-09-24:
--
--   Properti Pega        Kolom                  Kolom layar
--   -------------------- ---------------------- ------------------------
--   .pyID                OPEN_PROTECTION_ID     No Proteksi
--   .PolicyNo            POLICY_NO              No Polis
--   .CaseID              CLAIM_NO               No Klaim
--   .PNCCaseID           ID_CLAIM               tidak ditampilkan
--   .TypeProtection      PROTECTION_TYPE_ID     Tipe Proteksi
--   .InputDate           CREATE_DATE            Tanggal Proteksi Dibuat
--   .Keterangan          NOTES                  Keterangan
--   .AcceptStatus        APPROVAL_STATUS        penyaring inti
--   .pxCreateOpName      CREATED_BY             User Create
--   .ObjectName          OBJECT_NAME            Object Name (form)
--   .BranchName          BRANCH_NAME            Branch Name (form)
--
-- Tiga kolom berganti nama pada revisi ini — `ID` -> `OPEN_PROTECTION_ID`,
-- `PROTECTION_TYPE` -> `PROTECTION_TYPE_ID`, `RESOLVED_DATE_TIME` -> `RESOLVED_DATETIME`.
-- Yang terakhir tidak disentuh berkas ini; ia milik `inboxacceptopenprotection` (`P-1`).
--
-- `CREATE_DATE` memikul DUA peran sekaligus — tanggal proteksi dibuat dan waktu baris
-- dibuat. Tabel hanya punya satu kolom waktu pembuatan, dan sistem lama pun tidak
-- membedakan keduanya.
--
-- ============================================================================
-- NAMA TIPE DATANG DARI MASTER, BUKAN DARI KODE
-- ============================================================================
--
-- `POOLDATA.M_CLAIM_PROTECTION_TYPE` diterima 2026-09-24 berisi kesembilan tipe beserta
-- namanya. Sebelum itu layar menampilkan kodenya apa adanya karena label '1', '3', '4',
-- '5', '6', dan '9' tidak ada di export mana pun (`R-16`).
--
-- Join-nya **LEFT**, dan itu bukan kelonggaran. Kode yang tidak ada di master tetap harus
-- tampil: INNER JOIN akan MENGHILANGKAN barisnya dari inbox — tanpa galat, tanpa gejala,
-- dan justru pada baris yang paling perlu diperiksa manusia.
--
-- Pembandingnya dibungkus TRIM di KEDUA sisi. Keduanya `VARCHAR2(2)` tanpa penyeragaman
-- apa pun, dan satu spasi di ujung akan membuat seluruh nama tipe hilang.
--
-- ============================================================================
-- SETIAP KEMUNCULAN BIND BERNOMOR SENDIRI
-- ============================================================================
--
-- Driver mengikat argumen menurut urutan KEMUNCULAN penanda, bukan menurut nomornya.
-- Memakai penanda yang sama dua kali lalu mengirim satu argumen menghasilkan
-- **ORA-01008: not all variables bound** — galat yang sama yang pernah menimpa modul Inbox
-- Outstanding dan diperbaiki dengan cara yang sama.
--
-- Karena itu nilai yang dipakai berkali-kali diberi nomor berbeda dan DIKIRIM BERULANG dari
-- Go. Lihat searchArgs di protection.go.
--
-- Pola pencarian pun dibentuk DI GO, bukan dirangkai di SQL: teks yang memuat tanda persen
-- atau garis bawah akan menjadi wildcard tanpa disengaja, dan pengguna yang mencari nomor
-- polis bertanda itu akan menerima hasil yang bukan miliknya.
--
-- ============================================================================
-- OLD_DATA / NEW_DATA: SEPASANG KOLOM, DUA ARTI
-- ============================================================================
--
-- Keduanya menyimpan "nilai sebelum" dan "nilai sesudah" dari apa yang diminta berubah,
-- dan artinya ditentukan PROTECTION_TYPE_ID:
--
--   tipe '7'  OLD_DATA = Current Date Of Loss     NEW_DATA = Next Date Of Loss
--   tipe '8'  OLD_DATA = Cause Of Loss sebelumnya NEW_DATA = Cause Of Loss dipilih
--
-- Bentuknya meniru `POOLDATA.T_OPENPROTECTION.OLDATA/NEW_DATA`, yang memakai pola yang sama
-- untuk tipe proteksinya sendiri.
--
-- TANGGAL DISIMPAN SEBAGAI TEKS `YYYY-MM-DD`, bukan DATE. Itu konsekuensi dari sepasang
-- kolom yang melayani dua tipe sekaligus, dan akibatnya disadari: "tampilkan permintaan
-- yang mengubah DOL ke bulan September" TIDAK dapat dijawab SQL. Ketiga layar yang dibangun
-- tidak menyaring maupun mengurutkan berdasarkan isi ini — ia hanya ditampilkan.
--
-- ============================================================================
-- APPROVAL_STATUS: NULL, BUKAN TEKS KOSONG
-- ============================================================================
--
-- Kueri daftar menyaring `APPROVAL_STATUS IS NULL`, meniru
-- `Report Definition/InboxReqOpenProtection_RD-RD.xml` apa adanya. Baris yang menyimpan
-- teks kosong TIDAK cocok dengan penyaring itu dan hilang dari inbox tanpa satu pun galat.
--
-- Sejak 2026-09-23 basis data ikut menjaganya lewat `T_CLAIM_OPENPROT_CK_STATUS`, yang
-- membolehkan hanya NULL, '1', dan '2'.
--
-- ============================================================================
-- SELURUH KUERI MENYARING STATUS_ACTIVE
-- ============================================================================
--
-- `D-66` menetapkan tidak ada penghapusan fisik; penghapusan dinyatakan lewat penanda.
-- Kolomnya ber-DEFAULT '1' (aktif). Satu kueri yang lupa menyaringnya akan menampilkan
-- baris yang seharusnya hilang — kelas cacat baru yang tidak ada di sistem lama
-- (`09-DATABASE-STRATEGY.md` §8.1). Dijaga uji di query_test.go.


-- name: protection_count
-- Jumlah seluruh permintaan yang cocok, untuk keterangan dan paginasi layar.
--
-- TIDAK ikut men-join master: yang dihitung barisnya, dan nama tipe tidak mengubah
-- jumlahnya. Join yang tidak dipakai hanya menambah kerja basis data.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( :1 IS NULL
         OR UPPER(OPEN_PROTECTION_ID) LIKE :2
         OR UPPER(POLICY_NO)          LIKE :3
         OR UPPER(CLAIM_NO)           LIKE :4 )


-- name: protection_list
-- Satu halaman permintaan yang belum diakseptasi.
--
-- Diurutkan MENURUN mengikuti RD rujukan. Nomor proteksi ikut menjadi kunci urut kedua
-- supaya urutannya tetap sama pada dua pemanggilan dengan waktu pembuatan identik — tanpa
-- itu, paginasi dapat menampilkan satu baris dua kali dan melewatkan baris lain.
--
-- `OFFSET … FETCH NEXT` dipakai, bukan `ROWNUM` (`09-DATABASE-STRATEGY.md` §4).
SELECT p.OPEN_PROTECTION_ID,
       p.POLICY_NO,
       p.CLAIM_NO,
       p.ID_CLAIM,
       p.PROTECTION_TYPE_ID,
       t.PROTECTION_TYPE_NAME,
       p.CREATE_DATE,
       p.NOTES,
       p.APPROVAL_STATUS,
       p.CREATED_BY,
       p.OLD_DATA,
       p.NEW_DATA,
       p.OBJECT_NAME,
       p.BRANCH_NAME,
       p.OBJECT_ID,
       p.OBJECT_COVERAGE_ID
  FROM POOLDATA.T_CLAIM_OPENPROTECTION p
  LEFT JOIN POOLDATA.M_CLAIM_PROTECTION_TYPE t
         ON TRIM(t.PROTECTION_TYPE_ID) = TRIM(p.PROTECTION_TYPE_ID)
 WHERE p.APPROVAL_STATUS IS NULL
   AND (p.STATUS_ACTIVE IS NULL OR TRIM(p.STATUS_ACTIVE) = '1')
   AND ( :1 IS NULL
         OR UPPER(p.OPEN_PROTECTION_ID) LIKE :2
         OR UPPER(p.POLICY_NO)          LIKE :3
         OR UPPER(p.CLAIM_NO)           LIKE :4 )
 ORDER BY p.CREATE_DATE DESC, p.OPEN_PROTECTION_ID DESC
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY


-- name: protection_get
-- Satu permintaan menurut nomornya.
--
-- TIDAK menyaring APPROVAL_STATUS: form harus tetap dapat dibuka untuk permintaan yang baru
-- saja diakseptasi, supaya pesannya dapat menjelaskan apa yang terjadi — bukan sekadar
-- "tidak ditemukan".
SELECT p.OPEN_PROTECTION_ID,
       p.POLICY_NO,
       p.CLAIM_NO,
       p.ID_CLAIM,
       p.PROTECTION_TYPE_ID,
       t.PROTECTION_TYPE_NAME,
       p.CREATE_DATE,
       p.NOTES,
       p.APPROVAL_STATUS,
       p.CREATED_BY,
       p.OLD_DATA,
       p.NEW_DATA,
       p.OBJECT_NAME,
       p.BRANCH_NAME,
       p.OBJECT_ID,
       p.OBJECT_COVERAGE_ID
  FROM POOLDATA.T_CLAIM_OPENPROTECTION p
  LEFT JOIN POOLDATA.M_CLAIM_PROTECTION_TYPE t
         ON TRIM(t.PROTECTION_TYPE_ID) = TRIM(p.PROTECTION_TYPE_ID)
 WHERE UPPER(TRIM(p.OPEN_PROTECTION_ID)) = :1
   AND (p.STATUS_ACTIVE IS NULL OR TRIM(p.STATUS_ACTIVE) = '1')


-- name: protection_type_list
-- Seluruh tipe proteksi beserta namanya, untuk pilihan pada form.
--
-- Tanpa penyaring aktif/nonaktif: masternya hanya punya dua kolom, dan menambahkan
-- penyaring yang tidak punya kolom berarti mengarang.
--
-- Diurutkan PANJANG dulu, baru nilainya. Kolomnya `VARCHAR2(2)`, sehingga pengurutan teks
-- apa adanya akan menaruh '10' sebelum '9' begitu tipe kesepuluh ditambahkan. `LENGTH`
-- portabel di Oracle maupun PostgreSQL.
SELECT PROTECTION_TYPE_ID,
       PROTECTION_TYPE_NAME
  FROM POOLDATA.M_CLAIM_PROTECTION_TYPE
 ORDER BY LENGTH(TRIM(PROTECTION_TYPE_ID)), TRIM(PROTECTION_TYPE_ID)


-- name: protection_duplicate
-- Proteksi ganda: polis dan tipe yang sama pada HARI KALENDER yang sama.
--
-- Aturannya dari `Activity/ValidationInputProtection-Act.xml`, yang menolak dengan pesan
-- "Sudah ada Open Protection dengan no polis dan tipe proteksi yang sama di hari ini".
--
-- Perbandingan harinya memakai `CAST(… AS DATE)`, bukan `TRUNC` — `TRUNC` tidak portabel
-- ke PostgreSQL (`09-DATABASE-STRATEGY.md` §4).
--
-- Penanda ketiga dan keempat membawa nomor yang DIKECUALIKAN, dikirim dua kali karena
-- muncul dua kali. Tanpa pengecualian itu, menyunting sebuah permintaan tanpa mengubah
-- polis maupun tipenya akan ditolak oleh dirinya sendiri — cacat yang hanya muncul saat
-- menyunting, tidak saat membuat.
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE UPPER(TRIM(POLICY_NO)) = :1
   AND TRIM(PROTECTION_TYPE_ID) = :2
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( :3 IS NULL OR UPPER(TRIM(OPEN_PROTECTION_ID)) <> :4 )
   AND CAST(CREATE_DATE AS DATE) = CAST(:5 AS DATE)


-- name: protection_next_sequence
-- Nomor urut berikutnya, dari sequence.
--
-- ============================================================================
-- SEQUENCE, BUKAN LAGI MAX+1
-- ============================================================================
--
-- Work Owner membuat `POOLDATA.CLAIM_PROTECTION_SEQ` pada 2026-09-24
-- (`START WITH 1`, `NOCACHE`, `NOCYCLE`, `NOORDER`). Versi sebelumnya menurunkan nomor dari
-- `MAX(...)+1` karena pencacahnya belum ada.
--
-- Perubahannya bukan kerapian. `MAX+1` membaca isi tabel, sehingga dua permintaan yang tiba
-- bersamaan dapat membaca nilai yang sama dan menerbitkan nomor yang sama — yang tertahan
-- hanya oleh primary key, dengan percobaan ulang sebagai penambalnya. Sequence menerbitkan
-- nilai berbeda untuk setiap pemanggil TANPA membaca tabel, sehingga percobaan ulang itu
-- tidak lagi diperlukan.
--
-- Satu hal yang HILANG, dan diterima: sequence tidak direset tiap tahun, sehingga nomor
-- urut menembus pergantian tahun (`OPCN.26.0009` diikuti `OPCN.27.0010`). Segmen tahun
-- menjadi penanda, bukan penghitung per tahun.
--
-- Satu hal lain: `NOCACHE` membuat setiap pemanggilan menulis ke kamus data — lebih lambat,
-- tetapi tidak membuang blok nomor saat basis data direstart. Untuk volume proteksi yang
-- ratusan per tahun, itu pertukaran yang benar.
--
-- ============================================================================
-- SATU-SATUNYA KUERI MODUL INI YANG TIDAK PORTABEL
-- ============================================================================
--
-- `NEXTVAL` bergaya Oracle dan `FROM DUAL` keduanya khas Oracle; padanan PostgreSQL-nya
-- `SELECT nextval('pooldata.claim_protection_seq')` tanpa klausa FROM.
--
-- Ia diterima DI SINI SAJA karena generator nomor memang satu-satunya tempat yang `D-22`
-- dan `D-71` akui sebagai sakelar dialek (`ADR-0005`). Dipagari uji di query_test.go supaya
-- pengecualian itu tidak menyebar diam-diam ke kueri lain.
SELECT POOLDATA.CLAIM_PROTECTION_SEQ.NEXTVAL FROM DUAL


-- name: protection_insert
-- Menyisipkan permintaan baru.
--
-- APPROVAL_STATUS sengaja TIDAK ada di daftar kolom: ia harus bernilai NULL, dan
-- membiarkannya tidak disebut adalah cara paling pasti memastikannya. Menuliskannya
-- eksplisit sebagai NULL pun benar, tetapi menghilangkannya membuat tidak ada tempat bagi
-- seseorang kelak menggantinya dengan teks kosong tanpa sengaja.
--
-- RESOLVED_BY dan RESOLVED_DATETIME juga tidak disebut — keduanya milik modul
-- inboxacceptopenprotection (`P-1`).
--
-- OBJECT_ID dan OBJECT_COVERAGE_ID (migrasi `0015`) menyimpan SASARAN perubahan tipe '8' —
-- baris coverage yang dipilih pemohon. Keduanya NULL untuk tipe lain, dan itu benar: hanya
-- permintaan perubahan Cause of Loss yang punya sasaran di tingkat coverage.
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION (
    OPEN_PROTECTION_ID, POLICY_NO, CLAIM_NO, ID_CLAIM, PROTECTION_TYPE_ID,
    CREATE_DATE, CREATED_BY, NOTES,
    OLD_DATA, NEW_DATA, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE,
    OBJECT_ID, OBJECT_COVERAGE_ID
) VALUES (
    :1, :2, :3, :4, :5,
    :6, :7, :8,
    :9, :10, :11, :12, '1',
    :13, :14
)


-- name: protection_update
-- Menyunting permintaan yang belum tertaut klaim dan belum diakseptasi.
--
-- Kedua syarat itu ada DI DALAM WHERE, bukan hanya diperiksa lebih dulu di Go. Pemeriksaan
-- di Go menjaga pengguna dari kesalahan; syarat di sini yang menahan permintaan kedua yang
-- tiba bersamaan.
--
-- OPEN_PROTECTION_ID, CREATE_DATE, dan CREATED_BY TIDAK ikut diubah: ketiganya menyatakan
-- asal-usul baris.
UPDATE POOLDATA.T_CLAIM_OPENPROTECTION
   SET POLICY_NO          = :1,
       CLAIM_NO           = :2,
       ID_CLAIM           = :3,
       PROTECTION_TYPE_ID = :4,
       NOTES              = :5,
       OLD_DATA           = :6,
       NEW_DATA           = :7,
       OBJECT_NAME        = :8,
       BRANCH_NAME        = :9,
       -- Ikut ditulis ulang, termasuk menjadi NULL ketika tipenya berubah dari '8' ke tipe
       -- lain. Membiarkannya terisi akan meninggalkan sasaran yang tidak lagi berarti apa-apa
       -- pada baris yang bukan permintaan perubahan Cause of Loss.
       OBJECT_ID          = :10,
       OBJECT_COVERAGE_ID = :11
 WHERE UPPER(TRIM(OPEN_PROTECTION_ID)) = :12
   AND APPROVAL_STATUS IS NULL
   AND (CLAIM_NO IS NULL OR TRIM(CLAIM_NO) IS NULL)
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')


-- name: claim_find
-- Mencari klaim yang hendak ditaut, untuk mengisi field TURUNAN pada form.
--
-- ============================================================================
-- KENAPA MODUL INI MEMBACA TABEL KLAIM SAMA SEKALI
-- ============================================================================
--
-- `Activity/OpenProtection-Act.xml` — yang di Pega dipicu field **No Klaim** — memuat
-- klaimnya lewat Report Definition `BrowseCaseList`, lalu MENYALIN KELUAR ke halaman
-- proteksi:
--
--     pyWorkPage.PolicyNo                 <- polis klaim
--     pyWorkPage.Policy.QQName            <- nama tertanggung
--     .ClaimDataProtect.BeforeDateOfLoss  <- TempPNCOPEN.ClaimData.DateOfLoss
--     pyWorkPage.ClaimDataProtect.ObjectList(...).ObjectName / BranchName
--
-- Jadi di sistem lama pun nilai-nilai itu TIDAK PERNAH diketik: ia ditimpa setiap kali klaim
-- dicari. Implementasi pertama modul ini keliru menjadikannya isian bebas.
--
-- `BrowseCaseList` TIDAK ADA di export (`R-16`), sehingga kueri ini disusun dari kolom yang
-- TERBUKTI TERISI di katalog — bukan disalin dari RD-nya.
--
-- ============================================================================
-- TIGA TABEL, KARENA SATU TABEL TIDAK CUKUP
-- ============================================================================
--
-- Versi pertama kueri ini membaca DATEOFLOSS dan CAUSEOFLOSS dari tabel kerja Pega. Hitungan
-- terhadap 2.634 klaim membantahnya: KEDUA kolom itu **nol terisi** di sana.
--
--     PC_ASM_FW_GCNMFW_WORK    POLICYNO 2219 · QQNAME 2207 · BRANCHNAME 2167
--                              DATEOFLOSS 0  · CAUSEOFLOSS 0
--     T_CLAIM_PNC              DATEOFLOSS 1740
--     T_CLAIM_OBJECTCOVERAGE   CAUSEOFLOSS 2429
--
-- Kalau kolom yang nol terisi itu dipakai, form akan menampilkan "Current Date Of Loss"
-- KOSONG pada setiap klaim — dan tidak ada galat yang memberi tahu sebabnya.
--
-- ============================================================================
-- SATU TABEL KLAIM: T_CLAIM_PNC. TABEL KERJA PEGA TIDAK DIBACA LAGI
-- ============================================================================
--
-- Work Owner menetapkan 2026-09-26, lalu menegaskannya kembali setelah akibatnya disampaikan
-- beserta angkanya:
--
--     "untuk input protection juga ambil datanya dari t_claim_pnc ya,
--      jangan pakai t_claimlist_admin"
--     "jangan gunakan t_claimlist_admin sama sekali, gunakan t_claim_pnc saja"
--
-- Kueri ini karena itu membaca `POOLDATA.T_CLAIM_PNC` saja sebagai sumber klaim.
-- `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` — yang dulu menjadi tabel utamanya — **tidak dibaca lagi**,
-- dan `T_CLAIMLIST_ADMIN` tidak pernah dibaca modul ini sama sekali.
--
-- Dua tabel anak tetap dibaca karena kolomnya memang hanya ada di sana, dan keduanya tabel
-- klaim yang sama: `T_CLAIM_OBJECTCOVERAGE` (Cause of Loss) dan `T_CLAIM_OBJECTLIST`
-- (Object Name).
--
-- ============================================================================
-- AKIBATNYA TERUKUR, DAN DITERIMA SECARA SADAR
-- ============================================================================
--
-- Hitungan langsung ke Oracle 2026-09-26, dibatasi `PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'`
-- (tabel kerja Pega dipakai BERSAMA sepuluh case type — dari 7.716 barisnya hanya 2.639 yang
-- klaim):
--
--     klaim Work-PNC seluruhnya                         2.639
--     DAPAT ditemukan lewat T_CLAIM_PNC                 1.397
--     TIDAK dapat ditemukan                             1.242   47%
--
-- Jadi klaim yang belum punya baris di `T_CLAIM_PNC` menjawab **"klaim tidak ditemukan"**.
-- Itu konsekuensi yang disampaikan lebih dulu beserta angkanya, dan Work Owner menegaskan
-- pilihannya. Penutupnya adalah **melengkapi `T_CLAIM_PNC`** — bukan mengembalikan tabel
-- kerja Pega ke dalam kueri ini.
--
-- ============================================================================
-- KUNCINYA CLAIMID, BUKAN CLAIMNO
-- ============================================================================
--
-- `T_CLAIM_PNC` punya DUA kolom yang sama-sama tampak seperti nomor klaim. Keduanya diukur,
-- dan hasilnya tidak berimbang:
--
--     kolom            terisi    kembar   klaim terjangkau
--     CLAIMNO           1.697         1            1.349
--     CLAIMID           2.176         0            1.397
--
-- `CLAIMNO` **kosong pada 479 baris**, **berulang pada satu pasang** yang CLAIMID-nya berbeda,
-- dan **berbeda isi dari nomor turunannya pada 11 baris**. `CLAIMID` terisi pada SETIAP baris
-- dan tidak pernah kembar.
--
-- Menambahkan `OR CLAIMNO = ...` hanya menambah **satu** klaim (1.398), dengan biaya predikat
-- yang tidak dapat memakai index. Tidak diambil.
--
-- **Yang paling menentukan:** kunci baca di sini menjadi kolom yang SAMA dengan kunci tulis
-- `UPDATE POOLDATA.T_CLAIM_PNC ... WHERE CLAIMID = :2` di
-- `inboxacceptopenprotection/claimsync.go`. Selama keduanya satu kolom, klaim yang dapat
-- dicari pasti dapat pula diubah saat proteksinya disetujui — tidak ada celah di mana
-- pencarian berhasil tetapi penerapannya gagal.
--
-- ============================================================================
-- DUA BENTUK CLAIMID, DUA ARGUMEN — BUKAN INSTR/SUBSTR
-- ============================================================================
--
-- `CLAIMID` hanya punya DUA bentuk, dan itu terukur — nol bentuk ketiga, nol spasi ganda:
--
--     ASM-FW-GCNMFW-WORK PNC-1865   2.167 baris   klaim warisan Pega
--     PNCN.26.0007                      9 baris   klaim sistem baru, tanpa spasi
--
-- Bentuk pertama yang terpikir adalah memotong prefix-nya di dalam SQL dengan
-- `SUBSTR(CLAIMID, INSTR(CLAIMID,' ') + 1)`. Itu **tidak diambil**, karena `INSTR` ada di
-- daftar pola terlarang `09-DATABASE-STRATEGY.md` §4 — dan penggantinya yang disebut di sana,
-- `POSITION(x IN y)`, **tidak didukung Oracle**. Diuji langsung:
--
--     SELECT POSITION(' ' IN 'AB CD') FROM DUAL   ->   ORA-00907
--
-- Jadi memakai `INSTR` akan melanggar `D-20`, dan memakai `POSITION` akan gagal hari ini.
-- Keduanya dihindari: prefix disusun **di Go** dan dikirim sebagai argumen kedua, sehingga
-- kueri ini hanya membandingkan kesamaan.
--
--     :1  nomor klaim                       PNC-1865
--     :2  prefix + nomor klaim              ASM-FW-GCNMFW-WORK PNC-1865
--
-- Jangkauannya **identik** dengan bentuk `INSTR` — 1.397 klaim, diukur berdampingan — dan
-- keduanya predikat kesamaan, bukan pola.
--
-- Argumennya dikirim DUA KALI meski nilainya berasal dari satu masukan: driver mengikat
-- menurut urutan KEMUNCULAN penanda, bukan menurut nomornya, sehingga `:1` yang dipakai
-- berulang menghasilkan ORA-01008.
--
-- Nama kelas Pega karena itu tidak tertulis di berkas SQL ini sama sekali; ia satu konstanta
-- bernama di `protection.go`, yang hilang bersama klaim warisannya kelak.
--
-- ============================================================================
-- HANYA MEMBACA
-- ============================================================================
--
-- Ketiga tabel dimiliki Pega selama masa paralel. `P-1` melarang dua sistem MENULIS satu
-- tabel; membaca tidak dilarang. Tidak ada satu pun pernyataan tulis ke tabel klaim di
-- seluruh modul ini.
--
-- Objek dan penyebab kerugian diambil SATU baris lewat subkueri ber-`FETCH FIRST`, bukan
-- lewat join yang menggandakan baris: satu klaim dapat punya banyak objek dan banyak
-- coverage, sedangkan panel Detail Perubahan hanya punya satu nilai untuk masing-masing.
-- Pega pun menyalinnya sebagai satu nilai.
--
-- ============================================================================
-- CLAIMID DIKEMBALIKAN UTUH — ia yang menjadi ID_CLAIM
-- ============================================================================
--
-- Kolom kedua adalah `CLAIMID` apa adanya, dan itulah yang disimpan sebagai
-- `T_CLAIM_OPENPROTECTION.ID_CLAIM`:
--
--     klaim Pega        CLAIM_NO PNC-1865       ID_CLAIM ASM-FW-GCNMFW-WORK PNC-1865
--     klaim sistem baru CLAIM_NO PNCN.26.0007   ID_CLAIM PNCN.26.0007
--
-- Ia dikembalikan UTUH dan tidak di-`COALESCE`: bentuk penuhnya yang dipakai sebagai kunci
-- saat permintaannya disetujui. Nomor klaimnya diturunkan di Go dari kolom yang sama, jadi
-- tidak ada kolom kedua untuk itu — satu nilai, satu sumber.
SELECT c.CLAIMID,
       c.NOPOLIS,
       c.QQNAME,
       c.DATEOFLOSS,
       (SELECT cv.CAUSEOFLOSS
          FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE cv
         WHERE cv.CLAIMID = c.CLAIMID
         ORDER BY cv.OBJECTID, cv.COVERAGEID
         FETCH FIRST 1 ROW ONLY),
       c.BRANCHNAME,
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = c.CLAIMID
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY),
       -- BUSINESSCODE tidak ditampilkan. Ia menyaring daftar "Next Cause Of Loss" supaya
       -- pemohon hanya melihat penyebab kerugian yang berlaku bagi lini bisnis klaimnya —
       -- padanan `BrowseCOLByBisnis_Sql` milik Pega.
       c.BUSINESSCODE
  FROM POOLDATA.T_CLAIM_PNC c
 WHERE UPPER(TRIM(c.CLAIMID)) IN (:1, :2)

-- name: claim_coverages
--
-- Seluruh coverage satu klaim beserta KUNCI barisnya — isi panel Detail Perubahan Cause Of
-- Loss, dan satu-satunya cara permintaan tipe '8' tahu baris mana yang hendak diubah.
--
-- ============================================================================
-- KENAPA SELURUHNYA, BUKAN SATU
-- ============================================================================
--
-- `claim_find` di atas mengambil SATU Cause of Loss lewat `FETCH FIRST 1 ROW ONLY`. Itu
-- memadai selama panelnya hanya menampilkan keterangan, dan TIDAK memadai begitu pemohon
-- harus memilih — ia akan memilih dari satu-satunya baris yang ditawarkan, dan baris itu
-- belum tentu yang dimaksudnya.
--
-- Layar Pega `InputProtectionFlow` (`OPC-221`) menampilkan empat baris untuk satu klaim,
-- tiga di antaranya bernama `JackHugh / Resiko A` dengan Cause of Loss berbeda. Nama karena
-- itu tidak menunjuk baris; yang membedakannya hanya OBJECTCOVERAGEID.
--
-- ============================================================================
-- KUNCINYA OBJECTID + OBJECTCOVERAGEID
-- ============================================================================
--
-- Bersama CLAIMID, keduanya kunci alami satu baris coverage — kunci yang SAMA dengan yang
-- dipakai `POOLDATA.T_CLAIM_SPREADING`. Keduanya `NOT NULL` di tabel warisan dan terisi pada
-- seluruh 2.630 baris (diukur 2026-10-05), sehingga berlaku untuk baris warisan Pega maupun
-- baris baru.
--
-- `URUTAN_OBJEK`/`URUTAN` TIDAK dipakai meski modul `registrasi` memakainya: keduanya
-- ditambahkan migrasi `0008` dan hanya terisi pada **32 dari 2.630** baris.
--
-- ============================================================================
-- OBJECTNAME DIAMBIL DARI TABEL OBJEK, BUKAN DARI BARIS COVERAGE
-- ============================================================================
--
-- `T_CLAIM_OBJECTCOVERAGE` punya kolom OBJECTNAME sendiri — ditambahkan
-- `docs/ddl/tc_pnc_object_tree.sql`, dan karena itu baru. `T_CLAIM_OBJECTLIST.OBJECTNAME`
-- adalah kolom warisan yang terisi pada baris lama, sehingga LEFT JOIN ke sana membuat
-- nama objek tetap tampil pada klaim Pega.
--
-- LEFT JOIN, bukan INNER: coverage yang objeknya tidak ditemukan tetap TAMPIL, dengan nama
-- kosong. INNER JOIN akan menghilangkan barisnya dari panel tanpa satu pun gejala — dan
-- baris yang hilang dari panel adalah baris yang tidak akan pernah bisa diperbaiki.
--
-- Baris yang sudah ditandai terhapus dikecualikan (`DIHAPUS_PADA IS NULL`, `D-66`): memilih
-- coverage yang sudah dibuang berarti meminta perubahan atas sesuatu yang tidak lagi ada.
--
-- ============================================================================
-- DIJALANKAN TERHADAP ORACLE 2026-10-05 — HASILNYA SAMA PERSIS DENGAN LAYAR PEGA
-- ============================================================================
--
-- Work Owner menjalankan kueri ini untuk `ASM-FW-GCNMFW-WORK PNC-1452`:
--
--     1  1  JackHugh  Resiko A    ILLNESS
--     1  2  JackHugh  Resiko A    STORM
--     1  3  JackHugh  Katastropi  WINDSTORM
--     1  4  JackHugh  Resiko A    HURRICANE
--
-- Empat baris, urutan yang sama, isi yang sama dengan panel Detail Perubahan Cause Of Loss
-- pada layar `InputProtectionFlow` milik `OPC-221`. Ini bukti kesetaraan atas data nyata,
-- bukan terhadap data yang dikarang dari kode ini sendiri.
--
-- Tiga hal yang ikut terbukti, dan ketiganya tidak dapat dibuktikan sqlmock:
--
--   * `OBJECTCOVERAGEID` memang unik DI DALAM satu objek — 1, 2, 3, 4 pada OBJECTID yang
--     sama. Bersama OBJECTID dan CLAIMID ia menunjuk tepat satu baris.
--   * `LEFT JOIN` ke `T_CLAIM_OBJECTLIST` benar-benar menemukan namanya pada klaim
--     WARISAN — "JackHugh" terisi, bukan kosong.
--   * `JackHugh / Resiko A` muncul tiga kali, sehingga pencocokan lewat nama akan menunjuk
--     baris yang salah dua dari tiga kali.
--
-- ============================================================================
-- CAUSEOFLOSSID IKUT DIBACA — PENYEBAB KERUGIAN ADALAH SEPASANG NILAI
-- ============================================================================
--
-- Ditegaskan Work Owner 2026-10-05: *"ganti cause of loss itu ganti causeoflossid juga"*.
--
-- Pega melakukan keduanya dalam satu langkah (`InsertOpenProtectionCase`):
--
--     ObjectCoverageList(...).CauseOfLoss   <- deskripsi yang dipilih
--     ObjectCoverageList(...).CauseOfLossID <- id-nya
--
-- Membaca hanya deskripsinya akan membuat akseptasi menulis nama baru di atas id lama —
-- baris yang namanya berkata satu hal dan kodenya berkata hal lain, tanpa galat. Setiap
-- laporan yang mengelompokkan menurut kode akan ikut salah, dan tidak ada yang menandainya.
SELECT cv.OBJECTID,
       cv.OBJECTCOVERAGEID,
       o.OBJECTNAME,
       cv.COVERAGENAME,
       cv.CAUSEOFLOSS,
       cv.CAUSEOFLOSSID
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE cv
  LEFT JOIN POOLDATA.T_CLAIM_OBJECTLIST o
    ON TRIM(o.CLAIMID) = TRIM(cv.CLAIMID)
   AND TRIM(o.OBJECTID) = TRIM(cv.OBJECTID)
 WHERE UPPER(TRIM(cv.CLAIMID)) IN (:1, :2)
   AND cv.DIHAPUS_PADA IS NULL
 ORDER BY cv.OBJECTID, cv.OBJECTCOVERAGEID

-- name: cause_of_loss_options
--
-- Pilihan dropdown **Next Cause Of Loss**, disaring lini bisnis klaim.
--
-- ============================================================================
-- KOLOM BIASA, BUKAN JSONDATA — DITETAPKAN WORK OWNER 2026-10-05
-- ============================================================================
--
-- Dokumen JSON SUDAH DIKELUARKAN menjadi kolom. Bentuk tabel per 2026-10-05, dibaca
-- langsung dari katalog:
--
--     JSONDATA      CLOB            masih ada, TIDAK dipakai lagi
--     D_COL_ID      VARCHAR2(5)     NOT NULL — kunci, dan nilai yang disimpan
--     OLD_D_COL_ID  VARCHAR2(5)
--     M_COL_ID      VARCHAR2(1000)  golongan induk
--     DESCRIPTION   VARCHAR2(4000)  teks yang dibaca pengguna
--     STS_AKTIF     VARCHAR2(10)
--     LOSS_CODE     VARCHAR2(100)
--
-- Contoh isinya: "11997 · FIRE - OPEN FLAME · A" dan "12033 · WRECK REMOVAL".
--
-- JSONDATA sengaja TIDAK disebut di sini. Selama kolom dan dokumen hidup berdampingan,
-- keduanya dapat menyimpang — dan membaca yang satu sambil ada yang menulis yang lain
-- menghasilkan daftar yang benar hari ini lalu salah diam-diam besok.
--
-- ============================================================================
-- LINI BISNIS: TABEL ANAK, KARENA IA RELASI SATU-KE-BANYAK
-- ============================================================================
--
-- POOLDATA.D_CAUSE_OF_LOSS_BUSINESS — ditetapkan Work Owner 2026-10-05.
--
-- Satu penyebab kerugian berlaku bagi BEBERAPA lini bisnis, sehingga ia tidak muat sebagai
-- kolom pada tabel induk. Di dokumen JSON ia memang array: $.BISNISID[*] berisi {ID, Note}.
--
-- Dipasang sebagai EXISTS, bukan JOIN. Alasannya satu dan menentukan: sebuah penyebab
-- kerugian yang terdaftar pada tiga lini akan muncul TIGA KALI bila di-join, dan dropdown
-- yang menawarkan pilihan yang sama berulang kali membuat pengguna ragu ia memilih yang
-- benar.
--
-- Pega menyaringnya dengan merangkai potongan WHERE ke dalam teks SQL lewat
-- {ASIS:TempSearchBisnis.DESCRIPTION} (BrowseCOLByBisnis_Sql). Perangkaian itu TIDAK
-- dibawa; di sini nilainya lewat parameter binding (11-SECURITY.md §4.1).
--
-- TRIM di kedua sisi pembanding: keduanya VARCHAR2 tanpa penyeragaman, dan satu spasi di
-- ujung membuat dropdown kosong tanpa satu pun galat.
--
-- Kode bisnis KOSONG berarti tidak menyaring. Itu disengaja: klaim yang BUSINESSCODE-nya
-- kosong lebih baik menampilkan semua pilihan daripada daftar kosong yang terbaca sebagai
-- master yang rusak.
--
-- ============================================================================
-- STS_AKTIF TIDAK DISARING — MENGIKUTI PEGA
-- ============================================================================
--
-- BrowseCOLByBisnis_Sql tidak menyaringnya, dan detail_list milik modul detailpenyebab juga
-- tidak. Menambahkannya adalah PERUBAHAN PERILAKU: penyebab kerugian yang dinonaktifkan akan
-- hilang dari dropdown, dan permintaan lama yang memakainya tidak lagi dapat disunting. Bila
-- itu dikehendaki, ia butir P-5 tersendiri — bukan keputusan yang diselipkan ke dalam kueri.
SELECT d.D_COL_ID,
       d.DESCRIPTION,
       d.LOSS_CODE
  FROM POOLDATA.D_CAUSE_OF_LOSS d
 WHERE ( :1 IS NULL
         OR EXISTS (SELECT 1
                      FROM POOLDATA.D_CAUSE_OF_LOSS_BUSINESS b
                     WHERE TRIM(b.D_COL_ID) = TRIM(d.D_COL_ID)
                       AND TRIM(b.BISNISID) = :2) )
 ORDER BY d.DESCRIPTION
