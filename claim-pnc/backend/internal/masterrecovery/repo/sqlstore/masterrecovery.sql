-- Kueri Master Recovery.
--
-- Empat objek yang disentuh, seluruhnya milik POOLDATA dan seluruhnya sudah ada — tidak
-- ada satu pun migrasi yang dibutuhkan modul ini:
--
--     MST_RECOVERY_ASM_PENJAMINAN   batch recovery; PK BATCH
--     MST_VIRTUAL_ACCOUNT_PNC       master principal beserta rekening virtualnya
--     DATA_ATTACHFILE               Bukti Bayar; isinya di kolom BLOB ATTACHFILE
--     C_COUNTER_ATTACHMENT          penerbit DATAID lampiran
--
-- Bentuk seluruh kolom diverifikasi langsung ke ALL_TAB_COLUMNS portal ASM pada
-- 2026-09-19.
--
-- # Kenapa tidak lagi lewat procedure
--
-- `D-02` menetapkan logika stored procedure naik ke Go dan aplikasi tidak memanggil
-- procedure. Pada modul ini alasannya bukan sekadar kepatuhan — ketiga procedure yang
-- digantikan punya cacat yang tidak dapat dibawa:
--
--   * INSERTMASTERRECOVERYKLAIM.prc  melewati INSERT tanpa pesan apa pun bila BATCH sudah
--                                    ada, sehingga petugas melihat "berhasil" atas data
--                                    yang tidak pernah tersimpan.
--   * ADD_NEWMASTERVIRTUALACCOUNT.prc mengembalikan NOMOR VA lewat parameter bernama
--                                    ErrMsg — satu kolom untuk nilai berhasil dan pesan
--                                    galat sekaligus (`D-68`).
--   * SET_ATTACHMENT_64BIT.prc       menjalankan COMMIT sendiri, dan ROLLBACK-nya berada
--                                    sesudah commit itu sehingga tidak memulihkan apa pun.
--
-- # Aturan yang mengikat berkas ini
--
-- Kolom selalu disebut namanya; SELECT * dilarang. Nilai selalu lewat parameter binding —
-- dan di modul ini aturan itu menutup celah yang benar-benar ada, yaitu
-- `{ASIS:GeneratedVARecovery.NoteKomite}` pada `RDB List/GetDataVAbyValidasiVA-SQL.xml`,
-- yang merangkai nama principal ketikan pengguna langsung ke dalam klausa WHERE.
--
-- TO_CHAR, LPAD, NVL, SYSDATE, dan ROWNUM tidak dipakai; pemformatan dan pembentukan
-- nomor dikerjakan di Go (`09-DATABASE-STRATEGY.md` §4).

-- name: recovery_next_batch
--
-- Meniru `RDB List/GetMasterRecoveryClaimSPK-SQL.xml` persis, termasuk perlakuan tabel
-- kosong yang menghasilkan 1.
--
-- COALESCE, bukan NVL: keduanya sama artinya, tetapi COALESCE berlaku di Oracle maupun
-- PostgreSQL sehingga kueri ini tidak perlu ditulis dua kali saat pindah (`D-20`).
--
-- Dipakai di DUA tempat dengan arti berbeda, dan perbedaannya penting: di layar ia
-- PERKIRAAN yang ditampilkan, di dalam transaksi penyisipan ia nomor yang benar-benar
-- dipakai. Yang kedua dijalankan setelah baris terkunci, sehingga dua penyimpanan
-- bersamaan tidak dapat memperebutkan nomor yang sama — pengaman yang tidak ada sama
-- sekali di sistem lama.
SELECT COALESCE(MAX(BATCH), 0) + 1
  FROM POOLDATA.MST_RECOVERY_ASM_PENJAMINAN

-- name: recovery_lock_table
--
-- Mengunci tabel selama penerbitan nomor batch berlangsung.
--
-- # Kenapa mengunci TABEL, bukan baris
--
-- Yang dilindungi adalah `MAX(BATCH) + 1`, dan nilai itu tidak melekat pada satu baris
-- mana pun — ia sifat seluruh isi tabel. Mengunci baris tidak menolong: dua sesi yang
-- sama-sama membaca MAX pada saat tabel belum berubah akan memperoleh angka yang sama,
-- lalu yang kedua gagal pada kunci utama.
--
-- MODE EXCLUSIVE dipilih dan bukan SHARE karena dua penulis harus benar-benar bergantian.
-- Biayanya dapat diterima: tabel ini menerima beberapa baris per bulan, bukan per detik,
-- dan kuncinya dilepas begitu transaksi berakhir — sepersekian detik kemudian.
--
-- Sistem lama tidak punya pengaman ini sama sekali. Dua petugas yang menyimpan bersamaan
-- akan memperoleh BATCH yang sama, dan procedure-nya lalu MELEWATI penyisipan kedua tanpa
-- satu pun pesan.
LOCK TABLE POOLDATA.MST_RECOVERY_ASM_PENJAMINAN IN EXCLUSIVE MODE

-- name: recovery_insert
--
-- Kedua puluh kolom yang benar-benar diisi, sama persis dengan yang diisi
-- `INSERTMASTERRECOVERYKLAIM.prc`.
--
-- INSERTDATE sengaja TIDAK disebut: kolomnya ber-DEFAULT sysdate, diverifikasi dari
-- ALL_TAB_COLUMNS. Menuliskannya dari aplikasi akan memakai jam server aplikasi alih-alih
-- jam basis data, dan pada dua instans di belakang penyeimbang beban keduanya belum tentu
-- sama (`R-12`).
--
-- Sepuluh kolom lain pada tabel ini — AGENTID, BRANCHID, BUSINESSID, CLAIM_AMOUNT_ADJUST,
-- COVERAGEID, IBNR, MARKETINGID, NILAI_DEDUCTIBLE, NO_KTP_MASKING — juga tidak diisi.
-- Procedure lama pun tidak mengisinya, dan ketiga baris yang ada seluruhnya NULL di sana.
-- Mengisinya berarti mengarang arti yang tidak pernah ditetapkan siapa pun.
INSERT INTO POOLDATA.MST_RECOVERY_ASM_PENJAMINAN (
  BATCH, NAMAPRINCIPAL, TAHUN,
  NILAIKLAIM, NILAIRECOVERY, PEMBAYARAN, SISAKLAIM,
  KETERANGAN, POSISIKASUS, DOKUMENID, NOVA, USERNAME, CLIENTID,
  JSON_POLIS, NOHPLL, NOPOLIS,
  LBU_ID, LDC_ID, LAG_AGEN_ID, LMO_ID
) VALUES (
  :1, :2, :3,
  :4, :5, :6, :7,
  :8, :9, :10, :11, :12, :13,
  :14, :15, :16,
  :17, :18, :19, :20
)

-- name: recovery_check_table
--
-- Memastikan kolom yang dipakai modul ini ada dan dapat dibaca akun aplikasi, tanpa
-- mengambil satu baris pun. Aman dijalankan terhadap produksi, dan dipakai untuk
-- membedakan dua sebab kegagalan yang tampak mirip: tabelnya tidak ada di portal itu,
-- versus tidak punya hak baca atas tabel warisan.
SELECT BATCH, NAMAPRINCIPAL, TAHUN, NILAIKLAIM, NILAIRECOVERY,
       PEMBAYARAN, SISAKLAIM, KETERANGAN, POSISIKASUS, DOKUMENID,
       NOVA, USERNAME, CLIENTID, NOHPLL, NOPOLIS
  FROM POOLDATA.MST_RECOVERY_ASM_PENJAMINAN
 WHERE 1 = 0

-- name: principal_list
--
-- Mengisi pilihan "Nama Principal". Menggantikan pra-aktivitas `getDataAllMSTVA`.
--
-- Diurutkan menurut nama supaya pilihan yang dilihat petugas tidak berpindah-pindah
-- urutan; sistem lama tidak mengurutkannya sama sekali.
SELECT CLIENTID, CLIENTNAME, VIRTUALACCOUNTNUMBER, EMAILVA, STATUS, MESSAGE
  FROM POOLDATA.MST_VIRTUAL_ACCOUNT_PNC
 ORDER BY CLIENTNAME

-- name: principal_find
--
-- Mencari principal menurut Client ID DAN nama, tanpa membedakan besar-kecil huruf.
--
-- UPPER dipakai pada kedua sisi, meniru `ADD_NEWMASTERVIRTUALACCOUNT.prc` persis.
-- Menirunya penting: bila pencocokan di sini lebih ketat daripada di sana, aplikasi akan
-- menerbitkan VA kedua untuk principal yang menurut basis data sudah punya — dan dana
-- masuk ke rekening yang tidak diawasi siapa pun.
--
-- Nilainya lewat parameter binding, menggantikan perangkaian teks `{ASIS:...}` pada
-- kueri lama.
SELECT CLIENTID, CLIENTNAME, VIRTUALACCOUNTNUMBER, EMAILVA, STATUS, MESSAGE
  FROM POOLDATA.MST_VIRTUAL_ACCOUNT_PNC
 WHERE UPPER(TRIM(CLIENTID)) = UPPER(TRIM(:1))
   AND UPPER(TRIM(CLIENTNAME)) = UPPER(TRIM(:2))

-- name: principal_insert
--
-- Mencatat principal beserta VA yang baru diterbitkan.
--
-- Tidak ada cabang "bila sudah ada maka lewati" seperti pada procedure lama: pemeriksaan
-- itu sudah dikerjakan usecase sebelum layanan penerbit ditembak, dan menggandakannya di
-- sini hanya membuat dua tempat memutuskan hal yang sama.
INSERT INTO POOLDATA.MST_VIRTUAL_ACCOUNT_PNC (
  CLIENTID, CLIENTNAME, STATUS, MESSAGE, VIRTUALACCOUNTNUMBER, EMAILVA
) VALUES (:1, :2, :3, :4, :5, :6)

-- name: policy_reference
--
-- Mencari identitas lini bisnis, cabang, agen, dan marketing dari nomor polis.
--
-- Menggantikan `RDB List/GetRecoveryClaimData-SQL.xml` apa adanya, TERMASUK DB Link-nya.
--
-- # Utang yang disadari, bukan yang terlewat
--
-- `MST_DET_SALES@ASMD` adalah salah satu dari 64 pemakaian DB Link yang `D-25` tetapkan
-- diganti pemanggilan API — "API Master Sales" pada daftar enam API pengganti. API itu
-- BELUM ADA, dan yang membangunnya tim lain (`R-03`).
--
-- Selama masa paralel DB Link-nya masih hidup dan masih dibaca Pega, sehingga memakainya
-- di sini tidak menambah ketergantungan baru — ia meneruskan yang sudah ada. Ia sengaja
-- diisolasi sebagai SATU kueri bernama supaya penggantinya kelak menyentuh satu tempat
-- saja.
--
-- Kegagalannya diperlakukan sebagai "identitas tidak ditemukan", bukan sebagai kegagalan
-- penyimpanan — lihat alasannya di repo Go. Batch tetap tersimpan dengan keempat kolom
-- kosong, persis seperti sistem lama.
SELECT LBU_ID, LDC_ID, LAG_AGEN_ID, LMO_ID
  FROM MST_DET_SALES@ASMD.SINARMAS.CO.ID
 WHERE MDS_NO_POLIS = :1
 FETCH FIRST 1 ROW ONLY

-- name: attachment_next_sequence
--
-- Nomor urut lampiran. Urutan yang SAMA dengan yang dipakai procedure lama, supaya DATAID
-- yang diterbitkan aplikasi ini melanjutkan deret yang sudah ada dan tidak pernah
-- bertabrakan dengan yang pernah diterbitkan Pega.
--
-- FROM DUAL adalah satu-satunya bentuk khas Oracle di seluruh modul ini, dan ia tidak
-- terhindarkan: NEXTVAL menuntutnya. Ia sengaja diisolasi di kueri tersendiri —
-- perlakuannya sama dengan generator nomor klaim pada `ADR-0005`, satu-satunya tempat
-- lain yang dibenarkan memuat percabangan dialek.
SELECT POOLDATA.ATTACHFILE_SEQ.NEXTVAL
  FROM DUAL

-- name: attachment_counter_insert
--
-- Mencatat penerbitan DATAID, meniru `SET_ATTACHMENT_64BIT.prc` langkah tengahnya.
--
-- Tabel ini adalah jejak penerbitan nomor lampiran; ia ditulis meski nilainya juga dapat
-- dihitung dari urutan, karena Pega masih membacanya selama masa paralel.
--
-- KEY dibentuk di Go, bukan dengan `new_uuid` seperti procedure lama: fungsi itu milik
-- basis data dan mengikat kueri ini padanya tanpa alasan. Yang dibutuhkan hanyalah nilai
-- yang tidak pernah berulang, dan itu dapat dibuat di mana saja.
--
-- YEAR diisi dua digit tahun yang dibentuk di Go, menggantikan `to_char(sysdate,'yy')`.
INSERT INTO POOLDATA.C_COUNTER_ATTACHMENT (KEY, YEAR, RUNNO)
VALUES (:1, :2, :3)

-- name: attachment_insert
--
-- Menyimpan Bukti Bayar beserta isinya.
--
-- # Kenapa isinya masuk ke kolom BLOB, dan bukan ke API penyimpanan luar
--
-- `D-16` menetapkan dokumen disimpan lewat API storage internal, dan modul yang
-- membangunnya adalah `S-1` yang belum ada. Menunggunya akan membuat layar ini tidak
-- dapat menerima bukti bayar sama sekali.
--
-- Kolom ATTACHFILE bertipe BLOB dan memang disediakan untuk isi berkasnya — keberadaannya
-- diverifikasi ke ALL_TAB_COLUMNS pada 2026-09-19, dan procedure lama pun menyisipkan ke
-- tabel yang sama. Karena itu menyimpannya di sini bukan jalan pintas, melainkan memakai
-- jalur yang memang sudah ada.
--
-- IMAGEID sengaja DIKOSONGKAN. Ia penunjuk ke penyimpanan luar, dan mengisinya dengan
-- nilai karangan akan membuat pembaca mana pun mengira berkasnya ada di sana.
--
-- INPUTDATE tidak disebut: kolomnya TIMESTAMP dengan nilai dari basis data, sama alasannya
-- dengan INSERTDATE di atas.
INSERT INTO POOLDATA.DATA_ATTACHFILE (
  DATAID, ATTACHFILE, INPUTOPERATOR, ATTACHNAME, ATTACHNOTE,
  ATTACHMIMETYPE, CATEGORY, SUB_CATEGORY, IDPEGA
) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9)

-- name: attachment_check_table
--
-- Memastikan tabel lampiran dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT DATAID, ATTACHNAME, ATTACHMIMETYPE, CATEGORY
  FROM POOLDATA.DATA_ATTACHFILE
 WHERE 1 = 0

-- name: principal_check_table
--
-- Memastikan master Virtual Account dapat dibaca akun aplikasi.
SELECT CLIENTID, CLIENTNAME, VIRTUALACCOUNTNUMBER, EMAILVA
  FROM POOLDATA.MST_VIRTUAL_ACCOUNT_PNC
 WHERE 1 = 0
