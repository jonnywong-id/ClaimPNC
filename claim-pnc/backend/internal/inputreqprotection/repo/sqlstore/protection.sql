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
-- Kolom di bawah DIBACA DARI KATALOG Oracle, bukan disalin dari usulan. Tabelnya sempat
-- dibentuk ulang Work Owner, dan yang berlaku adalah bentuk terakhirnya: 16 kolom.
--
--   Properti Pega        Kolom                 Kolom layar
--   -------------------- --------------------- ------------------------
--   .pyID                ID                    No Proteksi
--   .PolicyNo            POLICY_NO               No Polis
--   .CaseID              CLAIM_NO               No Klaim
--   .PNCCaseID           ID_CLAIM               tidak ditampilkan
--   .TypeProtection      PROTECTION_TYPE        Tipe Proteksi
--   .InputDate           CREATE_DATE            Tanggal Proteksi Dibuat
--   .Keterangan          NOTES                 Keterangan
--   .AcceptStatus        APPROVAL_STATUS        penyaring inti
--   .pxCreateOpName      CREATED_BY             User Create
--   .ObjectName          OBJECT_NAME            Object Name (form)
--   .BranchName          BRANCH_NAME            Branch Name (form)
--
-- `CREATE_DATE` memikul DUA peran sekaligus — tanggal proteksi dibuat dan waktu baris
-- dibuat. Tabel hanya punya satu kolom waktu pembuatan, dan sistem lama pun tidak
-- membedakan keduanya.
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
-- dan artinya ditentukan PROTECTION_TYPE:
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
SELECT COUNT(*)
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( :1 IS NULL
         OR UPPER(ID)      LIKE :2
         OR UPPER(POLICY_NO) LIKE :3
         OR UPPER(CLAIM_NO) LIKE :4 )


-- name: protection_list
-- Satu halaman permintaan yang belum diakseptasi.
--
-- Diurutkan MENURUN mengikuti RD rujukan. `ID` ikut menjadi kunci urut kedua supaya
-- urutannya tetap sama pada dua pemanggilan dengan waktu pembuatan identik — tanpa itu,
-- paginasi dapat menampilkan satu baris dua kali dan melewatkan baris lain.
--
-- `OFFSET … FETCH NEXT` dipakai, bukan `ROWNUM` (`09-DATABASE-STRATEGY.md` §4).
SELECT ID,
       POLICY_NO,
       CLAIM_NO,
       ID_CLAIM,
       PROTECTION_TYPE,
       CREATE_DATE,
       NOTES,
       APPROVAL_STATUS,
       CREATED_BY,
       OLD_DATA,
       NEW_DATA,
       OBJECT_NAME,
       BRANCH_NAME
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE APPROVAL_STATUS IS NULL
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( :1 IS NULL
         OR UPPER(ID)      LIKE :2
         OR UPPER(POLICY_NO) LIKE :3
         OR UPPER(CLAIM_NO) LIKE :4 )
 ORDER BY CREATE_DATE DESC, ID DESC
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY


-- name: protection_get
-- Satu permintaan menurut nomornya.
--
-- TIDAK menyaring APPROVAL_STATUS: form harus tetap dapat dibuka untuk permintaan yang baru
-- saja diakseptasi, supaya pesannya dapat menjelaskan apa yang terjadi — bukan sekadar
-- "tidak ditemukan".
SELECT ID,
       POLICY_NO,
       CLAIM_NO,
       ID_CLAIM,
       PROTECTION_TYPE,
       CREATE_DATE,
       NOTES,
       APPROVAL_STATUS,
       CREATED_BY,
       OLD_DATA,
       NEW_DATA,
       OBJECT_NAME,
       BRANCH_NAME
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE UPPER(TRIM(ID)) = :1
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')


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
   AND TRIM(PROTECTION_TYPE) = :2
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
   AND ( :3 IS NULL OR UPPER(TRIM(ID)) <> :4 )
   AND CAST(CREATE_DATE AS DATE) = CAST(:5 AS DATE)


-- name: protection_next_sequence
-- Nomor urut berikutnya untuk tahun berjalan.
--
-- ============================================================================
-- KENAPA MAX+1, DAN APA YANG MEMBUATNYA AMAN
-- ============================================================================
--
-- Bentuk pencacah nomor BELUM DIPUTUSKAN (lihat docs/kolom-open-protection.md §6): tidak
-- ada sequence maupun tabel pencacah bernama terkait di POOLDATA — diperiksa lewat
-- ALL_OBJECTS, nol hasil.
--
-- Sampai salah satunya dibuat, nomor urut diturunkan dari nomor tertinggi tahun berjalan.
-- Ia dijalankan DI DALAM transaksi yang sama dengan penyisipannya, sehingga jendela
-- balapannya sempit — tetapi TIDAK NOL.
--
-- Yang menutup sisanya adalah CONSTRAINT UNIK pada kolom ID: bila dua permintaan tiba
-- bersamaan, yang kedua gagal ORA-00001 dan adapter mencobanya ulang.
--
-- CONSTRAINT ITU BELUM ADA. Tabel hari ini tanpa primary key (diperiksa lewat
-- ALL_CONSTRAINTS, nol hasil). Sampai ia dibuat, dua penyimpanan bersamaan DAPAT
-- menerbitkan nomor yang sama tanpa satu pun galat. Pernyataannya ada di
-- docs/kolom-open-protection.md §5, dan ia WAJIB — bukan penyempurnaan.
--
-- `SUBSTR(ID, 9)` mengambil segmen terakhir dari `OPCN.YY.xxxx`: empat karakter prefix,
-- satu titik, dua digit tahun, satu titik = 8 karakter sebelum nomor urutnya.
--
-- ============================================================================
-- SATU-SATUNYA KUERI MODUL INI YANG TIDAK PORTABEL
-- ============================================================================
--
-- `TO_NUMBER` khas Oracle. Ia diterima di sini karena GENERATOR NOMOR memang satu-satunya
-- tempat yang `D-22` dan `D-71` akui sebagai sakelar dialek (`ADR-0005`).
--
-- `COALESCE` dipakai, BUKAN `NVL`: untuk yang satu ini tidak ada alasan menambah
-- ketidakportabelan kedua (`09-DATABASE-STRATEGY.md` §4). Dijaga uji di query_test.go.
SELECT COALESCE(MAX(TO_NUMBER(SUBSTR(ID, 9))), 0) + 1
  FROM POOLDATA.T_CLAIM_OPENPROTECTION
 WHERE ID LIKE :1


-- name: protection_insert
-- Menyisipkan permintaan baru.
--
-- APPROVAL_STATUS sengaja TIDAK ada di daftar kolom: ia harus bernilai NULL, dan
-- membiarkannya tidak disebut adalah cara paling pasti memastikannya. Menuliskannya
-- eksplisit sebagai NULL pun benar, tetapi menghilangkannya membuat tidak ada tempat bagi
-- seseorang kelak menggantinya dengan teks kosong tanpa sengaja.
--
-- RESOLVED_BY dan RESOLVED_DATE_TIME juga tidak disebut — keduanya milik modul
-- inboxacceptopenprotection (`P-1`).
INSERT INTO POOLDATA.T_CLAIM_OPENPROTECTION (
    ID, POLICY_NO, CLAIM_NO, ID_CLAIM, PROTECTION_TYPE,
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
-- ID, CREATE_DATE, dan CREATED_BY TIDAK ikut diubah: ketiganya menyatakan asal-usul baris.
UPDATE POOLDATA.T_CLAIM_OPENPROTECTION
   SET POLICY_NO        = :1,
       CLAIM_NO        = :2,
       ID_CLAIM        = :3,
       PROTECTION_TYPE = :4,
       NOTES          = :5,
       OLD_DATA        = :6,
       NEW_DATA        = :7,
       OBJECT_NAME     = :8,
       BRANCH_NAME     = :9
 WHERE UPPER(TRIM(ID)) = :10
   AND APPROVAL_STATUS IS NULL
   AND (CLAIM_NO IS NULL OR TRIM(CLAIM_NO) IS NULL)
   AND (STATUS_ACTIVE IS NULL OR TRIM(STATUS_ACTIVE) = '1')
