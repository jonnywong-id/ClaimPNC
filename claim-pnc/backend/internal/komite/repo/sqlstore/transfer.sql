-- Kueri rincian "Lihat Detail Transfer" — pengganti `Section/ShowTransferDetail`.
--
-- ============================================================================
-- KEDUANYA HANYA MEMBACA.
-- ============================================================================
--
-- `POOLDATA.T_CLAIM_ADJUSTMENT` dan `POOLDATA.T_CLAIM_KOMITE_LIST` masih ditulis Pega, dan
-- `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem.
--
--
-- ============================================================================
-- KENAPA DATANYA DARI SINI, BUKAN DARI TEMPAT PEGA MEMBACANYA
-- ============================================================================
--
-- `ShowTransferDetail` membaca CLIPBOARD objek kerja, yang `OBJ-OPEN-BY-HANDLE` muat dari
-- BLOB Pega. Blob tidak dapat dibaca SQL.
--
-- Diverifikasi ke `ALL_TAB_COLUMNS`: dari 33 properti yang section itu tampilkan, hanya
-- ENAM yang benar-benar kolom `PC_ASM_FW_GCNMFW_WORK`. Dua puluh tujuh sisanya — termasuk
-- SELURUH nilai uangnya — tidak ada di sana.
--
-- Nilainya karena itu diambil dari tabel POOLDATA yang memuat data yang sama. Alasan
-- lengkapnya, beserta angka pengukurannya, ada di `internal/komite/transfer.go`.
--
--
-- ============================================================================
-- CASEIDKOMITE, BUKAN CLAIMID — DAN PERBEDAANNYA DIUKUR
-- ============================================================================
--
-- Keduanya ada di `T_CLAIM_ADJUSTMENT`, dan keduanya berarti hal yang BERBEDA. Diukur pada
-- dua belas case yang punya keduanya:
--
--     lewat CASEIDKOMITE   1 baris   pada dua belas-duanya
--     lewat CLAIMID        4–8 baris pada dua belas-duanya
--
-- Tidak satu pun sama. `CLAIMID` mengembalikan seluruh baris adjustment milik KLAIMNYA —
-- termasuk milik case komite lain. Memakainya berarti menaruh empat sampai delapan nilai
-- uang yang bukan urusan case ini pada layar tempat orang menyetujui uang.
--
-- `TestKueriTransferMemakaiKunciCaseKomite` menegakkan pilihan ini.
--
--
-- ============================================================================
-- GAYA SQL
-- ============================================================================
--
-- COALESCE bukan NVL · CASE WHEN bukan DECODE · kolom selalu disebut namanya · tanpa
-- TO_CHAR untuk tampilan. Mengikuti `09-DATABASE-STRATEGY.md` §4.


-- name: transfer_lines
--
-- Baris adjustment yang diputuskan satu case komite.
--
-- Diurutkan supaya hasilnya PASTI: dua baris pada case yang sama tidak boleh berpindah
-- tempat antar permintaan, karena pembacanya membandingkan angka uang antar baris.
SELECT j.CLAIMID,
       j.OBJECTID,
       j.OBJECTCOVERAGEID,
       j.NOAKSEPTASI,
       j.TGLAKSEPTASI,
       j.CURRENCY,
       j.PAYMENTTYPE,
       j.GROSSVALUE,
       j.PROPOSE_VALUE,
       j.NILAIAKSEPTASI,
       j.NILAI_SALVAGE_A,
       j.ASM_SHARE,
       j.ASM_SHARE_VALUE,
       j.INDIVIDUAL_RISK_VALUE,
       j.EXGRATIA,
       j.NOTES,
       j.CIRCUMCAUSEOFLOSS
  FROM POOLDATA.T_CLAIM_ADJUSTMENT j
 WHERE j.CASEIDKOMITE = :1
 ORDER BY j.OBJECTID, j.OBJECTCOVERAGEID, j.ADJUSTMENTID


-- name: transfer_committee
--
-- Keputusan komite MENURUT PEGA.
--
-- Diagregasi, bukan diambil apa adanya: `KOMITE_ID` tidak dijamin unik pada tabel berisi
-- 39 juta baris itu, dan dua baris untuk satu case akan membuat kueri ini mengembalikan
-- dua hasil pada tempat yang hanya menerima satu.
--
-- MAX dipakai seragam supaya "satu nilai per case" tidak bergantung pada baris mana yang
-- kebetulan dikembalikan lebih dulu.
SELECT MAX(k.NAMAKOMITE),
       MAX(k.KOMITEKE),
       MAX(k.TYPEKOMITE),
       MAX(k.PAYMENTTYPE),
       MAX(k.NOTEKOMITE),
       MAX(k.NILAIKLAIM),
       MAX(k.SHAREASM),
       MAX(k.TANGGALKOMITE),
       MAX(k.STATUSAPPROVE),
       COUNT(1)
  FROM POOLDATA.T_CLAIM_KOMITE_LIST k
 WHERE k.KOMITE_ID = :1


-- name: transfer_case
--
-- Dua medan pada BARIS KERJA case komite yang dibutuhkan JUDUL layar, bukan isinya.
--
-- `Section/ShowTransfer` memasang enam label bersyarat di sebelah "CLAIM COMMITTEE". Dua
-- di antaranya bersandar pada properti yang tidak ada di `T_CLAIM_KOMITE_LIST`:
--
--     IsTravel   GroupPanel = "005"                      -> GROUPPANEL_1
--     IsHE       BusinessType in ("HE","ContractorsPM")  -> BUSINESSTYPE
--
-- # Kenapa kueri tersendiri, bukan disatukan ke transfer_committee
--
-- Sudah dicoba sebagai subkueri skalar di dalam SELECT ber-agregat tanpa GROUP BY, dan
-- Oracle menolaknya: `ORA-00937: not a single-group group function`. Jadi pemisahan ini
-- bukan selera melainkan satu-satunya bentuk yang jalan.
--
-- Pemisahan itu sekaligus membuat kedua medan ini tetap terbaca pada case yang TIDAK punya
-- baris komite sama sekali — 218 dari 610 sejak 2024 berada dalam keadaan itu.
--
-- # Yang harus disadari tentang BUSINESSTYPE
--
-- Diukur pada basis data ASM: `GROUPPANEL_1` terisi 610 dari 610, `BUSINESSTYPE` terisi
-- **0 dari 610**. Kolomnya ada, tetapi Pega tidak menulisinya untuk kelas `Work-Komite`.
-- Ia tetap dibaca supaya cabang HE benar dengan sendirinya bila kolom itu kelak terisi,
-- dan supaya layar dapat MENYATAKAN bahwa cabang itu tidak dapat dinilai.
--
-- `MAX` dipakai supaya kueri ini selalu mengembalikan tepat satu baris, termasuk ketika
-- `PYID` tidak ditemukan sama sekali.
-- `PNCCASEID` diambil APA ADANYA, lengkap dengan prefix `ASM-FW-GCNMFW-WORK `.
--
-- Itu bukan kelalaian membersihkan: ia kunci join ke `T_CLAIM_PNC`, `T_CLAIM_OBJECTLIST`,
-- dan `T_CLAIM_OBJECTCOVERAGE`, yang ketiganya menyimpan `CLAIMID` BER-PREFIX. Memangkasnya
-- lebih dulu menghasilkan nol baris pada seluruh 189 case, tanpa satu pun galat.
--
-- Pemangkasannya terjadi di lapisan domain, setelah join selesai (`D-22`).
SELECT MAX(a.GROUPPANEL_1),
       MAX(a.BUSINESSTYPE),
       MAX(a.PNCCASEID)
  FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a
 WHERE a.PYID = :1
   AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'


-- name: transfer_claim
--
-- Data klaim yang `ShowTransferDetail` tampilkan lewat `.KomiteClaimData.*`.
--
-- Dikunci `CLAIMID` BER-PREFIX — lihat catatan pada `transfer_case`. Terbaca pada 189 dari
-- 189 case inbox; 189 case itu bermuara pada 72 klaim yang berbeda, karena satu klaim dapat
-- dikomitekan lebih dari sekali.
--
-- `MAX` dipakai supaya kueri ini selalu mengembalikan tepat satu baris, dan `COUNT(1)`
-- membedakan "tidak ada klaimnya" dari "ada tetapi kolomnya kosong".
SELECT MAX(p.DATEOFLOSS),
       MAX(p.REGISTERDATE),
       MAX(p.LOCATION),
       MAX(p.KRONOLOGI),
       MAX(p.STATUSCLAIM),
       MAX(p.REMARKRECOMENDATION),
       MAX(p.SHAREASM),
       MAX(p.COINSNAME),
       MAX(p.CURRENCY),
       MAX(p.EXGRATIA),
       COUNT(1)
  FROM POOLDATA.T_CLAIM_PNC p
 WHERE p.CLAIMID = :1


-- name: transfer_coverages
--
-- Blok analisis komite: di sinilah `.Komite.ExtentOfLoss`,
-- `.Komite.CircumtansesCouseOfLoss`, dan `.Komite.Remarks` sebenarnya tinggal.
--
-- Bukan di tabel kerja, dan bukan di `T_CLAIM_KOMITE_LIST` — keduanya sudah diperiksa.
--
-- # Tanpa batas baris, dan itu diukur lebih dulu
--
-- Satu case menghasilkan 1 sampai 3 baris, rerata 1,4; tidak satu pun melampaui tiga.
-- Memberi `FETCH NEXT` di sini hanya akan menyembunyikan baris ketiga tanpa alasan.
--
-- Baris yang sudah ditandai terhapus dibuang: `DIHAPUS_PADA` adalah penanda soft delete
-- pada tabel warisan ini, dan menampilkannya berarti menampilkan coverage yang sudah
-- dicabut dari klaimnya.
SELECT c.OBJECTID,
       c.OBJECTCOVERAGEID,
       c.OBJECTNAME,
       c.COVERAGENAME,
       c.CAUSEOFLOSS,
       c.SUMTSI,
       c.CURRENCY,
       c.CURICUMOFLOSS,
       c.EXTENTOFLOSS,
       c.LEGALLIABILITY,
       c.REMARKS,
       c.DIAGNOSE,
       c.INITIALNAME,
       c.TANGGALCOMITEE
  FROM POOLDATA.T_CLAIM_OBJECTCOVERAGE c
 WHERE c.CLAIMID = :1
   AND c.DIHAPUS_PADA IS NULL
 ORDER BY c.OBJECTID, c.OBJECTCOVERAGEID


-- name: transfer_check_table
--
-- Memastikan kedua tabel beserta kolomnya dapat dibaca akun aplikasi, tanpa mengambil satu
-- baris pun.
SELECT j.CASEIDKOMITE,
       j.CLAIMID,
       j.NILAIAKSEPTASI,
       j.ASM_SHARE_VALUE,
       j.CIRCUMCAUSEOFLOSS,
       k.KOMITE_ID,
       k.NAMAKOMITE,
       k.TYPEKOMITE,
       p.CLAIMID,
       p.KRONOLOGI,
       p.REMARKRECOMENDATION,
       c.CLAIMID,
       c.COVERAGENAME,
       c.EXTENTOFLOSS,
       c.DIHAPUS_PADA
  FROM POOLDATA.T_CLAIM_ADJUSTMENT j
  LEFT JOIN POOLDATA.T_CLAIM_KOMITE_LIST k
         ON k.KOMITE_ID = j.CASEIDKOMITE
  LEFT JOIN POOLDATA.T_CLAIM_PNC p
         ON p.CLAIMID = j.CLAIMID
  LEFT JOIN POOLDATA.T_CLAIM_OBJECTCOVERAGE c
         ON c.CLAIMID = j.CLAIMID
 WHERE 1 = 0
