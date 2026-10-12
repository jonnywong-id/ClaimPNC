import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { inputFrom } from './api'
import { ClausePage } from './ClausePage'

/**
 * Uji tambahan Master Pasal Kerugian: pemilih lini bisnis, galat simpan/muat/hapus, dan
 * tampilan baris yang isiannya kosong. Seluruh data KARANGAN (`D-69`).
 */

const ROUTE = '/api/master/pasal-kerugian'

const CATEGORY = {
  kategori: [
    { kode: '1', nama: 'Jaminan Polis' },
    { kode: '2', nama: 'Pengecualian' },
    { kode: '', nama: 'Notifikasi' },
  ],
}

const ROW_A = {
  id: '1',
  no_pasal: 'PSL-001',
  isi_pasal: '',
  deskripsi: '',
  kategori: '',
  kategori_label: 'Notifikasi',
  bisnis: [],
}

const ROW_B = {
  id: '2',
  no_pasal: 'PSL-002',
  isi_pasal: 'Isi pasal contoh.',
  deskripsi: 'Deskripsi contoh',
  kategori: '7',
  kategori_label: 'Kode Lama',
  bisnis: [],
}

const LIST = { portal: 'ASM', pasal_kerugian: [ROW_A, ROW_B] }

const DETAIL_B = {
  portal: 'ASM',
  pasal_kerugian: {
    ...ROW_B,
    bisnis: [
      { id: '2004', nama: 'Fire / Property' },
      { id: '', nama: '' },
    ],
  },
}

const DETAIL_A = { portal: 'ASM', pasal_kerugian: ROW_A }

const BUSINESS = {
  portal: 'ASM',
  bisnis: [
    { id: '2004', nama: 'Fire / Property' },
    { id: '2005', nama: 'Fire Industri' },
  ],
}

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

type Answer = Response | Promise<Response> | 'putus' | 'rusak' | undefined

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
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

function installFetch(answer: (call: Call) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
    }
    calls.push(call)
    const custom = answer(call)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom === 'rusak') return Promise.resolve(brokenResponse())
    if (custom) return Promise.resolve(custom)
    if (url.startsWith(`${ROUTE}/kategori`)) return Promise.resolve(json(200, CATEGORY))
    if (url.startsWith(`${ROUTE}/bisnis`)) return Promise.resolve(json(200, BUSINESS))
    if (call.method === 'GET' && url === ROUTE) return Promise.resolve(json(200, LIST))
    if (call.method === 'GET' && url === `${ROUTE}/1`) return Promise.resolve(json(200, DETAIL_A))
    if (call.method === 'GET') return Promise.resolve(json(200, DETAIL_B))
    if (call.method === 'DELETE') return Promise.resolve(json(200, { id: '1', portal: 'ASM' }))
    return Promise.resolve(json(200, DETAIL_B))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ClausePage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('inputFrom', () => {
  it('membuang kategori_label dan id dari baris', () => {
    expect(inputFrom({ ...ROW_B, bisnis: [{ id: '9', nama: 'X' }] })).toEqual({
      no_pasal: 'PSL-002',
      isi_pasal: 'Isi pasal contoh.',
      deskripsi: 'Deskripsi contoh',
      kategori: '7',
      bisnis: [{ id: '9', nama: 'X' }],
    })
  })
})

describe('daftar', () => {
  it('menampilkan tanda kosong untuk isian kosong dan lencana netral untuk kategori lain', async () => {
    installFetch()
    show()

    expect(screen.getByText('Memuat daftar pasal kerugian…')).toBeInTheDocument()
    const table = await screen.findByRole('table')
    const rowA = within(table).getByText('PSL-001').closest('tr')!
    expect(within(rowA).getAllByText('—')).toHaveLength(2)
    expect(within(rowA).getByText('Notifikasi')).toHaveClass('bg-slate-100')
    expect(within(table).getByText('Kode Lama')).toHaveClass('bg-slate-100')
  })

  it('menampilkan pesan kosong bila entitas belum punya pasal', async () => {
    installFetch((call) =>
      call.method === 'GET' && call.url === ROUTE
        ? json(200, { portal: 'ASM', pasal_kerugian: [] })
        : undefined,
    )
    show()

    expect(await screen.findByText('Belum ada pasal kerugian pada entitas ini.')).toBeInTheDocument()
  })

  it.each([
    { name: 'jaringan putus', answer: 'putus' as Answer, title: 'Server Claim PNC tidak dapat dihubungi' },
    {
      name: 'portal tidak disebut',
      answer: json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal tidak dikenal',
      answer: json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal belum siap',
      answer: json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      title: 'Basis data entitas ini belum tersedia',
    },
    {
      name: 'galat API lain',
      answer: json(500, { kode: 'galat_internal', pesan: 'x' }),
      title: 'Daftar tidak dapat dimuat',
      description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
    },
    {
      name: 'galat bukan API',
      answer: 'rusak' as Answer,
      title: 'Daftar tidak dapat dimuat',
      description: 'Coba beberapa saat lagi.',
    },
  ])('menampilkan galat muat: $name', async ({ answer, title, description }) => {
    installFetch((call) => (call.method === 'GET' && call.url === ROUTE ? answer : undefined))
    show()

    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
  })

  it('memuat ulang daftar saat Refresh ditekan', async () => {
    installFetch()
    show()
    await screen.findByRole('table')
    const before = calls.filter((c) => c.url === ROUTE).length

    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => expect(calls.filter((c) => c.url === ROUTE).length).toBe(before + 1))
    expect(await screen.findByRole('button', { name: 'Refresh' })).toBeEnabled()
  })
})

describe('form ubah', () => {
  async function openB() {
    installFetch()
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))
    return screen.findByText('Fire / Property')
  }

  it('mempertahankan kode kategori lama lewat pilihan bayangan dan menandai butir tanpa nama', async () => {
    await openB()

    expect(screen.getByText('ID:')).toHaveTextContent('ID: 2')
    const select = screen.getByLabelText('Kategori') as HTMLSelectElement
    expect(select.value).toBe('7')
    expect(within(select).getByRole('option', { name: 'Notifikasi (kode lama 7)' })).toBeInTheDocument()
    expect(screen.getByText('(tanpa nama)')).toBeInTheDocument()
    expect(screen.getByText('2004')).toBeInTheDocument()
  })

  it('menghapus butir bisnis lalu mengirim PUT dan menutup form', async () => {
    await openB()

    await userEvent.click(
      screen.getByRole('button', { name: 'Hapus butir tanpa nama dari daftar bisnis' }),
    )
    await userEvent.click(
      screen.getByRole('button', { name: 'Hapus Fire / Property dari daftar bisnis' }),
    )
    expect(
      screen.getByText('Belum ada lini bisnis. Pasal tetap dapat disimpan tanpa ini.'),
    ).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Isi Pasal'), ' Tambahan.')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() =>
      expect(screen.queryByRole('form', { name: 'Ubah Pasal Kerugian' })).not.toBeInTheDocument(),
    )
    const sent = calls.find((c) => c.method === 'PUT')
    expect(sent?.url).toBe(`${ROUTE}/2`)
    expect(sent?.body).toEqual({
      no_pasal: 'PSL-002',
      isi_pasal: 'Isi pasal contoh. Tambahan.',
      deskripsi: 'Deskripsi contoh',
      kategori: '7',
      bisnis: [],
    })
  })

  it('menyorot pelanggaran validasi server pada isiannya', async () => {
    installFetch((call) =>
      call.method === 'PUT'
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'Isian belum benar.',
            detail: [
              { field: 'no_pasal', pesan: 'No pasal ditolak.' },
              { field: 'deskripsi', pesan: 'Deskripsi terlalu panjang.' },
              { field: 'kategori', pesan: 'Kategori tidak dikenal.' },
              { field: 'isi_pasal', pesan: 'Isi pasal terlalu panjang.' },
            ],
          })
        : undefined,
    )
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))
    await screen.findByText('Fire / Property')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    for (const text of [
      'No pasal ditolak.',
      'Deskripsi terlalu panjang.',
      'Kategori tidak dikenal.',
      'Isi pasal terlalu panjang.',
    ]) {
      expect(await screen.findByText(text)).toBeInTheDocument()
    }
    expect(screen.getByLabelText('Isi Pasal')).toHaveAttribute('aria-invalid', 'true')
    // Galat yang menunjuk isian tidak ditampilkan lagi sebagai kotak pesan.
    expect(screen.queryByText('Belum dapat disimpan')).not.toBeInTheDocument()
  })

  it.each([
    {
      name: 'validasi tanpa isian',
      answer: json(422, { kode: 'validasi_gagal', pesan: 'Ditolak tanpa rincian.' }),
      texts: ['Belum dapat disimpan', 'Ditolak tanpa rincian.'],
    },
    {
      name: 'sudah tidak ada',
      answer: json(404, { kode: 'tidak_ditemukan', pesan: 'x' }),
      texts: ['Pasal ini sudah tidak ada'],
    },
    {
      name: 'portal tidak disebut',
      answer: json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      texts: ['Pilih portal entitas di bagian atas halaman, lalu simpan lagi.'],
    },
    {
      name: 'portal tidak dikenal',
      answer: json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      texts: ['Pilih portal entitas di bagian atas halaman, lalu simpan lagi.'],
    },
    {
      name: 'portal belum siap',
      answer: json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      texts: ['Basis data entitas ini belum tersedia'],
    },
    {
      name: 'galat API lain',
      answer: json(500, { kode: 'galat_internal', pesan: 'x' }),
      texts: ['Terjadi kesalahan pada sistem'],
    },
    {
      name: 'jaringan putus',
      answer: 'putus' as Answer,
      texts: ['Server Claim PNC tidak dapat dihubungi'],
    },
  ])('menampilkan galat simpan: $name dan form tetap terbuka', async ({ answer, texts }) => {
    installFetch((call) => (call.method === 'PUT' ? answer : undefined))
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))
    await screen.findByText('Fire / Property')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    for (const text of texts) expect(await screen.findByText(text)).toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Ubah Pasal Kerugian' })).toBeInTheDocument()
  })

  it('tidak menampilkan kotak pesan untuk galat yang bukan galat API', async () => {
    installFetch((call) => (call.method === 'PUT' ? 'rusak' : undefined))
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))
    await screen.findByText('Fire / Property')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Simpan' })).toBeEnabled())
    expect(calls.some((c) => c.method === 'PUT')).toBe(true)
    expect(screen.queryByText('Terjadi kesalahan pada sistem')).not.toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Ubah Pasal Kerugian' })).toBeInTheDocument()
  })

  it('menyebut pemuatan dan mengunci isian sampai detail selesai dibaca', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((call) =>
      call.method === 'GET' && call.url === `${ROUTE}/2`
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))

    expect(await screen.findByText('Memuat isi pasal…')).toBeInTheDocument()
    expect(screen.getByLabelText('No Pasal')).toBeDisabled()
    expect(screen.queryByLabelText('Tambah lini bisnis')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Simpan' })).toBeDisabled()

    release(json(200, DETAIL_B))
    expect(await screen.findByText('Fire / Property')).toBeInTheDocument()
    expect(screen.queryByText('Memuat isi pasal…')).not.toBeInTheDocument()
  })

  it('menutup form lewat Batal', async () => {
    await openB()
    await userEvent.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('form', { name: 'Ubah Pasal Kerugian' })).not.toBeInTheDocument()
  })
})

describe('pemilih lini bisnis', () => {
  async function openCreate() {
    installFetch()
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    return screen.getByLabelText('Tambah lini bisnis')
  }

  it('tidak mencari untuk kata kunci yang terlalu pendek', async () => {
    const input = await openCreate()
    await userEvent.type(input, 'F')

    expect(screen.queryByText(/apa adanya/)).not.toBeInTheDocument()
    expect(calls.some((c) => c.url.startsWith(`${ROUTE}/bisnis`))).toBe(false)
  })

  it('menambah butir dari master dan menyembunyikan kode yang sudah dipilih', async () => {
    const input = await openCreate()
    await userEvent.type(input, 'Fi')

    await userEvent.click(await screen.findByRole('button', { name: /Fire \/ Property/ }))
    expect(screen.getByText('2004')).toBeInTheDocument()
    expect(input).toHaveValue('')
    expect(calls.some((c) => c.url === `${ROUTE}/bisnis?cari=Fi`)).toBe(true)

    await userEvent.type(input, 'Fi')
    expect(await screen.findByRole('button', { name: /Fire Industri/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Fire \/ Property/ })).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('No Pasal'), 'PSL-020')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))
    await waitFor(() => expect(calls.some((c) => c.method === 'POST')).toBe(true))
    expect(calls.find((c) => c.method === 'POST')?.body).toMatchObject({
      bisnis: [{ id: '2004', nama: 'Fire / Property' }],
    })
  })

  it('menerima nama yang diketik bebas sebagai butir tanpa kode', async () => {
    const input = await openCreate()
    await userEvent.type(input, '  Lini Karangan  ')

    await userEvent.click(
      await screen.findByRole('button', { name: /Tambahkan “Lini Karangan” apa adanya/ }),
    )
    expect(screen.getByText('Lini Karangan')).toBeInTheDocument()
    expect(screen.getByText('tanpa kode')).toBeInTheDocument()
  })

  it('menampilkan keadaan mencari lalu galat pencarian', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((call) =>
      call.url.startsWith(`${ROUTE}/bisnis`)
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Tambah lini bisnis'), 'Ma')

    expect(await screen.findByText('Mencari…')).toBeInTheDocument()
    release(json(500, { kode: 'galat_internal', pesan: 'x' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Pencarian gagal. Coba beberapa saat lagi.',
    )
  })
})

describe('hapus', () => {
  it('menampilkan galat penghapusan', async () => {
    installFetch((call) =>
      call.method === 'DELETE' ? json(404, { kode: 'tidak_ditemukan', pesan: 'x' }) : undefined,
    )
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Hapus PSL-002' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Hapus permanen' }))

    expect(
      await screen.findByText(
        'Penghapusan gagal. Baris mungkin sudah dihapus petugas lain — muat ulang daftarnya.',
      ),
    ).toBeInTheDocument()
    expect(screen.getByRole('alertdialog')).toBeInTheDocument()
  })

  it('menutup form bila baris yang sedang disunting justru yang dihapus', async () => {
    installFetch()
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah PSL-001' }))
    expect(await screen.findByRole('form', { name: 'Ubah Pasal Kerugian' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus PSL-001' }))
    // Membuka konfirmasi tidak menutup form.
    expect(screen.getByRole('form', { name: 'Ubah Pasal Kerugian' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hapus permanen' }))

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(screen.queryByRole('form', { name: 'Ubah Pasal Kerugian' })).not.toBeInTheDocument()
  })

  it('tetap membuka form bila baris lain yang dihapus', async () => {
    installFetch()
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))
    await screen.findByText('Fire / Property')

    await userEvent.click(screen.getByRole('button', { name: 'Hapus PSL-001' }))
    await userEvent.click(screen.getByRole('button', { name: 'Hapus permanen' }))

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(screen.getByRole('form', { name: 'Ubah Pasal Kerugian' })).toBeInTheDocument()
  })
})
