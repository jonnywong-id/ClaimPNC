import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxRCLPage } from './InboxRCLPage'
import type { DaftarResponse, KeteranganResponse, TugasRCL } from './types'

/** Data uji seluruhnya KARANGAN (`D-69`). */
function tugas(partial: Partial<TugasRCL> = {}): TugasRCL {
  return {
    klaim_id: 'ASM-FW-GCNMFW-WORK PNCN.26.0412',
    nomor_case: 'PNCN.26.0412',
    nomor_polis: '26.002.2026.00412',
    nama_tertanggung: 'Andi Saputra Contoh',
    tanggal_masuk_inbox: '2026-09-22 10:00',
    deskripsi_analyst: 'Diagnosa tidak termasuk manfaat rawat inap.',
    dokter_rcl: 'DOKTERRCL01',
    status_proses: 'Open',
    operator_penerima: 'DOKTERRCL01',
    ...partial,
  }
}

/** Kelima judul diambil dari rule `pyCaption …` pada `Harness/RCL_Harness-Harness.xml`. */
const keteranganResponse: KeteranganResponse = {
  portal: 'ASM',
  kolom: [
    { kunci: 'nomor_case', judul: 'Nomor Case' },
    { kunci: 'nomor_polis', judul: 'No Polis' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
    { kunci: 'tanggal_masuk_inbox', judul: 'Tanggal Masuk Inbox' },
    { kunci: 'deskripsi_analyst', judul: 'Deskripsi Analyst' },
  ],
  selisih_terencana: ['Kotak cari Nomor Case dan No Polis adalah TAMBAHAN.'],
  keterbatasan: ['Antrean disaring dengan identitas LAMA Anda.'],
  ukuran_halaman: 25,
}

function daftarResponse(partial: Partial<DaftarResponse> = {}): DaftarResponse {
  return {
    portal: 'ASM',
    data: [tugas()],
    total: 1,
    lewati: 0,
    batas: 25,
    cari: '',
    identitas_lama_ditemukan: true,
    ...partial,
  }
}

let calls: string[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(daftar: DaftarResponse | (() => Response)) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    if (url.includes('/keterangan')) return Promise.resolve(jsonResponse(200, keteranganResponse))
    return Promise.resolve(typeof daftar === 'function' ? daftar() : jsonResponse(200, daftar))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InboxRCLPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function urlDaftar(): string {
  const daftar = calls.filter((c) => !c.includes('/keterangan'))
  return daftar[daftar.length - 1] ?? ''
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-28T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('menggambar kelima judul kolom dari server beserta isinya', async () => {
  stubFetch(daftarResponse())
  renderPage()

  expect(await screen.findByText('PNCN.26.0412')).toBeInTheDocument()
  for (const judul of [
    'Nomor Case',
    'No Polis',
    'Nama Tertanggung',
    'Tanggal Masuk Inbox',
    'Deskripsi Analyst',
  ]) {
    expect(screen.getAllByText(judul).length).toBeGreaterThan(0)
  }
  expect(screen.getByText('2026-09-22 10:00')).toBeInTheDocument()
  expect(screen.getByText('Diagnosa tidak termasuk manfaat rawat inap.')).toBeInTheDocument()
})

it('identitas lama tidak ditemukan hanya grid kosong, tanpa peringatan maupun catatan', async () => {
  stubFetch(daftarResponse({ data: [], total: 0, identitas_lama_ditemukan: false }))
  renderPage()

  expect(await screen.findByText('Tidak ada klaim RCL untuk Anda saat ini.')).toBeInTheDocument()
  expect(screen.queryByText('Identitas lama Anda tidak ditemukan')).not.toBeInTheDocument()
  expect(screen.queryByText('Perbedaan yang disengaja terhadap layar lama')).not.toBeInTheDocument()
  expect(screen.queryByText('Yang perlu diketahui')).not.toBeInTheDocument()
})

it('pencarian dikirim ke server', async () => {
  stubFetch(daftarResponse())
  renderPage()
  await screen.findByText('PNCN.26.0412')

  await userEvent.type(screen.getByLabelText(/Cari Nomor Case/), '26.005')

  await waitFor(() => expect(urlDaftar()).toContain('cari=26.005'))
})

it('meminta portal lebih dulu bila belum dipilih, tanpa menembak server', () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(daftarResponse())
  renderPage()

  expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})

it('menampilkan pesan galat dari server', async () => {
  stubFetch(() =>
    jsonResponse(409, { kode: 'profil_pemanggil_tidak_lengkap', pesan: 'Identitas Anda tidak terbaca.' }),
  )
  renderPage()

  expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
})
