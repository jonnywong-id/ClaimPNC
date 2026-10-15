import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimHistoryPage } from './ClaimHistoryPage'
import type { ClaimHistory, SearchType } from './types'

/**
 * Uji tambahan View History Claim: placeholder per tipe, kolom tambahan, paginasi, dan
 * cabang galat gerbang/pencarian. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/riwayat-klaim'

function textType(kode: string, label: string, extra: Partial<SearchType> = {}): SearchType {
  return {
    kode,
    label,
    pakai_teks: true,
    pakai_tanggal_pencarian: false,
    pakai_tanggal_lahir: false,
    kolom_tambahan: [],
    tersedia: true,
    ...extra,
  }
}

const TYPES: SearchType[] = [
  textType('1', 'No Polis'),
  textType('2', 'Nama Customer'),
  textType('3', 'Nama Objek', { kolom_tambahan: ['nama_objek'] }),
  textType('4', 'No PLA'),
  textType('5', 'No DLA'),
  textType('7', 'No Klaim'),
  textType('8', 'No Akseptasi', { kolom_tambahan: ['nomor_akseptasi'] }),
  textType('11', 'No Survey'),
  textType('13', 'No Balai Lelang', { kolom_tambahan: ['nomor_balai_lelang'] }),
  textType('14', 'Lainnya'),
  {
    ...textType('6', 'Tgl Kejadian'),
    pakai_teks: false,
    pakai_tanggal_pencarian: true,
  },
  {
    ...textType('9', 'Tanggal Lahir', { kolom_tambahan: ['tanggal_lahir'] }),
    pakai_teks: false,
    pakai_tanggal_lahir: true,
  },
  textType('15', 'Belum Ada', { tersedia: false }),
]

function claim(partial: Partial<ClaimHistory> = {}): ClaimHistory {
  return {
    referensi: 'REF-1',
    nomor_klaim: 'PNC-9001',
    nomor_polis: 'POL-CONTOH-0001',
    nama_tertanggung: 'PT CONTOH',
    tanggal_kejadian: null,
    bisnis: 'Fire',
    cabang: 'Cabang Contoh',
    status: 'Open',
    posisi_klaim: '',
    tanggal_close: null,
    catatan_close: '',
    pic_teknis: 'PIC',
    nomor_akseptasi: 'AKS-0001',
    nomor_balai_lelang: 'BL-01',
    nama_objek: 'Gudang',
    tanggal_lahir: '1990-01-02',
    ...partial,
  }
}

let urls: string[] = []

type Answer = Response | Promise<Response> | undefined

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function open(remaining = 49): Response {
  return json(200, {
    tipe_pencarian: TYPES,
    proteksi: { jatah_total: 50, jatah_terpakai: 1, jatah_sisa: remaining },
    portal: 'ASM',
  })
}

function result(rows: ClaimHistory[], halaman = 1, total = rows.length, totalPages = 1): Response {
  return json(200, {
    klaim: rows,
    halaman: { halaman, ukuran: 20, total, total_halaman: totalPages },
    portal: 'ASM',
  })
}

function installFetch(answer: (url: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    const custom = answer(url)
    if (custom) return Promise.resolve(custom)
    if (url === `${PATH}/buka`) return Promise.resolve(open())
    return Promise.resolve(result([claim()]))
  })
}

function show() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <ClaimHistoryPage />
    </QueryClientProvider>,
  )
}

async function choose(kode: string) {
  const select = await screen.findByLabelText('Tipe Pencarian')
  await within(select).findByRole('option', { name: 'No Polis' })
  await userEvent.selectOptions(select, kode)
}

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('isian pencarian', () => {
  it.each([
    { kode: '1', placeholder: 'POL-0000-0000', hint: 'persis' },
    { kode: '2', placeholder: 'nama tertanggung', hint: 'sebagian' },
    { kode: '3', placeholder: 'nama objek pertanggungan', hint: 'sebagian' },
    { kode: '4', placeholder: 'PLA-0000', hint: 'sebagian' },
    { kode: '5', placeholder: 'DLA-0000', hint: 'sebagian' },
    { kode: '7', placeholder: 'PNC-0000 atau PNCN.26.0000', hint: 'persis' },
    { kode: '8', placeholder: 'AKS-0000', hint: 'persis' },
    { kode: '11', placeholder: 'SRV-0000', hint: 'persis' },
    { kode: '13', placeholder: 'BL-00', hint: 'persis' },
    { kode: '14', placeholder: '', hint: 'persis' },
  ])('memberi contoh dan cara pencocokan untuk tipe $kode', async ({ kode, placeholder, hint }) => {
    installFetch()
    show()
    await choose(kode)

    const input = screen.getByLabelText('Nama Pencarian')
    expect(input).toHaveAttribute('placeholder', placeholder)
    expect(screen.getByText(new RegExp(`Dicocokkan ${hint}`))).toBeInTheDocument()
  })

  it('tidak menampilkan alasan bila tipe belum tersedia tanpa alasan', async () => {
    installFetch()
    show()
    await choose('15')

    expect(screen.getByRole('button', { name: /Cari/ })).toBeDisabled()
    expect(screen.getByRole('option', { name: 'Belum Ada — belum tersedia' })).toBeInTheDocument()
  })

  it('mengirim tanggal kejadian dan tanggal lahir pada tipe bertanggal', async () => {
    installFetch()
    show()
    await choose('6')
    fireEvent.change(screen.getByLabelText('Tanggal Pencarian'), { target: { value: '2026-03-12' } })
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))
    await waitFor(() =>
      expect(urls).toContain(`${PATH}?tipe=6&tanggal_pencarian=2026-03-12`),
    )

    await choose('9')
    fireEvent.change(screen.getByLabelText('Tanggal Lahir'), { target: { value: '1990-01-02' } })
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))
    await waitFor(() => expect(urls).toContain(`${PATH}?tipe=9&tanggal_lahir=1990-01-02`))
    // Kolom tambahan Tanggal Lahir tampil untuk tipe ini.
    expect(await screen.findByRole('columnheader', { name: /^Tanggal Lahir/ })).toBeInTheDocument()
  })
})

describe('gerbang', () => {
  it('memperingatkan jatah yang tinggal sedikit', async () => {
    installFetch((url) => (url === `${PATH}/buka` ? open(3) : undefined))
    show()

    expect(await screen.findByText(/Jatah pencarian Anda tinggal/)).toHaveTextContent(
      'Jatah pencarian Anda tinggal 3 kali lagi.',
    )
  })

  it('menampilkan pesan server untuk galat gerbang yang lain', async () => {
    installFetch((url) =>
      url === `${PATH}/buka` ? json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }) : undefined,
    )
    show()

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Basis data sibuk.')).toBeInTheDocument()
  })
})

describe('hasil', () => {
  it('menampilkan kolom tambahan akseptasi dan tanda hubung untuk tanggal kosong', async () => {
    installFetch()
    show()
    await choose('8')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), ' AKS-0001 ')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: /^No Akseptasi/ })).toBeInTheDocument()
    expect(within(table).getByText('AKS-0001')).toBeInTheDocument()
    const row = within(table).getByText('PNC-9001').closest('tr')!
    // Tgl Kejadian, Posisi Klaim, Tanggal Close, dan Catatan Close kosong.
    expect(within(row).getAllByText('—')).toHaveLength(4)
    expect(urls).toContain(`${PATH}?tipe=8&nilai=AKS-0001`)
  })

  it('menyebut keterbatasan Nama Objek dan menampilkan kolom balai lelang pada tipe 13', async () => {
    installFetch()
    show()
    await choose('3')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'Gudang')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    expect(
      await screen.findByText('Pada pencarian Nama Objek, kolom Posisi Klaim kosong — sama seperti sistem lama.'),
    ).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /^Nama Objek/ })).toBeInTheDocument()

    await choose('13')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'BL-01')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))
    expect(await screen.findByRole('columnheader', { name: /^No Balai Lelang/ })).toBeInTheDocument()
  })

  it('mengurutkan menurut setiap kolom', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}?`)
        ? result([
            claim({ referensi: 'R1', nomor_klaim: 'PNC-2', tanggal_kejadian: '2026-02-01', tanggal_close: '2026-03-01', posisi_klaim: 'Paid' }),
            claim({ referensi: 'R2', nomor_klaim: 'PNC-1' }),
          ])
        : undefined,
    )
    show()
    await choose('9')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))
    const table = await screen.findByRole('table')

    for (const header of within(table).getAllByRole('columnheader')) {
      await userEvent.click(within(header).getByRole('button'))
      expect(header).toHaveAttribute('aria-sort', 'ascending')
    }
    const nomor = within(table).getByRole('columnheader', { name: /^No Klaim/ })
    await userEvent.click(within(nomor).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('PNC-1')
  })

  it('berpindah halaman maju dan mundur', async () => {
    installFetch((url) => {
      if (!url.startsWith(`${PATH}?`)) return undefined
      const page = new URL(url, 'https://x').searchParams.get('halaman') ?? '1'
      return result([claim({ nomor_klaim: `PNC-H${page}` })], Number(page), 45, 3)
    })
    show()
    await choose('2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'PT')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    expect(await screen.findByRole('status')).toHaveTextContent('Menampilkan 1–1 dari 45 klaim.')
    expect(screen.getByRole('button', { name: 'Sebelumnya' })).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    expect(await screen.findByText('PNC-H2')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Menampilkan 21–21 dari 45 klaim.')
    expect(urls).toContain(`${PATH}?tipe=2&nilai=PT&halaman=2`)

    await userEvent.click(screen.getByRole('button', { name: 'Sebelumnya' }))
    expect(await screen.findByText('PNC-H1')).toBeInTheDocument()
  })

  it('menyebut nol baris bila halaman tidak berisi', async () => {
    installFetch((url) => (url.startsWith(`${PATH}?`) ? result([], 1, 5, 1) : undefined))
    show()
    await choose('2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'PT')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    expect(await screen.findByRole('status')).toHaveTextContent('Menampilkan 0–0 dari 5 klaim.')
    expect(screen.getByRole('button', { name: 'Berikutnya' })).toBeDisabled()
  })

  it('menampilkan galat pencarian yang bukan galat validasi', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}?`) ? json(500, { kode: 'galat_internal', pesan: 'Kueri gagal.' }) : undefined,
    )
    show()
    await choose('2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'PT')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    expect(await screen.findByText('Pencarian tidak dapat dijalankan')).toBeInTheDocument()
    expect(screen.getByText('Kueri gagal.')).toBeInTheDocument()
  })

  it('menyorot galat validasi yang memakai nama kolom', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}?`)
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'x',
            detail: [{ kolom: 'nilai_pencarian', pesan: 'Nilai terlalu pendek.' }, { pesan: 'tanpa nama' }],
          })
        : undefined,
    )
    show()
    await choose('2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'P')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    expect(await screen.findByText('Nilai terlalu pendek.')).toBeInTheDocument()
    expect(screen.queryByText('Pencarian tidak dapat dijalankan')).not.toBeInTheDocument()
  })
})
