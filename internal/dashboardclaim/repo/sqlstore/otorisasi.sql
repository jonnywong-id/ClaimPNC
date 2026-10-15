-- Grup akses pengguna — menentukan link mana yang dipasang pada nomor klaim.
--
-- ============================================================================
-- KENAPA MODUL INI MEMBACANYA SENDIRI
-- ============================================================================
--
-- Tempat yang lebih tepat bagi grup pengguna adalah SESI, dibaca sekali saat masuk. Itu
-- TIDAK dikerjakan di sini karena modul Login termasuk yang tidak boleh diubah (Isolasi
-- Protektif), dan menambah field ke sesi berarti menyuntingnya.
--
-- Akibat yang diterima: setiap modul yang kelak membutuhkan grup akan membaca ulang tabel
-- yang sama. Begitu `F-3` membawa grup ke dalam sesi, kueri ini dibuang dan pemanggilnya
-- berpindah ke sesi — satu tempat, bukan sepuluh.
--
-- ============================================================================
-- SUMBERNYA, DAN SATU YANG SENGAJA TIDAK DIPAKAI
-- ============================================================================
--
--     POOLDATA.M_LOGIN_GROUP_PNC     LOGIN_ID -> GROUP_ID      67 baris, 31 pengguna
--     POOLDATA.M_OTORISASI_PNC       GROUP_ID -> MENU_ID       395 baris
--     POOLDATA.M_MENU_APLIKASI_PNC   MENU_ID  -> nama menu      84 baris
--
-- `POOLDATA.T_ACCESS_GROUP_PNC` **tidak dipakai** — keputusan Work Owner 2026-10-12. Tabel
-- itu juga memuat kolom `ACCESS_GROUP` dan mudah terpakai keliru; ia sengaja disebut di sini
-- supaya penggantinya tidak dicari-cari lagi.
--
-- Nama grup di sini TANPA awalan `GCNMFW:` yang dipakai rule Pega — `PncPICTeknik`, bukan
-- `GCNMFW:PncPICTeknik`.
--
-- ============================================================================
-- KUNCINYA IDENTITAS, BUKAN OPERATOR ID
-- ============================================================================
--
-- `LOGIN_ID` sepadan dengan `POOLDATA.M_LOGIN_PNC.LOGIN_ID` — 31 dari 31, diukur. Dan
-- `internal/auth/identity.go:15` menyatakan bagi pengguna tabel itu, **identitasnya adalah
-- `LOGIN_ID`**. Jadi yang dikirim ke kueri ini adalah `Caller.User.Identity`.
--
-- `OperatorID` TIDAK dipakai, dan tidak bisa: `ADR-0024` pertanyaan terbuka nomor 4 mencatat
-- cara mencocokkan identitas HCC/HCQ dengan `OPERATOR_ID` belum ditetapkan, sehingga field
-- itu sengaja dibiarkan KOSONG di sesi — bukan ditebak.
--
-- Bentuk `LOGIN_ID` beragam: 27 dari 31 berupa surel, 4 berupa Operator ID. Keduanya dibaca
-- apa adanya; tidak ada penyusunan kunci di sini.

-- name: grup_pengguna
-- Seluruh grup milik satu pengguna.
--
-- Mengembalikan BANYAK baris, dan itu bukan kelonggaran: 22 dari 31 pengguna punya lebih
-- dari satu grup, terbanyak LIMA. Mengambil satu baris saja akan salah pada mayoritas
-- pengguna, dan salahnya tidak terlihat — ia hanya memberi hak yang kurang.
--
-- `UPPER` dipasang di KEDUA sisi karena bentuk `LOGIN_ID` beragam dan surel tidak peka
-- besar-kecil huruf. Tabelnya 67 baris, sehingga index tidak menjadi pertimbangan.
--
-- `STS_AKTIF` TIDAK disaring: kolom itu tidak ada pada tabel ini. Penonaktifan pengguna
-- ditangani `M_LOGIN_PNC.ACTIVE_STATUS`, dan pengguna nonaktif tidak akan punya sesi.
SELECT G.GROUP_ID
  FROM POOLDATA.M_LOGIN_GROUP_PNC G
 WHERE UPPER(TRIM(G.LOGIN_ID)) = UPPER(TRIM(:1))
 ORDER BY G.GROUP_ID

-- name: grup_pengguna_check
-- Membuktikan tabel grup terbaca akun aplikasi, beserta KEDUA kolom yang dipakai.
--
-- Probe yang hanya membuktikan tabelnya ada akan lulus terhadap kolom yang tidak ada —
-- kegagalan yang sudah terjadi dua kali di modul ini (`TOTAL_JOB`, lalu `CASEID_1`).
SELECT G.LOGIN_ID, G.GROUP_ID
  FROM POOLDATA.M_LOGIN_GROUP_PNC G
 WHERE 1 = 0
