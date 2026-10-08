import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { BandingHargaTabs } from './BandingHargaTabs'
import { DecisionConfirm } from './DecisionConfirm'
import { DecisionPanel } from './DecisionPanel'
import { DocumentPanel } from './DocumentPanel'
import { InboxBandingHargaSalvagePage } from './InboxBandingHargaSalvagePage'
import type { AppealRow, DecisionColumn, MetadataResponse, Tab } from './types'

/**
 * Uji tambahan Inbox Banding Harga Salvage: galat halaman, Refresh, paginasi, dan cabang
 * panel-panelnya. Seluruh nilai KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-banding-harga-salvage'

const TAB_REQUEST: Tab = {
  kode: 'request-banding-harga',
  nama: 'Request Banding Harga',
  keterangan: 'Belum diputus.',
  parameter_pega: '1',
  kolom: [
    { kunci: 'no_klaim', judul: 'No Klaim', angka: false },
    // Kolom Detail Object yang menjadikan tab ini tab keputusan, bukan tab riwayat.
    { kunci: 'detail_object', judul: 'Detail Object', angka: false },
    { kunci: 'harga_barang', judul: 'Harga Barang', angka: true },
  ],
}

const META: MetadataResponse = {
  tab: [TAB_REQUEST],
  tab_bawaan: 'request-banding-harga',
  label_cari: 'Cari No Klaim',
  petunjuk_cari: '',
  selisih_terencana: [],
  keterbatasan: [],
  kolom_rincian: [],
  portal: 'ASM',
}

function row(over: Partial<AppealRow> = {}): AppealRow {
  return {
    no_klaim: 'PNCN.26.0451',
    tanggal_request: '2026-08-30',
    detail_object: 'PNC-0451/1',
    nama_barang: '',
    harga_barang: 'abc',
    harga_request: '9750000',
    note_request: '',
    note_checker: '',
    aging: '',
    object_name: '',
    lokasi_salvage: '',
    pic: '',
    id_salvage: '451',
    nama_komite: 'KOMITE',
    ...over,
  }
}

type Answer = { status?: number; body?: unknown; fail?: boolean }
let calls: string[] = []

function stub(answer: (url: string, init?: RequestInit) => Answer | null) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push(url)
    const reply = answer(url, init) ?? defaults(url)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function defaults(url: string): Answer {
  if (url === `${PATH}/tab`) return { body: META }
  if (url === `${PATH}/ringkas`) return { body: { baris: [], portal: 'ASM' } }
  const page = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
  return {
    body: {
      tab: TAB_REQUEST,
      baris: [row({ no_klaim: `PNCN.26.0${page}` }), row({ no_klaim: 'PNCN.26.9999', harga_barang: '100', detail_object: 'X/2' })],
      paginasi: { halaman: page, ukuran: 2, total: 4, total_halaman: 2 },
      penyaring: { cari: '' },
      portal: 'ASM',
    },
  }
}

function wrap(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('halaman', () => {
  it('menyatakan layar tidak dapat dibuka saat bentuknya gagal dimuat', async () => {
    stub((url) => (url === `${PATH}/tab` ? { status: 500, body: { kode: 'x', pesan: 'Bentuk rusak.' } } : null))
    wrap(<InboxBandingHargaSalvagePage />)

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Bentuk rusak.')).toBeInTheDocument()
  })

  it('memakai pesan umum saat antrean gagal karena jaringan', async () => {
    stub((url) => (url.startsWith(`${PATH}?`) || url === PATH ? { fail: true } : null))
    wrap(<InboxBandingHargaSalvagePage />)

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })

  it('memuat ulang daftar dan ringkasan, mengurutkan, dan berpindah halaman', async () => {
    stub(() => null)
    const user = userEvent.setup()
    wrap(<InboxBandingHargaSalvagePage />)

    const table = await screen.findByRole('table', { name: 'Daftar Request Banding Harga' })
    // Harga yang bukan angka digambar apa adanya, bukan nol.
    expect(within(table).getByText('abc')).toBeInTheDocument()

    await user.click(within(table).getByRole('button', { name: 'Harga Barang' }))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('PNCN.26.9999')

    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => {
      const fresh = calls.slice(before)
      expect(fresh.some((url) => url === `${PATH}/ringkas`)).toBe(true)
      expect(fresh.some((url) => url.startsWith(PATH + '?') || url === PATH)).toBe(true)
    })

    await user.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
    expect(await within(table).findByText('PNCN.26.02')).toBeInTheDocument()
    expect(calls.some((url) => url.includes('halaman=2'))).toBe(true)
  })

  it('menutup kabar keputusan lewat tombol Tutup', async () => {
    stub((url, init) =>
      init?.method === 'POST' && url === `${PATH}/keputusan`
        ? { body: { tersimpan: true, harga_diterapkan: false, dokumen_ditandai: false, pesan: 'Keputusan tersimpan.', portal: 'ASM' } }
        : null,
    )
    const user = userEvent.setup()
    wrap(<InboxBandingHargaSalvagePage />)

    await screen.findByRole('table', { name: 'Daftar Request Banding Harga' })
    await user.click(screen.getAllByRole('button', { name: 'Approve' })[0]!)
    await user.click(screen.getByRole('button', { name: 'Simpan Approve' }))
    const notice = await screen.findByText('Keputusan tersimpan.')

    await user.click(within(notice.parentElement as HTMLElement).getByRole('button', { name: 'Tutup' }))
    expect(screen.queryByText('Keputusan tersimpan.')).not.toBeInTheDocument()
  })
})

describe('DecisionConfirm', () => {
  it('menggabungkan pelanggaran per isian ke pesan validasi', async () => {
    stub(() => ({
      status: 422,
      body: {
        kode: 'validasi_gagal',
        pesan: 'Perbaiki yang ditandai.',
        detail: [{ field: 'detail_object', pesan: 'Detail object wajib.' }],
      },
    }))
    const user = userEvent.setup()
    wrap(<DecisionConfirm row={row()} approve={false} onCancel={() => {}} onDecided={() => {}} />)

    expect(screen.getByText('Barang tanpa nama · Detail Object PNC-0451/1')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Simpan Reject' }))
    expect(await screen.findByText('Keputusan belum dapat disimpan')).toBeInTheDocument()
    expect(screen.getByText('Perbaiki yang ditandai. Detail object wajib.')).toBeInTheDocument()
  })

  it('memakai pesan umum untuk galat jaringan', async () => {
    stub(() => ({ fail: true }))
    const user = userEvent.setup()
    wrap(<DecisionConfirm row={row()} approve onCancel={() => {}} onDecided={() => {}} />)

    await user.click(screen.getByRole('button', { name: 'Simpan Approve' }))
    expect(await screen.findByText('Keputusan tidak tersimpan')).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })
})

describe('DecisionPanel', () => {
  const COLUMNS: DecisionColumn[] = [
    { kunci: 'detail_object', judul: 'Detail Object', angka: false },
    { kunci: 'harga_barang', judul: 'Harga Barang', angka: true },
    { kunci: 'jawaban_checker', judul: 'Jawaban Checker', angka: false },
    { kunci: 'nama_checker', judul: 'Nama Checker', angka: false },
  ]

  it('mewarnai lencana per kode, menulis tanda pisah, dan mengurutkan', async () => {
    stub(() => ({
      body: {
        no_klaim: 'PNCN.26.0440',
        baris: [
          { detail_object: 'B/2', harga_barang: 'xx', jawaban_checker_kode: '0', jawaban_checker: 'Tidak setuju', nama_checker: '' },
          { detail_object: 'A/1', harga_barang: '1000', jawaban_checker_kode: '9', jawaban_checker: 'PROSES', nama_checker: 'KOMITE' },
        ],
        portal: 'ASM',
      },
    }))
    const user = userEvent.setup()
    wrap(<DecisionPanel claimNo="PNCN.26.0440" columns={COLUMNS} />)

    const table = await screen.findByRole('table', { name: 'Keputusan banding klaim PNCN.26.0440' })
    expect(within(table).getByText('Tidak setuju').className).toContain('bg-rose-50')
    expect(within(table).getByText('PROSES').className).toContain('bg-slate-100')
    expect(within(table).getByText('xx')).toBeInTheDocument()
    expect(within(table).getByText('—')).toBeInTheDocument()

    await user.click(within(table).getByRole('button', { name: 'Detail Object' }))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('A/1')
  })

  it('memakai pesan umum saat riwayat gagal karena jaringan', async () => {
    stub(() => ({ fail: true }))
    wrap(<DecisionPanel claimNo="PNCN.26.0440" columns={COLUMNS} />)

    expect(await screen.findByText('Riwayat tidak dapat dimuat')).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })
})

describe('DocumentPanel', () => {
  const SCOPE = { no_klaim: 'PNCN.26.0451', detail_object: 'PNC-0451/1', id_salvage: '451', nama_barang: '' }

  it('menamai berkas dan barang tanpa nama, lalu mengurutkan', async () => {
    stub(() => ({
      body: {
        baris: [
          { id: '2', kategori: 'Zeta', nama: '', tanggal_unggah: '2026-09-02' },
          { id: '1', kategori: 'Alfa', nama: 'b.pdf', tanggal_unggah: '2026-09-01' },
        ],
        portal: 'ASM',
      },
    }))
    const user = userEvent.setup()
    wrap(<DocumentPanel scope={SCOPE} onClose={() => {}} />)

    expect(screen.getByText(/Barang tanpa nama · Detail Object/)).toBeInTheDocument()
    const table = await screen.findByRole('table', { name: 'Dokumen banding PNCN.26.0451' })
    const unnamed = within(table).getByRole('link', { name: 'Berkas tanpa nama' })
    expect(unnamed).not.toHaveAttribute('download')

    await user.click(within(table).getByRole('button', { name: 'Kategori' }))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('Alfa')
    await user.click(within(table).getByRole('button', { name: 'Nama' }))
    await user.click(within(table).getByRole('button', { name: 'Tanggal' }))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('b.pdf')
  })

  it('memakai pesan umum saat dokumen gagal karena jaringan', async () => {
    stub(() => ({ fail: true }))
    wrap(<DocumentPanel scope={SCOPE} onClose={() => {}} />)

    expect(await screen.findByText('Dokumen tidak dapat dimuat')).toBeInTheDocument()
  })
})


describe('BandingHargaTabs — lencana jumlah', () => {
  const TABS = [
    {
      kode: 'request-banding-harga',
      nama: 'Request Banding Harga',
      keterangan: '',
      parameter_pega: '1',
      kolom: [],
    },
    {
      kode: 'history-cheker',
      nama: 'History Cheker',
      keterangan: '',
      parameter_pega: '2',
      kolom: [],
    },
  ]

  /*
    Diuji sebagai KOMPONEN, bukan lewat halaman.

    Uji tingkat halaman sempat ditulis untuk ini dan ternyata membuktikan nol: ia menegaskan
    lencananya tidak ada, padahal `/ringkas` memang belum tiba pada saat itu — jadi ia lulus
    apa pun aturannya. Ketahuan saat aturannya dilumpuhkan sementara dan ujinya tetap lulus.

    Dengan prop yang diserahkan langsung, tidak ada yang perlu ditunggu.
  */
  it('tidak menggambar lencana ketika jumlahnya nol', () => {
    render(
      <BandingHargaTabs
        tabs={TABS}
        active="request-banding-harga"
        onSelect={() => {}}
        counts={[
          { status_salvage: 'Request Banding Harga', tab: 'request-banding-harga', jumlah: 0 },
          { status_salvage: 'History Cheker', tab: 'history-cheker', jumlah: 0 },
        ]}
      />,
    )

    expect(screen.getByRole('tab', { name: 'Request Banding Harga' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'History Cheker' })).toBeInTheDocument()
  })

  // Yang BERISI tetap dilencanai — di situlah angkanya menjawab "berapa yang menunggu saya".
  it('menggambar lencana ketika antreannya berisi', () => {
    render(
      <BandingHargaTabs
        tabs={TABS}
        active="request-banding-harga"
        onSelect={() => {}}
        counts={[
          { status_salvage: 'Request Banding Harga', tab: 'request-banding-harga', jumlah: 3 },
          { status_salvage: 'History Cheker', tab: 'history-cheker', jumlah: 0 },
        ]}
      />,
    )

    expect(screen.getByRole('tab', { name: 'Request Banding Harga 3' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'History Cheker' })).toBeInTheDocument()
  })

  // Tanpa angka sama sekali — keadaan sebelum `/ringkas` tiba.
  it('tidak menggambar lencana sebelum angkanya tiba', () => {
    render(
      <BandingHargaTabs tabs={TABS} active="request-banding-harga" onSelect={() => {}} />,
    )

    expect(screen.getByRole('tab', { name: 'Request Banding Harga' })).toBeInTheDocument()
  })
})
