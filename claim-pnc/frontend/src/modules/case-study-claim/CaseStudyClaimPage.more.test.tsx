import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CaseStudyClaimPage } from './CaseStudyClaimPage'
import type { CaseStudyRow, MetadataResponse } from './types'

/**
 * Uji tambahan Case Study Claim: ekspor, paginasi, rentang tahun, dan cabang galat. Seluruh
 * data KARANGAN (`D-69`).
 */

const PATH = '/api/case-study-claim'

const METADATA: MetadataResponse = {
  bisnis: [{ kode: '002', label: 'PA' }],
  status: [{ kode: 'reject', label: 'REJECT' }],
  kolom: [
    { kunci: 'nomor_klaim', judul: 'PNC Case ID', jenis: 'teks' },
    { kunci: 'remark', judul: 'Remark', jenis: 'catatan' },
  ],
  ambang_nilai_klaim: 500_000_000_000,
  catatan_periode: '',
  catatan_kolom_kembar: '',
}

function row(nomor: string, remark = ''): CaseStudyRow {
  return {
    nomor_klaim: nomor,
    nomor_polis: '',
    nama_tertanggung: '',
    cob: '',
    periode_polis: '',
    bulan_klaim: '',
    tanggal_kejadian: '',
    sob: '',
    posisi_reasuransi: '',
    nature_of_loss: '',
    cause_of_loss: '',
    tsi_100: null,
    share_asm: null,
    deductible: null,
    nilai_share_asm: null,
    nilai_klaim_100: null,
    adjuster_fee_100: null,
    nilai_klaim_net_100: null,
    nilai_klaim_net_share_asm: null,
    lack_of_doc: null,
    cabang: '',
    status_klaim: '',
    kronologi: '',
    remark,
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

function list(rows: CaseStudyRow[], total = rows.length, awal = '2024', akhir = '2024') {
  return json(200, { baris: rows, total, periode: { tahun_awal: awal, tahun_akhir: akhir } })
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
    if (url === `${PATH}/penyaring`) return Promise.resolve(json(200, METADATA))
    if (url.startsWith(`${PATH}/unduh`)) {
      return Promise.resolve(
        new Response('a\n', { status: 200, headers: { 'Content-Disposition': 'attachment; filename="cs.csv"' } }),
      )
    }
    if (method === 'PUT') return Promise.resolve(json(200, { nomor_klaim: 'STD-1', catatan: 'x' }))
    return Promise.resolve(list([row('STD-1')]))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <CaseStudyClaimPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSelectedPortal.getState().clear()
})

async function search() {
  await screen.findByRole('option', { name: 'PA' })
  fireEvent.change(screen.getByLabelText('Awal'), { target: { value: '2024-01-01' } })
  fireEvent.change(screen.getByLabelText('Akhir'), { target: { value: '2025-12-31' } })
  await userEvent.selectOptions(screen.getByLabelText('Status'), 'reject')
  await userEvent.selectOptions(screen.getByLabelText('Bisnis'), '002')
  await userEvent.click(screen.getByRole('button', { name: 'Lihat Data' }))
}

describe('ekspor', () => {
  it('mengunduh berkas dengan penyaring yang berlaku', async () => {
    installFetch()
    show()
    await search()
    await screen.findByText('STD-1')

    await userEvent.click(screen.getByRole('button', { name: 'Export Data' }))

    await waitFor(() =>
      expect(urls).toContain(
        `GET ${PATH}/unduh?dari=2024-01-01&sampai=2025-12-31&bisnis=002&status=reject`,
      ),
    )
    expect(await screen.findByRole('button', { name: 'Export Data' })).toBeEnabled()
  })

  it('menampilkan galat ekspor', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/unduh`) ? json(500, { kode: 'x', pesan: 'Berkas gagal.' }) : undefined,
    )
    show()
    await search()
    await screen.findByText('STD-1')
    await userEvent.click(screen.getByRole('button', { name: 'Export Data' }))

    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
    expect(screen.getByText('Berkas gagal.')).toBeInTheDocument()
  })
})

describe('daftar', () => {
  it('menyebut rentang tahun dan berpindah halaman', async () => {
    installFetch((url) => {
      if (!url.startsWith(`${PATH}?`)) return undefined
      const lewati = new URL(url, 'https://x').searchParams.get('lewati') ?? '0'
      return list([row(`STD-L${lewati}`)], 45, '2024', '2025')
    })
    show()
    await search()

    expect(await screen.findByText('Disaring menurut tahun registrasi 2024–2025.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /halaman berikutnya|Berikutnya/i }))
    expect(await screen.findByText('STD-L20')).toBeInTheDocument()
  })

  it('menyebut pesan kosong umum bila keterangan layar belum termuat', async () => {
    installFetch((url) => (url === `${PATH}/penyaring` ? new Promise<Response>(() => {}) : list([])))
    show()

    await userEvent.click(screen.getByRole('button', { name: 'Lihat Data' }))
    expect(await screen.findByText('Tidak ada klaim yang cocok dengan penyaring ini.')).toBeInTheDocument()
  })

  it('menampilkan pesan umum untuk galat daftar yang bukan galat API', async () => {
    installFetch((url) => (url.startsWith(`${PATH}?`) ? 'rusak' : undefined))
    show()
    await search()

    expect(await screen.findByText('Data tidak dapat dimuat')).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })

  it('menutup layar bila keterangan gagal dimuat', async () => {
    installFetch((url) =>
      url === `${PATH}/penyaring` ? json(500, { kode: 'x', pesan: 'Keterangan rusak.' }) : undefined,
    )
    show()

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Keterangan rusak.')).toBeInTheDocument()
  })
})

describe('catatan', () => {
  it('menyebut tersimpan setelah berhasil, dan pesan umum untuk galat bukan API', async () => {
    let fail = true
    installFetch((_url, method) => {
      if (method !== 'PUT') return undefined
      return fail ? 'rusak' : json(200, { nomor_klaim: 'STD-1', catatan: 'Telaah.' })
    })
    show()
    await search()

    const box = await screen.findByLabelText('Remark untuk klaim STD-1')
    await userEvent.type(box, 'Telaah.')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Catatan gagal disimpan. Coba lagi beberapa saat lagi.',
    )

    fail = false
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(urls.filter((u) => u.startsWith('PUT')).length).toBe(2))
    // Draf masih berbeda dari remark baris (daftar belum memuat ulang), sehingga tanda
    // "Tersimpan" belum muncul — ia hanya muncul saat draf sama dengan isi baris.
    expect(screen.queryByText('Tersimpan')).not.toBeInTheDocument()
    await userEvent.clear(box)
    expect(await screen.findByText('Tersimpan')).toBeInTheDocument()
  })
})
