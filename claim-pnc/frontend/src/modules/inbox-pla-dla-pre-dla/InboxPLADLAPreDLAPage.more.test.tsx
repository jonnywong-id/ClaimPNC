import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { APIError, NetworkError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxPLADLAPreDLAPage } from './InboxPLADLAPreDLAPage'
import { formatTanggal, galatIsian, pesanGalat, pesanMuat } from './pesan'

/**
 * Uji tambahan Inbox PLA, DLA, Pre DLA: fungsi pesan, ekspor, paginasi, dan keadaan memuat,
 * galat, serta kosong pada kedua panel. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-pla-dla-pre-dla'

const KOLOM_ANTREAN = [
  { kunci: 'no_klaim', judul: 'No Klaim', tanggal: false },
  { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung', tanggal: false },
  { kunci: 'pic_teknik', judul: 'PIC Teknik', tanggal: false },
  { kunci: 'tanggal_advice', judul: 'Tanggal PLA', tanggal: true },
]

const KOLOM_RINCIAN = [
  { kunci: 'no_advice', judul: 'No PLA', tanggal: false },
  { kunci: 'reasuradur', judul: 'Reasuradur', tanggal: false },
  { kunci: 'terkirim', judul: 'Terkirim', tanggal: false },
  { kunci: 'tanggal_kirim', judul: 'Tanggal Kirim', tanggal: true },
]

const KOLOM_CETAK = [
  { kunci: 'no_advice', judul: 'NO DLA', tanggal: false },
  { kunci: 'reasuradur', judul: 'DLA REINSURER', tanggal: false },
  { kunci: 'terkirim', judul: 'Terkirim', tanggal: false },
  { kunci: 'tanggal_kirim', judul: 'Tgl Kirim', tanggal: true },
]

const PLA = {
  kode: 'pla',
  nama: 'PLA',
  keterangan: 'PLA yang belum dikirim.',
  kolom: KOLOM_ANTREAN,
  kolom_rincian: KOLOM_RINCIAN,
  punya_rincian: true,
  kolom_cetak: [],
  punya_cetak: false,
  label_aksi_baris: '',
  label_pencarian: 'No Klaim',
  label_tanggal: 'Tanggal PLA',
}

const PRE = {
  ...PLA,
  kode: 'pre-dla',
  nama: 'Pre DLA',
  keterangan: 'Pre-DLA tanpa Nomor Akseptasi.',
  kolom_rincian: [],
  punya_rincian: false,
  kolom_cetak: KOLOM_CETAK,
  punya_cetak: true,
  label_aksi_baris: 'Print Pre DLA',
}

const METADATA = { daftar: [PLA, PRE], daftar_bawaan: 'pla', portal: 'ASM' }

const BARIS = {
  kunci_klaim: 'ASM-FW-GCNMFW-WORK PNC-2001',
  no_klaim: 'PNC-2001',
  no_polis: 'POL-1',
  nama_tertanggung: 'PT Contoh',
  tanggal_register: '',
  tanggal_kejadian: '',
  pic_teknik: '',
  tanggal_advice: '2026-01-10',
}

const BARIS_B = { ...BARIS, kunci_klaim: 'K-B', no_klaim: 'PNC-1999', nama_tertanggung: 'PT Alfa' }

type Answer = Response | Promise<Response> | 'putus' | undefined

let urls: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function antrean(baris: unknown[], halaman = 1, total = baris.length, totalHalaman = 1) {
  return json(200, {
    daftar: PLA,
    baris,
    paginasi: { halaman, ukuran: 10, total, total_halaman: totalHalaman },
    penyaring: { cari: '', dari: '', sampai: '' },
    portal: 'ASM',
  })
}

function installFetch(answer: (url: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    const custom = answer(url)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom) return Promise.resolve(custom)
    if (url === `${PATH}/daftar`) return Promise.resolve(json(200, METADATA))
    if (url.includes('/ekspor')) return Promise.resolve(new Response('a,b\n', { status: 200 }))
    if (url.includes('/cetak/')) {
      return Promise.resolve(json(200, { daftar: PRE, kunci_klaim: BARIS.kunci_klaim, baris: [], portal: 'ASM' }))
    }
    if (url.includes('/klaim/')) {
      return Promise.resolve(json(200, { daftar: PLA, kunci_klaim: BARIS.kunci_klaim, baris: [], portal: 'ASM' }))
    }
    return Promise.resolve(antrean([BARIS, BARIS_B]))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <InboxPLADLAPreDLAPage />
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
  useSelectedPortal.getState().clear()
})

describe('pesan', () => {
  it('formatTanggal menandai kosong, membiarkan teks yang bukan tanggal, dan memformat WIB', () => {
    expect(formatTanggal('  ')).toBe('—')
    expect(formatTanggal('bukan tanggal')).toBe('bukan tanggal')
    expect(formatTanggal('2026-01-10T00:00:00Z')).toBe(
      new Date('2026-01-10T00:00:00Z').toLocaleDateString('id-ID', {
        timeZone: 'Asia/Jakarta',
        day: '2-digit',
        month: 'short',
        year: 'numeric',
      }),
    )
  })

  it('pesanGalat membedakan jaringan, galat API, dan galat lain', () => {
    expect(pesanGalat(new NetworkError())).toBe(
      'Server Claim PNC tidak dapat dihubungi. Periksa koneksi jaringan.',
    )
    expect(pesanGalat(new APIError('x', 'Pesan server.', 500))).toBe('Pesan server.')
    expect(pesanGalat('lain')).toBe('Terjadi kesalahan pada sistem.')
  })

  it.each([
    { error: new NetworkError(), title: 'Server Claim PNC tidak dapat dihubungi', tone: 'gangguan' },
    { error: new APIError('portal_tidak_disebut', 'x', 400), title: 'Portal entitas belum dipilih', tone: 'penolakan' },
    { error: new APIError('portal_tidak_dikenal', 'x', 400), title: 'Portal entitas belum dipilih', tone: 'penolakan' },
    { error: new APIError('portal_belum_siap', 'x', 503), title: 'Basis data entitas ini belum tersedia', tone: 'gangguan' },
    { error: new APIError('validasi_gagal', 'Tanggal salah.', 422), title: 'Penyaring belum benar', tone: 'penolakan' },
    { error: new APIError('galat_internal', 'Sibuk.', 500), title: 'Antrean tidak dapat dimuat', tone: 'gangguan' },
    { error: 'lain', title: 'Terjadi kesalahan pada sistem', tone: 'gangguan' },
  ])('pesanMuat: $title', ({ error, title, tone }) => {
    const pesan = pesanMuat(error)
    expect(pesan.title).toBe(title)
    expect(pesan.tone).toBe(tone)
  })

  it('galatIsian hanya membaca rincian yang menyebut nama isian', () => {
    expect(galatIsian('lain')).toEqual({})
    expect(
      galatIsian(
        new APIError('validasi_gagal', 'x', 422, [
          { field: 'dari', pesan: 'Tanggal awal salah.' },
          { kolom: 'sampai', pesan: 'Diabaikan.' },
        ]),
      ),
    ).toEqual({ dari: 'Tanggal awal salah.' })
  })
})

describe('layar', () => {
  it('menyebut keterangan layar yang sedang dimuat', () => {
    installFetch((url) => (url === `${PATH}/daftar` ? new Promise<Response>(() => {}) : undefined))
    show()

    expect(screen.getByText('Memuat keterangan layar…')).toBeInTheDocument()
  })

  it('mengirim rentang tanggal lalu membersihkan penyaring', async () => {
    installFetch()
    show()
    await screen.findByText('PNC-2001')

    fireEvent.change(screen.getByLabelText('Tanggal PLA — Dari'), { target: { value: '2026-01-01' } })
    fireEvent.change(screen.getByLabelText('Tanggal PLA — Sampai'), { target: { value: '2026-01-31' } })
    await userEvent.click(screen.getByRole('button', { name: 'CARI DATA' }))
    await waitFor(() => expect(urls).toContain(`${PATH}?daftar=pla&dari=2026-01-01&sampai=2026-01-31`))

    await userEvent.click(screen.getByRole('button', { name: 'Bersihkan' }))
    expect(screen.getByLabelText('Tanggal PLA — Dari')).toHaveValue('')
    expect(screen.getByLabelText('Tanggal PLA — Sampai')).toHaveValue('')
  })

  it('menyorot penyaring yang ditolak server dan menampilkan galat antrean', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}?`)
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'Penyaring salah.',
            detail: [{ field: 'sampai', pesan: 'Tanggal akhir sebelum tanggal awal.' }],
          })
        : undefined,
    )
    show()

    expect(await screen.findByText('Tanggal akhir sebelum tanggal awal.')).toBeInTheDocument()
    expect(screen.getByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Penyaring salah.')).toBeInTheDocument()
  })

  it('menandai sel kosong, memuat ulang, dan berpindah halaman', async () => {
    installFetch((url) => {
      if (!url.startsWith(`${PATH}?`)) return undefined
      const halaman = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
      return antrean([{ ...BARIS, no_klaim: `PNC-H${halaman}` }], halaman, 25, 3)
    })
    show()

    const row = (await screen.findByText('PNC-H1')).closest('tr')!
    expect(within(row).getByText('—')).toBeInTheDocument()

    const sebelum = urls.length
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(urls.length).toBe(sebelum + 1))

    await userEvent.click(screen.getByRole('button', { name: /halaman berikutnya|Berikutnya/i }))
    expect(await screen.findByText('PNC-H2')).toBeInTheDocument()
    expect(urls).toContain(`${PATH}?daftar=pla&halaman=2`)
  })

  it('mengurutkan antrean menurut kolomnya', async () => {
    installFetch()
    show()
    await screen.findByText('PNC-2001')
    const table = screen.getByRole('table', { name: 'Antrean PLA' })

    for (const header of within(table).getAllByRole('columnheader')) {
      const button = within(header).queryByRole('button')
      if (button) await userEvent.click(button)
    }
    const nomor = within(table).getByRole('columnheader', { name: /^No Klaim/ })
    await userEvent.click(within(nomor).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('PNC-1999')
  })
})

describe('ekspor', () => {
  it('mengunduh berkas dengan penyaring yang berlaku', async () => {
    const create = vi.fn(() => 'blob:uji')
    const revoke = vi.fn()
    vi.stubGlobal('URL', { ...URL, createObjectURL: create, revokeObjectURL: revoke })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    installFetch()
    show()
    await screen.findByText('PNC-2001')

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    await waitFor(() => expect(revoke).toHaveBeenCalledWith('blob:uji'))
    expect(urls).toContain(`${PATH}/ekspor?daftar=pla`)
    expect(click).toHaveBeenCalled()
    click.mockRestore()
  })

  it('menyebut ekspor yang gagal', async () => {
    installFetch((url) =>
      url.includes('/ekspor') ? json(500, { kode: 'galat_internal', pesan: 'Ekspor gagal.' }) : undefined,
    )
    show()
    await screen.findByText('PNC-2001')

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    expect(await screen.findByText('Berkas ekspor tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText('Ekspor gagal.')).toBeInTheDocument()
  })
})

describe('panel rincian', () => {
  async function openDocument() {
    await userEvent.click(await screen.findByRole('button', { name: 'PNC-2001' }))
    return screen.findByRole('region', { name: 'Rincian PLA klaim PNC-2001' })
  }

  it('menampilkan memuat lalu galat rincian', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((url) =>
      url.includes('/klaim/')
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    const panel = await openDocument()

    release(json(500, { kode: 'galat_internal', pesan: 'Rincian rusak.' }))
    expect(await within(panel).findByText('Rincian tidak dapat dimuat')).toBeInTheDocument()
    expect(within(panel).getByText('Rincian rusak.')).toBeInTheDocument()
  })

  it('menyatakan rincian kosong sebagai keadaan sah lalu menutup panel', async () => {
    installFetch()
    show()
    const panel = await openDocument()

    expect(await within(panel).findByText(/Klaim ini belum punya PLA sama sekali\./)).toBeInTheDocument()
    await userEvent.click(within(panel).getByRole('button', { name: 'Tutup rincian' }))
    expect(screen.queryByRole('region', { name: /Rincian PLA/ })).not.toBeInTheDocument()
  })

  it('menandai sel kosong dan mengurutkan rincian', async () => {
    installFetch((url) =>
      url.includes('/klaim/') && !url.endsWith('/kirim')
        ? json(200, {
            daftar: PLA,
            kunci_klaim: BARIS.kunci_klaim,
            baris: [
              { no_advice: 'PLA/2', reasuradur: '', terkirim: '', tanggal_kirim: '', tanggal_dokumen: '1' },
              { no_advice: 'PLA/1', reasuradur: 'Reas A', terkirim: '1', tanggal_kirim: '2026-01-13', tanggal_dokumen: '2' },
            ],
            portal: 'ASM',
          })
        : undefined,
    )
    show()
    const panel = await openDocument()
    const table = await within(panel).findByRole('table', { name: 'Rincian PLA' })

    const row = within(table).getByText('PLA/2').closest('tr')!
    expect(within(row).getAllByText('—')).toHaveLength(2)
    expect(within(row).getByText('Belum')).toBeInTheDocument()

    for (const title of ['No PLA', 'Reasuradur', 'Terkirim', 'Tanggal Kirim']) {
      const header = within(table).getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(header).getByRole('button'))
    }
    const no = within(table).getByRole('columnheader', { name: /^No PLA/ })
    await userEvent.click(within(no).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('PLA/1')
  })
})

describe('panel Print Pre DLA', () => {
  async function openPrint() {
    await userEvent.click(await screen.findByRole('tab', { name: 'Pre DLA' }))
    const table = await screen.findByRole('table', { name: 'Antrean Pre DLA' })
    const row = (await within(table).findByText('PNC-2001')).closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'Print Pre DLA' }))
    return screen.findByRole('region', { name: 'Print Pre DLA klaim PNC-2001' })
  }

  it('menampilkan memuat lalu galat panel', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((url) =>
      url.includes('/cetak/')
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    const panel = await openPrint()

    release(json(503, { kode: 'portal_belum_siap', pesan: 'Basis data belum siap.' }))
    expect(await within(panel).findByText('Panel Print Pre DLA tidak dapat dimuat')).toBeInTheDocument()
  })

  it('menyatakan panel kosong tanpa keterangan jumlah lalu menutupnya', async () => {
    installFetch()
    show()
    const panel = await openPrint()

    expect(await within(panel).findByText(/Belum ada Pre-DLA klaim ini/)).toBeInTheDocument()
    expect(within(panel).queryByText(/Pre-DLA yang dokumennya sudah terlampir\./)).not.toBeInTheDocument()
    await userEvent.click(within(panel).getByRole('button', { name: 'Tutup panel Print Pre DLA' }))
    expect(screen.queryByRole('region', { name: /Print Pre DLA klaim/ })).not.toBeInTheDocument()
  })

  it('menandai sel kosong dan mengurutkan isi panel', async () => {
    installFetch((url) =>
      url.includes('/cetak/') && !url.endsWith('/kirim')
        ? json(200, {
            daftar: PRE,
            kunci_klaim: BARIS.kunci_klaim,
            baris: [
              { no_advice: 'PRE/2', reasuradur: '', tipe: '', tanggal_kirim: '', terkirim: '0', kunci_lampiran: 'A2' },
              { no_advice: 'PRE/1', reasuradur: 'Reas', tipe: 'OR', tanggal_kirim: '2026-03-10', terkirim: '1', kunci_lampiran: 'A1' },
            ],
            portal: 'ASM',
          })
        : undefined,
    )
    show()
    const panel = await openPrint()
    const table = await within(panel).findByRole('table', { name: 'Print Pre DLA' })

    const row = within(table).getByText('PRE/2').closest('tr')!
    expect(within(row).getAllByText('—')).toHaveLength(2)
    expect(within(panel).getByText(/Menampilkan 2 Pre-DLA/)).toBeInTheDocument()

    for (const title of ['NO DLA', 'DLA REINSURER', 'Terkirim', 'Tgl Kirim']) {
      const header = within(table).getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(header).getByRole('button'))
    }
    const no = within(table).getByRole('columnheader', { name: /^NO DLA/ })
    await userEvent.click(within(no).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('PRE/1')
  })
})
