-- 0011 — Inbox Compliance: penomoran Post Audit (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Berkas ini TIDAK menyentuh satu pun objek milik sistem lama. Ia hanya membuat SATU
-- sequence baru. Tabel yang memakainya, `POOLDATA.T_CLAIM_COMPLIANCE_H`, dibuat terpisah
-- oleh DBA dan tidak disentuh di sini.
--
-- Meski begitu ia tetap DDL, sehingga tetap menempuh `D-63`: permintaan tertulis tim
-- pengembang, persetujuan Work Owner, pelaksanaan oleh DBA. Akun aplikasi tidak memiliki
-- hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
-- Ia harus dijalankan di BASIS DATA SETIAP ENTITAS, bukan hanya di portal utama —
-- `D-75` menetapkan satu basis data per entitas, dan tab Post Audit berlaku di keempatnya.
-- Entitas yang terlewat akan membuat pengiriman ke Post Audit gagal di entitas itu saja,
-- dengan galat yang menyebut sequence tidak ditemukan.
--
-- ============================================================================
-- KENAPA MULAI DARI 100001
-- ============================================================================
--
-- Kolom `CASEID` sudah berisi nomor terbitan Pega berbentuk `CPL-1` … `CPL-19`, dan Pega
-- MASIH menerbitkannya selama masa paralel. Nomor terbitan aplikasi baru karena itu harus
-- berada di rentang yang tidak mungkin dicapai Pega dalam waktu dekat.
--
-- Keputusan Work Owner 2026-09-24: bentuknya **meniru bentuk Pega** — `CPL-` diikuti angka,
-- bukan tiga segmen bertitik seperti `PNCN.YY.xxxx` (`D-71`) maupun `LPK.YY.xxxx` yang
-- dipakai modul Pelaporan Klaim. Pemisahannya dilakukan lewat RENTANG, bukan lewat bentuk.
--
-- Konsekuensi yang diterima secara sadar, dan tercatat supaya tidak ditemukan sebagai
-- kejutan:
--
--   1. Asal sebuah nomor TIDAK terbaca dari bentuknya. `CPL-100001` dan `CPL-19` terlihat
--      sejenis; yang membedakan hanya besarnya angka. Ini berbeda dari nomor klaim, yang
--      `D-22` sengaja buat dapat dibedakan tanpa tabel pemetaan.
--   2. Rentangnya harus DIJAGA. Bila Pega kelak menerbitkan `CPL-100001`, keduanya
--      bertabrakan — dan tabel ini tidak punya constraint unik yang akan menolaknya.
--      Jarak 100.000 nomor dipilih supaya itu tidak mungkin terjadi pada umur sistem
--      paralel.
--   3. Pengurutan tab Post Audit adalah pengurutan TEKS (lihat inboxcompliance.sql),
--      sehingga `CPL-100001` selalu berada di atas `CPL-19` — nomor baru tampil paling
--      atas. Itu kebetulan yang menguntungkan, bukan yang dirancang.
--
-- NOCACHE dipilih supaya tidak ada nomor yang hilang saat instans dimatikan. Lubang
-- penomoran bukan cacat teknis, tetapi pada nomor yang dibaca orang ia selalu menimbulkan
-- pertanyaan yang mahal dijawab.

CREATE SEQUENCE POOLDATA.CPNC_POST_AUDIT_SEQ
    START WITH 100001
    INCREMENT BY 1
    NOCACHE
    NOCYCLE;

-- Hak pakai untuk akun aplikasi.
--
-- Dijalankan DBA dengan mengganti CPNC_APP menjadi nama akun yang sebenarnya. Tanpa ini,
-- sequence-nya ada tetapi aplikasi tidak dapat memanggilnya, dan galatnya berbunyi
-- "sequence does not exist" — pesan yang menyesatkan karena objeknya sebenarnya ada.
--
-- GRANT SELECT ON POOLDATA.CPNC_POST_AUDIT_SEQ TO CPNC_APP;

-- Hak tulis pada tabel yang dibuat DBA terpisah.
--
-- Dicatat di sini karena keduanya dibutuhkan bersamaan: tanpa INSERT, pengiriman ke Post
-- Audit gagal meski sequence-nya sudah ada.
--
-- GRANT SELECT, INSERT ON POOLDATA.T_CLAIM_COMPLIANCE_H TO CPNC_APP;
