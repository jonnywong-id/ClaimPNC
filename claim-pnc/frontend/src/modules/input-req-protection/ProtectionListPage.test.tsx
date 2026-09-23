import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ProtectionListPage } from './ProtectionListPage'
import type { Protection, ProtectionListResponse } from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen.
 */
function protection(partial: Partial<Protection> = {}): Protection {
  return {
    nomor_proteksi: 'OPCN.26.0001',
    nomor_polis: '99.001.2026.00000001',
    nomor_klaim: '',
    tipe_proteksi: '1',
    tanggal_proteksi: '2026-09-16',
    keterangan: 'Permintaan contoh.',
    user_create: 'ADMINCONTOH',
    dapat_disunting: true,
    premi: false,
    ...partial,
  }
}

function listResponse(partial: Partial<ProtectionListResponse> = {}): ProtectionListResponse {
  return { proteksi: [protection()], total: 1, ...partial }
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    return Promise.resolve(answer(url, init))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProtectionListPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-30T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// Judul kolom mengikuti `Section/InboxReqProtection_Section-Section.xml` apa adanya
// (`D-13`). Uji ini menjaga kesetaraan itu: mengganti judul akan lulus uji lain tetapi
// membuat layar berbeda dari yang dikenal pengguna.
it('memakai ketujuh judul kolom yang sama dengan section rujukan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')

  // Ditanyakan lewat peran `columnheader`, bukan lewat teks: sebagian judul kolom juga
  // muncul sebagai label isian pada form, dan pencarian berbasis teks akan menemukan
  // keduanya lalu gagal karena ganda.
  const headers = screen.getAllByRole('columnheader').map((cell) => cell.textContent ?? '')

  for (const title of [
    'No Proteksi',
    'No Polis',
    'No Klaim',
    'Tipe Proteksi',
    'Tanggal Proteksi Dibuat',
    'Keterangan',
    'User Create',
  ]) {
    expect(headers.some((header) => header.includes(title))).toBe(true)
  }
})

// Aturannya dari `Section/InboxReqProtection_Section-Section.xml:8657`, yang menonaktifkan
// tautan baris ketika `.CaseID` terisi.
it('mematikan tautan baris yang sudah tertaut klaim', async () => {
  stubFetch(() =>
    jsonResponse(
      200,
      listResponse({
        proteksi: [
          protection({ nomor_proteksi: 'OPCN.26.0001', dapat_disunting: true }),
          protection({
            nomor_proteksi: 'OPCN.26.0002',
            nomor_klaim: 'PNCN.26.0007',
            dapat_disunting: false,
          }),
        ],
        total: 2,
      }),
    ),
  )
  renderPage()

  // Yang masih dapat disunting berupa tombol; yang terkunci bukan.
  expect(await screen.findByRole('button', { name: 'OPCN.26.0001' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'OPCN.26.0002' })).not.toBeInTheDocument()
  expect(screen.getByText('OPCN.26.0002')).toBeInTheDocument()
})

it('menyatakan proteksi yang belum tertaut klaim, bukan membiarkan selnya kosong', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('belum tertaut')).toBeInTheDocument()
})

// Kode tipe yang labelnya tidak diketahui ditampilkan APA ADANYA (`R-16`). Menebak label
// akan diterima pengguna begitu saja; kode mentah segera ditanyakan.
it('menampilkan kode tipe proteksi yang belum berlabel apa adanya', async () => {
  stubFetch(() =>
    jsonResponse(
      200,
      listResponse({
        proteksi: [
          protection({ nomor_proteksi: 'OPCN.26.0001', tipe_proteksi: '7' }),
          protection({ nomor_proteksi: 'OPCN.26.0002', tipe_proteksi: '5' }),
        ],
        total: 2,
      }),
    ),
  )
  renderPage()

  expect(await screen.findByText('Perubahan DOL')).toBeInTheDocument()
  expect(screen.getByText('5')).toBeInTheDocument()
})

// Pencarian dikerjakan SERVER: daftarnya hanya dibatasi "belum diakseptasi", sehingga
// menyaring satu halaman dari sepuluh akan memberi jawaban yang salah.
it('mengirim kata pencarian ke server, bukan menyaring di peramban', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')

  await userEvent.type(
    screen.getByLabelText('Cari No Proteksi / No Polis / No Klaim'),
    '99.001',
  )

  await waitFor(() => {
    expect(calls.some((call) => call.url.includes('cari=99.001'))).toBe(true)
  })
})

it('membuka form kosong saat tombol Input Open Protection ditekan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Input Open Protection' }))

  expect(screen.getByRole('heading', { name: 'Input Open Protection' })).toBeInTheDocument()
  // Pencocokan label PERSIS, bukan pola: judul kolom tabel memuat teks yang sama, dan pola
  // akan menemukan keduanya.
  expect(screen.getByLabelText('No Polis')).toHaveValue('')
})

// Pelanggaran validasi dikirim backend SEKALIGUS dan ditempelkan ke kolomnya masing-masing
// (`11-CROSSCUTTING` §1.2 butir 1). Menampilkannya sebagai satu pesan di atas form akan
// membuat pengguna menebak kolom mana yang dimaksud.
it('menempelkan pelanggaran validasi ke kolomnya masing-masing', async () => {
  stubFetch((_url, init) => {
    if (init?.method === 'POST') {
      return jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Isian belum lengkap atau belum benar.',
        detail: [
          { field: 'no_polis', pesan: 'No Polis wajib diisi.' },
          { field: 'keterangan', pesan: 'Keterangan wajib diisi.' },
        ],
      })
    }
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Input Open Protection' }))

  // Isian wajib diisi lebih dulu supaya peramban benar-benar MENGIRIM permintaannya —
  // isian ber-`required` yang kosong ditahan peramban, dan penolakan backend tidak akan
  // pernah terjadi. Yang diuji di sini adalah penolakan BACKEND, bukan penolakan peramban.
  await userEvent.type(screen.getByLabelText('No Polis'), '99.001.2026.00000001')
  await userEvent.type(screen.getByLabelText('No Klaim'), 'PNCN.26.0007')
  await userEvent.type(
    screen.getByLabelText('Referensi Klaim (hasil pencarian)'),
    'KLAIM-CONTOH-0007',
  )
  await userEvent.selectOptions(screen.getByLabelText('Tipe Proteksi'), '7')
  await userEvent.type(screen.getByLabelText('Next Date Of Loss'), '2026-08-17')
  await userEvent.type(screen.getByLabelText('Keterangan'), 'Keterangan contoh.')

  await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

  expect(await screen.findByText('No Polis wajib diisi.')).toBeInTheDocument()
  expect(screen.getByText('Keterangan wajib diisi.')).toBeInTheDocument()
})

// `TKT-F6-002`: permintaan tanpa portal ditolak server, dan layar tidak menembaknya lebih
// dulu hanya untuk menerima penolakan.
it('menuntun memilih entitas lebih dulu dan tidak menembak server', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})
