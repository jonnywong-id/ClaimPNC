-- Kueri modul Inbox OS Claim per Cabang (`MENU_ID 69`).
--
-- Nama kueri dan alias hasil berbahasa Inggris (`D-80`); nama tabel dan nama kolom tetap
-- seperti aslinya karena keduanya milik basis data — pengecualian `D-80`, dan perubahannya
-- menempuh `D-63`.
--
-- SELURUH tabel yang dibaca berkas ini milik sistem lama. Tidak ada satu pun pernyataan yang
-- menulis, dan memang tidak boleh ada: selama masa paralel setiap tabel hanya boleh ditulis
-- SATU sistem, dan tabel-tabel ini milik Pega (`P-1`).
--
-- ============================================================================
-- ASALNYA
-- ============================================================================
--
--   list          RDB List/GetDataOutstandingperCabang-SQL.xml
--   list_export   RDB List/GetDataOutstandingperCabangExport-SQL.xml
--   branch_of     RDB List/GetNamaCabangTelepon-SQL.xml   (kolom kunci diganti, lihat di sana)
--   (penanda)     RDB List/GetProgress1Sama-SQL.xml       -> gabungan PROGRESS_STALLED
--
-- Keempatnya dipanggil `Activity/OutstandingperCabang_PreAct-Act.xml` dan
-- `Activity/ExportDataOSCabang-Act.xml`.
--
-- ============================================================================
-- TABEL PENYARING OUTSTANDING DIPINDAH KE POOLDATA, DAN ITU MENGUBAH ISI LAYAR
-- ============================================================================
--
-- Kueri lama menyaring klaim yang masih berjalan lewat tabel engine Pega:
--
--   kueri lama   JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK w ON c.claimid = w.pzinskey
--   di sini      JOIN POOLDATA.T_CLAIMLIST_ADMIN     w ON c.claimid = w.pzinskey
--
-- Work Owner memutuskan pemindahannya (2026-09-28). Dasarnya `D-21`: tabel engine Pega
-- digantikan tabel milik aplikasi, dan `PC_ASM_FW_GCNMFW_WORK` ada di daftar itu.
--
-- ### Kunci gabungannya `PZINSKEY`, bukan `PNCCASEID` — ini sempat salah
--
-- `T_CLAIMLIST_ADMIN.PNCCASEID` berisi nomor REGISTER DOKUMEN (RCV), bukan nomor klaim.
-- Digabungkan lewat kolom itu hasilnya NOL baris, dan nol baris terbaca sebagai "cabang ini
-- memang tidak punya klaim" — bukan sebagai gabungan yang salah. Kunci yang benar
-- `PZINSKEY`, yang sama isinya dengan `T_CLAIM_PNC.CLAIMID`, dan ia UNIK di tabel itu
-- (1.023 baris, 1.023 nilai berbeda) sehingga tidak menggandakan baris.
--
-- ### Yang berubah bukan bentuk kuerinya, melainkan HIMPUNAN KLAIM DI LAYAR
--
-- Diukur langsung, penyaring dan gabungan lain sama persis:
--
--   lewat DATAPEGA.PC_ASM_FW_GCNMFW_WORK   953 baris
--   lewat POOLDATA.T_CLAIMLIST_ADMIN       443 baris
--   tidak ada di T_CLAIMLIST_ADMIN         517 klaim
--
-- Sebabnya bukan data yang basi — baris terbaru tabel itu bertanggal hari pengukuran.
-- Keduanya **himpunan yang berbeda**, dan masing-masing punya ratusan baris yang tidak ada di
-- yang lain.
--
-- #### Arah pertama: 517 klaim outstanding yang tidak ada di T_CLAIMLIST_ADMIN
--
-- Tahap penugasan terbukanya, dihitung per klaim:
--
--   Send To Analis 167 · Estimation 136 · Choose Surveyor 81 · tanpa penugasan 45
--   View Polis 37 · Send To PIC Teknik 19 · Input Estimasi 12 · Input Register 10
--   FixCorrespondence 6 · InputPanel 3 · Review Klaim OCR 1
--
-- Yang menguasai angka adalah tahap **Analis dan Estimator** — 303 klaim, 59% dari 517.
--
-- Dan `T_CLAIMLIST_ADMIN` **tidak sekadar terbatas empat label**, meski isinya memang hanya
-- empat (Input Register 351 · Choose Surveyor 340 · Input Estimasi 182 ·
-- InputReceiveDocument 142). Dipilah menurut apakah tahapnya termasuk keempatnya:
--
--   seluruh tugasnya DI LUAR empat label itu           411 klaim
--   seluruh tugasnya JUSTRU empat label itu            103 klaim   <-- tetap tidak ada
--   campuran                                             3 klaim
--
-- Seratus tiga klaim duduk di label yang tabel itu memang bawa, namun barisnya tetap tidak
-- ada. Tabelnya karena itu **tidak lengkap**, bukan hanya bersempit lingkup.
--
-- #### Arah kedua: 453 baris tabel itu yang tidak dapat ditampilkan — dan itu BUKAN kemunduran
--
-- Dari 874 baris Work-PNC outstanding di `T_CLAIMLIST_ADMIN`:
--
--   PZINSKEY tidak ada di T_CLAIM_PNC   371
--   registerdate NULL                    59
--   branchcode NULL                      23
--   cocok penuh                         421
--
-- Ketiga sebab itu **berlaku sama persis pada sumber lama**, karena tabel penggiring kedua
-- kueri adalah `T_CLAIM_PNC` — bukan tabel penugasan. Klaim tanpa baris di sana tidak pernah
-- muncul, sumber tabel apa pun yang dipakai. Diukur:
--
--   DATAPEGA outstanding Work-PNC            1.985 baris, 1.062 punya baris T_CLAIM_PNC
--   T_CLAIMLIST_ADMIN outstanding Work-PNC     874 baris,   503 punya baris T_CLAIM_PNC
--
-- Sumber lama justru membuang LEBIH BANYAK (923 berbanding 371) dengan sebab yang sama.
--
-- Ketiga ratus tujuh puluh satu yatim itu seluruhnya **klaim PNC** (`PYID` berbentuk
-- `PNC-nnnn`, bukan `RCV-nnnn`) dan seluruhnya ada di DATAPEGA. Tahapnya Input Register 294 ·
-- Input Estimasi 57 · Choose Surveyor 20 — pola klaim yang case-nya sudah dibuat tetapi
-- registrasinya belum pernah disubmit, sehingga `T_CLAIM_PNC` belum ditulis.
--
-- Karena itu ia **tidak dinyatakan sebagai selisih** ke pengguna: tidak ada yang berubah bagi
-- mereka.
--
-- Itu selisih perilaku, bukan pemeliharaan. Ia dinyatakan lewat
-- inboxosclaimpercabang.PlannedDifferences supaya pengguna yang membandingkan kedua layar
-- memperoleh jawaban (`D-54`), dan menunggu keputusan Work Owner — bukan disembunyikan.
--
-- Tiga kolom yang dibaca kueri lama TIDAK ADA di tabel baru dan karena itu tetap dibaca dari
-- `T_CLAIM_PNC` beserta gabungannya, persis seperti sebelumnya: penyebab kerugian, nilai
-- cadangan, dan tanggal pembaruan progres terakhir.
--
-- ============================================================================
-- SATU SELISIH YANG MENYANGKUT UANG — DIPERBAIKI ATAS KEPUTUSAN WORK OWNER
-- ============================================================================
--
-- Kolom yang digambar grid berjudul "Reserve Claim ASM Share", tetapi kedua kueri LAMA
-- menghitungnya BERBEDA untuk klaim yang sama:
--
--   GetDataOutstandingperCabang         SUM(estimationvalue)
--                                         -> tanpa kurs, tanpa porsi ASM
--   GetDataOutstandingperCabangExport   SUM(estimationvalue * kursvalue) * SHAREASM/100
--                                         -> dengan keduanya
--
-- Jadi angka di layar lama BUKAN porsi ASM sama sekali, padahal judulnya mengatakan begitu,
-- dan berkas ekspornya menampilkan angka lain untuk klaim yang sama.
--
-- Ukurannya bukan teoretis. Diukur pada 1.141 klaim outstanding (2026-10-10):
--   74 klaim  nilainya bukan rupiah           (ada baris estimasi berkurs <> 1)
--   305 klaim nilainya memuat porsi koasuransi (SHAREASM <> 100)
--
-- Work Owner memutuskan 2026-10-10: **layar disamakan dengan ekspor**. Kolom grid dan panel
-- ringkasan kini memakai ekspresi yang sama persis dengan `list_export`:
--
--   ROUND(SUM(estimationvalue * COALESCE(kursvalue,1)) * (c.shareasm/100) * 100)
--
-- Ini SELISIH TERHADAP PEGA yang disengaja — sejenis dengan 13 butir `D-49` — dan karena itu
-- dinyatakan di inboxosclaimpercabang.PlannedDifferences, bukan diperbaiki diam-diam.
--
-- DUA tempat lain yang sengaja TIDAK ikut diubah, karena keduanya elemen yang berbeda dan
-- tidak diminta:
--   detail_header     "Total Reserve" pada popup Detail
--   summary_treaty_or kartu porsi treaty OR (sudah tidak digambar di layar)
-- Keduanya tetap `SUM(estimationvalue)` apa adanya seperti Pega.
--
-- ============================================================================
-- EMPAT BENTUK ORACLE YANG DIGANTI, DAN KENAPA
-- ============================================================================
--
--   NVL(x, 0)                              -> COALESCE(x, 0)
--   TRUNC(SYSDATE) - TRUNC(registerdate)   -> dihitung di Go (lihat catatan Aging)
--   MAX(x) KEEP (DENSE_RANK LAST ORDER BY) -> subkueri ROW_NUMBER
--   PIVOT (…) FOR rn IN (1,2,3)            -> GROUP BY + HAVING (lihat PROGRESS_STALLED)
--   LISTAGG(name, ', ')                    -> kueri dominant_factors, dirangkai di Go
--
-- Kelimanya khas Oracle dan tidak punya padanan di PostgreSQL 17+ (`D-20`, `D-24`).
--
-- Yang kelima perlu keterangan tersendiri. `09-DATABASE-STRATEGY.md` §4 memetakan `LISTAGG`
-- ke `STRING_AGG`, tetapi **Oracle 19c tidak mengenal `STRING_AGG`** — sehingga tidak ada satu
-- bentuk pun yang berjalan di kedua basis data. Perangkaiannya karena itu pindah ke Go, dan
-- itu tidak menambah perjalanan per baris: `dominant_factors` mengambil seluruh faktor milik
-- satu cabang dalam satu kueri, lalu dipasangkan ke barisnya di memori.
--
-- ### Aging TIDAK dihitung di sini, dan itu bukan kelalaian
--
-- Padanan `TRUNC` yang dianjurkan `09-DATABASE-STRATEGY.md` §4 adalah `CAST(x AS DATE)`, dan
-- bentuk itu **tidak memangkas jam di Oracle**. Diukur langsung:
--
--   CAST(CURRENT_TIMESTAMP AS DATE) - CAST(registerdate AS DATE)  ->  0.8758…
--   TRUNC(SYSDATE)                  - TRUNC(registerdate)         ->  1
--
-- Baris itu klaim yang terdaftar KEMARIN; dipindai ke bilangan bulat ia menjadi nol hari.
-- Karena umur itulah yang menentukan baris digambar merah, ia dihitung di Go terhadap
-- tanggal WIB (`inboxosclaimpercabang.AgingDaysSince`, `F-5`).
--
-- Urutan barisnya tetap di sini: umur menurun sama persis dengan tanggal registrasi menaik,
-- sehingga `ORDER BY c.registerdate ASC` menghasilkan urutan yang identik dengan
-- `ORDER BY "AgingKlaim" DESC` milik kueri lama — tanpa perlu menghitung umurnya.
--
-- Kedua kueri Pega memang berakhir demikian:
--   RDB List/GetDataOutstandingperCabang-SQL.xml       ORDER BY "AgingKlaim" DESC
--   RDB List/GetDataOutstandingperCabangExport-SQL.xml ORDER BY "AgingKlaim" DESC
-- dengan "AgingKlaim" = TRUNC(SYSDATE) - TRUNC(c.registerdate). Keduanya memakai TRUNC dan
-- SYSDATE, yang terlarang di sini (`D-20`); menulisnya sebagai tanggal registrasi menaik
-- memberi urutan yang sama tanpa keduanya.
--
-- DIUKUR, bukan hanya disimpulkan (2026-10-10): pada cabang berbaris terbanyak — 108 klaim
-- outstanding — kedua urutan dibandingkan nomor per nomor terhadap Oracle, dan **0 posisi
-- berbeda**.
--
-- `c.claimno ASC` adalah pemisah tambahan yang TIDAK dimiliki kueri lama. Tanpa ia, dua klaim
-- bertanggal registrasi sama dapat berpindah posisi antar permintaan, sehingga satu baris
-- muncul di dua halaman sekaligus sementara baris lain tidak muncul sama sekali. Kueri lama
-- tidak terkena karena ia tidak memaginasi.
--
-- ============================================================================
-- PROGRESS_STALLED — penanda baris merah, ditulis ulang set-based
-- ============================================================================
--
-- Sistem lama menghitungnya SATU KUERI PER BARIS: `OutstandingperCabang_PreAct` langkah 5
-- mengulang seluruh hasil, menyetel `TempCari.CARI1 = .ClaimNo`, menjalankan
-- `GetProgress1Sama`, lalu menyetel `.Medicare = "1"` bila ada hasilnya. Untuk 953 baris
-- outstanding hari ini itu 953 perjalanan ke basis data.
--
-- Di sini ia satu gabungan. Pertanyaannya tidak berubah: **apakah tiga catatan progres
-- terakhir klaim ini bernilai `status_progress1` sama.**
--
--   HAVING COUNT(status_progress1) = 3    tepat tiga catatan, dan tak satu pun NULL
--      AND COUNT(DISTINCT status_progress1) = 1    ketiganya bernilai sama
--
-- `COUNT(kolom)` mengabaikan NULL, dan itu yang meniru perilaku lama: pada PIVOT, `r1 = r2`
-- bernilai UNKNOWN begitu salah satunya NULL, sehingga klaim itu tidak pernah ditandai.
--
-- ### Pemisah saat dua catatan berwaktu sama — `ID_UPDATE`, bukan `rowid`
--
-- Kueri lama mengurutkan `ORDER BY tgl_input DESC, rowid DESC`. `rowid` adalah alamat
-- penyimpanan FISIK baris — bukan fakta bisnis, tidak ada padanannya di PostgreSQL, dan tidak
-- dapat direproduksi setelah data dipindahkan. Ia karena itu tidak dapat dibawa.
--
-- Pemisah itu benar-benar terpakai: **271 klaim** punya dua catatan progres atau lebih yang
-- `tgl_input`-nya sama persis (diukur 2026-10-10).
--
-- #### Koreksi atas catatan serahan sebelumnya
--
-- Serahan sebelumnya MENOLAK `ID_UPDATE` sebagai pengganti dengan alasan *"1.374 nilai
-- berbeda untuk 20.615 baris, jadi ia bukan pengenal baris"*. **Alasan itu keliru** — ia
-- menguji keunikan di SELURUH TABEL, padahal pemisah ini hanya perlu unik **di dalam satu
-- klaim**, karena jendelanya `PARTITION BY pnccaseid`.
--
-- Diuji ulang dengan pertanyaan yang benar: pasangan `(pnccaseid, ID_UPDATE)` yang kembar
-- berjumlah **0**. Ia memang pencacah per klaim, bukan pengenal global — dan itu persis yang
-- dibutuhkan.
--
-- #### Hasil pengukuran terhadap Oracle (2026-10-10)
--
--   urutan                              ditandai   hanya-Pega   hanya-kita
--   tgl_input DESC            (lama)       129          3           23
--   tgl_input DESC, ID_UPDATE DESC         107          2            0
--   tgl_input DESC, POSISIID  DESC         131          1           23
--   tgl_input DESC, ID_UPDATE ASC          131          1           23
--   versi Pega (rowid DESC)                109          —            —
--
-- `ID_UPDATE DESC` karena itu dipakai: **nol** klaim yang ditandai merah padahal Pega tidak,
-- turun dari 23. Yang tersisa dua klaim yang Pega tandai dan kita tidak — di situ urutan
-- fisik barisnya memang berbeda dari urutan `ID_UPDATE`-nya, dan itu tidak dapat ditiru
-- tanpa `rowid`.
--
-- #### Akibat kedua klaim itu di layar: NOL
--
-- Keduanya diperiksa terhadap populasi layar: **kedua-duanya tampil**, dan **kedua-duanya
-- sudah digambar merah karena umurnya melewati ambang**. Baris merah punya DUA syarat —
-- umur atau progres mandek — dan syarat pertama sudah terpenuhi pada keduanya.
--
-- Jadi tidak ada satu baris pun yang warnanya berbeda dari layar lama. Yang berbeda hanya
-- keterangan pada `title` barisnya, yang di layar lama memang tidak ada sama sekali.
--
-- #### Pada kedua klaim itu, pilihan di sini justru LEBIH TEPAT
--
-- `ROWID` adalah alamat penyimpanan fisik: ia dapat berpindah ketika Oracle merapikan tabel,
-- dan tidak punya arti bisnis apa pun. `ID_UPDATE` adalah nomor urut pencatatan per klaim.
-- Aturannya berbunyi "tiga catatan progres TERAKHIR" — itu urutan pencatatan, bukan urutan
-- penyimpanan. Jadi pada dua klaim yang berbeda itu, yang memilih tiga catatan yang benar
-- adalah kueri ini, bukan kueri lama.
--
-- Karena akibatnya di layar nol dan mekanismenya lebih tepat, butirnya DICABUT dari
-- PlannedDifferences (Work Owner, 2026-10-10). Angka pengukurannya tinggal di sini.
--
-- ============================================================================
-- SETIAP NILAI UANG DIKEMBALIKAN DALAM SATUAN TERKECIL
-- ============================================================================
--
-- Yakni rupiah dikali 100, sudah dibulatkan ke bilangan bulat: `ROUND(<ekspresi> * 100)`.
--
-- `I-12` dan `09-DATABASE-STRATEGY.md` §5 menetapkan nilai uang tidak pernah `float`, dan
-- `internal/platform/money` mewakilinya sebagai int64 satuan terkecil. Mengembalikannya
-- sebagai rupiah berpecahan akan memaksa pemindai melewati `float64` — tepat bentuk yang
-- dilarang, dan `money.FromSQLValue` memang MENOLAKNYA.
--
-- Pembulatannya nyata terpakai, bukan jaga-jaga: 24 kolom treaty dihitung
-- `nilai × (shareasm/100) × bagian_treaty`, dan hasil perkalian tiga faktor itu hampir
-- selalu berpecahan. Membulatkannya ke SEN adalah pembulatan TAMPILAN — berkas ekspor
-- memang tampilan — dan nilai tersimpannya tidak disentuh sama sekali.
--
-- ============================================================================
-- EMPAT ALIAS WAJIB SAMA URUTANNYA DENGAN PEMINDAI
-- ============================================================================
--
-- Urutan DAN namanya, pada `list` maupun `list_export`. Dua hal bergantung padanya: pemindai
-- di inboxosclaimpercabang.go, dan susunan kolom berkas ekspor. Ketiganya dijaga query_test.go.

-- name: branch_of
-- Cabang pemanggil, diterjemahkan dari kode cabang RINCI yang dikirim HCQ.
-- — menggantikan RDB List/GetNamaCabangTelepon-SQL.xml
--
-- ### Kolom kuncinya BERBEDA dari kueri lama, dan itu perbaikan
--
--   kueri lama   where id    = {OperatorID.pyTelephone}
--   di sini      where oldid = :1
--
-- `OperatorID.pyTelephone` bukan sumber yang dapat dibawa: dari 4.173 operator hanya 731
-- yang mengisinya, dan isinya bercampur antara kode cabang, nomor ponsel sungguhan, dan
-- nilai seperti `00`. Penggantinya `Placement.DetailBranchCode` dari HCQ, yang Work Owner
-- tetapkan (2026-09-28) sama dengan `POOLDATA.BRANCH.OLDID`.
--
-- ### Terjemahannya tidak ambigu, dan tidak menyentuh DB Link
--
-- Diukur langsung: `BRANCH` berisi 803 baris, `OLDID` terisi pada 792 dan seluruhnya UNIK.
-- Jalur terjemahan lain — `GENERAL.LST_DET_CABANG@asmd` lewat `LDC_ID` ke `LDC_ID_PEGA` —
-- dibandingkan baris per baris: **791 sepakat, 0 berselisih**. Yang lokal ini dipilih
-- karena hasilnya sama dan ia tidak menambah satu pun ketergantungan DB Link (`D-25`).
--
-- Tiga belas `LDC_ID` tidak punya pasangan di `BRANCH`; seluruhnya **nol klaim**, dan
-- sebelas di antaranya memang tidak punya `LDC_ID_PEGA` sama sekali.
--
-- Kode dan nama diambil dari SATU baris. Mengambilnya lewat dua kueri membuka kemungkinan
-- judul layar menyebut cabang yang berbeda dari cabang barisnya.
--
-- ### Kenapa `FETCH NEXT 1 ROWS ONLY` tetap dipasang meski OLDID terbukti unik
--
-- Keunikan itu terbaca dari ISI tabel hari ini, bukan dari constraint di katalog. Tanpa
-- pembatas, satu baris kembar yang masuk kelak akan membuat pemindai baris tunggal gagal —
-- layar mati total — alih-alih mengambil salah satunya.
--
-- ### `TRIM` di kedua sisi, dan itu bukan kehati-hatian berlebihan
--
-- `OLDID` bertipe VARCHAR2 dan diisi sistem lama; kode dari HCQ datang lewat JSON. Satu spasi
-- di ujung salah satunya cukup membuat seluruh pemanggil cabang itu tertolak, dan pesan yang
-- muncul menyuruh mereka menghubungi Tim IT — bukan menunjuk spasinya.
--
-- Bind: :1 kode cabang rinci (DetailBranchCode)
SELECT a.id         AS BRANCH_CODE,
       a.branchname AS BRANCH_NAME
  FROM POOLDATA.BRANCH a
 WHERE TRIM(a.oldid) = TRIM(:1)
 FETCH NEXT 1 ROWS ONLY

-- name: list
-- Isi grid untuk satu cabang.
-- — RDB List/GetDataOutstandingperCabang-SQL.xml
--
-- ### Pencarian nomor klaim dan nomor polis
--
-- KEMAMPUAN BARU; layar lama tidak punya kotak cari sama sekali. Ia disaring DI SINI, bukan
-- di peramban, karena daftarnya sudah dibagi per halaman di server — menyaring halaman yang
-- sedang tampil akan melewatkan baris pada halaman lain dan gagalnya DIAM: pengguna diberi
-- tabel kosong lalu menyimpulkan klaimnya tidak ada.
--
-- `COUNT(*) OVER ()` ikut tersaring dengan sendirinya, sehingga jumlah baris yang dilaporkan
-- adalah jumlah yang COCOK — bukan jumlah seluruh cabang.
--
-- Kedua sisi perbandingan di-UPPER supaya huruf kecil ikut cocok. Karakter khusus LIKE
-- di-escape di sisi Go, dan `ESCAPE` dinyatakan di sini — tanpa itu, mencari polis berisi
-- `%` akan berubah menjadi pola yang mencocokkan apa saja.
--
-- Bind: :1 kode cabang · :2 ada-pencarian (NULL = tampilkan semua) · :3 pola nomor klaim ·
--       :4 pola nomor polis · :5 offset · :6 jumlah baris
SELECT c.branchname                                   AS BRANCH_NAME,
       c.branchcode                                   AS BRANCH_CODE,
       c.sobname                                      AS BUSINESS_SOURCE,
       CASE
          WHEN c.grouppanel = '002' THEN 'PA'
          WHEN c.grouppanel = '003' THEN 'Aneka'
          WHEN c.grouppanel = '004' THEN 'Marine Cargo'
          WHEN c.grouppanel = '005' THEN 'Travel'
          WHEN c.grouppanel = '006' THEN 'Fire'
          ELSE c.grouppanel
       END                                            AS BUSINESS_NAME,
       c.nopolis                                      AS POLICY_NUMBER,
       (SELECT t.theinsured
          FROM POOLDATA.T_GENERAL t
         WHERE t.nopolis = c.nopolis
         ORDER BY CAST(TRIM(t.prodke) AS NUMERIC) DESC
         FETCH FIRST 1 ROW ONLY)                      AS INSURED_NAME,
       c.claimno                                      AS CLAIM_NUMBER,
       c.registerdate                                 AS REGISTER_DATE,
       c.dateofloss                                   AS LOSS_DATE,
       c.remarkrecomendation                          AS REMARK_RECOMMENDATION,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * 100)
                                                      AS ESTIMATION_VALUE,
       p.tgl_input                                    AS LAST_PROGRESS_AT,
       m1.sts_progress1                               AS PROGRESS_STATUS_1,
       m2.sts_progress2                               AS PROGRESS_STATUS_2,
       c.picteknik                                    AS TECHNICAL_PIC,
       p.keterangan                                   AS PROGRESS_NOTE,
       s.surveyor_name                                AS ADJUSTER_NAME,
       col.causeofloss                                AS CAUSE_OF_LOSS,
       c.kronologi                                    AS CHRONOLOGY,
       CASE WHEN stalled.pnccaseid IS NULL THEN 0 ELSE 1 END
                                                      AS PROGRESS_STALLED,
       COUNT(*) OVER ()                               AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
       JOIN POOLDATA.T_CLAIMLIST_ADMIN w
            ON c.claimid = w.pzinskey
           AND w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
           AND w.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
       LEFT JOIN (SELECT pnccaseid, tgl_input, status_progress1, status_progress2, keterangan
                    FROM (SELECT g.pnccaseid,
                                 g.tgl_input,
                                 g.status_progress1,
                                 g.status_progress2,
                                 g.keterangan,
                                 ROW_NUMBER() OVER (PARTITION BY g.pnccaseid
                                                        ORDER BY g.tgl_input DESC) rn
                            FROM POOLDATA.GCNM_PROGRESS_CLAIM g
                           WHERE g.status_progress1 IS NOT NULL)
                   WHERE rn = 1) p
            ON p.pnccaseid = c.claimno
       LEFT JOIN (  SELECT id_progress, MAX(sts_progress1) AS sts_progress1
                      FROM POOLDATA.GCNM_MST_PROGRESS
                  GROUP BY id_progress) m1
            ON m1.id_progress = p.status_progress1
       LEFT JOIN (  SELECT id_mst, MAX(sts_progress2) AS sts_progress2
                      FROM POOLDATA.GCNM_MST_PROGRESS
                  GROUP BY id_mst) m2
            ON m2.id_mst = p.status_progress2
       LEFT JOIN (  SELECT claimid,
                           SUM(estimationvalue * COALESCE(kursvalue, 1)) AS reserves
                      FROM POOLDATA.T_CLAIM_ESTIMASI
                  GROUP BY claimid) e
            ON e.claimid = c.claimid
       LEFT JOIN (  SELECT pnccaseid, MAX(surveyor_name) AS surveyor_name
                      FROM POOLDATA.T_SURVEYORLIST
                     WHERE surveytype = '2'
                  GROUP BY pnccaseid) s
            ON s.pnccaseid = c.claimid
       LEFT JOIN (SELECT claimid, causeofloss
                    FROM (SELECT o.claimid,
                                 o.causeofloss,
                                 ROW_NUMBER() OVER (PARTITION BY o.claimid
                                                        ORDER BY o.createdatetime DESC) rn
                            FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE o)
                   WHERE rn = 1) col
            ON col.claimid = c.claimid
       LEFT JOIN (  SELECT pnccaseid
                      FROM (SELECT g.pnccaseid,
                                   g.status_progress1,
                                   ROW_NUMBER() OVER (PARTITION BY g.pnccaseid
                                                          ORDER BY g.tgl_input DESC,
                                                                   g.id_update DESC) rn
                              FROM POOLDATA.GCNM_PROGRESS_CLAIM g)
                     WHERE rn <= 3
                  GROUP BY pnccaseid
                    HAVING COUNT(status_progress1) = 3
                       AND COUNT(DISTINCT status_progress1) = 1) stalled
            ON stalled.pnccaseid = c.claimno
 WHERE c.registerdate IS NOT NULL
   AND c.branchcode = :1
   AND (:2 IS NULL
        OR UPPER(c.claimno) LIKE :3 ESCAPE '\'
        OR UPPER(c.nopolis) LIKE :4 ESCAPE '\')
 ORDER BY c.registerdate ASC, c.claimno ASC
OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY

-- name: list_export
-- Isi berkas ekspor untuk satu cabang.
-- — RDB List/GetDataOutstandingperCabangExport-SQL.xml
--
-- Ia MENAMBAH kolom di atas `list`, tidak menggantinya: 19 kolom pertama sama persis dengan
-- urutan yang sama, lalu kolom yang hanya ada di berkas. Awalan yang cocok itu dijaga uji,
-- supaya satu kolom yang disisipkan di tengah `list` tidak diam-diam menggeser isi berkas.
--
-- ### DB Link `@asmd` dipakai di sini, dan itu keputusan sadar
--
-- `D-25` menetapkan seluruh DB Link kelak diganti pemanggilan API, dan `R-03` mencatat API-nya
-- kemungkinan belum ada. Selama itu belum tiba, kueri ini memakai link yang sama dengan sistem
-- lama — objek yang sama, sambungan yang sama, basis data yang sama. Preseden yang sama sudah
-- berjalan di `inboxlaporanklaim/repo/sqlstore/branch.sql`.
--
-- Yang membuatnya dapat diganti tanpa menyentuh aturan modul: ia berada di balik seam
-- `inboxosclaimpercabang.Repo`. Saat API penggantinya tiba, yang berubah hanya pengisi seam.
--
-- Diuji hidup terhadap basis data dev: `treaty_loss@asmd.sinarmas.co.id` terbaca, 19 baris
-- ber-`no_klaim` berawalan `PNC-`.
--
-- ### `TO_NUMBER(prodke)` diganti, bukan disalin
--
-- Bentuk berargumen satu sah di Oracle dan TIDAK sah di PostgreSQL — di sana `to_number`
-- menuntut format mask. `CAST(... AS NUMERIC)` adalah bentuk ANSI dari hal yang sama dan
-- diterima Oracle. Perlakuan yang sama sudah dipakai modul Open Protection.
--
-- Kedua subkueri ke `T_GENERAL` WAJIB memakai urutan yang identik. Mengubah salah satunya akan
-- memasangkan nama bisnis dari satu perpanjangan polis dengan nama tertanggung dari
-- perpanjangan lain — tanpa satu pun gejala.
--
-- Bind: :1 kode cabang · :2 offset · :3 jumlah baris
SELECT c.branchname                                   AS BRANCH_NAME,
       c.branchcode                                   AS BRANCH_CODE,
       c.sobname                                      AS BUSINESS_SOURCE,
       CASE
          WHEN c.grouppanel = '002' THEN 'PA'
          WHEN c.grouppanel = '003' THEN 'Aneka'
          WHEN c.grouppanel = '004' THEN 'Marine Cargo'
          WHEN c.grouppanel = '005' THEN 'Travel'
          WHEN c.grouppanel = '006' THEN 'Fire'
          ELSE c.grouppanel
       END                                            AS BUSINESS_NAME,
       c.nopolis                                      AS POLICY_NUMBER,
       (SELECT t.theinsured
          FROM POOLDATA.T_GENERAL t
         WHERE t.nopolis = c.nopolis
         ORDER BY CAST(TRIM(t.prodke) AS NUMERIC) DESC
         FETCH FIRST 1 ROW ONLY)                      AS INSURED_NAME,
       c.claimno                                      AS CLAIM_NUMBER,
       c.registerdate                                 AS REGISTER_DATE,
       c.dateofloss                                   AS LOSS_DATE,
       c.remarkrecomendation                          AS REMARK_RECOMMENDATION,
       ROUND(COALESCE(e.reserves_plain, 0) * 100)     AS ESTIMATION_VALUE,
       p.tgl_input                                    AS LAST_PROGRESS_AT,
       m1.sts_progress1                               AS PROGRESS_STATUS_1,
       m2.sts_progress2                               AS PROGRESS_STATUS_2,
       c.picteknik                                    AS TECHNICAL_PIC,
       p.keterangan                                   AS PROGRESS_NOTE,
       s.surveyor_name                                AS ADJUSTER_NAME,
       col.causeofloss                                AS CAUSE_OF_LOSS,
       c.kronologi                                    AS CHRONOLOGY,
       CASE WHEN stalled.pnccaseid IS NULL THEN 0 ELSE 1 END
                                                      AS PROGRESS_STALLED,
       -- Mulai dari sini kolom yang HANYA ada di berkas ekspor.
       --
       -- CLAIM_KEY tidak ditulis ke berkas; ia kunci pemasangan faktor dominan. Nomor klaim
       -- TIDAK dipakai untuk itu karena ia tidak unik — lihat ExportRow.ClaimKey.
       c.claimid                                      AS CLAIM_KEY,
       (SELECT t.businessname
          FROM POOLDATA.T_GENERAL t
         WHERE t.nopolis = c.nopolis
         ORDER BY CAST(TRIM(t.prodke) AS NUMERIC) DESC
         FETCH FIRST 1 ROW ONLY)                      AS POLICY_BUSINESS_NAME,
       ROUND(COALESCE(e.reserves_plain, 0) * 100)     AS RESERVE_CLAIM_FULL,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * 100)
                                                      AS RESERVE_CLAIM_ASM,
       ROUND(COALESCE(e.reserves, 0) * ((100 - c.shareasm) / 100) * 100)
                                                      AS COINSURANCE,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_or, 0) * 100)        AS SHARE_OR,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_fac_out, 0) * 100)   AS SHARE_FACOUT,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_facob, 0) * 100)     AS SHARE_FACOB,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_qs, 0) * 100)        AS SHARE_QS,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_fspl, 0) * 100)      AS SHARE_FSPL,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_sspl, 0) * 100)      AS SHARE_SSPL,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_er1, 0) * 100)       AS SHARE_ER1,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_er2, 0) * 100)       AS SHARE_ER2,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_bppdan, 0) * 100)    AS SHARE_BPPDAN,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_psrqs, 0) * 100)     AS SHARE_PSRQS,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_psrspl, 0) * 100)    AS SHARE_PSRSPL,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_ors, 0) * 100)       AS SHARE_ORS,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_xl, 0) * 100)        AS SHARE_XL,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_psror, 0) * 100)     AS SHARE_PSROR,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_qsor, 0) * 100)      AS SHARE_QSOR,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_pss, 0) * 100)       AS SHARE_PSS,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_prgbi, 0) * 100)     AS SHARE_PRGBI,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_pfra, 0) * 100)      AS SHARE_PFRA,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_fsplnsri, 0) * 100)  AS SHARE_FSPLNSRI,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_psplnsri, 0) * 100)  AS SHARE_PSPLNSRI,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_fsplnsor, 0) * 100)  AS SHARE_FSPLNSOR,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_psplnsor, 0) * 100)  AS SHARE_PSPLNSOR,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_facobsrb, 0) * 100)  AS SHARE_FACOBSRB,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * COALESCE(tl.claim_facobindt, 0) * 100) AS SHARE_FACOBINDT,
       COUNT(*) OVER ()                               AS TOTAL_ROWS
  FROM POOLDATA.T_CLAIM_PNC c
       JOIN POOLDATA.T_CLAIMLIST_ADMIN w
            ON c.claimid = w.pzinskey
           AND w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
           AND w.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
       LEFT JOIN (SELECT pnccaseid, tgl_input, status_progress1, status_progress2, keterangan
                    FROM (SELECT g.pnccaseid,
                                 g.tgl_input,
                                 g.status_progress1,
                                 g.status_progress2,
                                 g.keterangan,
                                 ROW_NUMBER() OVER (PARTITION BY g.pnccaseid
                                                        ORDER BY g.tgl_input DESC) rn
                            FROM POOLDATA.GCNM_PROGRESS_CLAIM g
                           WHERE g.status_progress1 IS NOT NULL)
                   WHERE rn = 1) p
            ON p.pnccaseid = c.claimno
       LEFT JOIN (  SELECT id_progress, MAX(sts_progress1) AS sts_progress1
                      FROM POOLDATA.GCNM_MST_PROGRESS
                  GROUP BY id_progress) m1
            ON m1.id_progress = p.status_progress1
       LEFT JOIN (  SELECT id_mst, MAX(sts_progress2) AS sts_progress2
                      FROM POOLDATA.GCNM_MST_PROGRESS
                  GROUP BY id_mst) m2
            ON m2.id_mst = p.status_progress2
       LEFT JOIN (  SELECT claimid,
                           SUM(estimationvalue * COALESCE(kursvalue, 1)) AS reserves,
                           SUM(estimationvalue)                          AS reserves_plain
                      FROM POOLDATA.T_CLAIM_ESTIMASI
                  GROUP BY claimid) e
            ON e.claimid = c.claimid
       LEFT JOIN (  SELECT pnccaseid, MAX(surveyor_name) AS surveyor_name
                      FROM POOLDATA.T_SURVEYORLIST
                     WHERE surveytype = '2'
                  GROUP BY pnccaseid) s
            ON s.pnccaseid = c.claimid
       LEFT JOIN (SELECT claimid, causeofloss
                    FROM (SELECT o.claimid,
                                 o.causeofloss,
                                 ROW_NUMBER() OVER (PARTITION BY o.claimid
                                                        ORDER BY o.createdatetime DESC) rn
                            FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE o)
                   WHERE rn = 1) col
            ON col.claimid = c.claimid
       LEFT JOIN (  SELECT pnccaseid
                      FROM (SELECT g.pnccaseid,
                                   g.status_progress1,
                                   ROW_NUMBER() OVER (PARTITION BY g.pnccaseid
                                                          ORDER BY g.tgl_input DESC,
                                                                   g.id_update DESC) rn
                              FROM POOLDATA.GCNM_PROGRESS_CLAIM g)
                     WHERE rn <= 3
                  GROUP BY pnccaseid
                    HAVING COUNT(status_progress1) = 3
                       AND COUNT(DISTINCT status_progress1) = 1) stalled
            ON stalled.pnccaseid = c.claimno
       LEFT JOIN (SELECT no_klaim, claim_or, claim_fac_out, claim_facob, claim_qs,
                         claim_fspl, claim_sspl, claim_er1, claim_er2, claim_bppdan,
                         claim_psrqs, claim_psrspl, claim_ors, claim_xl, claim_psror,
                         claim_qsor, claim_pss, claim_prgbi, claim_pfra, claim_fsplnsri,
                         claim_psplnsri, claim_fsplnsor, claim_psplnsor, claim_facobsrb,
                         claim_facobindt
                    FROM (SELECT tr.no_klaim, tr.claim_or, tr.claim_fac_out, tr.claim_facob,
                                 tr.claim_qs, tr.claim_fspl, tr.claim_sspl, tr.claim_er1,
                                 tr.claim_er2, tr.claim_bppdan, tr.claim_psrqs,
                                 tr.claim_psrspl, tr.claim_ors, tr.claim_xl, tr.claim_psror,
                                 tr.claim_qsor, tr.claim_pss, tr.claim_prgbi, tr.claim_pfra,
                                 tr.claim_fsplnsri, tr.claim_psplnsri, tr.claim_fsplnsor,
                                 tr.claim_psplnsor, tr.claim_facobsrb, tr.claim_facobindt,
                                 ROW_NUMBER() OVER (PARTITION BY tr.no_klaim
                                                        ORDER BY tr.no_spk DESC) rn
                            FROM treaty_loss@asmd.sinarmas.co.id tr
                           WHERE tr.no_klaim LIKE 'PNC-%')
                   WHERE rn = 1) tl
            ON tl.no_klaim = c.claimno
 WHERE c.registerdate IS NOT NULL
   AND c.branchcode = :1
 ORDER BY c.registerdate ASC, c.claimno ASC
OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY

-- name: dominant_factors
-- Faktor dominan setiap klaim outstanding milik satu cabang.
--
-- Ia menggantikan `LISTAGG(m.name, ', ') WITHIN GROUP (ORDER BY t.idx_dominanfactor)` pada
-- kueri ekspor lama. Alasannya bukan gaya: `LISTAGG` khas Oracle, dan padanan yang ditunjuk
-- `09-DATABASE-STRATEGY.md` §4 — `STRING_AGG` — **tidak dikenal Oracle 19c**. Tidak ada satu
-- bentuk yang berjalan di keduanya, sehingga perangkaiannya pindah ke Go.
--
-- ### Kenapa satu kueri per CABANG, bukan per halaman maupun per baris
--
-- Per baris berarti mengulangi pola yang justru sedang ditinggalkan — `OutstandingperCabang_PreAct`
-- sudah menjalankan satu kueri per baris untuk penanda progres mandek. Per halaman menuntut
-- pengikatan daftar nomor klaim yang panjangnya berubah-ubah, dan daftar bind dinamis adalah
-- pintu belakang menuju perangkaian teks SQL.
--
-- Per cabang bertahan seiring data bertambah karena ia terikat pada klaim OUTSTANDING satu
-- cabang, bukan pada seluruh tabel: penyaring dan gabungannya sama persis dengan `list`.
-- Ukurannya hari ini: `T_CLAIM_DOMINANFACTOR` berisi 6 baris di seluruh basis data.
--
-- Barisnya berurut menurut klaim lalu `idx_dominanfactor`, sehingga perangkaian di Go cukup
-- menyusul urutan yang sudah benar — bukan mengurutkannya lagi dengan aturan yang dapat
-- berselisih dengan yang di sini.
--
-- Bind: :1 kode cabang
SELECT d.claimid   AS CLAIM_KEY,
       f.name      AS FACTOR_NAME
  FROM POOLDATA.T_CLAIM_DOMINANFACTOR d
       JOIN POOLDATA.M_DOMINAN_FACTOR f
            ON f.id = d.id_dominanfactor
       JOIN POOLDATA.T_CLAIM_PNC c
            ON c.claimid = d.claimid
       JOIN POOLDATA.T_CLAIMLIST_ADMIN w
            ON c.claimid = w.pzinskey
           AND w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
           AND w.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
 WHERE c.registerdate IS NOT NULL
   AND c.branchcode = :1
 ORDER BY d.claimid ASC, d.idx_dominanfactor ASC

-- name: check_tables
-- Memastikan tabel yang disentuh modul ini terbaca dari koneksi yang dipakai.
--
-- Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah hak
-- baca dan keberadaan tabelnya, bukan isinya.
--
-- `treaty_loss@asmd` SENGAJA tidak ikut: ia hanya dibutuhkan ekspor, dan DB Link yang sedang
-- padam tidak boleh membuat seluruh layar dinyatakan rusak.
SELECT COUNT(*) AS READABLE
  FROM POOLDATA.T_CLAIM_PNC c
       JOIN POOLDATA.T_CLAIMLIST_ADMIN w ON c.claimid = w.pzinskey
       LEFT JOIN POOLDATA.GCNM_PROGRESS_CLAIM g ON g.pnccaseid = c.claimno
       LEFT JOIN POOLDATA.GCNM_MST_PROGRESS m ON m.id_progress = g.status_progress1
       LEFT JOIN POOLDATA.T_CLAIM_ESTIMASI e ON e.claimid = c.claimid
       LEFT JOIN POOLDATA.T_SURVEYORLIST s ON s.pnccaseid = c.claimid
       LEFT JOIN POOLDATA.T_CLAIM_OBJECTCOVERAGE o ON o.claimid = c.claimid
       LEFT JOIN POOLDATA.BRANCH b ON b.id = c.branchcode
 WHERE 1 = 0

-- name: any_branch_with_claims
-- Satu cabang yang BENAR-BENAR punya klaim outstanding, beserta kode RINCI-nya.
--
-- Dipakai perintah `-periksa` saja, tidak pernah oleh layar. Ia ada supaya pemeriksaan
-- menjalankan kueri daftar dengan kode cabang NYATA: kode karangan akan selalu menghasilkan
-- nol baris, dan nol baris tidak membuktikan kuerinya berjalan — ia justru menyembunyikan
-- gabungan yang rusak.
--
-- KEDUA kode dikembalikan, dan itu yang membuat pemeriksaannya berarti. Dengan `OLDID` di
-- tangan, `-periksa` dapat menempuh jalur yang SAMA PERSIS dengan layar — kode rinci lalu
-- `branch_of` lalu `list` — bukan menyuntikkan kode klaim langsung dan melewati satu-satunya
-- langkah yang paling mungkin salah.
--
-- Cabang yang `OLDID`-nya kosong dibuang: ia tidak dapat dipakai menempuh jalur itu.
--
-- Yang dipilih adalah cabang dengan klaim TERBANYAK, bukan yang pertama menurut kode.
-- Cabang yang isinya satu baris tidak akan pernah melewati satu pun gabungan LEFT JOIN yang
-- benar-benar kosong, sehingga ia membuktikan lebih sedikit.
SELECT branchcode AS BRANCH_CODE,
       oldid      AS DETAIL_BRANCH_CODE
  FROM (  SELECT c.branchcode, b.oldid, COUNT(*) AS baris
            FROM POOLDATA.T_CLAIM_PNC c
                 JOIN POOLDATA.T_CLAIMLIST_ADMIN w
                      ON c.claimid = w.pzinskey
                     AND w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
                     AND w.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
                 JOIN POOLDATA.BRANCH b
                      ON b.id = c.branchcode
                     AND b.oldid IS NOT NULL
           WHERE c.registerdate IS NOT NULL
             AND c.branchcode IS NOT NULL
        GROUP BY c.branchcode, b.oldid
        ORDER BY COUNT(*) DESC, c.branchcode ASC)
 FETCH NEXT 1 ROWS ONLY

-- name: check_export_tables
-- Memastikan tabel yang HANYA dipakai ekspor terbaca, termasuk lewat DB Link `@asmd`.
--
-- Terpisah dari check_tables supaya kegagalan DB Link terbaca sebagai kegagalan EKSPOR, bukan
-- sebagai kegagalan layar. Keduanya punya tindak lanjut yang berbeda.
SELECT COUNT(*) AS READABLE
  FROM POOLDATA.T_CLAIM_PNC c
       LEFT JOIN POOLDATA.T_GENERAL t ON t.nopolis = c.nopolis
       LEFT JOIN POOLDATA.T_CLAIM_DOMINANFACTOR d ON d.claimid = c.claimid
       LEFT JOIN POOLDATA.M_DOMINAN_FACTOR f ON f.id = d.id_dominanfactor
       LEFT JOIN treaty_loss@asmd.sinarmas.co.id tr ON tr.no_klaim = c.claimno
 WHERE 1 = 0

-- ============================================================================
-- POPUP DETAIL — empat kueri
-- ============================================================================
--
-- Asalnya `Harness/View_DetailKlaimCabang_Harness-Harness.xml` +
-- `Section/DetailKlaimCabang_Sect-Section.xml`, diisi
-- `Activity/ViewStatusProgressCabang_act-Act.xml` (15 langkah).
--
-- ### Sistem lama membacanya lewat CLIPBOARD; di sini dibaca dari tabel
--
-- Langkah 3 menjalankan `Obj-Open-By-Handle` atas `"ASM-FW-GCNMFW-WORK " + param.Inskey`,
-- yang memuat SELURUH pohon klaim ke memori Pega — klaim, objek, coverage, survei, adjustment
-- — lalu langkah-langkah berikutnya memungutnya dari sana. Go tidak punya padanan itu, dan
-- memuatnya pun tidak diinginkan: popup hanya menggambar sebagian kecilnya.
--
-- Karena itu keempat kueri di bawah membaca TEPAT yang digambar, tidak lebih.
--
-- ### SEMUANYA menyaring cabang, termasuk yang menerima nomor klaim
--
-- Popup lama tidak menyaring karena ia hanya dapat dibuka dari baris yang sudah tampil.
-- Endpoint HTTP tidak punya pembatas itu. Tanpa penyaring cabang, popup menjadi jalan memutar
-- yang membocorkan nama tertanggung dan nilai uang antarbadan hukum (`R-20`).
--
-- Ketiga kueri anak menerima `CLAIMID`, bukan nomor klaim. Kuncinya sudah dipastikan milik
-- cabang pemanggil oleh detail_header, dan `CLAIMID` unik sedangkan `CLAIMNO` tidak.

-- name: detail_header
-- Delapan nilai ringkasan pada kepala popup, untuk satu klaim milik satu cabang.
--
-- ### Penyaringnya sama persis dengan kueri daftar, dan itu disengaja
--
-- Cabang, `registerdate IS NOT NULL`, dan penyaring outstanding ketiganya diulang di sini.
-- Bila popup lebih longgar daripada daftarnya, akan ada klaim yang tidak tampil di layar
-- tetapi isinya tetap dapat dibaca lewat nomor.
--
-- ### "Total Sum Insured" dan "Occupation" diambil dari SATU objek, bukan seluruhnya
--
-- Di Pega, `local.idxobj` disetel di dalam perulangan atas `ObjectList` (langkah 4) sehingga
-- nilainya berakhir pada objek TERAKHIR; langkah 6 dan 7 lalu mengisi `TempDetail.TSI` dan
-- `TempDetail.Occupation` dari objek itu saja. Label "Total Sum Insured" karena itu
-- menyesatkan — angkanya bukan total. Perilakunya direplikasi (`P-5`).
--
-- "Objek terakhir" di Pega berarti baris terakhir pada clipboard, dan urutan itu tidak dapat
-- direproduksi: `URUTAN` terisi pada **4 dari 2.610 baris** dan `OBJECTINDEX` pada **0**.
-- Yang dipakai `OBJECTID` terbesar — satu-satunya kolom yang terisi pada seluruh baris.
-- Selisihnya dinyatakan lewat DetailPlannedDifferences.
--
-- ### Occupation menempuh dua jalur, dan keduanya kosong pada data hari ini
--
-- Langkah 6 memakai `OCCUPATIONNAME` objek; langkah 10 mencarinya lewat `OCCUPATIONID` ke
-- tabel `OCCUPATION`. Diukur langsung: **keduanya NULL pada seluruh 2.610 baris**, sehingga
-- nilai ini akan kosong di lingkungan ini. Jalurnya tetap dibangun karena produksi dapat
-- berbeda, dan kekosongannya dilaporkan — bukan disamarkan menjadi tanda hubung.
--
-- Jalur ketiga milik Pega — langkah 8, lewat `CoverageList(1).DeductibleList(1)` — TIDAK
-- dibawa: daftar deductible tidak punya tabel yang terbaca dari export.
--
-- Bind: :1 kode cabang · :2 nomor klaim
SELECT c.claimno                                AS CLAIM_NUMBER,
       c.claimid                                AS CLAIM_KEY,
       CASE
          WHEN c.grouppanel = '002' THEN 'PA'
          WHEN c.grouppanel = '003' THEN 'Aneka'
          WHEN c.grouppanel = '004' THEN 'Marine Cargo'
          WHEN c.grouppanel = '005' THEN 'Travel'
          WHEN c.grouppanel = '006' THEN 'Fire'
          ELSE c.grouppanel
       END                                      AS BUSINESS_NAME,
       COALESCE(obj.occupationname, occ.notes)  AS OCCUPATION,
       ROUND(COALESCE(cvg.sumtsi, 0) * 100)     AS TOTAL_SUM_INSURED,
       c.kronologi                              AS CHRONOLOGY,
       ROUND(COALESCE(e.reserves, 0) * 100)     AS ESTIMATION_VALUE,
       c.registerdate                           AS REGISTER_DATE,
       c.remarkrecomendation                    AS REMARK_RECOMMENDATION,
       p.keterangan                             AS PROGRESS_NOTE
  FROM POOLDATA.T_CLAIM_PNC c
       JOIN POOLDATA.T_CLAIMLIST_ADMIN w
            ON c.claimid = w.pzinskey
           AND w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
           AND w.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
       LEFT JOIN (SELECT claimid, objectid, occupationname, occupationid
                    FROM (SELECT o.claimid,
                                 o.objectid,
                                 o.occupationname,
                                 o.occupationid,
                                 ROW_NUMBER() OVER (PARTITION BY o.claimid
                                                        ORDER BY o.objectid DESC) rn
                            FROM POOLDATA.T_CLAIM_OBJECTLIST o)
                   WHERE rn = 1) obj
            ON obj.claimid = c.claimid
       LEFT JOIN OCCUPATION occ
            ON occ.id = obj.occupationid
       LEFT JOIN (SELECT claimid, objectid, sumtsi
                    FROM (SELECT v.claimid,
                                 v.objectid,
                                 v.sumtsi,
                                 ROW_NUMBER() OVER (PARTITION BY v.claimid, v.objectid
                                                        ORDER BY v.objectcoverageid DESC) rn
                            FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE v)
                   WHERE rn = 1) cvg
            ON cvg.claimid = c.claimid
           AND cvg.objectid = obj.objectid
       LEFT JOIN (  SELECT claimid, SUM(estimationvalue) AS reserves
                      FROM POOLDATA.T_CLAIM_ESTIMASI
                  GROUP BY claimid) e
            ON e.claimid = c.claimid
       LEFT JOIN (SELECT pnccaseid, keterangan
                    FROM (SELECT g.pnccaseid,
                                 g.keterangan,
                                 ROW_NUMBER() OVER (PARTITION BY g.pnccaseid
                                                        ORDER BY g.tgl_input DESC) rn
                            FROM POOLDATA.GCNM_PROGRESS_CLAIM g
                           WHERE g.status_progress1 IS NOT NULL)
                   WHERE rn = 1) p
            ON p.pnccaseid = c.claimno
 WHERE c.registerdate IS NOT NULL
   AND c.branchcode = :1
   AND c.claimno = :2
 FETCH NEXT 1 ROWS ONLY

-- name: detail_objects
-- Isi grid objek pertanggungan.
--
-- SELURUH kolom ketiga varian dikembalikan sekaligus. Yang memilih varian di Pega adalah when
-- rule `IsAneka`, `IsTravel`, `IsPA`, dan `IsMarineCargo` — dan **keempatnya tidak ada di
-- export** (`R-16`). Memilihnya di sini berarti menebak; memilihnya di layar dari lini bisnis
-- dapat diperbaiki tanpa menyentuh penyimpanan begitu keempat when rule tiba.
--
-- `LOKASI`, bukan `OBJECTLOCATION`. Properti Pega bernama `.ObjectLocation`, tetapi kolom
-- dengan nama itu TIDAK ADA di `T_CLAIM_OBJECTLIST` — diverifikasi ke katalog.
--
-- Urutannya `OBJECTID`, satu-satunya kolom yang terisi pada seluruh baris. Lihat catatan
-- detail_header.
--
-- Bind: :1 kunci internal klaim (CLAIMID)
SELECT o.objectid                AS OBJECT_ID,
       o.objectname              AS OBJECT_NAME,
       o.lokasi                  AS OBJECT_LOCATION,
       o.objectjob               AS OBJECT_JOB,
       o.dateofbirth             AS OBJECT_DATE_OF_BIRTH,
       o.objectidcard            AS OBJECT_ID_CARD,
       o.objectparticipantstatus AS OBJECT_PARTICIPANT_STATUS
  FROM POOLDATA.T_CLAIM_OBJECTLIST o
 WHERE o.claimid = :1
 ORDER BY o.objectid ASC

-- name: detail_object_coverages
-- Isi panel yang terbuka ketika sebuah baris objek dibuka.
--
-- # Grid objek di Pega MEMANG dapat dibuka
--
-- Mekanismenya master-detail, bukan `pyExpandable` — dan itu sempat terlewat pada pembacaan
-- pertama, yang keliru menyimpulkan layar lama tidak punya kemampuan ini:
--
--   Section/DetailKlaimCabang_Sect-Section.xml
--     pyRowEditing = masterDetail    pada ketiga varian grid objek (:5616, :7193, :9798)
--     pyEditAction = ViewObjectItem                               (:5625, :7179, :9815)
--
-- `Flow Action/ViewObjectItem-FlowAction.xml` merender
-- `Section/ViewObjectCoverage-Section.xml`, yang menggambar `.ObjectCoverageList` dengan
-- **tiga** properti: `.Currency`, `.SumTSI`, `.CoverageNote`. Hanya ketiganya yang diambil —
-- menambah kolom di luar itu adalah penambahan, bukan penyamaan (`P-5`).
--
-- `COVERAGENAME`, bukan `REMARKS`. Properti Pega bernama `.CoverageNote`, tetapi kolom yang
-- digambarnya berjudul **"Coverage"** dan berisi NAMA jaminan, bukan catatan. Namanya
-- menyesatkan — pola yang sama dengan `.ObjectLocation` yang tersimpan di `LOKASI`.
--
-- Diukur, bukan ditebak: nilai contoh dari layar lama (`FLEXAS`) muncul pada **358 baris
-- `COVERAGENAME` dan NOL baris `REMARKS`**. `COVERAGENAME` juga terisi pada 9.534 dari 9.704
-- baris, sedangkan `REMARKS` hanya 249 — kolom yang hampir selalu kosong tidak mungkin yang
-- digambar sebagai kolom pertama.
--
-- `DIHAPUS_PADA IS NULL` menjaga penghapusan lunak (`D-66`), sejalan dengan modul lain yang
-- membaca tabel ini. Coverage yang sudah dibuang tidak boleh muncul kembali hanya karena
-- barisnya masih ada.
--
-- SELURUH coverage satu klaim diambil dalam SATU kueri, lalu dikelompokkan di Go menurut
-- OBJECT_ID. Satu kueri per baris yang dibuka akan menghasilkan N+1 permintaan pada layar
-- yang justru dirancang untuk dibuka-tutup berkali-kali.
--
-- Bind: :1 kunci internal klaim (CLAIMID)
-- # Mata uang adalah KODE, dan harus diterjemahkan
--
-- `T_CLAIM_OBJECTCOVERAGE.CURRENCY` menyimpan kode internal (`10026`), bukan simbol. Layar
-- lama menampilkan `IDR`, sehingga menggambar kolom itu apa adanya memunculkan angka yang
-- tidak berarti bagi siapa pun — dan itu benar-benar terjadi pada serahan pertama.
--
-- Masternya `POOLDATA.CURRENCY`, bukan `M_CURRENCY`: yang kedua memang ada dan ber-ID sama,
-- tetapi kolom simbolnya KOSONG pada seluruh 35 barisnya. Diukur: `CURRENCY.ID = '10026'`
-- menghasilkan `IDR`, `'10001'` menghasilkan `USD`, dan dari 9.704 baris coverage hanya
-- **satu** yang kodenya tidak cocok (7.217 lainnya memang NULL sejak awal).
--
-- LEFT JOIN: kode yang kosong atau tidak dikenal tetap menampilkan barisnya, dengan mata
-- uang kosong — bukan menghilangkan coverage-nya.
SELECT v.objectid         AS OBJECT_ID,
       v.objectcoverageid AS OBJECT_COVERAGE_ID,
       v.coveragename     AS COVERAGE_NAME,
       k.isosymbol        AS COVERAGE_CURRENCY,
       ROUND(COALESCE(v.sumtsi, 0) * 100) AS COVERAGE_SUM_TSI
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE v
       LEFT JOIN POOLDATA.CURRENCY k ON TRIM(k.id) = TRIM(v.currency)
 WHERE v.claimid = :1
   AND v.dihapus_pada IS NULL
 ORDER BY v.objectid ASC, v.urutan ASC, v.objectcoverageid ASC

-- name: detail_object_items
-- Grid "Object Item" — tingkat ketiga, yang terbuka saat baris coverage dibuka.
--
-- # Barisnya datang dari ESTIMASI, bukan dari tabel item
--
-- Ini kesimpulan yang menyelamatkan grid ini dari selalu kosong, dan ia diukur:
--
--	POOLDATA.T_CLAIM_OBJECTITEMLIST   **1 baris di seluruh tabel**
--	POOLDATA.T_CLAIM_ESTIMASI         51.532 baris, OBJECTITEMID terisi SELURUHNYA (0 NULL)
--	klaim ber-estimasi                22.004; yang juga punya baris item: **1**
--
-- Jadi menarik grid ini dari tabel item akan menghasilkan nol baris untuk hampir setiap
-- klaim — padahal layar lama MENAMPILKAN barisnya. Yang ditampilkan Pega adalah satu baris
-- per `OBJECTITEMID` yang muncul di estimasi, dengan sel nama dan deskripsi KOSONG karena
-- tabel itu memang tidak memuatnya. Contoh PA dari Work Owner memperlihatkan persis itu:
-- baris ada, selnya kosong.
--
-- `OBJECTITEMID` adalah NOMOR URUT item di dalam satu coverage, bukan kode jenis: 51.370
-- dari 51.532 baris bernilai `1`.
--
-- Nama dan deskripsi tetap di-LEFT JOIN ke tabel item, sehingga satu baris yang memang ada
-- akan tampil namanya — bukan dibuang hanya karena tetangganya kosong.
--
-- Bind: :1 kunci internal klaim (CLAIMID)
SELECT e.objectid         AS OBJECT_ID,
       e.objectcoverageid AS OBJECT_COVERAGE_ID,
       e.objectitemid     AS OBJECT_ITEM_ID,
       MAX(i.objectname)       AS ITEM_NAME,
       MAX(i.deskripsiobject)  AS ITEM_DESCRIPTION
  FROM POOLDATA.T_CLAIM_ESTIMASI e
       LEFT JOIN POOLDATA.T_CLAIM_OBJECTITEMLIST i
              ON i.claimid = e.claimid
             AND TRIM(i.objectid) = TRIM(e.objectid)
             AND TRIM(i.objectitemid) = TRIM(e.objectitemid)
 WHERE e.claimid = :1
 GROUP BY e.objectid, e.objectcoverageid, e.objectitemid
 ORDER BY e.objectid ASC, e.objectcoverageid ASC, e.objectitemid ASC

-- name: detail_estimations
-- Grid "Estimasi" — tingkat keempat, yang terbuka saat baris Object Item dibuka.
--
-- Keenam kolomnya diambil dari layar lama apa adanya (`D-13`):
--
--	Estimasi Ke        <- ESTIMASIID, nomor urut estimasi pada item itu
--	Tanggal Estimasi   <- COALESCE(ESTIMATIONDATE, coverage.CREATEDATETIME)
--	Tipe Estimasi      <- ESTIMATIONTYPE  (kerap kosong; contoh PA memperlihatkannya kosong)
--
-- # Tanggal estimasi menempuh DUA sumber, dan urutannya penting
--
-- Dua serahan sebelumnya keduanya salah, dan keduanya ketahuan dari layar:
--
--	ESTIMATIONDATE saja   -> kolomnya KOSONG; ia terisi pada 32 dari 51.535 baris
--	INSERTDATE            -> kolomnya terisi TANGGAL HARI INI, bukan tanggal estimasi
--
-- `INSERTDATE` yang paling berbahaya: ia terisi seluruhnya, sehingga tampak benar — padahal
-- isinya kapan baris itu ditulis ulang job konversi. Pada klaim contoh Work Owner nilainya
-- hari pemeriksaan, sementara layar lama menuliskan `08/05/23 11:45`.
--
-- Sumber yang benar dicari dengan memindai **121 kolom tanggal** milik seluruh tabel POOLDATA
-- yang berkunci `CLAIMID`, lalu memeriksa mana yang memuat menit itu untuk klaim tersebut.
-- Hasilnya tepat satu: **`T_CLAIM_OBJECTCOVERAGE.CREATEDATETIME`**.
--
-- Tetapi ia tidak cukup sendirian: contoh Fire dari Work Owner memperlihatkan LIMA waktu
-- berbeda pada satu coverage, sedangkan `CREATEDATETIME` tentu sama untuk semuanya. Diukur
-- pada coverage yang estimasinya banyak — saat `ESTIMATIONDATE` terisi, ia memang berbeda per
-- baris (`09:00` versus `19:00`) sementara `CREATEDATETIME` keduanya sama.
--
-- Karena itu: `ESTIMATIONDATE` bila ada, `CREATEDATETIME` bila tidak. Keterisian
-- `CREATEDATETIME` **9.750 dari 9.750** — ia tidak pernah kosong, sehingga kolom ini tidak
-- akan pernah kosong lagi.
--
-- `ESTIMATIONTYPE` terisi pada 8.490 dari 51.535 baris; kolom yang kosong pada contoh PA
-- karena itu BENAR, bukan cacat.
--	Mata Uang          <- KURSID, diterjemahkan lewat POOLDATA.CURRENCY
--	Nilai Kurs (IDR)   <- KURSVALUE
--	Nilai Estimasi     <- ESTIMATIONVALUE
--
-- Nilai negatif memang terjadi dan WAJIB dibawa apa adanya: contoh Fire dari Work Owner
-- memuat pasangan `100 / -100 / 200 / -200 / 100`, yakni koreksi yang saling meniadakan.
-- Menyaring yang negatif akan membuat jumlahnya tidak pernah cocok dengan layar lama.
--
-- Bind: :1 kunci internal klaim (CLAIMID)
SELECT e.objectid         AS OBJECT_ID,
       e.objectcoverageid AS OBJECT_COVERAGE_ID,
       e.objectitemid     AS OBJECT_ITEM_ID,
       e.estimasiid       AS ESTIMATION_SEQUENCE,
       COALESCE(e.estimationdate, v.createdatetime) AS ESTIMATION_DATE,
       e.estimationtype   AS ESTIMATION_TYPE,
       k.isosymbol        AS ESTIMATION_CURRENCY,
       ROUND(COALESCE(e.kursvalue, 0) * 100)       AS ESTIMATION_RATE,
       ROUND(COALESCE(e.estimationvalue, 0) * 100) AS ESTIMATION_VALUE
  FROM POOLDATA.T_CLAIM_ESTIMASI e
       LEFT JOIN POOLDATA.CURRENCY k ON TRIM(k.id) = TRIM(e.kursid)
       LEFT JOIN POOLDATA.T_CLAIM_OBJECTCOVERAGE v
              ON v.claimid = e.claimid
             AND TRIM(v.objectid) = TRIM(e.objectid)
             AND TRIM(v.objectcoverageid) = TRIM(e.objectcoverageid)
 WHERE e.claimid = :1
 ORDER BY e.objectid ASC, e.objectcoverageid ASC, e.objectitemid ASC,
          e.estimasiid ASC

-- name: detail_coverage_spreading
-- Grid "List Spreading" — melekat pada COVERAGE, bukan pada adjustment.
--
-- # Kenapa coverage, dan bukan adjustment
--
-- Sempat disimpulkan sebaliknya dari `Activity/CalculatedSpredingForClaimKomite-Act.xml`,
-- yang memang menghitung per `AdjustmentList(Param.idadjustment)`. **Activity itu milik layar
-- Komite, dan popup ini tidak memanggilnya.** Yang mengisi popup adalah
-- `Activity/GetObjectFromTable-Act.xml`, dan di sana daftarnya melekat pada coverage:
-- `ObjectCoverageList(<LAST>).SpreadingList`. Kunci tabelnya pun sejalan —
-- `CLAIMID + OBJECTID + OBJECTCOVERAGEID`, tanpa adjustment.
--
-- # Dua kolom uangnya DIHITUNG, karena memang tidak tersimpan
--
-- `TSISPREADED` dan `PREMIUMSPREADED` **NULL pada seluruh 7.115 baris** tabel ini. Jadi
-- "Estimasi Value" dan "Result Value" tidak mungkin dibaca dari sini; keduanya dihitung di Go
-- setelah estimasi coverage terkumpul:
--
--	Estimasi Value = jumlah estimasi coverage itu
--	Result Value   = Estimasi Value x SHAREPERCENTAGE / 100
--
-- Dicocokkan ke contoh Fire dari Work Owner: estimasi `100 / -100 / 200 / -200 / 100`
-- berjumlah **100**, dan layar lama menuliskan Estimasi Value `100,00` serta Result Value
-- `100,00` pada share `100,0000%`. Keduanya cocok persis.
--
-- `TREATYNAME`, bukan `TREATYTYPE`, untuk kolom "Tipe Treaty": isinya `ORS`, `QS`, `FAC-OUT`,
-- `FSPL`, `OR` — dan `FAC-OUT` itulah yang tertulis di contoh Work Owner. `TREATYTYPE`
-- berisi kode (`10007`).
--
-- SHARE dikembalikan sebagai BILANGAN BULAT berskala 10.000 (`100%` -> `1000000`), supaya
-- perkalian persentase terhadap nilai uang tidak pernah menempuh float (`I-12`). Diukur:
-- `SHAREPERCENTAGE` per coverage berjumlah tepat 100 pada 7.108 dari 7.112 kelompok.
--
-- Bind: :1 kunci internal klaim (CLAIMID)
SELECT s.objectid         AS OBJECT_ID,
       s.objectcoverageid AS OBJECT_COVERAGE_ID,
       s.treatyname       AS TREATY_NAME,
       ROUND(COALESCE(s.sharepercentage, 0) * 10000) AS SHARE_SCALED
  FROM POOLDATA.T_CLAIM_SPREADING s
 WHERE s.claimid = :1
 ORDER BY s.objectid ASC, s.objectcoverageid ASC, s.urutan ASC

-- name: detail_coverage_comember
-- Grid "CO MEMBER" — daftar koasuransi.
--
-- # Ia berkunci POLIS, bukan klaim
--
-- `T_COINSLIST` berkunci `NOPOLIS + PRODKE`, sehingga satu daftar berlaku untuk SELURUH
-- coverage pada klaim itu. Yang berbeda per coverage hanyalah nilai uangnya, karena ia
-- dihitung dari estimasi coverage masing-masing.
--
-- Nomor polis diambil dari klaimnya sendiri, bukan diterima dari pemanggil: menerimanya dari
-- luar berarti daftar koasuransi dapat ditukar dengan milik polis lain lewat satu parameter.
--
-- Dua kolom uangnya dihitung di Go, pola yang sama dengan Spreading:
--
--	Estimasi Value = jumlah estimasi coverage itu
--	Result Value   = Estimasi Value x PERCENT_SHARE / 100
--
-- Bentuk itu sama persis dengan rumus pada jalur Komite
-- (`TSIShare = .PercentShare * Local.grossvalue / 100`), hanya basisnya berbeda.
--
-- `FLAGDELETE` disaring: baris yang sudah dibuang tidak boleh muncul kembali.
--
-- Bind: :1 kunci internal klaim (CLAIMID)
SELECT k.coinsname AS COINS_NAME,
       ROUND(COALESCE(k.percent_share, 0) * 10000) AS SHARE_SCALED
  FROM POOLDATA.T_COINSLIST k
       JOIN POOLDATA.T_CLAIM_PNC c
            ON TRIM(c.nopolis) = TRIM(k.nopolis)
           AND TRIM(c.prodke) = TRIM(k.prodke)
 WHERE c.claimid = :1
   AND COALESCE(k.flagdelete, '0') <> '1'
 ORDER BY k.leader DESC, k.coinsname ASC

-- name: detail_progress
-- Isi grid riwayat progres klaim.
-- — RDB List/GetCommunicationList-SQL.xml
--
-- ### Status Progress 1 dibaca dari tabel yang BERBEDA dari kueri daftar
--
-- Di sini `GCNM_MST_PROGRESS_KLAIM`; pada kueri daftar `GCNM_MST_PROGRESS`. Itu bukan salah
-- salin — kueri lamanya memang memakai tabel yang berbeda di kedua tempat, dan keduanya
-- dibawa apa adanya (`P-5`).
--
-- Gabungan ke `GCNM_MST_PROGRESS` untuk Status Progress 2 menempuh DUA kolom sekaligus,
-- `ID_MST` dan `ID_PROGRESS`. Menghilangkan salah satunya mengubah baris mana yang cocok.
--
-- Kunci penyaringnya `PNCCASEID`, yang pada tabel ini berisi NOMOR klaim — bukan `CLAIMID`.
-- Kueri lama menyetel `TempSearch.CARI1 := param.Inskey`, dan `Inskey` diisi `.ClaimNo`.
--
-- Bind: :1 nomor klaim
SELECT a.tgl_input     AS RECORDED_AT,
       a.pnccaseid     AS CLAIM_NUMBER,
       b.sts_progress1 AS PROGRESS_STATUS_1,
       c.sts_progress2 AS PROGRESS_STATUS_2,
       a.user_input    AS ENTERED_BY,
       a.next_followup AS NEXT_FOLLOW_UP_AT,
       a.status        AS PROGRESS_STATUS,
       a.keterangan    AS PROGRESS_NOTE
  FROM POOLDATA.GCNM_PROGRESS_CLAIM a
       LEFT JOIN POOLDATA.GCNM_MST_PROGRESS_KLAIM b
            ON a.status_progress1 = b.id_progress
       LEFT JOIN POOLDATA.GCNM_MST_PROGRESS c
            ON a.status_progress2 = c.id_mst
           AND b.id_progress = c.id_progress
 WHERE a.pnccaseid = :1
 ORDER BY a.tgl_input DESC

-- name: detail_messages
-- Isi grid "KOMUNIKASI DENGAN LOSS ADJUSTER".
-- — RDB List/GetInboxKomunikasi_OS_Cabang-SQL.xml
--
-- ### Kueri lama merangkai daftar kunci sebagai TEKS, dan itu celah injeksi
--
-- Langkah 12 memutari `SurveyResults` merangkai `getkomunikasi.M_SURVEY_ID` menjadi
-- `'a','b','c'`, langkah 13 menambahkan `pzInsKey`, lalu kuerinya menyisipkannya mentah:
--
--   WHERE a.caseid in ({ASIS:getkomunikasi.D_SURVEY_ID})
--
-- Pola `{ASIS:…}` adalah perangkaian SQL dari nilai, yang dilarang tanpa perkecualian
-- (`08-TECHNICAL-STRATEGY.md` §4.3). Di sini daftar itu menjadi SUBKUERI, sehingga tidak ada
-- satu pun nilai yang menyentuh teks SQL — sekaligus menghapus perjalanan per baris survei.
--
-- ### INTERNAL vs EXTERNAL dibawa, tetapi tidak lagi menentukan urutan
--
-- Kueri lama memecah hasilnya menjadi tiga UNION ALL — pesan internal, pesan eksternal, dan
-- balasan — lalu mengurutkannya. Di sini satu baris membawa pesan DAN balasannya sekaligus,
-- persis seperti tabelnya menyimpannya, dan penandanya dikirim sebagai kolom. Layar yang
-- menempatkannya di sisi kiri atau kanan.
--
-- `ROWNUM` pada kueri lama dipakai mengambil satu nama pengirim internal untuk SELURUH baris
-- — sehingga semua baris menampilkan nama yang sama. Itu tidak dibawa: setiap baris membawa
-- nama pengirimnya sendiri.
--
-- Bind: :1 kunci internal klaim (CLAIMID), dipakai dua kali → :1 dan :2
SELECT k.sendername      AS SENDER_NAME,
       k.createddate     AS SENT_AT,
       k.message         AS MESSAGE,
       k.createdatereply AS REPLIED_AT,
       k.replymessage    AS REPLY,
       CASE
          WHEN k.sender IN (SELECT u.operator_id
                              FROM POOLDATA.MST_USER_TEKNIK u
                             WHERE u.sts_aktif = '1')
          THEN 1 ELSE 0
       END               AS IS_INTERNAL
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.caseid = :1
    OR k.caseid IN (SELECT s.caseid
                      FROM POOLDATA.T_SURVEYORLIST s
                     WHERE s.pnccaseid = :2)
 ORDER BY k.createddate DESC

-- name: detail_dominant_factors
-- Faktor dominan satu klaim, sudah berurut.
-- — RDB List/GetDataDominanFactorListOS-SQL.xml
--
-- Kueri lama merangkainya dengan `LISTAGG`, yang tidak ada di PostgreSQL 17+, sementara
-- padanan yang dianjurkan `09-DATABASE-STRATEGY.md` §4 — `STRING_AGG` — tidak ada di Oracle
-- 19c. Tidak ada satu bentuk pun yang berjalan di keduanya, sehingga perangkaiannya pindah ke
-- Go. Alasan lengkapnya di kepala berkas ini.
--
-- Bind: :1 kunci internal klaim (CLAIMID)
SELECT m.name AS FACTOR_NAME
  FROM POOLDATA.T_CLAIM_DOMINANFACTOR d
       JOIN POOLDATA.M_DOMINAN_FACTOR m
            ON m.id = d.id_dominanfactor
 WHERE d.claimid = :1
 ORDER BY d.idx_dominanfactor ASC

-- name: check_detail_tables
-- Memastikan tabel yang HANYA dipakai popup Detail terbaca dari koneksi yang dipakai.
--
-- Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun.
--
-- Terpisah dari check_tables supaya kegagalannya terbaca sebagai kegagalan POPUP, bukan
-- sebagai kegagalan daftar. Yang pertama membuat satu tombol tidak bekerja; yang kedua
-- menghentikan seluruh layar cabang.
SELECT COUNT(*) AS READABLE
  FROM POOLDATA.T_CLAIM_OBJECTLIST o
       LEFT JOIN POOLDATA.T_CLAIM_OBJECTCOVERAGE v ON v.claimid = o.claimid
       LEFT JOIN OCCUPATION occ ON occ.id = o.occupationid
       LEFT JOIN POOLDATA.GCNM_MST_PROGRESS_KLAIM b ON b.id_progress = o.claimid
       LEFT JOIN POOLDATA.M_KOMUNIKASI_PNC k ON k.caseid = o.claimid
       LEFT JOIN POOLDATA.MST_USER_TEKNIK u ON u.operator_id = k.sender
 WHERE 1 = 0

-- ============================================================================
-- RINGKASAN MONITORING — dua kueri
-- ============================================================================
--
-- Melayani panel ringkasan di atas grid: kartu angka, sebaran umur, rincian per COB, dan
-- rincian per sumber bisnis.
--
-- ### Ia TIDAK ada di Pega
--
-- Layar lama hanya punya judul, tombol ekspor, dan grid. Panel ini diminta Work Owner
-- (2026-10-08) dan karena itu **tidak punya pembanding untuk uji kesetaraan** — sama halnya
-- dengan `F-3` dan `S-5` (`D-56`). Yang dapat diuji hanyalah bahwa angkanya konsisten dengan
-- grid di bawahnya, dan itulah yang dijaga query_test.go.
--
-- ### Kenapa baris mentah, bukan GROUP BY di basis data
--
-- Pengelompokan umur menuntut pemotongan tanggal, dan `CAST(x AS DATE)` **tidak memangkas jam
-- di Oracle** — alasan yang sama yang membuat kolom Aging dihitung di Go (lihat catatan Aging
-- di kepala berkas ini). Mengelompokkan di SQL berarti menulis aturan umur untuk KEDUA kalinya,
-- dengan bentuk yang tidak dapat dibuat sama persis.
--
-- Biayanya terukur dan kecil: outstanding per cabang paling banyak **83 baris**, rerata **9,2**
-- (diukur atas 50 cabang, 469 klaim). Yang diambil hanya lima kolom sempit per baris.
--
-- ### Dua kueri, dan kenapa dipisah
--
-- `summary_rows` tidak menyentuh DB Link sama sekali. `summary_treaty_or` menyentuhnya, dan
-- itu satu-satunya alasan ia terpisah: `@asmd` yang sedang padam harus membuat SATU kartu
-- angka kosong, bukan seluruh panel hilang.

-- name: summary_rows
-- Satu baris sempit per klaim outstanding cabang, untuk diringkas pemanggil.
--
-- Penyaringnya SAMA PERSIS dengan kueri `list` — cabang, `registerdate IS NOT NULL`, dan
-- penyaring outstanding. Bila berbeda, kartu angka di atas tidak akan cocok dengan jumlah
-- baris grid di bawahnya, dan pengguna tidak punya cara menjelaskan selisihnya.
--
-- Nilai uangnya WAJIB dihitung sama persis dengan kolom grid — `SUM(estimationvalue *
-- kursvalue) * SHAREASM/100` sejak keputusan Work Owner 2026-10-10. Panel ini menyatakan
-- dirinya "Dijumlahkan dari kolom Reserve Claim ASM Share di bawah"; bila rumusnya berbeda,
-- kalimat itu menjadi bohong dan tidak ada cara pengguna menjelaskan selisihnya.
--
-- TIDAK dipaginasi, dan itu disengaja: ringkasan atas sebagian baris adalah ringkasan yang
-- salah.
--
-- Bind: :1 kode cabang
SELECT c.registerdate                        AS REGISTER_DATE,
       CASE
          WHEN c.grouppanel = '002' THEN 'PA'
          WHEN c.grouppanel = '003' THEN 'Aneka'
          WHEN c.grouppanel = '004' THEN 'Marine Cargo'
          WHEN c.grouppanel = '005' THEN 'Travel'
          WHEN c.grouppanel = '006' THEN 'Fire'
          ELSE c.grouppanel
       END                                   AS BUSINESS_NAME,
       c.sobname                             AS BUSINESS_SOURCE,
       ROUND(COALESCE(e.reserves, 0) * (c.shareasm / 100) * 100)
                                             AS ESTIMATION_VALUE
  FROM POOLDATA.T_CLAIM_PNC c
       JOIN POOLDATA.T_CLAIMLIST_ADMIN w
            ON c.claimid = w.pzinskey
           AND w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
           AND w.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
       LEFT JOIN (  SELECT claimid,
                           SUM(estimationvalue * COALESCE(kursvalue, 1)) AS reserves
                      FROM POOLDATA.T_CLAIM_ESTIMASI
                  GROUP BY claimid) e
            ON e.claimid = c.claimid
 WHERE c.registerdate IS NOT NULL
   AND c.branchcode = :1
 ORDER BY c.registerdate ASC

-- name: summary_treaty_or
-- Total porsi treaty OR seluruh klaim outstanding cabang, dalam satuan terkecil.
--
-- ### Rumus dan penyaringnya disalin PERSIS dari kueri ekspor Pega
--
-- Termasuk `no_klaim LIKE 'PNC-%'` dan pemilihan satu baris per klaim menurut `no_spk`
-- terbesar — keduanya ada di `RDB List/GetDataOutstandingperCabangExport-SQL.xml`, bukan
-- karangan di sini. Dengan begitu angka kartu sama dengan jumlah kolom OR pada berkas ekspor.
--
-- ### Hasilnya akan NOL di hampir semua cabang, dan itu bukan cacat
--
-- `treaty_loss@asmd` memuat 10.152 baris, tetapi hanya **19** berawalan `PNC-` (17 klaim, 12
-- ber-OR bukan nol); sisanya `CLM-` 7.611, `KC72` 2.017, dan seterusnya. Dari 469 klaim
-- outstanding, hanya **satu cabang** yang total OR-nya bukan nol.
--
-- Pega menghitungnya dengan cara yang sama persis, sehingga kolom OR pada berkas ekspornya pun
-- sudah nol selama ini. Angkanya dibawa apa adanya (`P-5`) dan kekosongannya dinyatakan di
-- layar — bukan disamarkan menjadi tanda hubung.
--
-- ### Satu-satunya kueri modul ini yang menyentuh DB Link saat layar dibuka
--
-- Terpisah supaya `@asmd` yang padam mengosongkan SATU kartu, bukan seluruh panel.
--
-- Bind: :1 kode cabang
SELECT ROUND(COALESCE(SUM(COALESCE(e.reserves, 0)
                          * (c.shareasm / 100)
                          * COALESCE(tl.claim_or, 0)), 0) * 100) AS TREATY_OR
  FROM POOLDATA.T_CLAIM_PNC c
       JOIN POOLDATA.T_CLAIMLIST_ADMIN w
            ON c.claimid = w.pzinskey
           AND w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
           AND w.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'
       LEFT JOIN (  SELECT claimid, SUM(estimationvalue) AS reserves
                      FROM POOLDATA.T_CLAIM_ESTIMASI
                  GROUP BY claimid) e
            ON e.claimid = c.claimid
       LEFT JOIN (SELECT no_klaim, claim_or
                    FROM (SELECT tr.no_klaim,
                                 tr.claim_or,
                                 ROW_NUMBER() OVER (PARTITION BY tr.no_klaim
                                                        ORDER BY tr.no_spk DESC) rn
                            FROM treaty_loss@asmd.sinarmas.co.id tr
                           WHERE tr.no_klaim LIKE 'PNC-%')
                   WHERE rn = 1) tl
            ON tl.no_klaim = c.claimno
 WHERE c.registerdate IS NOT NULL
   AND c.branchcode = :1
