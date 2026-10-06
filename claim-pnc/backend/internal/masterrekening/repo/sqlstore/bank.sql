-- Kueri tabel GENERAL.LST_BANK_GROUP — daftar bank.
--
-- Tabel milik sistem lain; aplikasi ini hanya MEMBACA (ADR-0004, penulis tunggal per
-- tabel). Sumbernya sama dengan yang dipakai Report Definition BrowseBankGroup dan
-- RDB List SearchCodeBank_sql pada sistem lama.

-- name: bank_list
SELECT LBG_ID,
       BANK_GROUP
  FROM GENERAL.LST_BANK_GROUP
 ORDER BY BANK_GROUP

-- name: bank_list_pooldata
--
-- Sumber CADANGAN, dipakai hanya bila kueri di atas gagal karena objeknya tidak
-- terlihat (ORA-00942). Lihat BankRepo.List untuk kapan ia dijalankan.
--
-- # Kenapa ada dua
--
-- Pada basis data pengembangan `DEV_PEGA83G`, pengguna POOLDATA **tidak melihat satu pun
-- objek skema GENERAL** — `SELECT COUNT(*) FROM ALL_OBJECTS WHERE OWNER='GENERAL'`
-- mengembalikan nol baris, dan tidak ada sinonim bernama LST_BANK_GROUP. Yang tersedia
-- adalah view pendamping di skema POOLDATA sendiri, dengan **kolom yang sama persis**:
--
--	POOLDATA.VH_GENERAL_LST_BANK_GROUP  →  LBG_ID, BANK_GROUP, … (225 baris)
--
-- Namanya terbaca sebagai "view helper untuk GENERAL.LST_BANK_GROUP", dan isinya memang
-- daftar bank yang sama.
--
-- # Satu yang SENGAJA tidak disaring
--
-- View itu punya kolom STATUS, dan seluruh 225 barisnya bernilai `1`. Penyaringannya
-- tetap TIDAK dipasang, karena kueri utama pun tidak menyaring apa pun — begitu juga
-- `SearchCodeBank_sql` pada sistem lama. Menyaring di satu sumber saja akan membuat
-- daftar bank berbeda antara dua lingkungan, dan selisih itu baru ketahuan sebagai bank
-- yang hilang dari dropdown.
SELECT LBG_ID,
       BANK_GROUP
  FROM POOLDATA.VH_GENERAL_LST_BANK_GROUP
 ORDER BY BANK_GROUP
