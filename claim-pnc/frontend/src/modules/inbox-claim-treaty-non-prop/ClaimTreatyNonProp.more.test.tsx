import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimTreatyNonPropPage } from './ClaimTreatyNonPropPage'
import type { MetadataResponse, PageInfo, Tab, WorkItem } from './types'

const PATH = '/api/inbox-claim-treaty-non-prop'
const SELECT_NAME = /Antrean klaim treaty non-proporsional/

const SAMPLE_PROFILE = {
  identitas: '90000002',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminnonprop1',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/** Antrean milik pemanggil dengan kedua checkbox — tanpa `judul_grid`. */
const TAB_MINE: Tab = {
  kode: '1',
  nama: 'Treaty-In Admin',
  keterangan: 'Klaim milik Anda.',
  kolom: [
    { kunci: 'no_klaim', judul: 'Claim.ID' },
    { kunci: 'nama_tertanggung', judul: 'Insured Name' },
    { kunci: 'tanggal_kejadian', judul: 'Date of Loss' },
  ],
  hanya_milik_saya: true,
  pakai_lihat_semua: true,
  pakai_lihat_tba: true,
  terhalang: false,
}

/** Antrean bersama yang tetap mengenal "See All Claim". */
const TAB_SHARED: Tab = {
  kode: '2',
  nama: 'Treaty-In Bersama',
  keterangan: 'Antrean bersama.',
  kolom: TAB_MINE.kolom,
  hanya_milik_saya: false,
  pakai_lihat_semua: true,
  pakai_lihat_tba: false,
  terhalang: false,
}

/** Antrean terhalang tanpa pemilik penghalang. */
const TAB_BLOCKED: Tab = {
  kode: '3',
  nama: 'Komite',
  keterangan: 'Antrean komite.',
  kolom: [],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  pakai_lihat_tba: false,
  terhalang: true,
  alasan_terhalang: 'Kuerinya tidak ada di export.',
}

const META: MetadataResponse = {
  tab: [TAB_MINE, TAB_SHARED, TAB_BLOCKED],
  tab_bawaan: '1',
  selisih_terencana: [],
  portal: 'ASM',
}

/** Seluruh isinya KARANGAN — `D-69`. */
function item(overrides: Partial<WorkItem> = {}): WorkItem {
  return {
    referensi: 'REF-1',
    no_klaim: 'CLMNP-1',
    master_id: '',
    id_master: '',
    no_polis: '',
    tanggal_kejadian: '2026-08-14',
    nama_bisnis: '',
    sumber_bisnis: '',
    ceding_co: '',
    nama_tertanggung: 'Contoh Tertanggung',
    dibuat_pada: '',
    aging: 1,
    operator_pembuat: '',
    operator_pengubah: '',
    ...overrides,
  }
}

function listBody(tab: Tab, baris: WorkItem[], paginasi: Partial<PageInfo> = {}) {
  return {
    tab,
    baris,
    paginasi: { halaman: 1, ukuran: 25, total: baris.length, total_halaman: 1, ...paginasi },
    penyaring: { lihat_semua: false, lihat_tba: false },
    portal: 'ASM',
  }
}

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

type Answer = Response | Error | null

let calls: string[] = []
let clicked: string[] = []

/** stub menjawab permintaan; `null` memakai jawaban bawaan, `Error` menolak permintaan. */
function stub(answer: (url: string) => Answer) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const chosen = answer(url)
    if (chosen instanceof Error) return Promise.reject(chosen)
    if (chosen) return Promise.resolve(chosen)
    if (url === `${PATH}/tab`) return Promise.resolve(json(200, META))
    return Promise.resolve(json(200, listBody(TAB_MINE, [item()])))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-claim-treaty-non-prop']}>
        <Routes>
          <Route path="/inbox-claim-treaty-non-prop" element={<ClaimTreatyNonPropPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  clicked = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
  vi.stubGlobal(
    'URL',
    Object.assign(URL, {
      createObjectURL: () => 'blob:uji',
      revokeObjectURL: () => undefined,
    }),
  )
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push(this.download)
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('keadaan layar sebelum antrean tampil', () => {
  it('meminta pengguna memilih entitas lebih dulu tanpa meminta apa pun ke server', () => {
    useSelectedPortal.getState().clear()
    stub(() => null)
    renderPage()

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export Data' })).toBeDisabled()
    expect(calls).toEqual([])
  })

  it('menyatakan layar tidak dapat dibuka saat bentuknya gagal dimuat', async () => {
    stub((url) => (url === `${PATH}/tab` ? json(500, { kode: 'x', pesan: 'Bentuk rusak.' }) : null))
    renderPage()

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Bentuk rusak.')).toBeInTheDocument()
  })

  it('menyebut galat antrean dengan pesan dari server', async () => {
    stub((url) => (url === PATH ? json(500, { kode: 'x', pesan: 'Antrean rusak.' }) : null))
    renderPage()

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Antrean rusak.')).toBeInTheDocument()
  })
})

describe('grid dan penyaring', () => {
  it('memakai nama tab sebagai judul grid bila judul gridnya tidak dikirim', async () => {
    stub(() => null)
    renderPage()

    expect(await screen.findByRole('heading', { name: 'Treaty-In Admin' })).toBeInTheDocument()
  })

  it('menulis tanda pisah untuk nomor klaim dan sel yang kosong', async () => {
    stub((url) =>
      url === PATH
        ? json(200, listBody(TAB_MINE, [item({ no_klaim: '', nama_tertanggung: '' })]))
        : null,
    )
    renderPage()

    const table = await screen.findByRole('table')
    await within(table).findByText('14 Agustus 2026')
    expect(within(table).getAllByText('—')).toHaveLength(2)
    expect(within(table).queryByRole('link')).not.toBeInTheDocument()
  })

  it('menjelaskan antrean kosong menurut keempat kombinasi penyaringnya', async () => {
    stub((url) => (url === `${PATH}/tab` ? null : json(200, listBody(TAB_MINE, []))))
    const user = userEvent.setup()
    renderPage()

    expect(
      await screen.findByText(/Tidak ada pekerjaan klaim treaty non-prop milik Anda/),
    ).toBeInTheDocument()
    expect(screen.getByText('Antrean ini hanya berisi pekerjaan milik Anda.')).toBeInTheDocument()

    await user.click(screen.getByLabelText('See TBA Claim'))
    expect(
      await screen.findByText(/belum terbit pada penyaring ini\. Centang "See All Claim"/),
    ).toBeInTheDocument()

    await user.click(screen.getByLabelText('See All Claim'))
    expect(
      await screen.findByText(
        'Tidak ada klaim yang nomor polisnya belum terbit pada penyaring ini.',
      ),
    ).toBeInTheDocument()
    expect(screen.getByText(/milik seluruh petugas yang nomor polisnya BELUM/)).toBeInTheDocument()

    await user.click(screen.getByLabelText('See TBA Claim'))
    expect(
      await screen.findByText(
        'Menampilkan penugasan seluruh petugas, termasuk yang bukan milik Anda.',
      ),
    ).toBeInTheDocument()
    expect(await screen.findByText('Antrean ini sedang kosong.')).toBeInTheDocument()
  })

  it('menyebut antrean bersama pada tab yang hanya mengenal "See All Claim"', async () => {
    stub((url) => (url.includes('tab=2') ? json(200, listBody(TAB_SHARED, [])) : null))
    const user = userEvent.setup()
    renderPage()

    await screen.findByRole('option', { name: 'Treaty-In Bersama' })
    await user.selectOptions(screen.getByRole('combobox', { name: SELECT_NAME }), '2')

    expect(
      await screen.findByText('Antrean bersama — belum diambil siapa pun.'),
    ).toBeInTheDocument()
    expect(await screen.findByText('Antrean ini sedang kosong.')).toBeInTheDocument()
  })

  it('menampilkan alasan terhalang tanpa menyebut pemilik yang tidak ada', async () => {
    stub(() => null)
    const user = userEvent.setup()
    renderPage()

    await screen.findByRole('option', { name: /Komite/ })
    await user.selectOptions(screen.getByRole('combobox', { name: SELECT_NAME }), '3')

    expect(await screen.findByText('Kuerinya tidak ada di export.')).toBeInTheDocument()
    expect(screen.queryByText('Menunggu:')).not.toBeInTheDocument()
  })
})

describe('paginasi', () => {
  it('berpindah halaman maju dan mundur', async () => {
    stub((url) => {
      if (url === `${PATH}/tab`) return null
      const page = url.includes('halaman=2') ? 2 : 1
      return json(
        200,
        listBody(TAB_MINE, [item({ no_klaim: `CLMNP-${page}` })], {
          halaman: page,
          ukuran: 1,
          total: 2,
          total_halaman: 2,
        }),
      )
    })
    const user = userEvent.setup()
    renderPage()

    expect(await screen.findByText('Menampilkan 1–1 dari 2 baris.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sebelumnya' })).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'Berikutnya' }))
    expect(await screen.findByText('Menampilkan 2–2 dari 2 baris.')).toBeInTheDocument()
    expect(calls).toContain(`${PATH}?halaman=2`)

    await user.click(screen.getByRole('button', { name: 'Sebelumnya' }))
    expect(await screen.findByText('Menampilkan 1–1 dari 2 baris.')).toBeInTheDocument()
  })

  it('menyebut nol baris tampil bila halaman yang terbuka kosong', async () => {
    stub((url) =>
      url === PATH ? json(200, listBody(TAB_MINE, [], { total: 3, total_halaman: 1 })) : null,
    )
    renderPage()

    expect(await screen.findByText('Menampilkan 0–0 dari 3 baris.')).toBeInTheDocument()
  })
})

describe('tombol ekspor', () => {
  async function exportOnce(answer: Answer) {
    stub((url) => (url.startsWith(`${PATH}/ekspor`) ? answer : null))
    const user = userEvent.setup()
    renderPage()

    const button = await screen.findByRole('button', { name: 'Export Data' })
    await waitFor(() => expect(button).toBeEnabled())
    await user.click(button)
  }

  it('memakai pesan dari server saat ekspor ditolak', async () => {
    await exportOnce(json(403, { pesan: 'Ekspor tidak diizinkan.' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Ekspor tidak diizinkan.')
  })

  it('memakai pesan bawaan saat jawaban galat bukan JSON', async () => {
    await exportOnce(new Response('rusak', { status: 500 }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Berkas ekspor tidak dapat diambil.')
  })

  it('memakai pesan umum saat galatnya tidak berpesan', async () => {
    await exportOnce(new Error(''))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.',
    )
  })

  it('memakai nama cadangan tanpa Content-Disposition', async () => {
    await exportOnce(new Response('x', { status: 200 }))
    await waitFor(() => expect(clicked).toEqual(['klaim-treaty-non-prop.csv']))
    expect(calls).toContain(`${PATH}/ekspor`)
  })

  it('memakai nama cadangan saat Content-Disposition tidak menyebut nama berkas', async () => {
    await exportOnce(
      new Response('x', { status: 200, headers: { 'Content-Disposition': 'attachment' } }),
    )
    await waitFor(() => expect(clicked).toEqual(['klaim-treaty-non-prop.csv']))
  })
})

describe('tombol buat klaim', () => {
  it('memberi tahu bila server ternyata sudah dapat membuat klaim', async () => {
    stub((url) => (url === `${PATH}/klaim` ? json(201, {}) : null))
    const user = userEvent.setup()
    renderPage()

    await user.click(screen.getByRole('button', { name: 'Create Claim Treaty Non Prop' }))
    expect(
      await screen.findByText(
        'Pembuatan klaim treaty non-prop sudah tersedia. Muat ulang layar ini.',
      ),
    ).toBeInTheDocument()
  })

  it('menampilkan pesan galat jaringan saat permintaan gagal', async () => {
    stub((url) => (url === `${PATH}/klaim` ? new TypeError('Failed to fetch') : null))
    const user = userEvent.setup()
    renderPage()

    await user.click(screen.getByRole('button', { name: 'Create Claim Treaty Non Prop' }))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Create Claim Treaty Non Prop' })).toBeEnabled(),
    )
    expect(screen.getByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
  })
})
