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
       p.BRANCH_NAME
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
       p.BRANCH_NAME
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
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION (
    OPEN_PROTECTION_ID, POLICY_NO, CLAIM_NO, ID_CLAIM, PROTECTION_TYPE_ID,
    CREATE_DATE, CREATED_BY, NOTES,
    OLD_DATA, NEW_DATA, OBJECT_NAME, BRANCH_NAME, STATUS_ACTIVE
) VALUES (
    :1, :2, :3, :4, :5,
    :6, :7, :8,
    :9, :10, :11, :12, '1'
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
       BRANCH_NAME        = :9
 WHERE UPPER(TRIM(OPEN_PROTECTION_ID)) = :10
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
-- SELURUHNYA LEFT JOIN, DAN ITU DISENGAJA
-- ============================================================================
--
-- Hanya 1.393 dari 2.634 klaim punya baris di `T_CLAIM_PNC`, dan 1.321 punya objek.
-- `INNER JOIN` akan membuat separuh klaim tampak TIDAK ADA — dan pengguna menerima "klaim
-- tidak ditemukan" untuk klaim yang jelas-jelas ada.
--
-- Klaim yang ditemukan tetapi tanpa DOL adalah keadaan yang sah dan ditampilkan apa adanya.
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
SELECT w.PYID,
       COALESCE(w.POLICYNO, c.NOPOLIS),
       COALESCE(w.QQNAME, c.QQNAME),
       c.DATEOFLOSS,
       (SELECT cv.CAUSEOFLOSS
          FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE cv
         WHERE cv.CLAIMID = w.PZINSKEY
         ORDER BY cv.OBJECTID, cv.COVERAGEID
         FETCH FIRST 1 ROW ONLY),
       COALESCE(w.BRANCHNAME, c.BRANCHNAME),
       (SELECT o.OBJECTNAME
          FROM POOLDATA.T_CLAIM_OBJECTLIST o
         WHERE o.CLAIMID = w.PZINSKEY
         ORDER BY o.OBJECTID
         FETCH FIRST 1 ROW ONLY)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK w
  LEFT JOIN POOLDATA.T_CLAIM_PNC c
         ON c.CLAIMID = w.PZINSKEY
 WHERE UPPER(TRIM(w.PYID)) = :1
   AND w.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
