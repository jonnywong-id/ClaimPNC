import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { HalamanBeranda } from '@/modules/beranda/HalamanBeranda'
import { HalamanMasuk } from '@/modules/masuk/HalamanMasuk'
import { HalamanStatusProgres1 } from '@/modules/master-status-progres/HalamanStatusProgres1'
import { GalatAPI } from '@/api/klien'
import { KodeGalat } from '@/api/tipe'
import { gunakanSesi } from '@/app/sesi'

import { PenjagaSesi } from './PenjagaSesi'
import { PeringatanSesi } from './PeringatanSesi'

/**
 * Sesi yang ditolak server di tengah pekerjaan dibersihkan di satu tempat ini.
 *
 * Tanpa penanganan terpusat, setiap layar harus mengingat memeriksanya sendiri — dan
 * satu layar yang lupa akan menampilkan halaman kosong alih-alih mengembalikan pengguna
 * ke layar masuk.
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
      <Route
        path="/"
        element={
          <PenjagaSesi anak={<Beranda />} />
        }
      />
      {/* Master Status Progres 1.
          Belum ada tautan menuju ke sini: peta menu mengikuti izin peran, dan tabel 22
          peran beserta 51 izin menu adalah TKT-F3-004 yang masih terhalang. Keputusan
          Work Owner 2026-09-17: rute lebih dulu, tanpa menyentuh modul Beranda. Sampai
          kerangka menu TKT-U1-001 dibangun, layar ini dibuka lewat alamatnya. */}
      <Route
        path="/master/status-progres-1"
        element={<PenjagaSesi anak={<Layar anak={<HalamanStatusProgres1 />} />} />}
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

function Beranda() {
  return (
    <div className="min-h-screen bg-white">
      <PeringatanSesi />
      <HalamanBeranda />
    </div>
  )
}

/**
 * Layar membungkus satu halaman dengan bagian yang berlaku untuk seluruh layar dalam
 * sesi — sekarang baru peringatan sesi hampir habis.
 *
 * Ia bukan kerangka portal yang sebenarnya: navigasi samping dan jejak lokasi adalah
 * TKT-U1-001. Yang dijaminnya sekarang hanyalah satu hal yang tidak boleh terlewat —
 * peringatan sesi tampil di layar modul, bukan hanya di beranda. Tanpa itu, sesi habis
 * di tengah mengisi form akan datang tanpa peringatan.
 */
function Layar({ anak }: { anak: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-white">
      <PeringatanSesi />
      {anak}
    </div>
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
