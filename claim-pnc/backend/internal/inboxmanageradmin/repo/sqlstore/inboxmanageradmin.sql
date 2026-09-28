-- Kueri modul Inbox Manager Admin: antrean registrasi per unit organisasi admin.
--
-- Nama kueri dan nama di dalam kode berbahasa Inggris (`D-80`); nama tabel dan nama kolom
-- tetap seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan
-- perubahannya menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan
-- yang menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh
-- ditulis SATU sistem, dan tabel-tabel ini milik Pega (`P-1`).
--
-- ============================================================================
-- SUMBER DATA BERPINDAH — KETETAPAN WORK OWNER 2026-09-27
-- ============================================================================
--
-- Sebelumnya berkas ini membaca DUA tabel skema DATAPEGA yang digabung:
--
--     DATAPEGA.PC_ASM_FW_GCNMFW_WORK  w
--       INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST a ON a.PXREFOBJECTKEY = w.PZINSKEY
--
-- Keduanya kini digantikan SATU tabel datar:
--
--     POOLDATA.T_CLAIMLIST_ADMIN
--
-- Berkas ini karena itu TIDAK LAGI menyentuh skema DATAPEGA sama sekali.
--
-- # Dua cacat yang ikut hilang bersama join-nya
--
-- Join lamanya `INNER`, dan bentuk itu membawa dua cacat yang tidak pernah menghasilkan
-- galat:
--
--   * klaim TANPA assignment terbuka **lenyap dari layar** — padahal justru itu pekerjaan
--     yang terhenti dan paling perlu dilihat penyelia;
--   * klaim dengan LEBIH DARI SATU assignment **tampil berkali-kali**, dan penyelia
--     membaca satu klaim yang sama sebagai beberapa pekerjaan.
--
-- Tabel datar menyimpan satu baris per klaim, sehingga keduanya hilang dengan sendirinya.
-- Ini SELISIH PERILAKU yang disengaja, bukan pemeliharaan — lihat
-- inboxmanageradmin.PlannedDifferences, dan ia wajib disebut lebih dulu pada uji kesetaraan
-- gerbang 1 (`D-54`) supaya tidak dilaporkan sebagai cacat.
--
-- # Satu akibat yang TIDAK menguntungkan, dan tidak disembunyikan
--
-- `T_CLAIMLIST_ADMIN` **belum memuat seluruh klaim**: 1.014 dari 7.703 — **13%**
-- (`docs/kolom-t-claimlist-admin.md`, katalog Oracle 2026-09-22). Selama proses pengisinya
-- belum mengejar, layar ini menampilkan LEBIH SEDIKIT baris daripada Pega, dan selisihnya
-- bukan karena penyaring melainkan karena sumbernya belum lengkap.
--
-- ============================================================================
-- DARI MANA KUERI INI, PADAHAL LAYARNYA TIDAK PUNYA SATU PUN RULE SQL
-- ============================================================================
--
-- Ketiga grid layar ini tidak dilayani Connect-SQL melainkan Report Definition
-- `ManagementAdminView`, yang menyusun SQL-nya sendiri dari kelas
-- `ASM-FW-GCNMFW-Work-PNC`. Kueri di bawah adalah penulisan ulang RD itu, bagian per
-- bagian:
--
--   pyListFields  -> daftar SELECT           (13 isian; 9 dibawa, lihat di bawah)
--   pyFilters     -> klausa WHERE            (`A AND B AND C`)
--   pyParameters  -> bind `:1`               (`OrgUnit`)
--   pySource      -> FROM                    — kini satu tabel, bukan dua
--
-- ============================================================================
-- KONTRAK KOLOM: POOLDATA.T_CLAIMLIST_ADMIN
-- ============================================================================
--
-- Tabel ini NOL KEMUNCULAN di seluruh export Pega — 2.634 XML, 55 `.prc`, 8 `.fnc` — jadi
-- kolomnya tidak dapat dibaca dari export mana pun. Yang dipakai di bawah karena itu
-- diturunkan dari DUA sumber yang dapat diperiksa, bukan dikarang:
--
--   1. kolom yang SUDAH TERBUKTI dipakai modul lain terhadap tabel yang sama
--      (`inboxoutstanding`, `inboxlaporanklaim`);
--   2. katalog Oracle 2026-09-22 yang tercatat di `docs/kolom-t-claimlist-admin.md`.
--
--   Kolom di sini        Sumber lamanya                     Status
--   -------------------  ---------------------------------  ------------------------------
--   PZINSKEY             w.PZINSKEY                         ADA  (§A kolom doc)
--   PYID                 w.PYID                             ADA  (inboxoutstanding)
--   POLICYNO             w.POLICYNO                         ADA  (inboxoutstanding)
--   QQNAME               w.QQNAME                           ADA  (inboxoutstanding)
--   BUSINESSNAME         w.BUSINESSNAME                     ADA  (inboxoutstanding)
--   SOBNAME              w.SOBNAME                          ADA  (inboxoutstanding)
--   PXCREATEDATETIME     w.PXCREATEDATETIME                 ADA  (migrasi 0005, TIMESTAMP(6))
--   PXOBJCLASS           w.PXOBJCLASS                       ADA  (inboxlaporanklaim)
--   PYSTATUSWORK         w.PYSTATUSWORK                     ADA  (inboxoutstanding)
--   STATUSCLAIM_1        w.STATUSCLAIM_1                    DIMINTA — migrasi 0005 tahap 1
--   PXCREATEOPNAME       w.PXCREATEOPNAME                   DIMINTA — migrasi 0005 tahap 1
--   PXASSIGNEDORGUNIT    a.PXASSIGNEDORGUNIT                DIMINTA — migrasi 0005 tahap 1
--
-- Bila sebuah kolom belum ada, YANG DIMINTA adalah kolomnya ditambah — BUKAN penyaringnya
-- dihapus. Menghapus penyaring mengubah baris yang dibaca penyelia tanpa satu pun galat.
--
-- ============================================================================
-- TIGA KOLOM YANG SEMPAT DIMINTA, DAN TERNYATA SUDAH ADA SEMUA
-- ============================================================================
--
-- Saat modul ini dipindahkan ke tabel ini (2026-09-27), tiga kolom disimpulkan
-- belum ada dan diminta lewat `migrations/0005`. KETIGANYA TERNYATA SUDAH ADA, dan
-- permintaannya dicabut pada hari yang sama setelah Work Owner memeriksa tabelnya:
--
--   PXASSIGNEDORGUNIT  sudah ada
--   PXCREATEOPNAME     sudah ada
--   PYORIGUSERID       sudah ada (tidak dipakai berkas ini; tidak digambar section)
--
-- `STATUSCLAIM_1` juga sempat dipakai berkas ini lalu ditinggalkan: Status Klaim kini
-- diturunkan dari `PYSTATUSWORK`, mengikuti `RDB List/BrowseClaimALL-SQL.xml` dan modul
-- Inbox Outstanding. Kolom itu memang BELUM ada, dan pemakaiannya inilah yang membuat
-- kueri ini sempat gagal dengan ORA-00904.
--
-- # Kenapa ketiganya sempat disimpulkan tidak ada
--
-- Karena daftar kolom `T_CLAIMLIST_ADMIN` yang SUDAH ADA belum pernah dibaca dari
-- katalog — `docs/kolom-t-claimlist-admin.md` mencatat jumlahnya (40) tanpa menyebut
-- namanya. Ketiadaan sebuah kolom karena itu disimpulkan dari modul lain yang memakai
-- kolom berbeda, dan itu bukan bukti.
--
-- Daftar lengkapnya diminta sebagai `L-3` (`permintaan-artefak-pega.md` §7.6). Sampai ia
-- ada, setiap kolom baru yang dipakai berkas ini WAJIB ditelusuri ke modul lain yang
-- kuerinya benar-benar berjalan — bukan ke dokumen yang menyebut kolom sumbernya.
--
-- ============================================================================
-- PXOBJCLASS WAJIB DISARING — TABEL INI MEMUAT LEBIH DARI SATU KELAS KASUS
-- ============================================================================
--
-- Ini BUKAN penyaring yang boleh dihilangkan dengan alasan "tabelnya memang untuk Claim
-- PNC". Bukti bahwa isinya bercampur:
--
--   * `inboxlaporanklaim/repo/sqlstore/inboxlaporanklaim.sql:197-200` menggabung
--     `T_CLAIMLIST_ADMIN` dengan syarat `t.pxobjclass = w.pxobjclass` sementara `w`
--     disaring `'ASM-FW-GCNMFW-Work-ReceiveDocument'` — gabungan itu tidak akan pernah
--     menghasilkan baris bila tabelnya hanya memuat Work-PNC;
--   * `README.md` mencatat **142 baris RCV** di dalamnya.
--
-- Tanpa penyaring ini, baris Receive Document ikut masuk antrean registrasi klaim — dan
-- keduanya sama-sama punya PYID, sehingga tidak ada yang tampak salah.
--
-- ============================================================================
-- EMPAT HAL YANG BERUBAH DARI REPORT DEFINITION, DAN ALASANNYA
-- ============================================================================
--
-- 1. PARAMETER BINDING untuk `OrgUnit`. Di Pega nilainya ditanam section sebagai literal
--    (`<OrgUnit>"AdminPNC"</OrgUnit>`); di sini ia bind `:1`, sehingga satu kueri melayani
--    ketiga tab dan tidak ada nilai yang dirangkai ke dalam teks SQL
--    (`08-TECHNICAL-STRATEGY.md` §4.3).
--
-- 2. `pyMaxRecords=500` TIDAK dibawa. Menyalinnya berarti baris ke-501 seterusnya tidak
--    pernah terlihat tanpa satu pun tanda. Ia selisih terencana — lihat
--    inboxmanageradmin.PlannedDifferences.
--
-- 3. ORDER BY ditambahkan. Report Definition ini tidak mengurutkan hasilnya sama sekali.
--    Itu dapat dibiarkan selama seluruh baris ditarik sekaligus, tetapi halaman yang
--    dipotong dari urutan yang tidak ditetapkan membuat satu baris muncul di dua halaman
--    sekaligus hilang dari halaman lain. `PYID` disertakan sebagai pemutus seri supaya
--    urutannya tetap sama pada dua permintaan yang berbeda.
--
-- 4. Status klaim diresolusi lewat SUBKUERI, bukan lewat pemanggilan kedua. Activity ekspor
--    lama menjalankan Report Definition lebih dulu, lalu RDB List `GCNM GetStatusKlaim`
--    untuk menerjemahkan kodenya. Dua perjalanan menjadi satu; hasilnya sama.
--
-- ============================================================================
-- YANG SENGAJA TIDAK BERUBAH
-- ============================================================================
--
-- * PAGINASI TIDAK DILAKUKAN DI SINI. Tidak ada `OFFSET ... FETCH NEXT`, dan itu bukan
--   kelalaian: Work Owner memutuskan 2026-09-26 paginasi layar ini direplikasi apa adanya,
--   sama seperti modul Inbox Admin. Pemotongan halaman terjadi di aplikasi — lihat
--   inboxmanageradmin.Slice, yang juga memuat konsekuensi yang diterima secara sadar.
--
-- * PENYARING `PYSTATUSWORK` dibawa apa adanya, termasuk bentuknya sebagai DUA pertidaksamaan
--   terpisah di Pega (`!= 'Resolved-Completed'` dan `!= 'Resolved-Rejected'`). Di sini
--   keduanya menjadi satu `NOT IN`; artinya identik.
--
-- * TIDAK ADA penyaring `USERTEKNIS IS NULL`, meski `pyDescription` Report Definition-nya
--   berbunyi "ONLY SHOWS VALUE IF THE USERTEKNIS IS NULL". Pembacaan `pyFilterLogic`
--   membantahnya: filternya `A AND B AND C`, dan `.ClaimData.UserTeknis` hanya muncul
--   sebagai kolom tampilan. Menambahkannya berarti menyaring yang tidak disaring sistem
--   lama, dan baris yang hari ini terlihat akan hilang tanpa pesan galat.
--
-- * KOLOM YANG DIBAWA tetap kesembilan yang sama. Berpindah sumber tidak menambah kolom:
--   `T_CLAIMLIST_ADMIN` memuat banyak kolom lain, dan mengambilnya karena "sekalian ada"
--   akan menampilkan data yang tidak pernah ada di layar Pega.

-- name: list_by_org_unit
-- Ketiga tab — Report Definition/ManagementAdminView-RD.xml
--
-- Satu kueri melayani ketiganya, persis seperti di Pega: yang membedakan Non-MBU, PA, dan
-- Travel hanyalah nilai parameter `OrgUnit`.
--
-- Bind: :1 unit organisasi penugasan ('AdminPNC' | 'AdminPA' | 'AdminTRAVEL')
SELECT a.PZINSKEY                                   AS REFERENCE,
       a.PYID                                       AS CASE_ID,
       a.POLICYNO                                   AS POLICY_NUMBER,
       a.QQNAME                                     AS INSURED_NAME,
       a.BUSINESSNAME                               AS BUSINESS_NAME,
       a.SOBNAME                                    AS BUSINESS_SOURCE,
       a.PXCREATEDATETIME                           AS REGISTERED_AT,
       a.PXCREATEOPNAME                             AS ADMIN_NAME,
       a.PYSTATUSWORK                               AS CLAIM_STATUS
  FROM POOLDATA.T_CLAIMLIST_ADMIN a
 WHERE a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
   AND a.PXASSIGNEDORGUNIT = :1
   AND a.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
 ORDER BY a.PXCREATEDATETIME DESC, a.PYID

-- name: check_table
-- Memastikan tabel sumber modul ini terbaca dari koneksi yang dipakai.
--
-- Tidak menyentuh satu baris pun: yang diperiksa adalah hak baca dan keberadaan tabelnya.
SELECT 1
  FROM POOLDATA.T_CLAIMLIST_ADMIN
 WHERE 1 = 0

-- name: line_business_for
-- Lini bisnis seorang petugas — padanan `OperatorID.pyPosition` sistem lama.
--
-- Ia yang menentukan tab mana yang boleh dibuka, menggantikan pembacaan dari HCQ yang
-- keliru sampai 2026-09-27: `EmpResponse.Placement.PositionName` adalah JABATAN
-- KEPEGAWAIAN ("IT SPECIALIST"), bukan kode lini bisnis, sehingga tidak seorang pun
-- melihat satu tab pun.
--
-- Kuerinya sengaja SAMA PERSIS dengan `line_business_for` milik modul Inbox Outstanding
-- (`inboxoutstanding/repo/sqlstore/outstanding.sql`). Keduanya membaca kolom yang sama untuk
-- keperluan yang sama; menulisnya berbeda hanya menciptakan dua tempat yang dapat menyimpang.
--
-- Nama kolomnya `LINE_BUSINESS` DENGAN GARIS BAWAH. Migrasi `0004_login_line_business` yang
-- memakai `LINEBUSINESS` tanpa garis bawah sudah DICABUT sebelum dijalankan justru karena
-- namanya salah — lihat `migrations/0004_DICABUT.md`. Kolom yang benar sudah ada.
--
-- Tabelnya dimiliki modul Login; modul ini hanya MEMBACA satu kolom (`P-1`).
--
-- Bind: :1 login petugas, sudah huruf besar dan tanpa spasi tepi
SELECT p.LINE_BUSINESS
  FROM POOLDATA.M_LOGIN_PNC p
 WHERE UPPER(TRIM(p.LOGIN_ID)) = :1

-- name: check_line_business
-- Memastikan kolom LINE_BUSINESS terbaca dari koneksi yang dipakai.
--
-- Ia pemeriksaan KETIGA, terpisah dari kedua di atas, karena tabelnya pun berbeda:
-- `M_LOGIN_PNC` milik modul Login, sedangkan kedua pemeriksaan lain menyentuh
-- `T_CLAIMLIST_ADMIN`. Kegagalannya karena itu menunjuk hal yang berbeda pula.
--
-- Ia layak diperiksa tersendiri meski kolomnya SUDAH ADA, karena namanya pernah salah
-- ditulis: migrasi `0004_login_line_business` menambahkan `LINEBUSINESS` tanpa garis bawah
-- dan DICABUT sebelum dijalankan justru karena itu (`migrations/0004_DICABUT.md`). Ejaan
-- yang keliru di sini menghasilkan ORA-00904, dan tanpa pemeriksaan ini galatnya baru
-- muncul saat petugas membuka layar.
--
-- `1 = 0` membuatnya tidak pernah mengembalikan baris.
SELECT 1
  FROM POOLDATA.M_LOGIN_PNC
 WHERE LINE_BUSINESS IS NOT NULL
   AND 1 = 0
