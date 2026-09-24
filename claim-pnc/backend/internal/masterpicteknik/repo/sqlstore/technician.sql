-- Kueri master PIC Teknik.
--
-- # DUA OBJEK, dan pembagiannya disengaja
--
--   POOLDATA.V_MST_USER_TEKNIS   DIBACA untuk daftar. Memuat TOTAL_JOB, yang tidak ada
--                                di tabelnya.
--   POOLDATA.MST_USER_TEKNIK     DIBACA untuk satu baris, dan SATU-SATUNYA yang DITULIS.
--
-- Pembagian itu bukan pilihan gaya — ia meniru sistem lama persis:
--
--   Report Definition/BrowseVMstUserTeknis_RD  kelasnya ASM-FW-GCNMFW-Int-V_MST_USER_TEKNIS
--                                              → grid membaca VIEW
--   RDB List/GetMasterPICTeknis-SQL.xml        FROM POOLDATA.MST_USER_TEKNIK
--                                              → form membaca TABEL
--   Database/PEGA_MST_USER_TEKNIS.prc          INSERT/UPDATE POOLDATA.mst_user_teknik
--                                              → penulisan ke TABEL
--
-- Perhatikan ejaannya berbeda dan itu BUKAN salah ketik: view berakhiran **TEKNIS**,
-- tabel berakhiran **TEKNIK**. Keduanya terbaca apa adanya dari export.
--
-- # PEMETAAN KOLOM
--
-- Alias sistem lama menyesatkan dan TIDAK dibawa masuk. Berkas ini satu-satunya tempat
-- ketiganya dapat dibandingkan:
--
--   kolom            alias Pega lama     nama domain
--   --------------------------------------------------------------------------
--   OPERATOR_ID      OPERATOR_ID         OperatorID
--                    "MCL_Name" pada BrowseEmailUserTeknis — alias yang menyebut
--                    kolom NAMA padahal isinya ID
--   MCL_NAME         MCL_NAME            Name          (diturunkan dari direktori)
--   EMAIL            EMAIL               Email
--   TYPE_BUSINESS    TYPE_BUSINESS       BusinessLine
--   TEAM_GROUP       TEAM_GROUP          Group
--   ATASAN           ATASAN              Supervisor    (berisi OPERATOR_ID, bukan nama)
--   COUNTER_QUOTA    COUNTER_QUOTA       Quota
--   COUNTER_QUOTA2   "OLD_OPERATOR_ID"   ExternalQuota (ANGKA, bukan identitas)
--   TOTAL_JOB        TOTAL_JOB           Workload      (hanya di view; hanya dibaca)
--   GROUPPANEL       "IBNR"              PanelGroup    (hanya di tabel; hanya dibaca)
--   STS_AKTIF        STS_AKTIF           Active        ('1' = aktif)
--
-- # BELUM DIVERIFIKASI KE BASIS DATA — nama kolom view
--
-- Definisi POOLDATA.V_MST_USER_TEKNIS tidak ada di export; yang terbaca hanyalah NAMA
-- PROPERTI kelas Pega yang memetakannya. Pega memetakan properti ke kolom bernama sama,
-- sehingga kolom view diduga bernama `OLD_OPERATOR_ID` — bukan `COUNTER_QUOTA2` seperti
-- di tabelnya.
--
-- Dugaan itu harus dikonfirmasi DBA sebelum dipakai terhadap Oracle:
--
--     SELECT text FROM all_views
--      WHERE owner = 'POOLDATA' AND view_name = 'V_MST_USER_TEKNIS';
--
-- Bila ternyata berbeda, yang berubah hanya kueri `technician_list` di bawah — tidak ada
-- satu baris pun kode Go yang ikut berubah. Dicatat terbuka di
-- docs/catatan-pengembangan.md.
--
-- # Kenapa tidak lagi lewat PEGA_MST_USER_TEKNIS
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. `D-68` menambahkan alasan yang lebih keras, dan procedure ini mengulang cacat
-- yang sama persis: parameter keluarannya bernama `ErrMsg` tetapi pada jalur BERHASIL ia
-- berisi kalimat "Data Sudah Disimpan dengan ID : ..." — sehingga pemanggil tidak dapat
-- membedakan berhasil dari gagal tanpa membaca teks.
--
-- Dua hal lagi yang tidak dibawa:
--
--   - Procedure menghitung `id_site || lpad(MST_USER_TEKNIS_SEQ.nextval, 6, '0')` ke
--     variabel `id_mst_user_teknis`, lalu TIDAK PERNAH memakainya — INSERT-nya memakai
--     `IDPega`. Generator itu kode mati.
--   - Cabang UPDATE-nya menulis `SET OPERATOR_ID = IDPega ... WHERE OPERATOR_ID = IDPega`,
--     yaitu menyetel kolom ke nilainya sendiri. Ia tidak mengubah apa pun dan hanya
--     membuat pembaca mengira kunci ini dapat berpindah.
--
-- Kolom selalu disebut namanya; SELECT * dilarang.

-- name: technician_list
--
-- Daftar untuk grid. Menggantikan Report Definition `BrowseVMstUserTeknis_RD` beserta
-- KEDUA penyaringnya yang digabung `A AND B`.
--
-- Penyaring B dipatok langsung di Report Definition sebagai `STS_AKTIF = '1'`, sehingga
-- grid Pega TIDAK PERNAH menampilkan petugas nonaktif. Keputusan Work Owner 2026-09-19
-- mempertahankannya persis. Nilainya tetap lewat parameter binding, bukan ditulis di
-- dalam teks SQL, supaya sandi aktif hidup di satu tempat saja — masterpicteknik.ActiveCode.
--
-- Penyaring A (`TYPE_BUSINESS = Param.type_business`) TIDAK dibawa: nilainya dipasok
-- pemanggil, dan seluruh pemanggil yang ada di export mengisinya kosong — yang pada Report
-- Definition Pega berarti penyaringnya tidak berlaku. Menerapkannya di sini akan
-- mengosongkan daftar.
SELECT OPERATOR_ID,
       MCL_NAME,
       EMAIL,
       TYPE_BUSINESS,
       TEAM_GROUP,
       ATASAN,
       COUNTER_QUOTA,
       OLD_OPERATOR_ID,
       TOTAL_JOB,
       STS_AKTIF
  FROM POOLDATA.V_MST_USER_TEKNIS
 WHERE STS_AKTIF = :1
 ORDER BY OPERATOR_ID

-- name: technician_get
--
-- Satu baris untuk form. Menggantikan `GetMasterPICTeknis`, termasuk apa yang TIDAK
-- disaringnya: tidak ada penyaring STS_AKTIF sama sekali di sana.
--
-- Itulah yang membuat petugas nonaktif — yang sudah hilang dari daftar — masih dapat
-- dibuka dan diaktifkan kembali. Menambahkan penyaring aktif di sini akan mengubah
-- keadaan "tidak muncul di daftar" menjadi "tidak dapat dijangkau sama sekali".
--
-- Dibaca dari TABEL, bukan view, karena dua alasan: GROUPPANEL hanya ada di tabel, dan
-- membaca dari tabel yang sama dengan yang ditulis membuat apa yang disimpan dan apa yang
-- dibaca kembali pasti sama.
--
-- TOTAL_JOB tidak ikut — ia tidak ada di tabel. Form mendapatkannya dari baris daftar,
-- sama seperti layar Pega: `GetMasterPICTeknis` pun tidak menyertakannya.
--
-- UPPER di kedua sisi adalah PERBAIKAN yang disengaja. Procedure lama membandingkan
-- `operator_id = IDPega` tanpa UPPER sementara pencarian namanya memakai UPPER, sehingga
-- "BUDI" dan "budi" dapat menjadi dua baris berbeda untuk orang yang sama.
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

-- name: technician_insert
--
-- GROUPPANEL sengaja tidak ikut: procedure lama pun tidak pernah menulisnya, pada cabang
-- INSERT maupun UPDATE. Urutan kolomnya mengikuti INSERT procedure lama supaya
-- perbandingan baris per baris saat uji kesetaraan tetap mudah dibaca.
INSERT INTO POOLDATA.MST_USER_TEKNIK
       (OPERATOR_ID, COUNTER_QUOTA, TYPE_BUSINESS, EMAIL, STS_AKTIF,
        TEAM_GROUP, ATASAN, COUNTER_QUOTA2, MCL_NAME)
VALUES (:1, :2, :3, :4, :5,
        :6, :7, :8, :9)

-- name: technician_update
--
-- OPERATOR_ID tidak ikut diubah, berbeda dari procedure lama yang menyetelnya ke nilainya
-- sendiri. Mencantumkannya tidak mengubah apa pun dan hanya menyesatkan pembaca.
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

-- name: technician_check_table
--
-- Memastikan seluruh kolom yang dipakai modul ini ada dan dapat dibaca akun aplikasi,
-- tanpa mengambil satu baris pun. Aman dijalankan terhadap produksi, dan dipakai mode
-- periksa untuk membedakan dua sebab kegagalan yang tampak mirip: tabelnya tidak ada di
-- portal itu, versus tidak punya hak baca.
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

-- name: technician_check_view
--
-- Pasangan pemeriksaan di atas untuk VIEW-nya, dan ia ada karena satu sebab yang nyata:
-- nama kolom view belum dapat diverifikasi dari export (lihat catatan di kepala berkas).
-- Bila dugaan `OLD_OPERATOR_ID` keliru, kegagalannya terbaca di mode periksa — bukan
-- ditemukan pengguna sebagai daftar yang gagal dimuat.
SELECT OPERATOR_ID,
       MCL_NAME,
       EMAIL,
       TYPE_BUSINESS,
       TEAM_GROUP,
       ATASAN,
       COUNTER_QUOTA,
       OLD_OPERATOR_ID,
       TOTAL_JOB,
       STS_AKTIF
  FROM POOLDATA.V_MST_USER_TEKNIS
 WHERE 1 = 0
