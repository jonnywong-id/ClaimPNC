-- Keanggotaan grup pengguna — POOLDATA.M_LOGIN_GROUP_PNC (LOGIN_ID, GROUP_ID). Dibaca saja.
--
-- Dicocokkan UPPER(TRIM(...)) di kedua sisi seperti modul menu: LOGIN_ID bertipe VARCHAR2
-- tanpa penyeragaman huruf, dan satu spasi di ujung akan menghilangkan seluruh peran.

-- name: grup_login
SELECT g.GROUP_ID
  FROM POOLDATA.M_LOGIN_GROUP_PNC g
 WHERE UPPER(TRIM(g.LOGIN_ID)) = UPPER(TRIM(:1))
 ORDER BY g.GROUP_ID
