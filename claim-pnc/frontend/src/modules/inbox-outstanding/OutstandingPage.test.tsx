import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { OutstandingPage } from './OutstandingPage'
import type { OutstandingClaim, OutstandingListResponse } from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen.
 */
function claim(partial: Partial<OutstandingClaim> = {}): OutstandingClaim {
  return {
    klaim_id: 'KLM-0001',
    nomor_klaim: 'PNCN.26.0001',
    nomor_polis: 'POL-FIRE-0001',
    nama_tertanggung: 'PT Bumi Contoh Sentosa',
    nama_bisnis: 'Fire',
    sumber_bisnis: 'Broker',
    nama_cabang: 'JKT',
    group_panel: '006',
    tanggal_pendaftaran: '2026-09-16',
    tanggal_kejadian: '2026-09-08',
    tanggal_lapor: '2026-09-10',
    // Nilai `PYSTATUSWORK` sebagaimana benar-benar tersimpan — bukan 'BERJALAN', yang
    // berasal dari tabel yang sempat salah dipakai.
    status_proses: 'New',
    status_tampil: 'On Progress',
    status_klaim: '1147',
    pic_teknik: 'BUDISANTOSO',
    admin_pnc: 'ADMINPNC',
    umur_hari: 4,
    aging_hari: 2,
    posisi_klaim: 'On Progress',
    tahap_kini: 'Input Estimasi',
    pemegang_tugas: 'BUDISANTOSO',
    ...partial,
  }
}

function listResponse(partial: Partial<OutstandingListResponse> = {}): OutstandingListResponse {
  return {
    klaim: [claim()],
    total: 1,
    batas_lini: { tanpa_batas: false, group_panel: ['006'] },
    ...partial,
  }
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
  // Cache dimatikan supaya tiap uji berdiri sendiri; retry dimatikan supaya jalur galat
  // tidak menunggu percobaan ulang.
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <OutstandingPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: '2026-09-20T12:00:00Z',
  })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// Judul kolom mengikuti `Section/InboxRegister_Section-Section.xml` apa adanya (`D-13`).
//
// Uji ini menjaga kesetaraan itu: mengganti judul menjadi bahasa Indonesia akan lulus uji
// lain tetapi membuat layar berbeda dari yang dikenal pengguna.
it('memakai judul kolom yang sama dengan section rujukan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')

  for (const title of [
    'Claim no',
    'Policy no',
    'Insured name',
    'Business Name',
    'Business source',
    'Branch name',
    'Admin name',
    'Register Date',
    'Date of loss',
    'Total Aging',
    'Aging',
    'Claim status',
    'Status ASM',
    'ASM PIC',
  ]) {
    // Nama dicocokkan PERSIS, bukan sebagian: "Aging" dan "Total Aging" adalah dua kolom
    // berbeda, dan pencocokan sebagian akan menemukan keduanya sekaligus.
    expect(screen.getByRole('columnheader', { name: title })).toBeInTheDocument()
  }
})

// Kolom "Aging" yang belum terisi tampil sebagai tanda hubung, bukan "0".
//
// Keduanya berbeda: nol hari adalah angka yang diketahui, belum terisi bukan.
it('membedakan Aging yang belum terisi dari nol hari', async () => {
  stubFetch(() =>
    jsonResponse(
      200,
      listResponse({
        klaim: [claim({ klaim_id: 'A', aging_hari: 0 }), claim({ klaim_id: 'B', aging_hari: null })],
        total: 2,
      }),
    ),
  )
  renderPage()

  expect(await screen.findByText('0 hari')).toBeInTheDocument()
})

it('menampilkan isi kolom klaim', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('PNCN.26.0001')).toBeInTheDocument()
  expect(screen.getByText('PT Bumi Contoh Sentosa')).toBeInTheDocument()
  expect(screen.getByText('POL-FIRE-0001')).toBeInTheDocument()
  expect(screen.getByText('ADMINPNC')).toBeInTheDocument()
  expect(screen.getByText('BUDISANTOSO')).toBeInTheDocument()
  expect(screen.getByText('1147')).toBeInTheDocument()
})

// "Status ASM" yang kosong adalah keadaan NYATA, bukan kasus tepi yang dikarang: baris
// produksi yang diserahkan Work Owner tidak menyertakan kolom `STATUSLOCK_1` sama sekali.
//
// Sel yang benar-benar kosong membuat baris tampak rusak; ia dinyatakan dengan tanda hubung.
it('menandai Status ASM yang tidak terisi, bukan membiarkan selnya kosong', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ klaim: [claim({ status_klaim: '' })] })))
  renderPage()

  await screen.findByText('PNCN.26.0001')
  expect(screen.getByRole('cell', { name: '—' })).toBeInTheDocument()
})

it('menampilkan sumber bisnis', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('Broker')).toBeInTheDocument()
})

it('mengirim portal aktif pada setiap permintaan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')

  const request = calls.find((c) => c.url.startsWith('/api/inbox-outstanding'))
  expect(request).toBeDefined()

  const header = new Headers(request?.init?.headers)
  expect(header.get('X-Portal')).toBe('ASM')
})

// Klaim yang belum lolos Input Register belum bernomor. Menampilkannya sebagai sel kosong
// membuat layar tampak rusak; ia dinyatakan dengan kata-kata.
it('menyatakan klaim yang belum bernomor, bukan membiarkannya kosong', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ klaim: [claim({ nomor_klaim: '' })] })))
  renderPage()

  expect(await screen.findByText('belum bernomor')).toBeInTheDocument()
})

// Pemegang tugas dan tahap TETAP diterima dari server, tetapi TIDAK ditampilkan sebagai
// kolom — `Section/InboxRegister_Section-Section.xml` tidak punya kolomnya.
//
// Uji ini menjaga keputusan itu tetap disengaja. Tanpa uji, kolomnya mudah ditambahkan
// kembali karena datanya memang ada di tangan.
it('tidak menampilkan tahap maupun pemegang tugas sebagai kolom', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')

  expect(screen.queryByRole('columnheader', { name: /Tahap/ })).not.toBeInTheDocument()
  expect(screen.queryByRole('columnheader', { name: /[Pp]emegang/ })).not.toBeInTheDocument()
  expect(screen.queryByText('Input Estimasi')).not.toBeInTheDocument()
})

// Inilah uji yang menjaga janji terpenting layar ini.
//
// Work Owner menetapkan pengguna tanpa lini bisnis tetap melihat SELURUH lini, persis
// perilaku Pega. Yang berbeda dari Pega: keadaannya dinyatakan, sehingga datanya cepat
// dilengkapi admin.
it('menyatakan ketika penyaringan lini bisnis belum berlaku', async () => {
  stubFetch(() =>
    jsonResponse(
      200,
      listResponse({ batas_lini: { tanpa_batas: true, group_panel: [] } }),
    ),
  )
  renderPage()

  expect(await screen.findByText('Penyaringan lini bisnis belum berlaku')).toBeInTheDocument()
})

it('menyebutkan lini yang berlaku ketika penyaringan aktif', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText(/Menampilkan lini bisnis/)).toBeInTheDocument()
  expect(screen.getByText('006')).toBeInTheDocument()
})

// Pencarian dikerjakan SERVER. Bila ia dikerjakan di peramban, hasilnya hanya menyentuh
// halaman yang sedang terbuka — dan pengguna diberi tahu bahwa klaimnya tidak ada padahal
// ia ada di halaman lain.
it('mengirim kata pencarian ke server, tidak menyaring di peramban', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')
  calls = []

  await userEvent.type(screen.getByRole('searchbox'), 'POL-FIRE')

  await waitFor(() => {
    const request = calls.find((c) => c.url.includes('cari='))
    expect(request).toBeDefined()
  })
})

it('meminta unduhan dengan penyaring yang sedang berlaku', async () => {
  stubFetch((url) => {
    if (url.includes('/unduh')) {
      return new Response('No Klaim\nPNCN.26.0001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="inbox-outstanding-20260920.csv"',
        },
      })
    }
    return jsonResponse(200, listResponse())
  })

  // jsdom tidak menyediakan keduanya; unduhan memakai objek URL untuk menyerahkan berkas
  // ke peramban.
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:uji'),
    revokeObjectURL: vi.fn(),
  })

  renderPage()
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: /Unduh CSV/ }))

  await waitFor(() => {
    const request = calls.find((c) => c.url.includes('/unduh'))
    expect(request).toBeDefined()
    expect(new Headers(request?.init?.headers).get('X-Portal')).toBe('ASM')
  })
})

it('menuntun pengguna memilih entitas ketika portal belum dipilih', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(() => jsonResponse(200, listResponse()))

  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()

  // Tidak ada permintaan yang ditembakkan: menembaknya hanya untuk menerima penolakan
  // adalah perjalanan jaringan yang sia-sia.
  expect(calls.filter((c) => c.url.startsWith('/api/inbox-outstanding'))).toHaveLength(0)
})

it('menampilkan pesan ketika daftar gagal dimuat', async () => {
  stubFetch(() =>
    jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' }),
  )
  renderPage()

  expect(await screen.findByText('Daftar klaim tidak dapat dimuat')).toBeInTheDocument()
})

it('menyembunyikan paginasi ketika tidak ada klaim', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ klaim: [], total: 0 })))
  renderPage()

  expect(await screen.findByText('Tidak ada klaim yang masih berjalan.')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Berikutnya' })).not.toBeInTheDocument()
})
