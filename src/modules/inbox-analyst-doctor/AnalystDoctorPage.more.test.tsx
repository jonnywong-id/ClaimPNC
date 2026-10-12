import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useParams } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AnalystDoctorPage } from './AnalystDoctorPage'
import type { KeteranganResponse, TugasAnalystDoctor } from './types'

/** Uji tambahan Inbox Analyst Doctor: tautan klaim, sel kosong, dan muat ulang. Nilai KARANGAN. */

const ROW: TugasAnalystDoctor = {
  klaim_id: 'K-1',
  nomor_case: 'PNCN.26.0400',
  nomor_polis: '',
  nama_tertanggung: '',
  nama_cabang: '',
  nama_admin: '',
  komentar_pic_teknis: 'Mohon dinilai.',
  pic_teknis: '',
  tanggal_pendaftaran: '',
  lama_hari: 2,
  status_proses: 'Open',
  operator_penerima: 'X',
}

const KETERANGAN: KeteranganResponse = {
  portal: 'ASM',
  kolom: [
    { kunci: 'nomor_case', judul: 'Nomor Case' },
    { kunci: 'nomor_polis', judul: 'No Polis' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
    { kunci: 'nama_cabang', judul: 'Nama Cabang' },
    { kunci: 'tanggal_pendaftaran', judul: 'Tanggal Pendaftaran' },
    { kunci: 'nama_admin', judul: 'Nama Admin' },
    // Tanpa keterangan: sel kosongnya tetap diberi judul kosong.
    { kunci: 'komentar_pic_teknis', judul: 'Komentar dari PIC Teknis' },
    { kunci: 'lama_hari', judul: 'Lama Waktu Klaim' },
    // Kunci yang tidak dikenal dilewati, bukan digambar kosong.
    { kunci: 'kolom_baru', judul: 'Kolom Baru' },
  ],
  keterbatasan: [],
  ukuran_halaman: 25,
}

let calls: string[] = []

function stubFetch(rows: TugasAnalystDoctor[], failList = false) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    if (url.includes('/keterangan')) {
      return Promise.resolve(new Response(JSON.stringify(KETERANGAN), { status: 200 }))
    }
    if (failList) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(
        JSON.stringify({ portal: 'ASM', data: rows, total: rows.length, lewati: 0, batas: 25, cari: '' }),
        { status: 200 },
      ),
    )
  })
}

function ViewClaim() {
  const { nomor } = useParams()
  return <p>Membuka klaim {nomor}</p>
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-analyst-doctor']}>
        <Routes>
          <Route path="/inbox-analyst-doctor" element={<AnalystDoctorPage />} />
          <Route path="/view-claim/:nomor" element={<ViewClaim />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-24T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('membuka klaim lewat nomor case, dan menulis tanda pisah pada sel kosong', async () => {
  stubFetch([ROW, { ...ROW, klaim_id: 'K-2', nomor_case: '' }])
  const user = userEvent.setup()
  renderPage()

  expect(await screen.findByText('belum bernomor')).toBeInTheDocument()
  expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(10)
  expect(screen.queryByRole('columnheader', { name: 'Kolom Baru' })).not.toBeInTheDocument()
  // Komentar tanpa PIC tidak menuliskan nama siapa pun.
  expect(screen.getAllByText('Mohon dinilai.')[0]?.textContent).toBe('Mohon dinilai.')

  await user.click(screen.getByRole('button', { name: 'PNCN.26.0400' }))
  expect(await screen.findByText('Membuka klaim PNCN.26.0400')).toBeInTheDocument()
})

it('memuat ulang antrean lewat tombolnya', async () => {
  stubFetch([ROW])
  const user = userEvent.setup()
  renderPage()

  await screen.findByRole('button', { name: 'PNCN.26.0400' })
  const before = calls.length
  await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
  await waitFor(() => expect(calls.length).toBeGreaterThan(before))
})

it('memakai pesan umum saat antrean gagal karena jaringan', async () => {
  stubFetch([], true)
  renderPage()

  expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
  expect(
    screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
  ).toBeInTheDocument()
})

it('memberi keterangan pada komentar yang belum terbawa meski server tidak menjelaskannya', async () => {
  stubFetch([{ ...ROW, komentar_pic_teknis: '' }])
  renderPage()

  const cell = await screen.findByText('belum terbawa')
  expect(cell).toHaveAttribute('title', '')
})
