import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { XOLPage } from './XOLPage'
import { committeeLabel, committeeTone, formatMoney } from './format'

/**
 * Uji tambahan Master XOL: cabang galat muat/simpan/hapus, form tambah, dan baris anak.
 * Seluruh data KARANGAN (`D-69`).
 */

const ROUTES = '/api/master/xol'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const BASE = {
  remark_pic: '',
  pic: '',
  komite: '',
  bisnis: [],
  layer: [],
}

const SAMPLE = [
  {
    ...BASE,
    id: '10001',
    nama: 'Section 1',
    tahun: '2018',
    kurs: 13500,
    tipe: '1',
    tipe_label: 'Property',
    status_komite: '0',
    remark_komite: 'ok',
    layer: [{ id: '1', nama: 'L1', limit: 1, excess: 1, limit_idr: 1, reas: [] }],
  },
  {
    ...BASE,
    id: '10002',
    nama: 'Section Dua',
    tahun: '2017',
    kurs: 14500,
    tipe: '',
    tipe_label: '',
    status_komite: 'X',
    remark_komite: '',
  },
]

const DETAIL = {
  ...SAMPLE[0],
  bisnis: [{ id: '10013', nama: 'FIRE' }],
  layer: [
    {
      id: '10001',
      nama: 'Sub Layer',
      limit: 100,
      excess: 50,
      limit_idr: 1350000,
      reas: [{ id: 'R1', nama: 'REAS CONTOH', share: 100 }],
    },
  ],
}

const FORM_OPTION = {
  tahun: ['2026', '2018'],
  tipe: [
    { kode: '1', label: 'Property' },
    { kode: '2', label: 'PA / GA' },
  ],
  portal: 'ASM',
}

const BUSINESS_OPTION = {
  bisnis: [
    { id: '10013', nama: 'FIRE' },
    { id: '10009', nama: 'ENGINEERING' },
  ],
  total: 2,
  portal: 'ASM',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

type Answer = (url: string, init?: RequestInit) => Response | Promise<Response> | undefined

/** Peladen tiruan; `answer` boleh mengembalikan undefined untuk jawaban bawaan. */
function installFetch(answer: Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    const custom = answer(url, init)
    if (custom) return Promise.resolve(custom)
    if (url === `${ROUTES}/form`) return Promise.resolve(jsonResponse(200, FORM_OPTION))
    if (url.startsWith(`${ROUTES}/bisnis`)) {
      return Promise.resolve(jsonResponse(200, BUSINESS_OPTION))
    }
    const method = init?.method ?? 'GET'
    if (method === 'DELETE') return Promise.resolve(new Response(null, { status: 204 }))
    if (method === 'POST') {
      return Promise.resolve(jsonResponse(201, { xol: { ...SAMPLE[0], id: '10009' }, portal: 'ASM' }))
    }
    if (method === 'PUT') return Promise.resolve(jsonResponse(200, { xol: DETAIL, portal: 'ASM' }))
    if (url === `${ROUTES}/10001`) {
      return Promise.resolve(jsonResponse(200, { xol: DETAIL, portal: 'ASM' }))
    }
    return Promise.resolve(jsonResponse(200, { xol: SAMPLE, total: SAMPLE.length, portal: 'ASM' }))
  })
}

/** Jawaban yang membuat callAPI melempar galat selain APIError/NetworkError. */
function brokenResponse(): Response {
  return {
    get status(): number {
      throw new TypeError('jawaban rusak')
    },
  } as unknown as Response
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <XOLPage />
    </QueryClientProvider>,
  )
}

function isList(url: string, init?: RequestInit) {
  return url === ROUTES && (init?.method ?? 'GET') === 'GET'
}

async function openEdit() {
  await screen.findByText('Section 1')
  await userEvent.click(screen.getByRole('button', { name: 'Ubah master XOL 10001' }))
  await screen.findByDisplayValue('Sub Layer')
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

describe('format', () => {
  it('memformat angka dan menjaga nilai yang bukan bilangan', () => {
    expect(formatMoney(14782500000)).toBe('14.782.500.000')
    expect(formatMoney(Number.NaN)).toBe('0')
  })

  it('menampilkan status komite yang tidak dikenal apa adanya dengan warna netral', () => {
    expect(committeeLabel('9')).toBe('9')
    expect(committeeTone('9')).toBe('bg-slate-100 text-slate-600 ring-slate-200')
    expect(committeeTone('')).toBe('bg-slate-100 text-slate-600 ring-slate-200')
    expect(committeeTone('0')).toContain('amber')
    expect(committeeTone('1')).toContain('emerald')
  })
})

describe('daftar', () => {
  it('meminta portal dipilih lebih dulu bila belum ada portal', async () => {
    useSelectedPortal.getState().clear()
    installFetch()
    show()

    expect(screen.getByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(calls.some((call) => call.url === ROUTES)).toBe(false)
  })

  it.each([
    {
      name: 'jaringan putus',
      reply: () => Promise.reject(new TypeError('putus')),
      title: 'Tidak dapat menghubungi server',
    },
    {
      name: 'portal tidak disebut',
      reply: () => jsonResponse(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal tidak dikenal',
      reply: () => jsonResponse(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal belum siap',
      reply: () => jsonResponse(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      title: 'Basis data entitas ini belum tersedia',
    },
    {
      name: 'galat API lain',
      reply: () => jsonResponse(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }),
      title: 'Daftar master XOL gagal dimuat',
    },
  ])('menampilkan galat muat: $name', async ({ reply, title }) => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      if (isList(url, init)) return reply()
      return Promise.resolve(jsonResponse(200, FORM_OPTION))
    })
    show()

    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('menampilkan pesan server pada galat API yang tidak dikenali', async () => {
    installFetch((url, init) =>
      isList(url, init)
        ? jsonResponse(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' })
        : undefined,
    )
    show()

    expect(await screen.findByText('Basis data sibuk.')).toBeInTheDocument()
  })

  it('menampilkan pesan umum untuk galat yang bukan galat API', async () => {
    installFetch((url, init) => (isList(url, init) ? brokenResponse() : undefined))
    show()

    expect(await screen.findByText('Terjadi kesalahan pada sistem. Coba muat ulang.')).toBeInTheDocument()
  })

  it('menyaring dan mengurutkan baris menurut kolomnya', async () => {
    installFetch()
    show()
    await screen.findByText('Section 1')

    // Status komite yang tidak dikenal ditampilkan apa adanya.
    expect(screen.getByText('X')).toBeInTheDocument()

    await userEvent.type(screen.getByPlaceholderText('Cari ID, nama, tahun, atau status'), 'Dua')
    expect(await screen.findByText('1 dari 2 baris cocok.')).toBeInTheDocument()
    expect(screen.queryByText('Section 1')).not.toBeInTheDocument()

    await userEvent.clear(screen.getByPlaceholderText('Cari ID, nama, tahun, atau status'))
    await screen.findByText('Section 1')

    const header = screen.getByRole('columnheader', { name: /Kurs IDR/ })
    await userEvent.click(within(header).getByRole('button'))
    expect(header).toHaveAttribute('aria-sort', 'ascending')
    for (const title of ['ID', 'Nama', 'Type XOL', 'Status Komite', 'Remark Komite']) {
      const column = screen.getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(column).getByRole('button'))
      expect(column).toHaveAttribute('aria-sort', 'ascending')
    }
  })

  it('memuat ulang daftar saat tombol Muat ulang ditekan', async () => {
    installFetch()
    show()
    await screen.findByText('Section 1')
    const before = calls.filter((call) => isList(call.url, call.init)).length

    await userEvent.click(screen.getByRole('button', { name: 'Muat ulang' }))

    await waitFor(() =>
      expect(calls.filter((call) => isList(call.url, call.init)).length).toBe(before + 1),
    )
    expect(await screen.findByRole('button', { name: 'Muat ulang' })).toBeEnabled()
  })
})

describe('hapus induk', () => {
  it('menyebut jumlah layer dan menutup form induk yang sedang diubah', async () => {
    installFetch()
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus master XOL 10001' }))
    const dialog = await screen.findByRole('alertdialog')
    expect(dialog).toHaveTextContent('Hapus master XOL 10001 — Section 1?')
    expect(dialog).toHaveTextContent('layer (1)')

    await userEvent.click(within(dialog).getByRole('button', { name: 'Ya, hapus' }))

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(screen.queryByText('Memperbaharui Data')).not.toBeInTheDocument()
  })

  it('tetap membuka form induk lain saat menghapus induk yang berbeda', async () => {
    installFetch()
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus master XOL 10002' }))
    const dialog = await screen.findByRole('alertdialog')
    expect(dialog).not.toHaveTextContent('layer (')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Ya, hapus' }))

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(screen.getByText('Memperbaharui Data')).toBeInTheDocument()
  })

  it.each([
    {
      name: 'sudah tidak ada',
      reply: () => jsonResponse(404, { kode: 'xol_tidak_ditemukan', pesan: 'x' }),
      text: 'Master XOL ini sudah tidak ada. Muat ulang daftarnya.',
    },
    {
      name: 'galat API lain',
      reply: () => jsonResponse(409, { kode: 'konflik', pesan: 'Sedang dipakai komite.' }),
      text: 'Sedang dipakai komite.',
    },
    {
      name: 'jaringan putus',
      reply: () => Promise.reject(new TypeError('putus')),
      text: 'Tidak dapat menghubungi server. Periksa koneksi lalu coba lagi.',
    },
    {
      name: 'galat bukan API',
      reply: () => Promise.resolve(brokenResponse()),
      text: 'Terjadi kesalahan pada sistem. Coba lagi.',
    },
  ])('menampilkan galat hapus: $name lalu membersihkannya saat dibatalkan', async ({ reply, text }) => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      if (init?.method === 'DELETE') return reply()
      if (url === `${ROUTES}/form`) return Promise.resolve(jsonResponse(200, FORM_OPTION))
      return Promise.resolve(jsonResponse(200, { xol: SAMPLE, total: 2, portal: 'ASM' }))
    })
    show()
    await screen.findByText('Section 1')

    await userEvent.click(screen.getByRole('button', { name: 'Hapus master XOL 10002' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Ya, hapus' }))

    expect(await screen.findByText(text)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Batal' }))
    expect(screen.queryByText(text)).not.toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })
})

describe('form tambah', () => {
  it('menambah baris anak, menghitung total share, lalu mengirim POST dan menutup form', async () => {
    installFetch()
    show()
    await screen.findByText('Section 1')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(await screen.findByText('Menambah Data')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()
    expect(
      screen.getByText('Pilih Type XOL lebih dulu — pilihan grup bisnis mengikuti jenisnya.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Belum ada grup bisnis.')).toBeInTheDocument()
    expect(screen.getByText('Belum ada layer.')).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Nama'), 'Section Baru')
    await userEvent.selectOptions(screen.getByLabelText('Tahun'), '2026')
    await userEvent.selectOptions(screen.getByLabelText('Type XOL'), '1')
    expect(screen.getByText('Grup bisnis yang dicakup treaty ini.')).toBeInTheDocument()
    await userEvent.clear(screen.getByLabelText('Kurs IDR'))
    await userEvent.type(screen.getByLabelText('Kurs IDR'), '15000')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah bisnis' }))
    const groupSelect = await screen.findByLabelText('Grup bisnis 1')
    await within(groupSelect).findByRole('option', { name: 'ENGINEERING (10009)' })
    // Nilai pilihan memakai pemisah karakter NUL antara ID dan nama (lihat XOLForm).
    await userEvent.selectOptions(groupSelect, '10009\u0000ENGINEERING')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah layer' }))
    await userEvent.type(await screen.findByLabelText('Nama Layer'), 'Layer Baru')
    await userEvent.clear(screen.getByLabelText('Limit (USD)'))
    await userEvent.type(screen.getByLabelText('Limit (USD)'), '2')
    // 2 × 15.000 = 30.000
    expect(screen.getByText('30.000')).toBeInTheDocument()

    expect(screen.getByText('Belum ada reasuradur pada layer ini.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah reas' }))
    await userEvent.type(screen.getByLabelText('Reasuransi'), 'REAS BARU')
    await userEvent.clear(screen.getByLabelText('Share (%)'))
    await userEvent.type(screen.getByLabelText('Share (%)'), '100')
    expect(screen.getByText('Total share 100%')).toHaveClass('bg-emerald-50')
    expect(screen.queryByText(/Total share belum 100%/)).not.toBeInTheDocument()

    // Share yang DIKETIK tersimpan sebagai teks (register tanpa valueAsNumber) sehingga
    // skema menolaknya tanpa pesan di layar — dicatat sebagai temuan, tidak diuji di sini.
    // Baris reas karena itu dibuang lebih dulu sebelum menyimpan.
    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))
    expect(await screen.findByText('Total share 0%')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(screen.queryByText('Menambah Data')).not.toBeInTheDocument())
    const sent = calls.find((call) => call.init?.method === 'POST')
    expect(sent?.url).toBe(ROUTES)
    const body = JSON.parse(String(sent?.init?.body)) as Record<string, unknown>
    expect(body).toMatchObject({
      nama: 'Section Baru',
      tahun: '2026',
      tipe: '1',
      kurs: 15000,
      bisnis: [{ id: '10009', nama: 'ENGINEERING' }],
    })
    const layer = (body['layer'] as { nama: string; limit: number; reas: { nama: string }[] }[])[0]
    expect(layer?.nama).toBe('Layer Baru')
    expect(layer?.limit).toBe(2)
    expect(layer?.reas).toEqual([])
  })

  it('membuang baris anak yang belum tersimpan tanpa memanggil server', async () => {
    installFetch()
    show()
    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.click(await screen.findByRole('button', { name: 'Tambah bisnis' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah layer' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Tambah reas' }))

    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))
    expect(await screen.findByText('Belum ada reasuradur pada layer ini.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hapus grup bisnis baris 1' }))
    expect(await screen.findByText('Belum ada grup bisnis.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hapus layer baris 1' }))
    expect(await screen.findByText('Belum ada layer.')).toBeInTheDocument()

    expect(calls.some((call) => call.init?.method === 'DELETE')).toBe(false)
  })

  it('menolak kurs negatif dan kurs kosong di layar tanpa memanggil server', async () => {
    installFetch()
    show()
    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const kurs = await screen.findByLabelText('Kurs IDR')
    await userEvent.clear(kurs)
    await userEvent.type(kurs, '-5')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))
    expect(await screen.findByText('Kurs tidak boleh negatif.')).toBeInTheDocument()

    await userEvent.clear(kurs)
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))
    expect(await screen.findByText('Kurs harus diisi angka.')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tambah layer' }))
    await userEvent.type(await screen.findByLabelText('Nama Layer'), 'x'.repeat(51))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))
    expect(
      await screen.findByText('Nama layer paling panjang 50 karakter.'),
    ).toBeInTheDocument()
    expect(calls.some((call) => call.init?.method === 'POST')).toBe(false)
  })

  it('menutup form tanpa menyimpan lewat tombol Tutup', async () => {
    installFetch()
    show()
    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Tutup' }))

    expect(screen.queryByText('Menambah Data')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeEnabled()
  })
})

describe('form ubah', () => {
  it('menampilkan keadaan memuat lalu galat bila detail gagal dimuat', async () => {
    let fail: (r: Response) => void = () => {}
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      if (url === `${ROUTES}/10001`) {
        return new Promise<Response>((resolve) => {
          fail = resolve
        })
      }
      if (url === `${ROUTES}/form`) return Promise.resolve(jsonResponse(200, FORM_OPTION))
      if (url.startsWith(`${ROUTES}/bisnis`)) {
        return Promise.resolve(jsonResponse(200, BUSINESS_OPTION))
      }
      return Promise.resolve(jsonResponse(200, { xol: SAMPLE, total: 2, portal: 'ASM' }))
    })
    show()
    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah master XOL 10001' }))

    expect(await screen.findByText('Memuat isi master XOL…')).toBeInTheDocument()
    fail(jsonResponse(500, { kode: 'galat_internal', pesan: 'x' }))
    expect(await screen.findByText('Isi master XOL gagal dimuat')).toBeInTheDocument()
  })

  it('menyatakan tersimpan tanpa catatan bila server tidak mengirim peringatan', async () => {
    installFetch()
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Tersimpan dan diajukan ke komite.')).toBeInTheDocument()
    // Form ubah TIDAK ditutup setelah tersimpan.
    expect(screen.getByText('Memperbaharui Data')).toBeInTheDocument()
    expect(calls.find((call) => call.init?.method === 'PUT')?.url).toBe(`${ROUTES}/10001`)
  })

  it.each([
    {
      name: 'validasi berisi detail',
      reply: () =>
        jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [{ field: 'nama', pesan: 'Nama terlalu panjang.' }],
        }),
      texts: ['Isian belum benar.', 'Nama terlalu panjang.'],
    },
    {
      name: 'validasi tanpa detail',
      reply: () => jsonResponse(422, { kode: 'validasi_gagal', pesan: 'Isian ditolak.' }),
      texts: ['Gagal menyimpan', 'Isian ditolak.'],
    },
    {
      name: 'nomor bentrok',
      reply: () => jsonResponse(409, { kode: 'id_xol_sudah_dipakai', pesan: 'x' }),
      texts: ['Nomor bentrok'],
    },
    {
      name: 'jaringan putus',
      reply: () => Promise.reject(new TypeError('putus')),
      texts: ['Perubahan belum tersimpan. Periksa koneksi lalu tekan Simpan lagi.'],
    },
    {
      name: 'galat bukan API',
      reply: () => Promise.resolve(brokenResponse()),
      texts: ['Terjadi kesalahan pada sistem. Coba lagi.'],
    },
  ])('menampilkan galat simpan: $name', async ({ reply, texts }) => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      if (init?.method === 'PUT') return reply()
      if (url === `${ROUTES}/form`) return Promise.resolve(jsonResponse(200, FORM_OPTION))
      if (url.startsWith(`${ROUTES}/bisnis`)) {
        return Promise.resolve(jsonResponse(200, BUSINESS_OPTION))
      }
      if (url === `${ROUTES}/10001`) {
        return Promise.resolve(jsonResponse(200, { xol: DETAIL, portal: 'ASM' }))
      }
      return Promise.resolve(jsonResponse(200, { xol: SAMPLE, total: 2, portal: 'ASM' }))
    })
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    for (const text of texts) expect(await screen.findByText(text)).toBeInTheDocument()
  })

  // Hapus baris tersimpan dikirim SEKETIKA ke server. Yang diperiksa di sini hanya jenis dan
  // induknya; nilai ID baris sengaja tidak di-assert (lihat temuan di laporan).
  it('menghapus baris anak tersimpan seketika lewat server', async () => {
    installFetch()
    show()
    await openEdit()

    expect(screen.getByText('Total share 100%')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))
    await waitFor(() =>
      expect(
        calls.some(
          (call) =>
            call.init?.method === 'DELETE' &&
            call.url.startsWith(`${ROUTES}/10001/layer/`) &&
            call.url.endsWith('/reas/R1'),
        ),
      ).toBe(true),
    )

    await userEvent.click(screen.getByRole('button', { name: 'Hapus grup bisnis baris 1' }))
    await waitFor(() =>
      expect(
        calls.some(
          (call) => call.init?.method === 'DELETE' && call.url.startsWith(`${ROUTES}/10001/bisnis/`),
        ),
      ).toBe(true),
    )

    await userEvent.click(screen.getByRole('button', { name: 'Hapus layer baris 1' }))
    await waitFor(() =>
      expect(
        calls.some(
          (call) =>
            call.init?.method === 'DELETE' &&
            call.url.startsWith(`${ROUTES}/10001/layer/`) &&
            !call.url.includes('/reas/'),
        ),
      ).toBe(true),
    )
  })

  it('memberi tahu bila baris anak gagal dihapus di server', async () => {
    installFetch((_url, init) =>
      init?.method === 'DELETE'
        ? jsonResponse(500, { kode: 'galat_internal', pesan: 'x' })
        : undefined,
    )
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Baris gagal dihapus di server. Muat ulang layar untuk melihat keadaan yang sebenarnya.',
    )
  })
})
