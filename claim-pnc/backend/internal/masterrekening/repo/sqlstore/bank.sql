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
