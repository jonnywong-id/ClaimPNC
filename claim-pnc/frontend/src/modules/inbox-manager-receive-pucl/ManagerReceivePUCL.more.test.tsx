import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InputReceiveDocumentPage } from './InputReceiveDocumentPage'
import { ManagerReceivePUCLPage } from './ManagerReceivePUCLPage'
import type { MetadataResponse, Tab, WorkItem } from './types'

/**
 * Uji tambahan Inbox Manager Receive/PUCL: galat, tab terhalang, ekspor, paginasi, sel
 * nomor case, dan layar kerja dokumen. Seluruh nilai KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-manager-receive-pucl'

const TAB_RECEIVE: Tab = {
  kode: '1',
  nama: 'Receive',
  keterangan: 'Berkas penerimaan dokumen.',
  kolom: [
    { kunci: 'no_case', judul: 'CaseID' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
  ] as Tab['kolom'],
  antrean_bersama: false,
  buka_layar_kerja: true,
  buka_layar_klaim: false,
  terhalang: false,
}

const TAB_BLOCKED: Tab = {
  ...TAB_RECEIVE,
  kode: '3',
  nama: 'Receive Lain',
  terhalang: true,
  alasan_terhalang: 'Report Definition-nya hilang dari export.',
}

const META: MetadataResponse = {
  tab: [TAB_RECEIVE, TAB_BLOCKED, { ...TAB_BLOCKED, kode: '4', nama: 'Tanpa Pemilik' }],
  tab_bawaan: '1',
  portal: 'ASM',
}
// Tab terhalang pertama menyebut pemiliknya; yang kedua tidak.
;(META.tab[1] as Tab).pemilik_penghalang = 'Tim Pega'

function item(over: Partial<WorkItem> = {}): WorkItem {
  return {
    referensi: 'REF 1',
    no_case: 'RCV-1',
    no_polis: '',
    no_klaim_pnc: '',
    nama_tertanggung: 'Tertanggung Satu',
    tanggal_kejadian: '',
    jenis_klaim: '',
    nama_pengirim: '',
    tanggal_terima_dokumen: '',
    jumlah_lembar_dokumen: '',
    tanggal_masuk_inbox: '',
    deskripsi_analyst: '',
    rcl_pucl: '',
    status_rcl_pucl: '',
    tanggal_cetak_surat: '',
    lama_klaim: '',
    status_kadaluarsa: '',
    layar_klaim_siap: true,
    ...over,
  }
}

type Answer = Response | { status?: number; body?: unknown; fail?: boolean | 'empty' } | null
let calls: string[] = []

function stub(answer: (url: string) => Answer) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const reply = answer(url) ?? defaults(url)
    if (reply instanceof Response) return Promise.resolve(reply)
    if (reply.fail === 'empty') return Promise.reject(new Error(''))
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function defaults(url: string): NonNullable<Answer> {
  if (url === `${PATH}/tab`) return { body: META }
  const page = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
  return {
    body: {
      tab: TAB_RECEIVE,
      baris: [
        item({ no_case: `RCV-${page}` }),
        item({ referensi: '', no_case: 'RCV-TANPA-KUNCI' }),
        item({ referensi: 'REF 3', no_case: '' }),
      ],
      paginasi: { halaman: page, ukuran: 3, total: 6, total_halaman: 2 },
      portal: 'ASM',
    },
  }
}

function renderAt(path: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/inbox-manager-receive-pucl" element={<ManagerReceivePUCLPage />} />
          <Route
            path="/inbox-manager-receive-pucl/dokumen/:referensi"
            element={<InputReceiveDocumentPage />}
          />
          <Route
            path="/inbox-manager-receive-pucl/dokumen"
            element={<InputReceiveDocumentPage />}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

let clicked: string[] = []

beforeEach(() => {
  calls = []
  clicked = []
  Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() })
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push(this.download)
  })
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSession.getState().clear()
})

describe('ManagerReceivePUCLPage', () => {
  it('menyatakan layar tidak dapat dibuka saat bentuknya gagal dimuat', async () => {
    stub((url) =>
      url === `${PATH}/tab` ? { status: 500, body: { kode: 'x', pesan: 'Bentuk rusak.' } } : null,
    )
    renderAt('/inbox-manager-receive-pucl')

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Bentuk rusak.')).toBeInTheDocument()
  })

  it('menyebut galat jaringan pada antrean', async () => {
    stub((url) => (url === `${PATH}/tab` ? null : { fail: true }))
    renderAt('/inbox-manager-receive-pucl')

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
  })

  it('menggambar sel nomor case menurut kuncinya dan berpindah halaman', async () => {
    stub(() => null)
    const user = userEvent.setup()
    renderAt('/inbox-manager-receive-pucl')

    const table = await screen.findByRole('table', { name: 'Antrean Receive' })
    expect(within(table).getByRole('button', { name: 'RCV-1' })).toBeInTheDocument()
    // Tanpa kunci teknis nomornya teks biasa; tanpa nomor ia tanda pisah.
    expect(within(table).queryByRole('button', { name: 'RCV-TANPA-KUNCI' })).not.toBeInTheDocument()
    expect(within(table).getByText('RCV-TANPA-KUNCI')).toBeInTheDocument()
    expect(within(table).getAllByText('—').length).toBeGreaterThan(0)

    await user.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
    expect(await within(table).findByRole('button', { name: 'RCV-2' })).toBeInTheDocument()
    expect(calls).toContain(`${PATH}?tab=1&halaman=2`)
  })

  it('menjelaskan tab terhalang tanpa meminta isinya, dengan dan tanpa pemilik', async () => {
    stub(() => null)
    const user = userEvent.setup()
    renderAt('/inbox-manager-receive-pucl')

    await screen.findByRole('table', { name: 'Antrean Receive' })
    const blockedTab = screen.getByRole('tab', { name: /Receive Lain/ })
    expect(blockedTab).toHaveAttribute('title', 'Report Definition-nya hilang dari export.')
    expect(within(blockedTab).getByText('belum tersedia')).toBeInTheDocument()

    const before = calls.length
    await user.click(blockedTab)
    expect(await screen.findByText('Receive Lain belum tersedia')).toBeInTheDocument()
    expect(screen.getByText('Tim Pega')).toBeInTheDocument()
    expect(calls.slice(before).some((url) => url.includes('tab=3'))).toBe(false)

    await user.click(screen.getByRole('tab', { name: /Tanpa Pemilik/ }))
    expect(await screen.findByText('Tanpa Pemilik belum tersedia')).toBeInTheDocument()
    expect(screen.queryByText('Menunggu:')).not.toBeInTheDocument()
  })

  it('tidak menggambar tabel saat tab bawaan tidak diketahui', async () => {
    stub((url) => (url === `${PATH}/tab` ? { body: { ...META, tab_bawaan: '' } } : null))
    renderAt('/inbox-manager-receive-pucl')

    await screen.findByRole('tab', { name: /^Receive$/ })
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })
})

describe('InputReceiveDocumentPage', () => {
  const DOC = {
    referensi: 'REF 1',
    no_case: 'RCV-1',
    no_klaim_pnc: '',
    jenis_klaim: '',
    status_kerja: '',
    kelompok: [
      {
        judul: 'Kejadian',
        isian: [
          { kunci: 'kronologis', judul: 'Kronologis', bertingkat: true },
          {
            kunci: 'polis_leader',
            judul: 'Polis Leader',
            terhalang: true,
            alasan_terhalang: 'Di blob.',
          },
        ],
      },
    ],
    nilai: { kronologis: '' },
    tindakan: [],
    portal: 'ASM',
  }

  it('menulis tanda pisah untuk isian kosong dan tidak menyebut pemilik yang tidak ada', async () => {
    stub(() => ({ body: DOC }))
    renderAt('/inbox-manager-receive-pucl/dokumen/REF%201')

    expect(await screen.findByText('Berkas RCV-1')).toBeInTheDocument()
    expect(screen.getByText('belum diregistrasi')).toBeInTheDocument()
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(3)
    expect(screen.getByText('1 isian belum terbawa dari Pega')).toBeInTheDocument()
    expect(screen.queryByText('Menunggu:')).not.toBeInTheDocument()
  })

  it('kembali ke antrean tanpa parameter bila alamatnya tidak membawa tab', async () => {
    stub((url) => (url.startsWith(`${PATH}/dokumen`) ? { body: DOC } : null))
    const user = userEvent.setup()
    renderAt('/inbox-manager-receive-pucl/dokumen/REF%201')

    await screen.findByText('Berkas RCV-1')
    await user.click(screen.getByRole('button', { name: 'Kembali ke antrean' }))
    expect(await screen.findByRole('table', { name: 'Antrean Receive' })).toBeInTheDocument()
  })

  it('menyebut galat jaringan dengan pesannya sendiri', async () => {
    stub(() => ({ fail: true }))
    renderAt('/inbox-manager-receive-pucl/dokumen/REF%201')

    expect(await screen.findByText('Berkas tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
  })
})
