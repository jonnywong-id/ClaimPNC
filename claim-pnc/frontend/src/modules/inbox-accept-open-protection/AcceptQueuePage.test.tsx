import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AcceptQueuePage } from './AcceptQueuePage'
import type { Protection, ProtectionDetail, ProtectionListResponse } from './types'

/** Data uji seluruhnya KARANGAN (`D-69`). */
function protection(partial: Partial<Protection> = {}): Protection {
  return {
    nomor_proteksi: 'OPCN.26.0001',
    nomor_polis: '99.001.2026.00000001',
    nomor_klaim: 'PNCN.26.0007',
    tipe_proteksi: '1',
    nama_tipe_proteksi: 'General',
    tanggal_proteksi: '2026-09-17',
    keterangan: 'Menunggu akseptasi.',
    user_create: 'ADMINCONTOH',
    antrean: 'non-premi',
    ...partial,
  }
}

function detail(partial: Partial<ProtectionDetail> = {}): ProtectionDetail {
  return {
    ...protection(),
    nama_tertanggung: 'Tertanggung Contoh',
    polis_mulai: '2026-01-01',
    polis_akhir: '2026-12-31',
    status_akseptasi: '',
    tanggal_akseptasi: '',
    diaksep_oleh: '',
    menunggu_keputusan: true,
    ...partial,
  }
}

function listResponse(partial: Partial<ProtectionListResponse> = {}): ProtectionListResponse {
  return { proteksi: [protection()], total: 1, antrean: 'non-premi', ...partial }
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
        <AcceptQueuePage />
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

it('membuka antrean NON PREMI lebih dulu, bukan PREMI', async () => {
  // Arahnya dipilih sengaja: antrean PREMI menyangkut penagihan premi dan pemiliknya satu
  // peran tertentu.
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')
  expect(calls.some((call) => call.url.includes('antrean=non-premi'))).toBe(true)
})

it('meminta antrean PREMI ke server saat tabnya dipilih', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ antrean: 'premi' })))
  renderPage()

  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('tab', { name: 'Proteksi Klaim PREMI' }))

  await waitFor(() => {
    expect(calls.some((call) => call.url.includes('antrean=premi'))).toBe(true)
  })
})

it('memakai ketujuh judul kolom yang sama dengan section rujukan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')
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

it('mengirim keputusan setuju ke jalur akseptasi', async () => {
  stubFetch((url, init) => {
    if (init?.method === 'PUT') return jsonResponse(200, detail({ status_akseptasi: '1' }))
    if (url.includes('/OPCN.26.0001')) return jsonResponse(200, detail())
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))
  await userEvent.click(await screen.findByRole('button', { name: 'Setujui' }))

  await waitFor(() => {
    const decision = calls.find((call) => call.init?.method === 'PUT')
    expect(decision?.url).toContain('/OPCN.26.0001/akseptasi')
    expect(decision?.init?.body).toContain('setuju')
  })
})

it('mengirim keputusan tolak, bukan setuju, saat tombol Tolak ditekan', async () => {
  stubFetch((url, init) => {
    if (init?.method === 'PUT') return jsonResponse(200, detail({ status_akseptasi: '2' }))
    if (url.includes('/OPCN.26.0001')) return jsonResponse(200, detail())
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))
  await userEvent.click(await screen.findByRole('button', { name: 'Tolak' }))

  await waitFor(() => {
    const decision = calls.find((call) => call.init?.method === 'PUT')
    expect(decision?.init?.body).toContain('tolak')
  })
})

// Antrean BERSAMA: baris dapat diputuskan petugas lain kapan saja. Menampilkan tombol yang
// pasti ditolak hanya membuat pengguna mencobanya.
it('menyembunyikan tombol keputusan untuk proteksi yang sudah diputuskan', async () => {
  stubFetch((url) => {
    if (url.includes('/OPCN.26.0001')) {
      return jsonResponse(
        200,
        detail({
          status_akseptasi: '1',
          diaksep_oleh: 'KOLEKSICONTOH',
          tanggal_akseptasi: '2026-09-20',
          menunggu_keputusan: false,
        }),
      )
    }
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))

  expect(await screen.findByText(/disetujui/)).toBeInTheDocument()
  expect(screen.getByText(/KOLEKSICONTOH/)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Setujui' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Tolak' })).not.toBeInTheDocument()
})

// Petugas kedua yang kalah cepat BUKAN sedang melakukan kesalahan; pesannya harus
// menyatakan keputusannya sudah diambil, bukan galat teknis.
it('menyampaikan bahwa keputusan sudah diambil petugas lain', async () => {
  stubFetch((url, init) => {
    if (init?.method === 'PUT') {
      return jsonResponse(409, {
        kode: 'konflik',
        pesan:
          'Permintaan proteksi ini sudah diakseptasi. Muat ulang daftar untuk melihat keputusannya.',
      })
    }
    if (url.includes('/OPCN.26.0001')) return jsonResponse(200, detail())
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))
  await userEvent.click(await screen.findByRole('button', { name: 'Setujui' }))

  expect(await screen.findByText(/sudah diakseptasi/)).toBeInTheDocument()
})

it('menuntun memilih entitas lebih dulu dan tidak menembak server', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})
