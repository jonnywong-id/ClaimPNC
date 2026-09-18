import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { HalamanBeranda } from '@/modules/beranda/HalamanBeranda'
import { HalamanMasuk } from '@/modules/masuk/HalamanMasuk'
import { HalamanStatusProgres1 } from '@/modules/master-status-progres/HalamanStatusProgres1'
import { GalatAPI } from '@/api/klien'
import { KodeGalat } from '@/api/tipe'
import { gunakanSesi } from '@/app/sesi'

import { Kerangka } from './Kerangka'
import { PenjagaSesi } from './PenjagaSesi'

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
      {/* Kedua rute di bawah dibungkus Kerangka, sehingga menu utama dan peringatan sesi
          tampil di keduanya.

          Beranda memakai aksiDiHalaman: ia sudah memuat pemilih portal dan tombol keluar
          di header-nya sendiri, dan menampilkannya dua kali akan membingungkan. Berkas
          modul Beranda sendiri TIDAK disentuh — yang berubah hanya pembungkusnya di sini,
          dan berkas ini kerangka, bukan modul (keputusan Work Owner 2026-09-17). */}
      <Route
        path="/"
        element={<PenjagaSesi anak={<Kerangka anak={<HalamanBeranda />} aksiDiHalaman />} />}
      />
      <Route
        path="/master/status-progres-1"
        element={<PenjagaSesi anak={<Kerangka anak={<HalamanStatusProgres1 />} />} />}
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
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
