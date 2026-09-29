import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DokumenPenunjangPanel } from './DokumenPenunjangPanel'

/**
 * Keadaan yang BERLAKU HARI INI: modul GCS belum disiapkan, jadi fiturnya mati.
 *
 * Berkas ini terpisah dari `DokumenPenunjangPanel.test.tsx` karena `vi.mock` berlaku untuk
 * satu berkas utuh dan tidak dapat dibalik di tengahnya. Keduanya menguji komponen yang
 * sama pada dua keadaan sakelarnya.
 *
 * Mock-nya ditulis eksplisit `false`, bukan dibiarkan memakai nilai aslinya: kalau
 * sakelarnya kelak dinyalakan, berkas ini harus GAGAL dan menarik perhatian — bukan
 * diam-diam ikut berubah arti.
 */
vi.mock('./fitur', () => ({ FITUR_DOKUMEN_PENUNJANG_AKTIF: false }))

let calls: string[] = []

function renderPanel(nomorKlaim: string | null) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <DokumenPenunjangPanel nomorKlaim={nomorKlaim} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-30T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    return Promise.resolve(new Response('{}', { status: 200 }))
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// Work Owner: "munculkan saja dulu tombol unggah file penunjang tanpa fungsinya".
//
// Menyembunyikannya akan membuat layar tampak tidak punya fitur unggah sama sekali, dan
// pertanyaan "kok hilang?" berulang.
it('TETAP menampilkan judul dan tombolnya', async () => {
  renderPanel('PNC-1865')

  expect(screen.getByText('Unggah File Penunjang')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Pilih berkas' })).toBeInTheDocument()
})

// Tombol hidup yang tidak melakukan apa-apa terbaca sebagai rusak, dan pengguna akan
// menekannya berkali-kali sebelum melaporkannya.
it('tombolnya MATI, bukan hidup tanpa akibat', async () => {
  renderPanel('PNC-1865')

  expect(screen.getByRole('button', { name: 'Pilih berkas' })).toBeDisabled()
})

it('menyebutkan sebabnya, bukan membiarkan panelnya kosong', async () => {
  renderPanel('PNC-1865')

  expect(
    screen.getByText(/Layanan penyimpanan dokumen belum tersedia/),
  ).toBeInTheDocument()
})

// Inilah yang membedakan "dimatikan" dari "ditampilkan tetapi tetap menembak server".
it('tidak menembak server sama sekali', async () => {
  renderPanel('PNC-1865')

  await waitFor(() => expect(calls).toHaveLength(0))
})

// Tanpa input berkas, tidak ada jalan membuka pemilih berkas — termasuk lewat papan ketik
// atau skrip. Tombol mati saja tidak cukup bila input-nya masih ada di DOM.
it('tidak memasang input berkas sama sekali', async () => {
  const { container } = renderPanel('PNC-1865')

  expect(container.querySelector('input[type="file"]')).toBeNull()
})

// Pesan "pilih No Klaim dulu" hanya bermakna ketika fiturnya hidup. Menampilkannya saat
// fiturnya mati menyuruh pengguna melakukan sesuatu yang tidak akan mengubah apa pun.
it('tidak meminta memilih klaim ketika fiturnya memang mati', async () => {
  renderPanel(null)

  expect(screen.queryByText(/Pilih No Klaim lebih dulu/)).not.toBeInTheDocument()
  expect(
    screen.getByText(/Layanan penyimpanan dokumen belum tersedia/),
  ).toBeInTheDocument()
})
