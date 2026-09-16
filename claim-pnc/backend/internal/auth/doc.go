// Package auth adalah inti modul Identitas & Akses (`F-3`).
//
// # Isi paket ini
//
// Aturan dan tipe yang dimiliki modul — identitas pengguna, sesi, dan catatan pengguna
// lokal — beserta SEAM yang dideklarasikannya: `Identitas`, `PenggunaRepo`, dan
// `SesiRepo`. Antarmuka dideklarasikan di sini, di paket yang memakainya, dan dipenuhi
// subpaket di bawahnya.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul (`platform/waktu`). Aturan ini adalah bentuk
// modul dari arah ketergantungan `ADR-0001`: ketergantungan hanya mengarah ke dalam.
//
// # Susunan subpaket
//
//	auth/            aturan modul + seam          ← paket ini
//	auth/usecase/    orkestrasi masuk, periksa, perpanjang, keluar
//	auth/provider/   pengisi seam Identitas       — tiruan, dan kelak HCC/HCQ
//	auth/repo/       pengisi seam penyimpanan     — sqlstore, memori
//	auth/http/       lapisan transport modul ini  — handler, dto, middleware, rute
//
// Seluruh subpaket mengimpor paket ini; paket ini tidak mengimpor satu pun dari mereka.
//
// # Yang BUKAN urusan paket ini
//
// Otorisasi. Autentikasi didelegasikan ke sistem identitas luar (`ADR-0024`); kewenangan
// dimiliki aplikasi ini sendiri dan menjadi lingkup `TKT-F3-004` serta `TKT-F3-005`,
// yang keduanya belum dikerjakan.
package auth
