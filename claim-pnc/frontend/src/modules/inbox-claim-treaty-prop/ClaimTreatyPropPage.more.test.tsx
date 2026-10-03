import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimTreatyPropPage } from './ClaimTreatyPropPage'
import type { Tab, WorkItem } from './types'

/**
 * Uji tambahan Inbox Claim Treaty Prop: penyaring See All Claim, paginasi, tombol Create,
 * dan cabang galat. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-claim-treaty-prop'

const TAB_SAYA: Tab = {
  kode: '4',
  nama: 'Prop Treaty-in Saya',
  keterangan: 'Pekerjaan milik Anda.',
  kolom: [
    { kunci: 'claim_id', judul: 'Claim ID' },
    { kunci: 'nama_tertanggung', judul: 'Insured Name' },
  ],
  hanya_milik_saya: true,
  pakai_lihat_semua: true,
  terhalang: false,
}

const TAB_HALANG: Tab = {
  kode: '5',
  nama: 'Terhalang Tanpa Pemilik',
  keterangan: 'Belum dapat dibaca.',
  kolom: [],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  terhalang: true,
  alasan_terhalang: 'Sumbernya belum ada.',
}

const METADATA = { tab: [TAB_SAYA, TAB_HALANG], tab_bawaan: '4', selisih_terencana: [], portal: 'ASM' }

function row(claim: string): WorkItem {
  return {
    referensi: `REF-${claim}`,
    claim_id: claim,
    id_master: '',
    no_polis: '',
    tanggal_kejadian: '',
    nama_bisnis: '',
    sumber_bisnis: '',
    ceding_co: '',
    nama_tertanggung: 'PT Contoh',
    subjectivity: '',
    operator_pengubah: '',
    status_kerja: '',
  }
}

type Answer = Response | Promise<Response> | 'rusak' | undefined

let urls: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function list(rows: WorkItem[], halaman = 1, total = rows.length, totalHalaman = 1) {
  return json(200, {
    tab: TAB_SAYA,
    baris: rows,
    paginasi: { halaman, ukuran: 25, total, total_halaman: totalHalaman },
    penyaring: { lihat_semua: false },
    portal: 'ASM',
  })
}

function installFetch(answer: (url: string, method: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    urls.push(`${method} ${url}`)
    const custom = answer(url, method)
    if (custom === 'rusak') {
      return Promise.resolve({
        get status(): number {
          throw new TypeError('rusak')
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (url === `${PATH}/tab`) return Promise.resolve(json(200, METADATA))
    if (url === `${PATH}/klaim`) return Promise.resolve(json(200, {}))
    return Promise.resolve(list([row('CLMP-1')]))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ClaimTreatyPropPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const isList = (entry: string) => entry === `GET ${PATH}` || entry.startsWith(`GET ${PATH}?`)

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('penyaring See All Claim', () => {
  it('menyebut cakupan milik sendiri lalu mengirim lihat_semua saat dicentang', async () => {
    installFetch((url) => (url.startsWith(`${PATH}?`) || url === PATH ? list([]) : undefined))
    show()

    expect(await screen.findByText('Antrean ini hanya berisi pekerjaan milik Anda.')).toBeInTheDocument()
    expect(
      await screen.findByText(
        'Tidak ada pekerjaan klaim treaty milik Anda. Centang "See All Claim" untuk melihat penugasan seluruh petugas.',
      ),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('checkbox', { name: 'See All Claim' }))

    expect(
      await screen.findByText('Menampilkan penugasan seluruh petugas, termasuk yang bukan milik Anda.'),
    ).toBeInTheDocument()
    expect(screen.queryByText('Antrean ini hanya berisi pekerjaan milik Anda.')).not.toBeInTheDocument()
    await waitFor(() => expect(urls).toContain(`GET ${PATH}?lihat_semua=1`))
    expect(await screen.findByText('Antrean ini sedang kosong.')).toBeInTheDocument()
  })
})

describe('paginasi', () => {
  it('maju dan mundur satu halaman', async () => {
    installFetch((url) => {
      if (!(url.startsWith(`${PATH}?`) || url === PATH)) return undefined
      const halaman = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
      return list([row(`CLMP-H${halaman}`)], halaman, 30, 2)
    })
    show()

    expect(await screen.findByText('Menampilkan 1–1 dari 30 baris.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    expect(await screen.findByText('CLMP-H2')).toBeInTheDocument()
    expect(screen.getByText('Menampilkan 26–26 dari 30 baris.')).toBeInTheDocument()
    expect(urls).toContain(`GET ${PATH}?halaman=2`)

    await userEvent.click(screen.getByRole('button', { name: 'Sebelumnya' }))
    expect(await screen.findByText('CLMP-H1')).toBeInTheDocument()
  })

  it('menyebut nol baris bila halaman tidak berisi', async () => {
    installFetch((url) => (url.startsWith(`${PATH}?`) || url === PATH ? list([], 1, 3, 1) : undefined))
    show()

    expect(await screen.findByText('Menampilkan 0–0 dari 3 baris.')).toBeInTheDocument()
  })
})

describe('tab terhalang', () => {
  it('tidak menyebut pemilik penghalang bila server tidak mengirimnya', async () => {
    installFetch()
    show()

    await screen.findByRole('option', { name: 'Terhalang Tanpa Pemilik — belum tersedia' })
    await userEvent.selectOptions(
      screen.getByRole('combobox', { name: 'Antrean klaim treaty proporsional' }),
      '5',
    )
    expect(await screen.findByText('Terhalang Tanpa Pemilik belum tersedia')).toBeInTheDocument()
    expect(screen.getByText('Sumbernya belum ada.')).toBeInTheDocument()
    expect(screen.queryByText('Menunggu:')).not.toBeInTheDocument()
  })
})

describe('tombol Create Claim Treaty Prop', () => {
  it('menyebut keberhasilan bila server menerima permintaan', async () => {
    installFetch()
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Create Claim Treaty Prop' }))

    expect(await screen.findByText('Pembuatan klaim treaty sudah tersedia. Muat ulang layar ini.')).toBeInTheDocument()
    expect(urls).toContain(`POST ${PATH}/klaim`)
  })

  it('menyebut pesan umum bila galatnya bukan galat API', async () => {
    installFetch((url) => (url === `${PATH}/klaim` ? 'rusak' : undefined))
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Create Claim Treaty Prop' }))

    expect(
      await screen.findByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })
})

describe('galat', () => {
  it('menutup layar bila keterangan antrean gagal dimuat', async () => {
    installFetch((url) =>
      url === `${PATH}/tab` ? json(500, { kode: 'galat_internal', pesan: 'Keterangan rusak.' }) : undefined,
    )
    show()

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Keterangan rusak.')).toBeInTheDocument()
  })

  it('menampilkan galat antrean', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}?`) || url === PATH ? json(500, { kode: 'galat_internal', pesan: 'Antrean rusak.' }) : undefined,
    )
    show()

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Antrean rusak.')).toBeInTheDocument()
    expect(urls.filter(isList).length).toBeGreaterThan(0)
  })
})
