/** Satu butir menu yang dapat diklik. */
export type ButirMenu = {
  label: string
  jalur: string
}

/** Sekelompok butir menu di bawah satu judul. */
export type KelompokMenu = {
  /** Judul kelompok; null berarti butirnya tampil tanpa judul di atasnya. */
  judul: string | null
  butir: ButirMenu[]
}

/**
 * Peta menu aplikasi.
 *
 * # Kenapa ia data, bukan JSX yang ditulis tangan di komponen
 *
 * Menambah modul berarti menambah SATU baris di sini — sama seperti di backend, tempat
 * modul baru cukup menambah satu pemanggilan `Pasang(...)` di `cmd/claimpnc`. Kalau
 * menunya ditulis sebagai JSX, setiap modul baru menuntut menyunting tata letak, dan
 * pada 74 layar itu berubah menjadi tata letak yang berbeda-beda.
 *
 * # Yang BELUM ada di sini, dan harus disebut terang
 *
 * Daftar ini **tetap**, belum disaring izin peran. Menu yang benar mengikuti izin: 22
 * peran dan 51 item menu (`TKT-F3-004`), dan peta peran → menu di sistem lama hidup di
 * 34 When rule — lima di antaranya **hilang dari export**. Ditambah penghalang yang
 * lebih mendasar: penugasan operator ke peran **tidak ada di basis data**
 * (`POOLDATA.T_ACCESS_GROUP_PNC` hanya memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`),
 * sehingga tabel izinnya dapat dibangun tetapi belum dapat diisi.
 *
 * Karena itu menu ini **bukan kendali akses**. Yang menjadi kendali adalah pemeriksaan
 * di server pada setiap endpoint (`D-59`) — menyembunyikan menu hanyalah kenyamanan
 * tampilan, dan itulah justru cacat sistem lama yang tidak boleh diulang: di sana
 * `pyPrivilegeName` terisi pada 1 dari 902 activity.
 *
 * Begitu `TKT-F3-004` tersedia, yang berubah adalah penyaringan daftar ini terhadap izin
 * pengguna — bentuk datanya tidak perlu berubah.
 */
export const menuUtama: KelompokMenu[] = [
  {
    judul: null,
    butir: [{ label: 'Beranda', jalur: '/' }],
  },
  {
    judul: 'Master Data',
    butir: [{ label: 'Status Progres 1', jalur: '/master/status-progres-1' }],
  },
]
