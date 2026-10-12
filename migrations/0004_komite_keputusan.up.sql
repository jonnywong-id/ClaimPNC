-- 0004 — Keputusan Komite: tabel jejak baru milik aplikasi (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- # Apa yang dibuat, dan apa yang TIDAK disentuh
--
-- Berkas ini hanya MENAMBAH objek baru:
--
--   * POOLDATA.CPNC_KOMITE_KEPUTUSAN     tabel jejak keputusan komite
--   * dua indeks pendukung
--   * satu constraint unik
--
-- TIDAK ADA satu pun objek milik sistem lama yang diubah. Tidak ada ALTER, tidak ada
-- penggantian view, tidak ada perubahan hak pada tabel yang sedang melayani produksi.
--
-- Ia tetap menempuh prosedur `D-63`: permintaan tertulis tim pengembang, persetujuan Work
-- Owner, pelaksanaan DBA, lalu pengujian dengan MENJALANKAN PEGA DAN GO BERSAMAAN. Akun
-- aplikasi tidak memiliki hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
--
-- # Kenapa tabel baru, dan bukan T_CLAIM_KOMITE_LIST
--
-- Keputusan komite di sistem lama tersimpan di `POOLDATA.T_CLAIM_KOMITE_LIST`
-- (`STATUSAPPROVE`, `NOTEKOMITE`, `NAMAKOMITE`, `KOMITEKE`). Tabel itu **tidak dapat
-- dipakai**, dan alasannya bukan selera:
--
--   1. Ia masih DITULIS Pega lewat `INSERTDATAKOMITELIST`, dan `P-1` menetapkan satu
--      tabel hanya boleh ditulis satu sistem. Dua penulis dengan aturan validasi yang
--      berbeda menghasilkan konflik data yang hampir mustahil dilacak.
--   2. Ia juga DIBACA puluhan kueri Pega. Satu baris yang bentuknya sedikit berbeda dari
--      yang mereka harapkan dapat menghentikan alur yang sedang melayani produksi.
--
-- `TKT-B07-002` sudah menetapkan jalan keluarnya pada bagian migrasi skema: "Menambah
-- tabel jejak komite. Backward-compatible."
--
--
-- # Akibat yang HARUS diketahui sebelum modul ini menyala di produksi
--
-- Selama masa paralel, kasus yang sudah diputuskan di sistem baru **tetap terbuka di
-- Pega**: `PYSTATUSWORK` di sana tidak berubah, dan penugasan worklist-nya tidak dicabut.
-- Layar Inbox Komite menutupinya dengan menumpangkan keputusan kita di atas baris
-- warisan, tetapi **Pega tidak tahu apa-apa tentang itu**.
--
-- Dua akibatnya, dan keduanya sifat dari menjalankan dua sistem sekaligus — bukan cacat:
--
--   1. Orang yang sama dapat memutuskan kasus itu LAGI di Pega. Keputusan mana yang sah
--      adalah pertanyaan untuk Work Owner, bukan untuk kode ini.
--   2. Alur Pega tidak berlanjut ke jenjang berikutnya karena persetujuan kita.
--
--
-- # Yang HARUS dikerjakan DBA bersama tabel ini
--
--  1. PASTIKAN NAMANYA BELUM DIPAKAI:
--
--       SELECT object_name, object_type FROM all_objects
--        WHERE owner = 'POOLDATA' AND object_name LIKE 'CPNC_KOMITE%';
--
--     Berkas ini TIDAK memakai `CREATE OR REPLACE` dan tidak menimpa apa pun; bila
--     namanya sudah ada, pernyataannya gagal — dan itu memang yang diinginkan.
--
--  2. BERI HAK YANG TEPAT, DAN HANYA ITU. Tabel ini APPEND-ONLY (`ADR-0012`,
--     `09-DATABASE-STRATEGY.md` §8): keputusan komite tidak dapat dihapus maupun diubah.
--
--       GRANT SELECT, INSERT ON POOLDATA.CPNC_KOMITE_KEPUTUSAN TO <akun aplikasi>;
--
--     JANGAN memberikan UPDATE maupun DELETE. Aturan yang hanya ada di dalam kode dapat
--     dilanggar oleh kode berikutnya; aturan yang ada di hak akses tidak.
--
--     Ini lebih penting di sini daripada di tabel mana pun yang sudah ada: `D-59`
--     menghapus pemisahan tugas — satu orang dapat membuat, menyetujui, dan membayarkan
--     satu klaim bila perannya memiliki ketiga menu itu — sehingga jejak inilah
--     SATU-SATUNYA kontrol pengimbang yang tersisa.
--
--  3. RETENSI mengikuti retensi data klaim yang berlaku sekarang (`D-62`) — satu
--     kebijakan untuk keduanya, bukan kebijakan terpisah. ANGKANYA belum diserahkan, dan
--     sampai itu tiba tidak ada penghapusan berkala yang boleh dijadwalkan.
--
--
-- # Empat portal, empat kali
--
-- `D-75` menetapkan satu database per entitas. Migrasi ini karena itu dijalankan EMPAT
-- KALI — sekali per portal — dan gagal di salah satunya membuat portal itu tertinggal
-- versi. Inbox Komite pada portal yang tertinggal akan menampilkan kasus tetapi menolak
-- setiap keputusan dengan galat tabel tidak ditemukan.

-- ---------------------------------------------------------------------------
-- CPNC_KOMITE_KEPUTUSAN — satu baris per keputusan komite.
--
-- Bukan satu baris per kasus. Satu kasus dapat menempuh beberapa jenjang, dan setiap
-- jenjang meninggalkan barisnya sendiri. Keadaan kasus DIHITUNG dari baris-baris ini
-- (`komite.Evaluate`), tidak disimpan sebagai kolom — keadaan yang disimpan dapat
-- berselisih dengan kejadian yang membentuknya, dan selisihnya tidak terlihat sampai
-- seseorang membandingkan keduanya.
-- ---------------------------------------------------------------------------

CREATE TABLE POOLDATA.CPNC_KOMITE_KEPUTUSAN (
    -- Pengenal acak 128 bit dalam heksadesimal, dibangkitkan aplikasi.
    --
    -- Acak, bukan berurut: pengenal keputusan tidak boleh membocorkan berapa banyak
    -- keputusan yang sudah tercatat, dan tidak boleh dapat ditebak dari pengenal lain.
    ID                VARCHAR2(32)   NOT NULL,

    -- Nomor case komite — `pyID` pada kelas `ASM-FW-GCNMFW-Work-Komite`.
    --
    -- TIDAK ada foreign key ke tabel warisan, dan ketiadaannya disengaja: constraint dari
    -- tabel milik kita ke tabel milik Pega akan membuat penghapusan atau pengarsipan di
    -- sisi Pega GAGAL karena baris kita. Kita tidak boleh menghalangi sistem yang sedang
    -- melayani produksi.
    CASE_ID           VARCHAR2(64)   NOT NULL,

    -- Nomor klaim, disimpan SEBAGAI SALINAN.
    --
    -- Ia sudah tanpa prefix `ASM-FW-GCNMFW-WORK ` (`D-22`). Disalin, bukan dirujuk,
    -- supaya jejak ini tetap terbaca utuh bila kasus warisannya kelak diarsipkan.
    NOMOR_KLAIM       VARCHAR2(50),

    -- Jenjang keberapa yang diputuskan — `KOMITEKE` di sistem lama.
    --
    -- Ditetapkan SERVER dari keadaan kasus, tidak pernah dikirim klien.
    JENJANG           NUMBER(3)      NOT NULL,

    -- 'setuju', 'tolak', atau 'kembalikan'.
    --
    -- Disimpan sebagai TEKS, bukan angka seperti `STATUSAPPROVE` yang bernilai '1' atau
    -- bukan. Alasannya terbaca langsung dari data warisan: di sana kolom yang sama
    -- membawa "belum diputuskan" dan "ditolak" dalam satu nilai kosong, sehingga kueri
    -- lama menurunkan `else 'DITOLAK'` — dan klaim yang masih menunggu ikut terbaca
    -- sudah ditolak.
    --
    -- Nilainya berbahasa Indonesia karena ia sama dengan nilai pada kontrak API.
    KEPUTUSAN         VARCHAR2(20)   NOT NULL,

    -- Catatan komite. WAJIB diisi aplikasi pada 'tolak' dan 'kembalikan'.
    --
    -- Kewajibannya ditegakkan di Go, bukan di sini: ia bergantung pada nilai kolom lain,
    -- dan check constraint yang menyandingkan dua kolom jauh lebih sulit dibaca daripada
    -- aturan yang tertulis sebagai kalimat di `DecisionCommand.Validate`.
    CATATAN           VARCHAR2(1000),

    -- Login yang DIKETIK pengguna, dinormalkan huruf besar.
    --
    -- Inilah kunci yang Work Owner tetapkan untuk mencocokkan identitas sesi dengan
    -- `OPERATOR_ID` sistem lama (`keputusan-implementasi.md` §16.5).
    ACTOR_LOGIN       VARCHAR2(64)   NOT NULL,

    -- Nama pemutus, disimpan BERSAMA keputusannya.
    --
    -- Tidak dirujuk ke tabel pengguna: jejak yang namanya diambil lewat join akan BERUBAH
    -- ketika orangnya berganti nama atau catatannya dihapus — dan jejak yang dapat
    -- berubah bukan jejak.
    ACTOR_NAMA        VARCHAR2(100),

    -- Waktu keputusan, dalam UTC.
    --
    -- `DB-8` menetapkan seluruh waktu disimpan UTC, dan konversi ke WIB terjadi di SATU
    -- tempat saja (`F-5`). Ini meninggalkan pola lama yang menambahkan tujuh jam secara
    -- manual di 118 titik pada 36 activity — sebagian pada satu sisi sebuah perbandingan
    -- dan tidak pada sisi lainnya (`R-12`).
    PADA              TIMESTAMP(6) WITH TIME ZONE NOT NULL,

    CONSTRAINT CPNC_KOMITE_KEPUTUSAN_PK PRIMARY KEY (ID),

    -- Satu orang memutuskan SEKALI per kasus.
    --
    -- Lapisan usecase sudah memeriksanya lebih dulu, tetapi pemeriksaan itu dan
    -- penyimpanannya BUKAN satu operasi atomik: dua permintaan yang tiba bersamaan — dua
    -- tab yang terbuka, keduanya ditekan — dapat lolos keduanya. Constraint inilah yang
    -- menutup celah itu.
    --
    -- Namanya dipakai kode untuk menerjemahkan bentrok menjadi pesan yang dapat dibaca,
    -- lewat konstanta `sqlstore.UniqueKeyName`. Menggantinya di sini tanpa mengganti
    -- konstanta itu akan membuat keputusan ganda muncul sebagai galat 500.
    CONSTRAINT CPNC_KOMITE_KEPUTUSAN_UK UNIQUE (CASE_ID, ACTOR_LOGIN),

    -- Keputusan yang tidak dikenali tidak boleh pernah tersimpan.
    --
    -- Nilainya sudah divalidasi di Go. Diulang di sini karena tabel ini append-only dan
    -- tidak dapat diperbaiki: satu baris ber-KEPUTUSAN salah ketik akan ada selamanya,
    -- dan `komite.Evaluate` akan mengabaikannya diam-diam — kasusnya tampak belum
    -- diputuskan padahal barisnya ada.
    CONSTRAINT CPNC_KOMITE_KEPUTUSAN_CK_JENIS
        CHECK (KEPUTUSAN IN ('setuju', 'tolak', 'kembalikan')),

    CONSTRAINT CPNC_KOMITE_KEPUTUSAN_CK_JENJANG CHECK (JENJANG >= 1)
);

COMMENT ON TABLE POOLDATA.CPNC_KOMITE_KEPUTUSAN IS
    'Jejak keputusan komite (TKT-B07-002). Append-only: akun aplikasi tidak boleh punya hak UPDATE/DELETE';

-- Inbox membaca keputusan SELURUH kasus pada satu halaman sekaligus, dikunci CASE_ID.
CREATE INDEX POOLDATA.IX_CPNC_KOMITE_KEPUTUSAN_CASE
    ON POOLDATA.CPNC_KOMITE_KEPUTUSAN (CASE_ID);

-- Menelusuri seluruh keputusan seseorang — dipakai saat ada pertanyaan tentang siapa
-- menyetujui apa, yang pada modul ini adalah pertanyaan yang benar-benar akan diajukan.
CREATE INDEX POOLDATA.IX_CPNC_KOMITE_KEPUTUSAN_ACTOR
    ON POOLDATA.CPNC_KOMITE_KEPUTUSAN (ACTOR_LOGIN, PADA);
