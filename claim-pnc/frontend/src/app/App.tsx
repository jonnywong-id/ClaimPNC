import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { HalamanBeranda } from '@/modules/beranda/HalamanBeranda'
import { HalamanMasterRekening } from '@/modules/master-rekening/HalamanMasterRekening'
import { HalamanMasterStatusKlaim } from '@/modules/master-status-klaim/HalamanMasterStatusKlaim'
import { HalamanMasuk } from '@/modules/masuk/HalamanMasuk'
import { HalamanStatusProgres1 } from '@/modules/master-status-progres/HalamanStatusProgres1'
import { GalatAPI } from '@/api/klien'
import { KodeGalat } from '@/api/tipe'
import { gunakanSesi } from '@/app/sesi'

import { KerangkaHalaman } from './KerangkaHalaman'
import { PenjagaSesi } from './PenjagaSesi'
import { PeringatanSesi } from './PeringatanSesi'

/**
 * Sesi yang ditolak server di tengah pekerjaan dibersihkan di satu tempat ini.
 *
 * Tanpa penanganan terpusat, setiap layar harus mengingat memeriksanya sendiri — dan
 * satu layar yang lupa akan menampilkan halaman kosong alih-alih mengembalikan pengguna
 * ke layar masuk.
 * 
 */
function tanganiGalatSesi(galat: unknown): void {
  if (!(galat instanceof GalatAPI)) return
  if (galat.kode === KodeGalat.sesiTidakSah || galat.kode === KodeGalat.sesiKedaluwarsa) {
    gunakanSesi.getState().bersihkan()
  }
}

export function buatKlienKueri(): QueryClient {
  return new QueryClient({
    queryCache: new QueryCache({ onError: tanganiGalatSesi }),
    mutationCache: new MutationCache({ onError: tanganiGalatSesi }),
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false,
      },
      mutations: { retry: false },
    },
  })
}

export function Rute() {
  return (
    <Routes>
      <Route path="/masuk" element={<HalamanMasuk />} />
      {/* Seluruh rute di bawah dibungkus Terlindungi, sehingga bilah atas, menu, identitas
          pengguna, tombol keluar, dan peringatan sesi tampil sama di semuanya.

          Beranda tidak lagi perlu perlakuan khusus. Pemilih portal, tombol keluar, dan
          nama pengguna sudah pindah dari header-nya ke bilah atas pada penataan ulang
          2026-09-17; PemilihPortal yang tersisa di dalamnya ber-`sm:hidden`, melengkapi
          bilah atas yang justru menyembunyikannya di layar sempit. */}
      <Route
        path="/"
        element={<PenjagaSesi anak={<Terlindungi anak={<HalamanBeranda />} />} />}
      />
      {/*
        Modul berikutnya menempel sebagai satu baris di sini. Penjaga sesi adalah
        KENYAMANAN TAMPILAN; penegakan yang sebenarnya ada di server, yang memeriksa
        sesi pada setiap endpoint.
      */}
      <Route
        path="/master/status-progres-1"
        element={<PenjagaSesi anak={<Terlindungi anak={<HalamanStatusProgres1 />} />} />}
      />
      <Route
        path="/master/status-klaim"
        element={<PenjagaSesi anak={<Terlindungi anak={<HalamanMasterStatusKlaim />} />} />}
      />
      {/*
        Master rekening berada di balik penjaga sesi yang sama. Pemeriksaan kewenangan
        menu — siapa yang boleh membuka layar master mana — adalah TKT-F3-005 yang
        belum ada; sampai itu ada, setiap pengguna yang dapat masuk dapat membukanya.
      */}
      <Route
        path="/master-rekening"
        element={<PenjagaSesi anak={<Terlindungi anak={<HalamanMasterRekening />} />} />}
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

/**
 * Terlindungi membungkus SELURUH layar di balik sesi dengan kerangka yang sama: bilah
 * atas, menu, identitas pengguna, tombol keluar, dan peringatan sesi.
 *
 * Satu pembungkus untuk semuanya, bukan satu per layar. Itu yang membuat tombol Keluar
 * dan nama pengguna hanya ada di satu tempat — sebelumnya keduanya hidup di dalam
 * halaman beranda, sehingga layar lain tidak punya cara keluar.
 */
function Terlindungi({ anak }: { anak: ReactNode }) {
  return (
    <KerangkaHalaman
      anak={
        <>
          <PeringatanSesi />
          {anak}
        </>
      }
    />
  )
}

export function App() {
  // Klien dibuat sekali seumur hidup aplikasi; membuatnya ulang tiap render akan
  // membuang seluruh cache pada setiap perubahan state.
  const [klien] = useState(buatKlienKueri)

  return (
    <QueryClientProvider client={klien}>
      <BrowserRouter>

      
        <Rute />
      </BrowserRouter>
    </QueryClientProvider>
  )
}
