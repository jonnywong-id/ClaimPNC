# Migrasi 0004 dicabut — 2026-09-24

Berkas `0004_login_line_business.up.sql` dan `.down.sql` **dihapus tanpa pernah dijalankan**.

## Sebabnya satu, dan sudah cukup: nama kolomnya salah

Migrasi itu menambahkan `LINEBUSINESS`. Kolom yang sebenarnya ada di
`POOLDATA.M_LOGIN_PNC` bernama **`LINE_BUSINESS`** — dengan garis bawah, `VARCHAR2`,
terverifikasi dari `ALL_TAB_COLUMNS`. Menjalankannya akan menghasilkan **dua kolom berbeda
nama untuk satu hal yang sama**, dan kolom yang benar sudah ada sehingga tidak ada yang
perlu ditambahkan.

## Koreksi atas sebab kedua yang pernah ditulis di sini

Berkas ini semula menyebut sebab kedua: *"modul yang membutuhkannya ternyata tidak
membutuhkannya"*, karena `InboxRegister_RD` memperlakukan panel sebagai parameter dan bukan
sebagai pita tetap per pengguna.

**Pernyataan itu benar untuk DAFTAR, dan salah bila dibaca sebagai berlaku untuk seluruh
modul.** `RDB List/ExportDataDetailKlaim-SQL.xml` — jalur **unduhan** — tidak menyaring
operator sama sekali, dan yang membatasinya justru **lini bisnis petugas**, dipilih
`Activity/ExportDataDetailKlaim-Act.xml` dari `OperatorID.pyPosition`.

Jadi `LINE_BUSINESS` **dipakai modul ini**, hanya bukan oleh layarnya. Pembacaannya hidup
di `repo/sqlstore/outstanding.sql` sebagai kueri `line_business_for`, dan dirinci di
`docs/keputusan-implementasi.md` §40.

Pencabutan migrasinya tetap benar — bukan karena kolomnya tidak dibutuhkan, melainkan
karena **kolomnya sudah ada dengan nama yang berbeda**.

## Yang ikut dihapus, dan yang kembali

Dihapus dan tidak dihidupkan kembali: `repo/sqlstore/linebusiness.sql` dan `.go` — keduanya
memakai nama kolom yang salah.

Kembali dalam bentuk lain, khusus untuk unduhan: tipe `LineBusiness` beserta kelima
nilainya pada domain, dan method `Repo.LineBusinessFor`. Yang **tidak** kembali adalah
`LineScope` dan `ScopeFor` — keduanya membatasi DAFTAR, dan daftar memang tidak dibatasi
lini bisnis.

## Keadaan datanya hari ini

`LINE_BUSINESS` terisi pada **1 dari 1 baris** `M_LOGIN_PNC` (`NONMBU`). Petugas tanpa
nilainya mengunduh **cakupan penuh** — perilaku Pega yang dipertahankan (`P-5`), dan yang
memperbaikinya adalah mengisi kolomnya, bukan mengubah kode.
