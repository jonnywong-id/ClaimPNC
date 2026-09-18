import type { ReactNode } from 'react'

import { Tombol } from '@/components/Tombol'
import { gunakanKeluar } from '@/modules/masuk/api'
import { PemilihPortal } from '@/modules/portal/PemilihPortal'

import { NavigasiUtama } from './NavigasiUtama'
import { PeringatanSesi } from './PeringatanSesi'

type Props = {
  anak: ReactNode

  /**
   * Halaman sudah menyediakan pemilih portal dan tombol keluar di dalam dirinya sendiri,
   * sehingga kerangka tidak menampilkannya lagi.
   *
   * Ini penyesuaian sementara untuk Beranda. Beranda dibangun sebelum kerangka ini ada
   * dan memuat keduanya di header-nya sendiri; menampilkannya dua kali akan membingungkan
   * — dua pemilih portal pada satu layar tidak jelas mana yang berlaku.
   *
   * Keduanya seharusnya tinggal di kerangka saja, bukan di dalam halaman. Memindahkannya
   * menuntut menyunting `modules/beranda/HalamanBeranda.tsx`, dan modul Beranda
   * dinyatakan tidak boleh diubah (keputusan Work Owner 2026-09-17). Dicatat sebagai
   * utang teknis, bukan dikerjakan sepihak.
   */
  aksiDiHalaman?: boolean
}

/**
 * Kerangka adalah bingkai bersama seluruh layar di dalam sesi.
 *
 * Isinya tiga hal yang tidak boleh diulang di setiap layar: peringatan sesi hampir habis,
 * menu utama, dan aksi tingkat aplikasi (pemilih portal dan keluar).
 *
 * # Kenapa ia di app/, bukan di salah satu modul
 *
 * Bingkai ini milik kerangka, bukan milik satu modul mana pun. Menaruhnya di sebuah modul
 * akan membuat modul lain mengimpornya — dan aturan frontend melarang modul saling
 * mengimpor; kebutuhan bersama naik ke `app/`, `components/`, atau `api/`.
 *
 * # Kenapa peringatan sesi ada di sini
 *
 * Supaya ia tampil di SETIAP layar, bukan hanya di beranda. Pengguna sistem klaim mengisi
 * form panjang, dan sesi yang habis di tengah pengisian tanpa peringatan berarti pekerjaan
 * hilang. Sebelum kerangka ini ada, layar modul harus mengingat memasangnya sendiri — dan
 * satu layar yang lupa berarti peringatannya tidak pernah muncul di sana.
 *
 * # Yang BELUM ada di sini
 *
 * Jejak lokasi (breadcrumb) dan penanda peran pengguna. Keduanya bagian `TKT-U1-001`
 * yang sesungguhnya; kerangka ini cikal-bakalnya, bukan penggantinya.
 */
export function Kerangka({ anak, aksiDiHalaman = false }: Props) {
  const keluar = gunakanKeluar()

  return (
    <div className="flex min-h-screen flex-col bg-white">
      <PeringatanSesi />

      {!aksiDiHalaman && (
        <header className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
          <span className="text-sm font-semibold text-slate-900">Claim PNC</span>
          <div className="flex flex-wrap items-center gap-3">
            <PemilihPortal />
            <Tombol
              peran="sekunder"
              onClick={() => keluar.mutate()}
              sedangJalan={keluar.isPending}
              teksSedangJalan="Keluar…"
            >
              Keluar
            </Tombol>
          </div>
        </header>
      )}

      {/* md:flex-row menjadikan menu kolom samping di layar lebar dan baris atas di layar
          sempit. min-w-0 pada isi mencegah tabel lebar memaksa seluruh halaman melebar —
          tanpa itu, gulir mendatar milik TabelData tidak berfungsi. */}
      <div className="flex flex-1 flex-col md:flex-row">
        <NavigasiUtama />
        <div className="min-w-0 flex-1">{anak}</div>
      </div>
    </div>
  )
}
