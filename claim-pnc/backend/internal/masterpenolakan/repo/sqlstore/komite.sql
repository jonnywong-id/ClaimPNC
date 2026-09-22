-- Kueri tabel POOLDATA.MST_REJECTED_KOMITE — Master Penolakan Komite.
--
-- Tabel ini tidak sekerabat dengan kedua tabel di masterpenolakan.sql: kuncinya bertipe
-- angka, namanya tidak sepola, dan tidak ada satu pun kolom yang menghubungkannya. Yang
-- menyatukan keduanya hanyalah satu layar Pega dan satu butir menu.
--
-- Aturan yang mengikat berkas ini sama dengan masterpenolakan.sql. Satu yang perlu
-- ditegaskan ulang: TIDAK ADA DELETE — seluruh export tidak memuat satu pun DELETE
-- terhadap tabel ini, dan layar lama hanya punya tombol "ubah".

-- name: committee_rejection_list
--
-- Asal: RDB List/GetMasterRejectedKomites-SQL.xml
--
--   select IDMASTER as "IDMaster", NOTEMASTER as "NoteKasir", 'ubah' as "NOKTP"
--     from POOLDATA.MST_REJECTED_KOMITE order by IDMASTER
--
-- Kolom ketiga TIDAK dibawa. Ia bukan data melainkan label tombol yang ditumpangkan ke
-- hasil kueri supaya grid Pega punya teks untuk digambar — dan aliasnya `NOKTP` mengaku
-- sebagai nomor KTP. Tombolnya tetap ada di layar baru; yang tidak ikut hanyalah cara
-- basis data dipakai untuk mengirim labelnya.
--
-- ORDER BY IDMASTER dipertahankan apa adanya. Kolomnya bertipe NUMBER
-- (`Database/INSERTMASTERREJECTEDKOMITE.prc:1` mendeklarasikan parameternya `in number`),
-- sehingga pengurutannya numerik — berbeda dari kedua tabel di masterpenolakan.sql, yang
-- kuncinya bertipe teks dan karena itu terurut secara leksikografis.
SELECT IDMASTER,
       NOTEMASTER
  FROM POOLDATA.MST_REJECTED_KOMITE
 ORDER BY IDMASTER

-- name: committee_rejection_get
--
-- Membaca satu baris, dipakai sebelum memperbarui supaya "baris tidak ada" dapat
-- dibedakan dari "baris ada tetapi nilainya sama persis".
--
-- TIDAK ADA ASALNYA DI EXPORT: layar lama memuat modalnya dari baris grid yang sudah ada
-- di klipboard, bukan dengan membaca ulang. Membaca ulang lebih benar — baris dapat
-- berubah di antara pemuatan daftar dan penyimpanan — dan tidak mengubah apa yang
-- tersimpan.
--
-- TANPA TRIM, berbeda dari kueri kunci di masterpenolakan.sql: kolomnya NUMBER, sehingga
-- tidak ada pemadatan spasi yang perlu ditangani. Pemangkasan spasi pada nilai yang
-- masuk tetap dilakukan di Go sebelum sampai ke sini.
SELECT IDMASTER,
       NOTEMASTER
  FROM POOLDATA.MST_REJECTED_KOMITE
 WHERE IDMASTER = :1

-- name: committee_rejection_list_id_locked
--
-- Mengunci seluruh baris, lalu mengembalikan ID-nya untuk menurunkan nomor berikutnya.
--
-- Procedure lama membaca `max(IDMASTER)+1` (`INSERTMASTERREJECTEDKOMITE.prc:11`) sebagai
-- pernyataan lepas, lalu menyisipkan beberapa baris kemudian — dua penambahan bersamaan
-- karena itu dapat menerima nomor yang sama. FOR UPDATE membuat yang kedua menunggu lalu
-- membaca ulang.
SELECT IDMASTER
  FROM POOLDATA.MST_REJECTED_KOMITE
 FOR UPDATE

-- name: committee_rejection_insert
--
-- Asal: Database/INSERTMASTERREJECTEDKOMITE.prc:15
--
--   INSERT INTO POOLDATA.MST_REJECTED_KOMITE a (A.IDMASTER,A.NOTEMASTER)
--   values(count_data,tmasternote);
--
-- Kedua kolom itu memang seluruh isi tabelnya. Tidak ada kolom pencatat siapa dan kapan,
-- sehingga perubahan pada master ini tidak meninggalkan jejak sama sekali — keterbatasan
-- yang dicatat, bukan ditambal dengan kolom yang dikarang (D-63).
INSERT INTO POOLDATA.MST_REJECTED_KOMITE (IDMASTER, NOTEMASTER)
VALUES (:1, :2)

-- name: committee_rejection_update
--
-- Asal: Database/INSERTMASTERREJECTEDKOMITE.prc:19
--
--   UPDATE POOLDATA.MST_REJECTED_KOMITE set NOTEMASTER=tmasternote where IDMASTER=tIDMASTER;
--
-- Hanya NOTEMASTER yang di-SET. IDMASTER adalah kunci dan hanya dipakai sebagai penyaring,
-- persis seperti procedure lama.
UPDATE POOLDATA.MST_REJECTED_KOMITE
   SET NOTEMASTER = :1
 WHERE IDMASTER = :2

-- name: committee_rejection_check_table
--
-- Memastikan tabel ada dan dapat dibaca akun aplikasi, tanpa mengambil satu baris pun.
SELECT IDMASTER
  FROM POOLDATA.MST_REJECTED_KOMITE
 WHERE 1 = 0
