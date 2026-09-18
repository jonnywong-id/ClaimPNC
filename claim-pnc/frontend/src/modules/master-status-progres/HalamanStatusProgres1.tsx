// import { useState } from 'react'

// import { GalatAPI, GalatJaringan } from '@/api/klien'
// import { KodeGalat, type StatusProgres } from '@/api/tipe'
// import { PesanGalat, type NadaGalat } from '@/components/PesanGalat'
// import { TabelData, type Kolom } from '@/components/TabelData'
// import { Tombol } from '@/components/Tombol'
// import { gunakanPortalTerpilih } from '@/app/portal'

// import {
//   gunakanDaftarPosisiKlaim,
//   gunakanDaftarStatusProgres,
//   gunakanTambahStatusProgres,
//   gunakanUbahStatusProgres,
// } from './api'
// import { FormStatusProgres, type IsianForm } from './FormStatusProgres'

// /** Tidak ada form yang terbuka. */
// const TERTUTUP = 'tertutup'
// /** Form terbuka dalam mode tambah. */
// const TAMBAH = 'tambah'

// type KeadaanForm = typeof TERTUTUP | typeof TAMBAH | StatusProgres

// type IsiPesan = { judul: string; keterangan: string; nada: NadaGalat }

// function pesanMuat(galat: unknown): IsiPesan {
//   if (galat instanceof GalatJaringan) {
//     return {
//       judul: 'Server Claim PNC tidak dapat dihubungi',
//       keterangan: 'Periksa koneksi jaringan Anda, lalu muat ulang.',
//       nada: 'gangguan',
//     }
//   }
//   if (galat instanceof GalatAPI) {
//     switch (galat.kode) {
//       case KodeGalat.portalTidakDisebut:
//       case KodeGalat.portalTidakDikenal:
//         return {
//           judul: 'Portal entitas belum dipilih',
//           keterangan:
//             'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
//           nada: 'penolakan',
//         }
//       case KodeGalat.portalBelumSiap:
//         return {
//           judul: 'Basis data entitas ini belum tersedia',
//           keterangan:
//             'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.',
//           nada: 'gangguan',
//         }
//       default:
//         return {
//           judul: 'Daftar tidak dapat dimuat',
//           keterangan: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
//           nada: 'gangguan',
//         }
//     }
//   }
//   return {
//     judul: 'Daftar tidak dapat dimuat',
//     keterangan: 'Coba beberapa saat lagi.',
//     nada: 'gangguan',
//   }
// }

// /**
//  * Layar Master Status Progres 1.
//  *
//  * Pengganti `Harness/StatusProgress-Harness.xml` atas tabel
//  * POOLDATA.GCNM_MST_PROGRESS_KLAIM. Judul, susunan kolom, dan kedua tombolnya mengikuti
//  * layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna tidak perlu belajar
//  * ulang):
//  *
//  *   - Judul "Master Status Progres 1" — `Section/MasterStatusProgress-Section.xml`
//  *   - Tombol "Tambah" dan "Refresh"  — section yang sama
//  *   - Grid tiga kolom: ID, Status Progres, Posisi — `BrowseStatusProgress-Section.xml`
//  *
//  * Yang SENGAJA tidak ada: tombol hapus. Sistem lama tidak punya satu pun pernyataan
//  * DELETE terhadap tabel ini — sudah diperiksa ke seluruh export — dan tabelnya pun tidak
//  * punya kolom penanda terhapus yang dapat dipakai `D-66`. Menambahkannya berarti
//  * mengarang perilaku yang tidak pernah ada, sekaligus berisiko: baris ini dirujuk
//  * `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1` pada data klaim yang sudah berjalan.
//  */
// export function HalamanStatusProgres1() {
//   const portal = gunakanPortalTerpilih((keadaan) => keadaan.alias)
//   const [form, setForm] = useState<KeadaanForm>(TERTUTUP)

//   const daftar = gunakanDaftarStatusProgres()
//   const posisi = gunakanDaftarPosisiKlaim()
//   const tambah = gunakanTambahStatusProgres()
//   const ubah = gunakanUbahStatusProgres()

//   const disunting = typeof form === 'string' ? null : form
//   const sedangMenyimpan = tambah.isPending || ubah.isPending
//   const galatSimpan = disunting ? ubah.error : tambah.error

//   function bukaTambah() {
//     tambah.reset()
//     ubah.reset()
//     setForm(TAMBAH)
//   }

//   function bukaUbah(baris: StatusProgres) {
//     tambah.reset()
//     ubah.reset()
//     setForm(baris)
//   }

//   function tutupForm() {
//     tambah.reset()
//     ubah.reset()
//     setForm(TERTUTUP)
//   }

//   function simpan(isian: IsianForm) {
//     // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
//     // membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
//     // diketik, itu berarti mengetik ulang dari awal.
//     if (disunting) {
//       ubah.mutate({ id: disunting.id, isian }, { onSuccess: tutupForm })
//       return
//     }
//     tambah.mutate(isian, { onSuccess: tutupForm })
//   }

//   // `nilai` dipisah dari `tampil` mengikuti kontrak Kolom: yang dicari dan diurutkan
//   // adalah teks polos, yang dilihat pengguna boleh berisi markup. Menyatukannya akan
//   // membuat pencarian ikut menelusuri kelas CSS.
//   const kolom: Kolom<StatusProgres>[] = [
//     { kunci: 'id', judul: 'ID', lebar: 'w-20', nilai: (b) => b.id },
//     { kunci: 'nama', judul: 'Status Progres', nilai: (b) => b.nama },
//     {
//       kunci: 'posisi',
//       judul: 'Posisi',
//       lebar: 'w-40',
//       // Label posisi yang ditampilkan; kodenya ikut disebut karena itulah yang
//       // tersimpan di kolom STATUS dan yang dipakai saat menelusuri data. Keduanya ikut
//       // ke `nilai` supaya pencarian menemukan baris lewat kode maupun lewat labelnya.
//       nilai: (b) => `${b.nama_posisi} ${b.kode_posisi}`,
//       tampil: (b) => (
//         <span>
//           {b.nama_posisi}
//           <span className="ml-2 text-xs text-slate-500">{b.kode_posisi}</span>
//         </span>
//       ),
//     },
//     {
//       kunci: 'aksi',
//       judul: 'Aksi',
//       lebar: 'w-24',
//       // Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya
//       // tombol, bukan data.
//       tanpaUrut: true,
//       keKanan: true,
//       nilai: () => '',
//       tampil: (b) => (
//         <Tombol nada="kedua" onClick={() => bukaUbah(b)} aria-label={`Ubah ${b.nama}`}>
//           Ubah
//         </Tombol>
//       ),
//     },
//   ]

//   return (
//     <main className="mx-auto max-w-5xl px-4 py-8">
//       {/* Tidak ada tautan "kembali ke beranda" di sini: menu utama di kerangka sudah
//           menyediakannya, dan dua jalan ke tempat yang sama pada satu layar membuat
//           pengguna menebak mana yang dimaksud. */}
//       <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
//         <div>
//           <h1 className="text-xl font-semibold text-slate-900">Master Status Progres 1</h1>
//           <p className="text-sm text-slate-600">
//             Daftar status progres yang dapat dicatat petugas pada setiap posisi klaim.
//           </p>
//         </div>
//         <div className="flex flex-wrap items-center gap-2">
//           <Tombol
//             nada="kedua"
//             onClick={() => void daftar.refetch()}
//             disabled={daftar.isFetching}
//           >
//             {daftar.isFetching ? 'Memuat…' : 'Refresh'}
//           </Tombol>
//           <Tombol nada="utama" onClick={bukaTambah} disabled={form !== TERTUTUP}>
//             Tambah
//           </Tombol>
//         </div>
//       </header>

//       {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
//           empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
//           hanya diandaikan pengguna (ADR-0030, R-20). */}
//       <p className="mt-3 text-xs text-slate-500">
//         Portal entitas:{' '}
//         <span className="font-medium text-slate-700">{daftar.data?.portal ?? portal ?? '—'}</span>
//       </p>

//       {form !== TERTUTUP && (
//         <section className="mt-5">
//           <FormStatusProgres
//             disunting={disunting}
//             posisi={posisi.data?.posisi ?? []}
//             sedangMenyimpan={sedangMenyimpan}
//             galat={galatSimpan}
//             onSimpan={simpan}
//             onBatal={tutupForm}
//           />
//         </section>
//       )}

//       <section className="mt-6">
//         {portal === null ? (
//           <PesanGalat
//             judul="Portal entitas belum dipilih"
//             keterangan="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
//             nada="penolakan"
//           />
//         ) : daftar.isPending ? (
//           <p className="text-sm text-slate-500">Memuat daftar status progres…</p>
//         ) : daftar.isError ? (
//           (() => {
//             const pesan = pesanMuat(daftar.error)
//             return (
//               <PesanGalat
//                 judul={pesan.judul}
//                 keterangan={pesan.keterangan}
//                 nada={pesan.nada}
//               />
//             )
//           })()
//         ) : (
//           <TabelData
//             kolom={kolom}
//             baris={daftar.data.status_progres}
//             kunciBaris={(b) => b.id}
//             keterangan="Sumber: POOLDATA.GCNM_MST_PROGRESS_KLAIM"
//             pesanKosong="Belum ada status progres pada entitas ini."
//           />
//         )}
//       </section>
//     </main>
//   )
// }
