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
-- # Kenapa master HRD ikut dibaca di sini
--
-- `M_LOGIN_PNC` TIDAK punya satu pun kolom cabang. Akibatnya setiap pengguna yang masuk
-- lewat jalur ini berakhir tanpa cabang, dan SETIAP layar yang batas datanya cabang
-- menolaknya dengan "Cabang Anda belum terdaftar" — meskipun cabangnya tercatat di tempat
-- lain. Itu bukan penolakan yang benar; itu data yang ada tetapi tidak pernah dibaca.
--
-- `POOLDATA.V_HRD_M_MST` adalah master HRD **lokal** — bukan lewat DB Link, sehingga tidak
-- terhalang `R-03`. Kolom `LCA_ID`-nya berada di ruang kode yang SAMA dengan
-- `DetailBranchCode` dari HCQ, yakni `POOLDATA.BRANCH.OLDID`. Diukur langsung terhadap
-- katalog: mencocokkan `LCA_ID` ke `BRANCH.ID` menghasilkan **0 dari 27.966** baris,
-- sedangkan ke `BRANCH.OLDID` cocok. Jadi nilainya dapat dipakai apa adanya tanpa
-- terjemahan kedua.
--
-- LEFT JOIN, bukan JOIN: pengguna non-karyawan yang sebenarnya — broker dan surveyor
-- independen — memang TIDAK ada di HRD, dan mereka harus tetap dapat masuk. Yang mereka
-- dapat tetap cabang kosong, dan itu memang benar: mereka tidak punya cabang.
SELECT L.LOGIN_ID,
       L.LOGIN_NAME,
       H.LCA_ID AS DETAIL_BRANCH_CODE
  FROM POOLDATA.M_LOGIN_PNC L
       LEFT JOIN POOLDATA.V_HRD_M_MST H
              ON UPPER(TRIM(H.LOGIN_APLIKASI)) = UPPER(TRIM(L.LOGIN_ID))
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
