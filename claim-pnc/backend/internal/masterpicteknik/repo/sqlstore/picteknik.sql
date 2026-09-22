-- Kueri master PIC Teknik: POOLDATA.MST_USER_TEKNIK.
--
-- # PEMETAAN KOLOM
--
-- Alias sistem lama menyesatkan dan TIDAK dibawa masuk. Tabel ini satu-satunya tempat
-- ketiganya dapat dibandingkan:
--
--   kolom            alias Pega lama     nama domain
--   --------------------------------------------------------------------------
--   OPERATOR_ID      OPERATOR_ID         IDOperator
--                    "MCL_Name" pada BrowseEmailUserTeknis — alias yang menyebut
--                    kolom NAMA padahal isinya ID
--   MCL_NAME         MCL_NAME            Nama          (diturunkan dari direktori)
--   EMAIL            EMAIL               Email
--   TYPE_BUSINESS    TYPE_BUSINESS       LiniBisnis
--   TEAM_GROUP       TEAM_GROUP          Grup
--   ATASAN           ATASAN              Atasan
--   COUNTER_QUOTA    COUNTER_QUOTA       Kuota
--   COUNTER_QUOTA2   "OLD_OPERATOR_ID"   KuotaLuar     (ANGKA, bukan id operator)
--   GROUPPANEL       "IBNR"              GrupPanel     (hanya dibaca)
--   STS_AKTIF        STS_AKTIF           Aktif         ('1' = aktif)
--
-- # Kenapa tidak lagi lewat PEGA_MST_USER_TEKNIS
--
-- `D-02` menetapkan logika stored procedure naik ke Go. `D-68` menambahkan alasan yang
-- lebih keras, dan procedure ini mengulang cacat yang sama persis: parameter keluarannya
-- bernama `ErrMsg` tetapi pada jalur BERHASIL ia berisi kalimat
-- "Data Sudah Disimpan dengan ID : ..." — sehingga pemanggil tidak dapat membedakan
-- berhasil dari gagal tanpa membaca teks.
--
-- Satu hal lagi yang tidak dibawa: procedure menghitung
-- `id_site || lpad(MST_USER_TEKNIS_SEQ.nextval, 6, '0')` ke variabel
-- `id_mst_user_teknis`, lalu TIDAK PERNAH memakainya — INSERT-nya memakai `IDPega`.
-- Generator itu kode mati.
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding
-- — rule lama merangkai `{ASIS:InputCOL.OPERATOR_ID}` langsung ke dalam WHERE, dan itu
-- persis yang tidak diulang di sini.

-- name: pic_teknik_list
SELECT OPERATOR_ID,
       MCL_NAME,
       EMAIL,
       TYPE_BUSINESS,
       TEAM_GROUP,
       ATASAN,
       COUNTER_QUOTA,
       COUNTER_QUOTA2,
       GROUPPANEL,
       STS_AKTIF
  FROM POOLDATA.MST_USER_TEKNIK
 ORDER BY OPERATOR_ID

-- name: pic_teknik_get
SELECT OPERATOR_ID,
       MCL_NAME,
       EMAIL,
       TYPE_BUSINESS,
       TEAM_GROUP,
       ATASAN,
       COUNTER_QUOTA,
       COUNTER_QUOTA2,
       GROUPPANEL,
       STS_AKTIF
  FROM POOLDATA.MST_USER_TEKNIK
 WHERE UPPER(OPERATOR_ID) = UPPER(:1)

-- name: pic_teknik_insert
--
-- GROUPPANEL sengaja tidak ikut: procedure lama pun tidak pernah menulisnya, pada
-- cabang INSERT maupun UPDATE.
INSERT INTO POOLDATA.MST_USER_TEKNIK
       (OPERATOR_ID, MCL_NAME, EMAIL, TYPE_BUSINESS, TEAM_GROUP,
        ATASAN, COUNTER_QUOTA, COUNTER_QUOTA2, STS_AKTIF)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9)

-- name: pic_teknik_update
--
-- OPERATOR_ID tidak ikut diubah meski procedure lama menuliskannya
-- (`SET OPERATOR_ID = IDPega ... WHERE OPERATOR_ID = IDPega`) — menyetel kolom ke
-- nilainya sendiri tidak mengubah apa pun, dan mencantumkannya hanya membuat pembaca
-- mengira kunci ini dapat berpindah.
UPDATE POOLDATA.MST_USER_TEKNIK
   SET MCL_NAME       = :1,
       EMAIL          = :2,
       TYPE_BUSINESS  = :3,
       TEAM_GROUP     = :4,
       ATASAN         = :5,
       COUNTER_QUOTA  = :6,
       COUNTER_QUOTA2 = :7,
       STS_AKTIF      = :8
 WHERE UPPER(OPERATOR_ID) = UPPER(:9)

-- name: pic_teknik_check_table
--
-- Memastikan seluruh kolom dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
-- Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip: kolom
-- tidak ada versus tidak punya hak baca.
SELECT OPERATOR_ID,
       MCL_NAME,
       EMAIL,
       TYPE_BUSINESS,
       TEAM_GROUP,
       ATASAN,
       COUNTER_QUOTA,
       COUNTER_QUOTA2,
       GROUPPANEL,
       STS_AKTIF
  FROM POOLDATA.MST_USER_TEKNIK
 WHERE 1 = 0

-- name: operator_name
--
-- Direktori operator. Meniru `SelectMstUserTeknisMclName` apa adanya, termasuk
-- perbandingan UPPER di kedua sisi.
--
-- Tabel ini milik Pega (`DATAPEGA.PR_OPERATORS`) dan hanya DIBACA (ADR-0004). Ketika
-- Pega dimatikan, yang diganti adalah pengisi seam DirektoriOperator — bukan modul ini.
SELECT PYUSERNAME
  FROM DATAPEGA.PR_OPERATORS
 WHERE UPPER(PYUSERIDENTIFIER) = UPPER(:1)
   AND PXOBJCLASS = 'Data-Admin-Operator-ID'
