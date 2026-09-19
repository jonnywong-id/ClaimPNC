-- Kueri tabel POOLDATA.LST_ACCOUNT — master rekening tujuan pembayaran klaim.
--
-- Kolom selalu disebut namanya; SELECT * dilarang supaya kolom baru di basis data tidak
-- diam-diam mengubah perilaku aplikasi.
--
-- PEMETAAN KOLOM. Alias Pega di sistem lama menyesatkan secara aktif dan TIDAK dibawa
-- masuk. Yang dipakai adalah nama domain di internal/masterrekening. Tabel ini adalah
-- satu-satunya tempat ketiganya dapat dibandingkan:
--
--   kolom                 alias Pega lama    nama domain
--   ------------------------------------------------------------------
--   ACCOUNT_NO            NoAccount          NomorRekening
--   ACCOUNT_NAME          Name               NamaPemilik
--   BANK_NAME             NameOfBank         NamaBank
--   BANK_BRANCH           BranchOfBank       CabangBank
--   BANK_ADDRESS          Address            AlamatBank
--   BANKID                IDBank             KodeBank
--   ACCOUNT_TYPE          CoverID            TipeRekening      (bukan "cover")
--   STS_AKTIF             CaseID             Aktif             (bukan nomor kasus)
--   APPROVAL              pyID saat ditulis  Status
--   KOMITE_APPROVAL       pyCountry          KomiteApproval    (bukan negara)
--   TANGGALAPPROVEKOMITE  KOMISI             DiputuskanPada    (bukan komisi)
--   EMAIL                 EmailReceiver      Email
--   EMAILINPUT            NoHpUserAccount    EmailPenginput    (bukan nomor HP)
--   TELP                  Telephone          Telepon
--   NIK                   Nik                NIK
--   DOKUMENID             CoverInsKey        IDDokumen
--   NOTE                  pyContext          Catatan
--   USER_INPUT            pyID saat dibaca   DiinputOleh
--   TGL_INPUT             —                  DiinputPada
--   UPDATEBY              Updatingby         DiubahOleh
--   STS_SERVICE           DISC               StatusLayanan
--   ID_REKASIR            NoIdentitas        IDRekeningKasir
--   RESPONSE_KASIR        Notes              ResponsKasir
--   FLAGUPDATE            KOMISI             FlagPerubahan
--   OLDBANID              LbgIdOld           KodeBankLama
--   OLDACCOUNT_NO         AccountNoOld       NomorRekeningLama
--   ACCOUNTNAMEOLD        AccountNameOld     NamaPemilikLama
--
-- Perhatikan dua alias yang dipakai untuk DUA kolom berbeda pada rule yang berbeda:
-- pyID (USER_INPUT saat SELECT, APPROVAL saat UPDATE) dan KOMISI (FLAGUPDATE saat
-- SELECT, TANGGALAPPROVEKOMITE saat UPDATE). Itulah alasan alias lama tidak dibawa.
--
-- RESPONSE_KASIR tidak lagi dipangkas dengan SUBSTR/INSTR di dalam SQL seperti rule
-- lama. Memangkas pesan adalah urusan penyajian, dan INSTR bukan fungsi yang tersedia
-- sama di PostgreSQL kelak (09-DATABASE-STRATEGY §4). Pemangkasannya pindah ke
-- masterrekening.TrimCashierResponse.

-- name: account_list
SELECT ACCOUNT_NO,
       ACCOUNT_NAME,
       BANK_NAME,
       BANK_BRANCH,
       BANK_ADDRESS,
       BANKID,
       ACCOUNT_TYPE,
       STS_AKTIF,
       APPROVAL,
       KOMITE_APPROVAL,
       TANGGALAPPROVEKOMITE,
       EMAIL,
       EMAILINPUT,
       TELP,
       NIK,
       DOKUMENID,
       NOTE,
       USER_INPUT,
       TGL_INPUT,
       UPDATEBY,
       STS_SERVICE,
       ID_REKASIR,
       RESPONSE_KASIR,
       FLAGUPDATE,
       OLDBANID,
       OLDACCOUNT_NO,
       ACCOUNTNAMEOLD
  FROM POOLDATA.LST_ACCOUNT
 WHERE (:1  IS NULL OR APPROVAL = :2)
   AND (:3  IS NULL OR UPPER(ACCOUNT_NO)      LIKE '%' || UPPER(:4)  || '%')
   AND (:5  IS NULL OR UPPER(ACCOUNT_NAME)    LIKE '%' || UPPER(:6)  || '%')
   AND (:7  IS NULL OR UPPER(BANK_NAME)       LIKE '%' || UPPER(:8)  || '%')
   AND (:9  IS NULL OR UPPER(KOMITE_APPROVAL) =         UPPER(:10))
 ORDER BY TGL_INPUT DESC, ACCOUNT_NO
OFFSET :11 ROWS FETCH NEXT :12 ROWS ONLY

-- Catatan paginasi. OFFSET … FETCH dipakai, bukan ROWNUM: ia didukung Oracle 12c+ dan
-- PostgreSQL sekaligus (09-DATABASE-STRATEGY §3.3). Keyset pagination yang dituntut
-- §6.3 berlaku untuk inbox dan pencarian klaim berpuluh juta baris; master rekening
-- adalah tabel master yang sudah tersaring sempit, yang justru disebut §6.3 sebagai
-- tempat OFFSET … FETCH masih tepat.

-- name: account_count
SELECT COUNT(*)
  FROM POOLDATA.LST_ACCOUNT
 WHERE (:1 IS NULL OR APPROVAL = :2)
   AND (:3 IS NULL OR UPPER(ACCOUNT_NO)      LIKE '%' || UPPER(:4)  || '%')
   AND (:5 IS NULL OR UPPER(ACCOUNT_NAME)    LIKE '%' || UPPER(:6)  || '%')
   AND (:7 IS NULL OR UPPER(BANK_NAME)       LIKE '%' || UPPER(:8)  || '%')
   AND (:9 IS NULL OR UPPER(KOMITE_APPROVAL) =         UPPER(:10))

-- name: account_get
SELECT ACCOUNT_NO,
       ACCOUNT_NAME,
       BANK_NAME,
       BANK_BRANCH,
       BANK_ADDRESS,
       BANKID,
       ACCOUNT_TYPE,
       STS_AKTIF,
       APPROVAL,
       KOMITE_APPROVAL,
       TANGGALAPPROVEKOMITE,
       EMAIL,
       EMAILINPUT,
       TELP,
       NIK,
       DOKUMENID,
       NOTE,
       USER_INPUT,
       TGL_INPUT,
       UPDATEBY,
       STS_SERVICE,
       ID_REKASIR,
       RESPONSE_KASIR,
       FLAGUPDATE,
       OLDBANID,
       OLDACCOUNT_NO,
       ACCOUNTNAMEOLD
  FROM POOLDATA.LST_ACCOUNT
 WHERE ACCOUNT_NO = :1
   AND BANKID     = :2

-- name: account_find_by_number
SELECT ACCOUNT_NO,
       ACCOUNT_NAME,
       BANK_NAME,
       BANK_BRANCH,
       BANK_ADDRESS,
       BANKID,
       ACCOUNT_TYPE,
       STS_AKTIF,
       APPROVAL,
       KOMITE_APPROVAL,
       TANGGALAPPROVEKOMITE,
       EMAIL,
       EMAILINPUT,
       TELP,
       NIK,
       DOKUMENID,
       NOTE,
       USER_INPUT,
       TGL_INPUT,
       UPDATEBY,
       STS_SERVICE,
       ID_REKASIR,
       RESPONSE_KASIR,
       FLAGUPDATE,
       OLDBANID,
       OLDACCOUNT_NO,
       ACCOUNTNAMEOLD
  FROM POOLDATA.LST_ACCOUNT
 WHERE ACCOUNT_NO = :1
 ORDER BY BANKID

-- name: account_insert
INSERT INTO POOLDATA.LST_ACCOUNT
       (ACCOUNT_NO, ACCOUNT_NAME, BANK_NAME, BANK_BRANCH, BANK_ADDRESS, BANKID,
        ACCOUNT_TYPE, STS_AKTIF, APPROVAL, KOMITE_APPROVAL, EMAIL, EMAILINPUT, TELP,
        NIK, DOKUMENID, NOTE, USER_INPUT, TGL_INPUT, UPDATEBY, STS_SERVICE,
        OLDBANID, OLDACCOUNT_NO, ACCOUNTNAMEOLD)
VALUES (:1, :2, :3, :4, :5, :6,
        :7, :8, :9, :10, :11, :12, :13,
        :14, :15, :16, :17, :18, :19, :20,
        :21, :22, :23)

-- TGL_INPUT diisi dari aplikasi (:18), bukan dari SYSDATE seperti rule lama.
-- Alasannya mengikat seluruh aplikasi: waktu berasal dari satu seam (platform/clock)
-- sehingga dapat diuji deterministik, dan disimpan UTC. SYSDATE mengambil zona waktu
-- server basis data, yang di sistem lama justru menjadi sumber penanganan zona waktu
-- manual yang 08-TECHNICAL-STRATEGY §4.4 larang.

-- name: account_update
UPDATE POOLDATA.LST_ACCOUNT
   SET ACCOUNT_NAME         = :1,
       BANK_NAME            = :2,
       BANK_BRANCH          = :3,
       BANK_ADDRESS         = :4,
       ACCOUNT_TYPE         = :5,
       STS_AKTIF            = :6,
       APPROVAL             = :7,
       KOMITE_APPROVAL      = :8,
       TANGGALAPPROVEKOMITE = :9,
       EMAIL                = :10,
       EMAILINPUT           = :11,
       TELP                 = :12,
       NIK                  = :13,
       DOKUMENID            = :14,
       NOTE                 = :15,
       UPDATEBY             = :16,
       STS_SERVICE          = :17,
       ID_REKASIR           = :18,
       RESPONSE_KASIR       = :19,
       OLDBANID             = :20,
       OLDACCOUNT_NO        = :21,
       ACCOUNTNAMEOLD       = :22
 WHERE ACCOUNT_NO = :23
   AND BANKID     = :24

-- Rule lama memperbarui dengan `where account_no = {TempBank.pyEmailAddress}` — satu
-- kolom saja, dan lewat properti yang namanya menyebut alamat surel. Nomor rekening
-- yang sama dapat ada di dua bank berbeda, sehingga kunci yang benar adalah pasangan
-- ACCOUNT_NO + BANKID. Keduanya dipakai di sini.

-- name: account_clear_rejected
DELETE FROM POOLDATA.LST_ACCOUNT
 WHERE ACCOUNT_NO = :1
   AND BANKID     = :2
   AND APPROVAL   = '2'

-- Syarat APPROVAL = '2' ada di dalam kueri, bukan hanya diperiksa di Go. Rule lama
-- menghapus dengan `DELETE FROM LST_ACCOUNT where {ASIS:TempDataBank.City}` — klausa
-- WHERE-nya dirangkai dari properti klipboard, sehingga apa yang terhapus ditentukan
-- di luar teks SQL. Di sini batasnya tertulis dan tidak dapat digeser pemanggil.

-- name: account_check_table
SELECT COUNT(*)
  FROM ALL_TABLES
 WHERE OWNER      = 'POOLDATA'
   AND TABLE_NAME = 'LST_ACCOUNT'
