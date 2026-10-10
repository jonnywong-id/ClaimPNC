-- 0012 — Inbox Compliance: keputusan form Compliance Checker (Oracle 19c)
--
-- ============================================================================
-- BACA SELURUH BERKAS INI SEBELUM MENJALANKAN SATU PERNYATAAN PUN.
-- ============================================================================
--
-- Berkas ini membuat SATU tabel baru. Ia TIDAK menyentuh satu pun objek milik sistem lama.
--
-- Meski begitu ia tetap DDL, sehingga tetap menempuh `D-63`: permintaan tertulis tim
-- pengembang, persetujuan Work Owner, pelaksanaan oleh DBA. Akun aplikasi tidak memiliki
-- hak DDL.
--
-- BERKAS INI BELUM PERNAH DIJALANKAN DI LINGKUNGAN MANA PUN.
--
-- Ia harus dijalankan di BASIS DATA SETIAP ENTITAS, bukan hanya di portal utama (`D-75`).
-- Entitas yang terlewat membuat form Compliance Checker gagal menyimpan di entitas itu
-- saja, dengan galat yang menyebut tabelnya tidak ditemukan.
--
-- ============================================================================
-- KENAPA TABEL BARU, BUKAN MENULIS KE KLAIMNYA
-- ============================================================================
--
-- Di Pega, `Activity/SetComplianceResult` menyimpan keputusan Compliance PADA KLAIMNYA:
--
--     .ClaimData.PilihanCompliance        pilihan petugas
--     .ClaimData.NotePilihanCompliance    catatan singkat
--     .ClaimData.ComplianceRemark         catatan yang dibawa ke Post Audit
--     .ClaimData.CPLValidDate             waktu, bila pilihannya Bayar/Valid
--     .TanggalKirimPostAudit              waktu, bila pilihannya Bayar/PostAudit
--     .StatusClaim := '1151'              status klaim berubah
--     .ComplianceStatus per Objek         status per Objek Pertanggungan
--
-- Ketujuhnya tinggal di `POOLDATA.T_CLAIM_PNC` dan `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`.
--
-- Selama masa paralel, kedua tabel itu **dimiliki Pega** — `P-1` menetapkan setiap tabel
-- hanya boleh ditulis SATU sistem. Menulisnya dari aplikasi baru berarti dua sistem
-- menulis satu tabel dengan aturan validasi yang berbeda, yakni kelas kerusakan data yang
-- `P-1` ada untuk mencegahnya. Tabel klaim itu dibaca **116 rule Pega** yang sedang
-- melayani produksi.
--
-- Karena itu lima isian pertama disimpan di tabel ini, dan dua yang terakhir — perubahan
-- status klaim dan status per objek — **TIDAK DIKERJAKAN SAMA SEKALI**.
--
-- ============================================================================
-- APA YANG BELUM BERJALAN KARENA KEPUTUSAN DI ATAS
-- ============================================================================
--
-- Ini harus dibaca sebagai keterbatasan yang disadari, bukan sebagai fitur yang jalan:
--
--   1. Keputusan tersimpan dan terbaca di aplikasi baru, tetapi **klaimnya di Pega tidak
--      berpindah status** dan **tidak keluar dari antrean `CompliancePNC`**. Petugas yang
--      memutuskan di aplikasi baru akan tetap melihat klaim itu di antreannya, dan petugas
--      Pega juga.
--   2. Kedua Ticket rule yang `SetComplianceResult` picu — `SendtoAnalysator` bila PA,
--      `SendToPICTravel` bila Travel — tidak dijalankan. Klaim tidak berpindah ke Analyst
--      maupun ke PIC Teknik.
--   3. Baris riwayat `InsertHistoryClaimPNC` tidak ditulis.
--
-- Ketiganya hilang sampai kepemilikan tabel klaim berpindah ke aplikasi baru, yang menempuh
-- `D-63` dan `P-2`: modul `B-2` Registrasi Klaim harus sudah pindah lebih dulu, karena
-- ialah pemilik tabel itu.
--
-- Sampai saat itu, form Compliance Checker **mencatat keputusan — ia belum menjalankan
-- alurnya.** Kalimat itu perlu sampai ke petugas yang memakainya, bukan hanya ke DBA.
--
-- ============================================================================
-- KENAPA SATU BARIS PER KLAIM
-- ============================================================================
--
-- Karena `.ClaimData.PilihanCompliance` adalah SATU properti pada klaimnya, bukan daftar.
-- Petugas yang membuka form kedua kalinya dan mengubah pilihannya MENGUBAH keputusan itu;
-- ia tidak menambah keputusan kedua. Kuerinya karena itu `MERGE`, bukan `INSERT`.
--
-- Riwayat perubahannya TIDAK disimpan di sini. Di Pega ia ada di `InsertHistoryClaimPNC`,
-- tabel yang berbeda dan dimiliki Pega. Sampai modul `S-5` Jejak Audit ada, satu-satunya
-- jejak perubahan keputusan di aplikasi ini adalah baris log — dan `D-59` menjadikan jejak
-- audit satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas. Keputusan ini
-- menyangkut uang: pilihan `0` menolak klaim.

CREATE TABLE POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE (
    -- Kunci klaim, yakni `PZINSKEY` — BUKAN nomor klaim yang dibaca orang.
    --
    -- Nomor klaim tidak unik lintas sistem selama masa paralel; kunci teknis unik. Nama
    -- kolomnya tetap `NO_KLAIM` supaya seragam dengan `T_CLAIM_COMPLIANCE_H`, yang kolom
    -- bernama sama isinya juga kunci teknis.
    NO_KLAIM              VARCHAR2(255 BYTE) NOT NULL,

    -- Salah satu dari empat nilai pada `Property/PilihanCompliance_property.xml`:
    --
    --     0  Fraud / Tolak
    --     1  Bayar / Valid
    --     2  Bayar / PostAudit
    --     3  Lain-Lain
    --
    -- Disimpan sebagai TEKS, bukan angka, karena `pyStandardValue` memang teks dan
    -- menyimpannya sebagai angka akan menghapus pembedaan antara `'0'` dan belum diisi.
    PILIHAN               VARCHAR2(1 BYTE)   NOT NULL,

    -- `.ClaimData.NotePilihanCompliance` dan `.ClaimData.ComplianceRemark`.
    --
    -- Lebar 4000 adalah TEBAKAN: kedua properti itu tidak ada di export (`R-16`), sehingga
    -- lebar aslinya tidak terbaca. Yang dipakai adalah lebar `REMARKS` pada
    -- `T_CLAIM_COMPLIANCE_H`, satu-satunya kolom catatan di jalur ini yang dapat dibaca.
    -- Bila DDL aslinya kelak tiba dan lebarnya berbeda, kolom ini yang menyesuaikan.
    NOTE                  VARCHAR2(4000 BYTE),
    CATATAN               VARCHAR2(4000 BYTE),

    -- Siapa yang memutuskan.
    --
    -- Pega TIDAK menyimpan ini pada klaimnya — ia hanya menulis
    -- `OperatorID.pyUserIdentifier` ke baris riwayat. Kolom ini karena itu BUKAN tiruan;
    -- ia TAMBAHAN, dan alasannya ada di catatan "satu baris per klaim" di atas.
    DIPUTUSKAN_OLEH       VARCHAR2(128 BYTE),
    DIPUTUSKAN_PADA       TIMESTAMP(6)       NOT NULL,

    -- Terisi HANYA pada pilihan `1` — padanan `.ClaimData.CPLValidDate`.
    TGL_VALID             TIMESTAMP(6),

    -- Terisi HANYA pada pilihan `2` — padanan `.TanggalKirimPostAudit` dan
    -- `.ClaimData.PostAudtiTfAnalyst`, yang `SetComplianceResult` langkah 5 isi keduanya
    -- dengan waktu yang sama.
    TGL_KIRIM_POST_AUDIT  TIMESTAMP(6),

    -- ------------------------------------------------------------------------------
    -- Grid komentar — `.ClaimData.ComplianceList`, satu kolom JSON
    -- ------------------------------------------------------------------------------
    --
    -- Bentuknya senarai objek, dengan nomor baris, tanggal, dan isinya:
    --
    --     [{"urutan":1,"tanggal":"2026-10-07T09:00:00Z","komentar":"…"}, …]
    --
    -- # Kenapa SATU KOLOM, bukan tabel kedua
    --
    -- Rancangan pertama memakai tabel tersendiri `CPNC_KOMENTAR_COMPLIANCE`, karena grid
    -- ini memang berulang. Work Owner menanyakan apakah bisa satu tabel saja (2026-10-07),
    -- dan jawabannya bukan sekadar bisa — ia **lebih benar**:
    --
    --   1. **Atomisitas.** Dengan dua tabel, penyimpanan menempuh dua perintah terpisah
    --      yang BUKAN satu transaksi, karena seam Repo modul ini belum punya kepemilikan
    --      transaksi (`08-TECHNICAL-STRATEGY.md` §4.5). Bila yang kedua gagal, keputusan
    --      tersimpan tanpa komentarnya. Satu kolom menghapus kelas kegagalan itu — form
    --      menyimpan keduanya dalam satu tombol, dan kini juga dalam satu pernyataan.
    --   2. Tidak ada laporan maupun layar yang mencari komentar LINTAS klaim. Ia selalu
    --      dibaca bersama keputusannya, tidak pernah sendirian.
    --
    -- # Yang hilang, dan dinyatakan di muka
    --
    -- Komentar tidak dapat dicari maupun di-index satu per satu. Hari ini tidak ada yang
    -- melakukannya; bila kelak ada laporan atas komentar, ia menuntut migrasi — bukan
    -- sekadar kueri baru.
    --
    -- Nomor baris juga tidak lagi dijaga constraint basis data. Itu ringan: Go menomori
    -- ulang 1..n setiap kali menyimpan, setelah membuang baris kosong.
    --
    -- # Kenapa Go yang mengurainya, bukan JSON_TABLE
    --
    -- Supaya kuerinya tetap SQL biasa dan portabel apa adanya ke PostgreSQL (`D-20`).
    -- Basis data hanya menyimpan dan memvalidasi bentuknya; penguraian ada di Go, tempat
    -- aturan bisnis memang tinggal.
    --
    -- Lebar isi tiap komentar dibatasi **512** di aplikasi — `pyMaxLength` pada properti
    -- `Compliance`. CLOB dipilih karena 200 baris × 512 melampaui batas VARCHAR2.
    KOMENTAR_JSON         CLOB,

    -- Bentuknya dijaga basis data, bukan hanya kode.
    --
    -- Tanpa ini, satu cacat pembuat JSON di Go akan menyimpan teks rusak yang baru
    -- ketahuan saat dibaca — mungkin berbulan-bulan kemudian, pada klaim yang sudah
    -- diputuskan.
    CONSTRAINT CK_CPNC_KEPUTUSAN_KOMENTAR CHECK (KOMENTAR_JSON IS JSON),

    -- Primary key pada NO_KLAIM, dan ini SENGAJA berbeda dari `T_CLAIM_COMPLIANCE_H`.
    --
    -- Tabel itu tidak punya primary key sama sekali, dan ketiadaannya tercatat sebagai
    -- kelemahan: nomor yang bertabrakan tidak akan ditolak basis data. Tabel ini tidak
    -- mengulangi kelemahan itu — satu klaim satu keputusan, dan basis data yang
    -- menegakkannya, bukan hanya kode.
    CONSTRAINT PK_CPNC_KEPUTUSAN_COMPLIANCE PRIMARY KEY (NO_KLAIM)
);

-- Hak akses untuk akun aplikasi.
--
-- Empat hak, dan `DELETE` sengaja TIDAK diberikan: `D-66` menetapkan tidak ada penghapusan
-- fisik pada data bernilai bisnis. Keputusan Compliance yang dibatalkan diubah pilihannya,
-- tidak dibuang.
--
-- Ganti `APP_CLAIM_PNC` dengan nama akun aplikasi yang sebenarnya di tiap entitas.
-- ============================================================================
-- GRANT DI BAWAH SENGAJA DIKOMENTARI — JANGAN DIAKTIFKAN TANPA MEMBACA INI
-- ============================================================================
--
-- Menurut `claim-pnc/backend/.env` baris 60, aplikasi menyambung sebagai
-- POOLDATA_ASM_PENGGUNA=POOLDATA — yakni PEMILIK SKEMA-nya sendiri.
--
-- Dua akibatnya:
--
--   1. GRANT TIDAK DIBUTUHKAN. Pemilik skema selalu punya hak penuh atas objek yang ia
--      miliki; tidak ada hak yang perlu diberikan kepada dirinya sendiri.
--   2. Menjalankannya JUSTRU GAGAL. `APP_CLAIM_PNC` tidak ada di basis data ini, sehingga
--      Oracle melempar `ORA-01917: user or role 'APP_CLAIM_PNC' does not exist` — dan satu
--      pernyataan gagal dapat menghentikan skrip DBA di tengah jalan, meninggalkan
--      sebagian objek terbuat dan sebagian tidak.
--
-- AKTIFKAN baris di bawah HANYA bila aplikasi kelak memakai akun terpisah dari POOLDATA,
-- lalu ganti APP_CLAIM_PNC dengan nama akun itu.
--
-- # Satu pengaman yang TIDAK berlaku selama aplikasi menyambung sebagai POOLDATA
--
-- Ketiadaan `DELETE` pada GRANT dimaksudkan menegakkan `D-66` — soft delete — lewat hak
-- akses basis data, bukan lewat disiplin kode. Pengaman itu **tidak bekerja** pada pemilik
-- skema: POOLDATA dapat menghapus barisnya sendiri apa pun yang tertulis di sini.
--
-- Jadi selama konfigurasi ini berlaku, `D-66` ditegakkan HANYA oleh kode dan oleh review.
-- Itu kelemahan nyata, dan ia hilang begitu akun aplikasi terpisah dibuat.
-- GRANT SELECT, INSERT, UPDATE ON POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE TO APP_CLAIM_PNC;
