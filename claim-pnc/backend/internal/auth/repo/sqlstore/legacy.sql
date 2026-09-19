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
SELECT LOGIN_ID,
       LOGIN_NAME
  FROM POOLDATA.M_LOGIN_PNC
 WHERE LOGIN_ID = :1
   AND ACTIVE_STATUS = '1'
   AND UPPER(HASH_PASSWORD) = UPPER(:2)

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
