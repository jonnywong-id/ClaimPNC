import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { OSClaimPerCabangPage } from './OSClaimPerCabangPage'
import type { ListResponse, WorkItem } from './types'

/**
 * Uji tambahan Inbox OS Claim per Cabang dirender langsung: ekspor, paginasi, alasan
 * penandaan yang tidak dikenal, popup, dan cabang galat. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-os-claim-per-cabang'

function item(partial: Partial<WorkItem>): WorkItem {
  return {
    cabang: 'CILEGON',
    sumbis: '',
    cob: 'Aneka',
    no_polis: '',
    nama_insured: '',
    no_klaim: 'PNC-1',
    tanggal_registrasi: '',
    tanggal_kejadian: '',
    nilai_estimasi: '1000.00',
    tanggal_update_progres: '',
    status_progres_1: '',
    status_progres_2: '',
    pic: '',
    adjuster: '',
    col: '',
    aging_hari: 5,
    perlu_perhatian: false,
    progres_mandek: false,
    catatan_progres: '',
    ...partial,
  }
}

function list(rows: WorkItem[], halaman = 1, total = rows.length, totalHalaman = 1): ListResponse {
  return {
    data: rows,
    cabang: { kode: '100099', nama: '' },
    paginasi: { halaman, ukuran: 25, total, total_halaman: totalHalaman },
    ambang_aging: 180,
    selisih_terencana: [],
    portal: 'ASM',
  }
}

type Answer = Response | Promise<Response> | 'putus' | 'aneh' | undefined

let urls: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: (url: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    const custom = answer(url)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom === 'aneh') {
      return Promise.resolve({
        get status(): number {
          throw 'bukan galat'
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (url === `${PATH}/ekspor`) return Promise.resolve(new Response('a\n', { status: 200 }))
    if (url.startsWith(`${PATH}/PNC-`)) return Promise.resolve(json(404, { kode: 'klaim_tidak_ditemukan', pesan: 'Klaim tidak ada.' }))
    return Promise.resolve(json(200, list([item({ no_klaim: 'PNC-2', aging_hari: 9 }), item({})])))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <OSClaimPerCabangPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSelectedPortal.getState().clear()
})

describe('daftar', () => {
  it('meminta portal dipilih lebih dulu', () => {
    useSelectedPortal.getState().clear()
    installFetch()
    show()

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export To Excel' })).toBeDisabled()
  })

  it('memakai kode cabang bila namanya kosong, dan tidak menampilkan selisih kosong', async () => {
    installFetch()
    show()

    expect(await screen.findByText('( CABANG 100099 )')).toBeInTheDocument()
    expect(screen.queryByText('Yang berbeda dari layar lama')).not.toBeInTheDocument()
  })

  it('menyebut penandaan tanpa sebab yang dikenal apa adanya', async () => {
    installFetch((url) =>
      url === PATH ? json(200, list([item({ perlu_perhatian: true, aging_hari: 5, progres_mandek: false })])) : undefined,
    )
    show()

    const cell = await screen.findByText('PNC-1')
    expect(cell).toHaveAttribute('title', 'Perlu perhatian.')
  })

  it('mengurutkan menurut setiap kolom data', async () => {
    installFetch()
    show()
    const table = await screen.findByRole('table')

    const aging = within(table).getByRole('columnheader', { name: /^Aging/ })
    await userEvent.click(within(aging).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('PNC-1')

    for (const header of within(table).getAllByRole('columnheader')) {
      const button = within(header).queryByRole('button')
      if (button) await userEvent.click(button)
      if (button) expect(header).not.toHaveAttribute('aria-sort', 'none')
    }
  })

  it('berpindah halaman maju dan mundur, dan menyebut halaman tanpa baris', async () => {
    installFetch((url) => {
      if (!(url === PATH || url.startsWith(`${PATH}?`))) return undefined
      const halaman = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
      return json(200, list(halaman === 3 ? [] : [item({ no_klaim: `PNC-H${halaman}` })], halaman, 30, 3))
    })
    show()

    expect(await screen.findByText('Menampilkan 1–1 dari 30 baris.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    expect(await screen.findByText('PNC-H2')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    // Halaman tanpa baris: awalnya nol, akhirnya tetap dihitung dari halaman itu.
    expect(await screen.findByText('Menampilkan 0–50 dari 30 baris.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Sebelumnya' }))
    expect(await screen.findByText('PNC-H2')).toBeInTheDocument()
    expect(urls).toContain(`${PATH}?halaman=3`)
  })

  it.each([
    { name: 'galat API', answer: (): Answer => json(500, { kode: 'x', pesan: 'Basis data sibuk.' }), text: 'Basis data sibuk.' },
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
    { name: 'galat bukan Error', answer: (): Answer => 'aneh', text: 'Terjadi kesalahan yang tidak dikenali.' },
  ])('menampilkan galat daftar: $name', async ({ answer, text }) => {
    installFetch((url) => (url === PATH ? answer() : undefined))
    show()

    expect(await screen.findByText('Daftar tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })
})

describe('popup', () => {
  it('membuka popup dari tombol Detail dan menutupnya lagi', async () => {
    installFetch()
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Lihat detail klaim PNC-1' }))
    const dialog = screen.getByRole('dialog')
    expect(await within(dialog).findByText('Klaim tidak ditemukan')).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Tutup' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})

describe('ekspor', () => {
  it('memakai nama berkas bawaan bila server tidak menyebutnya', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
    const revoke = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const names: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      names.push(this.download)
    })
    installFetch()
    show()
    await screen.findByText('PNC-1')

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    await waitFor(() => expect(names).toEqual(['SummaryOS.csv']))
    expect(revoke).toHaveBeenCalledWith('blob:contoh')
  })

  it.each([
    { name: 'pesan dari server', answer: (): Answer => json(500, { pesan: 'Cabang tidak punya klaim.' }), text: 'Cabang tidak punya klaim.' },
    { name: 'badan bukan JSON', answer: (): Answer => new Response('rusak', { status: 500 }), text: 'Berkas ekspor tidak dapat diambil.' },
  ])('menampilkan galat ekspor: $name', async ({ answer, text }) => {
    installFetch((url) => (url === `${PATH}/ekspor` ? answer() : undefined))
    show()
    await screen.findByText('PNC-1')

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(text)
  })
})
