import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { RCLDokterPage } from './RCLDokterPage'
import type { DetailRCL } from './types'

/** Data uji seluruhnya KARANGAN (`D-69`). */
function detail(partial: Partial<DetailRCL> = {}): DetailRCL {
  return {
    portal: 'ASM',
    nomor_case: 'PNCN.26.31',
    nomor_polis: '26.002.2026.00031',
    nama_tertanggung: 'Andi Saputra Contoh',
    mode: '1',
    catatan_analyst: 'Diagnosa tidak dijamin.',
    alasan: 'Penyakit bawaan.',
    alasan_dokter: '',
    status_klaim: '1151',
    status_proses: 'New',
    operator_penerima: 'JONNY',
    tanggal_masuk_inbox: '2026-10-05 15:18',
    ...partial,
  }
}

let calls: string[] = []
let posted: { url: string; body: unknown }[] = []

function json(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function stub(status: number, body: unknown, decision?: { status: number; body: unknown }) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push(url)
    if (init?.method === 'POST') {
      posted.push({ url, body: JSON.parse(String(init.body)) })
      return Promise.resolve(json(decision?.status ?? 200, decision?.body ?? {}))
    }
    return Promise.resolve(json(status, body))
  })
}

function renderAt(nomor: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/inbox-rcl/klaim/${nomor}`]}>
        <Routes>
          <Route path="/inbox-rcl/klaim/:nomor" element={<RCLDokterPage />} />
          <Route path="/inbox-rcl" element={<p>Antrean RCL</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  posted = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-10-06T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('mode RCL menggambar isian dan tombol section RCLDokter', async () => {
  stub(200, detail())
  renderAt('PNCN.26.31')

  expect(await screen.findByText('Klaim PNCN.26.31')).toBeInTheDocument()
  expect(screen.getByLabelText('Catatan dari Analyst')).toHaveValue('Diagnosa tidak dijamin.')
  expect(screen.getByLabelText('Alasan Klaim Ditolak/RCL')).toHaveValue('Penyakit bawaan.')
  expect(screen.getByText('Apakah anda setuju untuk menolak klaim ini?')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Setuju' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Tidak Setuju' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Submit' })).not.toBeInTheDocument()
  expect(screen.queryByText('Lihat Dokumen')).not.toBeInTheDocument()
  expect(calls[0]).toContain('/api/inbox-rcl/klaim/PNCN.26.31')
})

it('mode MSIG menggambar Alasan Klaim MSIG dengan Back dan Submit', async () => {
  stub(200, detail({ mode: '3', alasan: 'Menunggu leader MSIG.' }))
  renderAt('PNCN.26.31')

  expect(await screen.findByLabelText('Alasan Klaim MSIG')).toHaveValue('Menunggu leader MSIG.')
  expect(screen.getByRole('button', { name: 'Back' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Submit' })).toBeInTheDocument()
  expect(screen.queryByText('Apakah anda setuju untuk menolak klaim ini?')).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Setuju' })).not.toBeInTheDocument()
})

it('klaim di luar antrean pemanggil menampilkan pesan dari server', async () => {
  stub(404, { kode: 'klaim_tidak_ditemukan', pesan: 'Klaim ini tidak ada di antrean RCL Anda.' })
  renderAt('PNCN.26.99')

  expect(await screen.findByText('Klaim tidak dapat dibuka')).toBeInTheDocument()
  expect(screen.getByText('Klaim ini tidak ada di antrean RCL Anda.')).toBeInTheDocument()
})

it('Setuju langsung mengirim SETUJU lalu kembali ke antrean', async () => {
  stub(200, detail(), { status: 200, body: { tahap_berikutnya: 'RCL/PUCL' } })
  renderAt('PNCN.26.31')

  fireEvent.click(await screen.findByRole('button', { name: 'Setuju' }))

  expect(await screen.findByText('Antrean RCL')).toBeInTheDocument()
  expect(posted).toHaveLength(1)
  expect(posted[0]!.url).toContain('/api/inbox-rcl/klaim/PNCN.26.31/keputusan')
  expect(posted[0]!.body).toEqual({ keputusan: 'SETUJU', alasan_dokter: '' })
})

it('Tidak Setuju membuka layar Alasan Dokter, Kirim mengirim TidakSetuju beserta alasannya', async () => {
  stub(200, detail(), { status: 200, body: {} })
  renderAt('PNCN.26.31')

  fireEvent.click(await screen.findByRole('button', { name: 'Tidak Setuju' }))
  expect(posted).toHaveLength(0)

  fireEvent.change(screen.getByLabelText('Alasan Dokter'), { target: { value: 'Diagnosa dijamin.' } })
  fireEvent.click(screen.getByRole('button', { name: 'Kirim' }))

  await waitFor(() => expect(posted).toHaveLength(1))
  expect(posted[0]!.body).toEqual({ keputusan: 'TidakSetuju', alasan_dokter: 'Diagnosa dijamin.' })
})

it('Back pada mode MSIG mengirim BackMSIG; Cancel kembali tanpa mengirim', async () => {
  stub(200, detail({ mode: '3' }), { status: 200, body: {} })
  renderAt('PNCN.26.31')

  fireEvent.click(await screen.findByRole('button', { name: 'Back' }))
  fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
  expect(await screen.findByRole('button', { name: 'Submit' })).toBeInTheDocument()
  expect(posted).toHaveLength(0)

  fireEvent.click(screen.getByRole('button', { name: 'Back' }))
  fireEvent.click(screen.getByRole('button', { name: 'Kirim' }))
  await waitFor(() => expect(posted).toHaveLength(1))
  expect(posted[0]!.body).toEqual({ keputusan: 'BackMSIG', alasan_dokter: '' })
})

it('keputusan yang ditolak server menampilkan pesannya dan tetap di layar kerja', async () => {
  stub(200, detail(), {
    status: 404,
    body: { kode: 'klaim_tidak_ditemukan', pesan: 'Klaim ini tidak ada di antrean RCL Anda.' },
  })
  renderAt('PNCN.26.31')

  fireEvent.click(await screen.findByRole('button', { name: 'Setuju' }))

  expect(await screen.findByText('Keputusan tidak tersimpan')).toBeInTheDocument()
  expect(screen.getByText('Klaim ini tidak ada di antrean RCL Anda.')).toBeInTheDocument()
  expect(screen.queryByText('Antrean RCL')).not.toBeInTheDocument()
})
