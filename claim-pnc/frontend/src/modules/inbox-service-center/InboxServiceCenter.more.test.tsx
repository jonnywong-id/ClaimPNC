import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxServiceCenterPage } from './InboxServiceCenterPage'
import { ServiceCenterDetailPage } from './ServiceCenterDetailPage'
import type { DetailResponse, MetadataResponse, ServiceClaim, Tab } from './types'

const PATH = '/api/inbox-service-center'

const SAMPLE_PROFILE = {
  identitas: '90000002',
  nama: 'Contoh Petugas Service Center',
  jenis: 'KARYAWAN',
  login: 'picsc',
  email: 'contoh.picsc@example.invalid',
  perusahaan: 'ASM',
}

const TAB: Tab = {
  kode: 'registrasi-sc',
  nama: 'Registrasi SC',
  keterangan: 'Klaim portal rekanan yang belum pernah diajukan ke komite.',
  parameter_pega: '',
  kolom: [
    { kunci: 'id', judul: 'ID' },
    { kunci: 'tanggal_input', judul: 'Tanggal Input' },
    { kunci: 'nasabah', judul: 'Nasabah' },
  ],
}

const META: MetadataResponse = {
  tab: [TAB],
  tab_bawaan: 'registrasi-sc',
  portal: 'ASM',
  keterbatasan: [],
}

/** Seluruh isinya KARANGAN — `D-69`. */
function row(overrides: Partial<ServiceClaim>): ServiceClaim {
  return {
    id: 'SC-1',
    tanggal_input: '2026-09-22',
    no_polis: 'POLIS-CONTOH',
    nasabah: 'Contoh Nasabah',
    tipe: 'Gadget',
    pic: 'PICSC',
    repair_id: 'R-1',
    no_klaim: '',
    imei: '',
    status_perbaikan: '',
    status_perbaikan_label: '',
    status_persetujuan: '',
    status_persetujuan_label: '',
    ...overrides,
  }
}

type Answer = Response | 'gagal' | null

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

let calls: string[] = []

/** stub menjawab permintaan; `null` berarti memakai jawaban bawaan. */
function stub(answer: (url: string) => Answer) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const chosen = answer(url)
    if (chosen === 'gagal') return Promise.reject(new TypeError('Failed to fetch'))
    if (chosen) return Promise.resolve(chosen)
    if (url === `${PATH}/tab`) return Promise.resolve(json(200, META))
    return Promise.resolve(json(404, {}))
  })
}

function renderAt(path: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/inbox-service-center" element={<InboxServiceCenterPage />} />
          <Route path="/inbox-service-center/:id" element={<ServiceCenterDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function listBody(baris: ServiceClaim[], aktif = true) {
  return {
    tab: TAB,
    baris,
    paginasi: { halaman: 1, ukuran: 10, total: baris.length, total_halaman: 1, aktif },
    penyaring: { cari: '' },
    portal: 'ASM',
  }
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('InboxServiceCenterPage', () => {
  it('meminta pengguna memilih entitas lebih dulu saat portal belum dipilih', () => {
    useSelectedPortal.getState().clear()
    stub(() => null)
    renderAt('/inbox-service-center')

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(calls).toEqual([])
  })

  it('menyatakan layar tidak dapat dibuka dengan pesan umum saat jaringan gagal', async () => {
    stub(() => 'gagal')
    renderAt('/inbox-service-center')

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })

  it('menyebut galat antrean dengan pesan dari server', async () => {
    stub((url) => (url === PATH ? json(500, { kode: 'x', pesan: 'Antrean sedang rusak.' }) : null))
    renderAt('/inbox-service-center')

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Antrean sedang rusak.')).toBeInTheDocument()
  })

  it('menulis tanda pisah untuk sel kosong dan baris tanpa ID, lalu mengurutkan kolom', async () => {
    stub((url) =>
      url === PATH
        ? json(200, listBody([row({ id: '', tanggal_input: null, nasabah: '' }), row({})]))
        : null,
    )
    const user = userEvent.setup()
    renderAt('/inbox-service-center')

    const table = await screen.findByRole('table')
    expect(await within(table).findByText('22 September 2026')).toBeInTheDocument()
    // Baris tanpa ID: sel ID, tanggal, dan nasabah kosong, ditambah tautan rincian.
    expect(within(table).getAllByText('—').length).toBeGreaterThanOrEqual(4)
    expect(within(table).getAllByRole('link', { name: 'Lihat Rincian' })).toHaveLength(1)

    await user.click(within(table).getByRole('button', { name: /Nasabah/ }))
    expect(within(table).getAllByRole('row').length).toBeGreaterThanOrEqual(3)
  })
})

describe('ServiceCenterDetailPage', () => {
  const DETAIL: DetailResponse = {
    klaim: {
      id: 'SC-1',
      nasabah: '',
      customer_name: '',
      status_approval_label: null,
      repair_status_label: '  ',
      detail_part_json: null,
      kosong: null,
    },
    riwayat_progres: [{ tanggal: null, catatan: '', oleh: '' }],
    kelompok: [
      { kode: 'g', judul: 'General Information', isian: [{ kunci: 'kosong', judul: 'X' }] },
    ],
    portal: 'ASM',
  }

  it('meminta pengguna memilih entitas lebih dulu saat portal belum dipilih', () => {
    useSelectedPortal.getState().clear()
    stub(() => null)
    renderAt('/inbox-service-center/SC-1')

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(screen.getByText('Rincian Klaim SC-1')).toBeInTheDocument()
  })

  it('memakai teks cadangan untuk nasabah dan status yang tidak tercatat', async () => {
    stub((url) => (url === `${PATH}/SC-1` ? json(200, DETAIL) : null))
    renderAt('/inbox-service-center/SC-1')

    expect(
      await screen.findByText(
        'Nasabah tidak tercatat · Status persetujuan tidak tercatat · Status perbaikan tidak tercatat',
      ),
    ).toBeInTheDocument()
    // Riwayat tanpa tanggal, catatan, dan pelaku digambar dengan tanda pisah.
    const history = screen.getByRole('table', { name: 'Riwayat progres klaim' })
    expect(within(history).getAllByText('—')).toHaveLength(3)
    // Detail Part kosong tidak digambar sama sekali.
    expect(screen.queryByText('Detail Part')).not.toBeInTheDocument()
  })

  it('memakai nama pelanggan bila nama nasabah kosong', async () => {
    stub((url) =>
      url === `${PATH}/SC-1`
        ? json(200, { ...DETAIL, klaim: { ...DETAIL.klaim, customer_name: 'Contoh Pelanggan' } })
        : null,
    )
    renderAt('/inbox-service-center/SC-1')

    expect(await screen.findByText(/^Contoh Pelanggan · /)).toBeInTheDocument()
  })

  it('menyebut rincian tidak dapat dimuat dengan pesan umum setelah percobaan ulang', async () => {
    stub(() => 'gagal')
    renderAt('/inbox-service-center/SC-1')

    // Galat selain 404 dicoba ulang dua kali sebelum pesannya tampil.
    expect(
      await screen.findByText('Rincian tidak dapat dimuat', undefined, { timeout: 8000 }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
    expect(calls.filter((url) => url === `${PATH}/SC-1`)).toHaveLength(3)
  }, 12000)
})
