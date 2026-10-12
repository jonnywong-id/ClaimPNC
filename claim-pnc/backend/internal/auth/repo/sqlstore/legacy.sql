-- Kueri tabel WARISAN milik sistem lama, dibaca apa adanya.
--
-- Aplikasi ini hanya MEMBACA ketiganya; penulisnya tetap sistem yang sekarang memiliki
-- tabel itu (ADR-0004, penulis tunggal per tabel).

-- name: local_login_find_active
--
-- Login non-karyawan: broker dan surveyor independen. Ketiga syarat digabung dalam satu
-- kueri dengan sengaja — memisahkannya membuat lamanya jawaban berbeda antara akun yang
-- ada dan yang tidak, dan selisih waktu itu membocorkan keberadaan akun.
--
-- HASH_PASSWORD disimpan heksadesimal huruf besar; UPPER() di kedua sisi menjaga
-- pencocokan tetap benar walau ada baris yang tersimpan huruf kecil.
--
-- # Cabang TIDAK dibaca di sini, dan itu keputusan Work Owner (2026-10-11)
--
-- `M_LOGIN_PNC` tidak punya satu pun kolom cabang, dan tidak ada tabel lokal yang boleh
-- dipakai menggantikannya. Kueri ini sempat `LEFT JOIN POOLDATA.V_HRD_M_MST` dan memakai
-- kolom `LCA_ID`-nya; penggabungan itu DICABUT.
--
-- Sebabnya: `V_HRD_M_MST` maupun `LCA_ID` **nol kemunculan** di seluruh export Pega — ia
-- pilihan sendiri, bukan sumber yang dipakai sistem lama. Rantai Pega yang sebenarnya ada di
-- `RDB List/GetIDCabang-SQL.xml` (dipanggil 13 activity):
--
--   login_aplikasi -> hrdasm.v_hrd_mst@asmd .NIK
--                  -> lst_user_asuransi@asmd .CAB_ID
--                  -> BRANCH.OLDID -> BRANCH.ID
--
-- Cabangnya datang dari `LST_USER_ASURANSI.CAB_ID`, bukan dari HRD, dan tabel itu hanya
-- terjangkau lewat DB Link. Penggantinya diukur dan TIDAK setara: dari 2.400 baris gabungan,
-- `LCA_ID` sama dengan `CAB_ID` hanya pada 1.625 (68%), dan 556 NIK punya lebih dari satu
-- `CAB_ID` sehingga jawaban Pega sendiri ambigu bagi mereka.
--
-- Work Owner menetapkan: **cabang HANYA datang dari layanan HCC**. Pengguna yang tidak
-- membawa data cabang dari sana memang TIDAK berhak membuka Inbox OS Claim per Cabang, dan
-- penolakannya benar — bukan cacat yang perlu ditambal dari tabel lain.
--
-- Akibat yang disengaja: setiap pengguna yang masuk lewat jalur ini berakhir tanpa cabang.
SELECT L.LOGIN_ID,
       L.LOGIN_NAME
  FROM POOLDATA.M_LOGIN_PNC L
 WHERE L.LOGIN_ID = :1
   AND L.ACTIVE_STATUS = '1'
   AND UPPER(L.HASH_PASSWORD) = UPPER(:2)

-- name: service_address
--
-- Alamat endpoint layanan luar. Penyaringnya adalah APP — alias portal — dan bukan
-- APPLICATIONIP seperti rule lama `RDB List/BrowseServiceName_sql-SQL.xml`, yang
-- membandingkan nama server lewat perangkaian string `{ASIS:...}`. Perubahan ini
-- menjadikan pembedaan entitas eksplisit (ADR-0030) sekaligus menutup celah injeksinya.
SELECT SERVICENAME
  FROM POOLDATA.GCNM_CONNECT_REST
 WHERE APP = :1
   AND TYPESERVICE = :2


-- name: local_login_check_table
SELECT LOGIN_ID
  FROM POOLDATA.M_LOGIN_PNC
 WHERE 1 = 0
