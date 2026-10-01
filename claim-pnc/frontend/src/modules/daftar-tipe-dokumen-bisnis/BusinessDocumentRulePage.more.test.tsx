import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { BusinessDocumentRulePage } from './BusinessDocumentRulePage'
import { idFromLabel, labelOf } from './BusinessDocumentRuleForm'

/**
 * Uji tambahan Daftar Tipe Dokumen Bisnis: cabang galat muat/simpan, form tambah dan ubah
 * di luar jalur utama, serta tampilan baris yang isiannya kosong. Data KARANGAN (`D-69`).
 */

const BUSINESS_LIST = {
  portal: 'ASM',
  total: 2,
  boleh_pilih_semua: true,
  bisnis: [
    { id: '001', nama_bisnis: 'Fire', dikecualikan_pilih_semua: false },
    { id: '009', nama_bisnis: '', dikecualikan_pilih_semua: false },
  ],
}

const BUSINESS_CHOICES = {
  portal: 'ASM',
  total: 2,
  boleh_pilih_semua: true,
  bisnis: [
    { id: '001', nama_bisnis: 'Fire', dikecualikan_pilih_semua: false },
    { id: '004', nama_bisnis: 'Marine Cargo', dikecualikan_pilih_semua: false },
  ],
}

const DOCUMENT_TYPES = { portal: 'ASM', total: 1, pilihan: [{ id: '20001', nama: 'REGISTER', id_induk: '' }] }
const DETAIL_DOCUMENTS = {
  portal: 'ASM',
  total: 1,
  pilihan: [{ id: '40001', nama: 'Laporan Kerugian', id_induk: '20001' }],
}
const OBJECT_DOCUMENTS = { portal: 'ASM', total: 1, pilihan: [{ id: '30001', nama: 'Bangunan', id_induk: '' }] }

const RULE = {
  id: '10001',
  id_bisnis: '001',
  nama_bisnis: '',
  // Kode yang tidak ada di master — layar wajib tetap menampilkan sesuatu.
  id_tipe_dokumen: '29999',
  tipe_dokumen: 'TAHAP LAMA',
  id_object_dokumen: '39999',
  object_dokumen: '',
  id_detail_dokumen: '40001',
  detail_dokumen: '',
  status_wajib: false,
  minimum_dokumen: 2,
  jenis_klaim: [] as string[],
}

const RULE_LIST = { portal: 'ASM', total: 1, tipe_dokumen_bisnis: [RULE] }

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

type Answer = Response | Promise<Response> | 'putus' | 'rusak' | undefined

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

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
    if (call.method !== 'GET') {
      return Promise.resolve(json(200, { portal: 'ASM', tipe_dokumen_bisnis: RULE }))
    }
    if (url.startsWith('/api/master/bisnis-pilihan')) return Promise.resolve(json(200, BUSINESS_CHOICES))
    if (url.startsWith('/api/master/tipe-dokumen-pilihan')) return Promise.resolve(json(200, DOCUMENT_TYPES))
    if (url.startsWith('/api/master/detail-dokumen-pilihan')) return Promise.resolve(json(200, DETAIL_DOCUMENTS))
    if (url.startsWith('/api/master/objek-dokumen-pilihan')) return Promise.resolve(json(200, OBJECT_DOCUMENTS))
    if (url.includes('/tipe-dokumen-bisnis/bisnis/')) return Promise.resolve(json(200, RULE_LIST))
    if (/\/tipe-dokumen-bisnis\/\d+$/.test(url)) {
      return Promise.resolve(json(200, { portal: 'ASM', tipe_dokumen_bisnis: RULE }))
    }
    return Promise.resolve(json(200, BUSINESS_LIST))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <BusinessDocumentRulePage />
    </QueryClientProvider>,
  )
}

const isBusinessList = (call: Call) =>
  call.method === 'GET' && call.url === '/api/master/tipe-dokumen-bisnis'
const isRuleList = (call: Call) => call.url.includes('/tipe-dokumen-bisnis/bisnis/')
const isRuleDetail = (call: Call) =>
  call.method === 'GET' && /\/tipe-dokumen-bisnis\/\d+$/.test(call.url)

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

async function openRules() {
  const table = await screen.findByRole('table')
  await userEvent.click(within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }))
}

async function openEdit() {
  await openRules()
  await userEvent.click(await screen.findByRole('button', { name: 'Ubah aturan 10001' }))
  return screen.findByRole('form', { name: 'Update Data' })
}

const ERRORS = [
  { name: 'jaringan putus', answer: (): Answer => 'putus', title: 'Server Claim PNC tidak dapat dihubungi' },
  {
    name: 'portal tidak disebut',
    answer: () => json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
    title: 'Portal entitas belum dipilih',
  },
  {
    name: 'portal tidak dikenal',
    answer: () => json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
    title: 'Portal entitas belum dipilih',
  },
  {
    name: 'portal belum siap',
    answer: () => json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
    title: 'Basis data entitas ini belum tersedia',
  },
]

describe('labelOf dan idFromLabel', () => {
  const list = [{ id: '1', nama: 'Satu', id_induk: '' }]

  it('menyusun label lalu mencocokkan label, nama, atau kode', () => {
    expect(labelOf(list[0]!)).toBe('Satu (1)')
    expect(idFromLabel(list, ' Satu (1) ')).toBe('1')
    expect(idFromLabel(list, 'Satu')).toBe('1')
    expect(idFromLabel(list, '1')).toBe('1')
    expect(idFromLabel(list, 'Lain')).toBe('')
    expect(idFromLabel(list, '   ')).toBe('')
  })
})

describe('daftar bisnis', () => {
  it.each([
    ...ERRORS,
    {
      name: 'galat API lain',
      answer: () => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }),
      title: 'Daftar tidak dapat dimuat',
    },
    { name: 'galat bukan API', answer: (): Answer => 'rusak', title: 'Daftar tidak dapat dimuat' },
  ])('menampilkan galat muat: $name', async ({ answer, title }) => {
    installFetch((call) => (isBusinessList(call) ? answer() : undefined))
    show()

    expect(screen.getByText('Memuat daftar bisnis…')).toBeInTheDocument()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('menampilkan pesan server dan pesan umum pada galat lain', async () => {
    installFetch((call) =>
      isBusinessList(call) ? json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }) : undefined,
    )
    show()
    expect(await screen.findByText('Basis data sibuk.')).toBeInTheDocument()
  })

  it('menampilkan pesan muat ulang untuk galat bukan API', async () => {
    installFetch((call) => (isBusinessList(call) ? 'rusak' : undefined))
    show()
    expect(await screen.findByText('Muat ulang halaman ini.')).toBeInTheDocument()
  })

  it('menampilkan pesan kosong bila belum ada bisnis beraturan', async () => {
    installFetch((call) =>
      isBusinessList(call) ? json(200, { ...BUSINESS_LIST, bisnis: [], total: 0 }) : undefined,
    )
    show()

    expect(
      await screen.findByText('Belum ada lini bisnis yang punya aturan dokumen pada entitas ini.'),
    ).toBeInTheDocument()
  })

  it('menandai bisnis tanpa nama dan memakai kodenya pada label tombol', async () => {
    installFetch()
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('(tanpa nama)')).toBeInTheDocument()
    await userEvent.click(within(table).getByRole('button', { name: 'Lihat dokumen bisnis 009' }))
    expect(await screen.findByRole('heading', { name: /Dokumen bisnis\s+009/ })).toBeInTheDocument()
  })

  it('memuat ulang daftar bisnis dan dokumen sekaligus lewat Refresh', async () => {
    installFetch()
    show()
    await openRules()
    await screen.findByText('TAHAP LAMA')
    const business = calls.filter(isBusinessList).length
    const rules = calls.filter(isRuleList).length

    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => expect(calls.filter(isBusinessList).length).toBe(business + 1))
    await waitFor(() => expect(calls.filter(isRuleList).length).toBe(rules + 1))
  })

  it('Refresh tanpa bisnis terpilih hanya memuat daftar bisnis', async () => {
    installFetch()
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => expect(calls.filter(isBusinessList).length).toBe(2))
    expect(calls.some(isRuleList)).toBe(false)
  })
})

describe('daftar dokumen bisnis', () => {
  it('menampilkan tanda kosong lalu menutup panel', async () => {
    installFetch()
    show()
    await openRules()

    const cell = await screen.findByText('TAHAP LAMA')
    const row = cell.closest('tr')!
    expect(within(row).getAllByText('—')).toHaveLength(2)
    expect(within(row).getByText('Tidak')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tutup' }))
    expect(screen.queryByText('TAHAP LAMA')).not.toBeInTheDocument()
  })

  it('menampilkan memuat, galat, dan kosong pada daftar dokumen', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((call) =>
      isRuleList(call)
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    await openRules()

    expect(await screen.findByText('Memuat dokumen…')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Copy' })).toBeDisabled()
    release(json(503, { kode: 'portal_belum_siap', pesan: 'x' }))
    expect(await screen.findByText('Basis data entitas ini belum tersedia')).toBeInTheDocument()
  })

  it('mematikan Copy bila bisnis belum punya aturan', async () => {
    installFetch((call) =>
      isRuleList(call) ? json(200, { portal: 'ASM', total: 0, tipe_dokumen_bisnis: [] }) : undefined,
    )
    show()
    await openRules()

    expect(
      await screen.findByText('Bisnis ini belum punya aturan dokumen. Tekan Tambah untuk membuatnya.'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Copy' })).toBeDisabled()
  })

  it('menyalin aturan dengan kode tak dikenal apa adanya ke form tambah', async () => {
    installFetch()
    show()
    await openRules()
    await screen.findByText('TAHAP LAMA')

    await userEvent.click(screen.getByRole('button', { name: 'Copy' }))

    const form = await screen.findByRole('form', { name: 'Tambah Data' })
    expect(within(form).getByLabelText('Tipe Dokumen')).toHaveValue('29999')
    expect(within(form).getByLabelText('Object Dokumen')).toHaveValue('39999')
    expect(within(form).getByLabelText('Detail Dokumen')).toHaveValue('Laporan Kerugian (40001)')
    expect(within(form).getByLabelText('Minimum Dokumen')).toHaveValue('2')
    // Tombol Tambah dan Copy dimatikan selama form terbuka.
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Copy' })).toBeDisabled()
  })
})

describe('form tambah', () => {
  async function openCreate() {
    installFetch(currentAnswer)
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    return screen.findByRole('form', { name: 'Tambah Data' })
  }

  let currentAnswer: (call: Call) => Answer = () => undefined

  beforeEach(() => {
    currentAnswer = () => undefined
  })

  it('mengelola pilihan bisnis dan baris dokumen sebelum menyimpan', async () => {
    const form = await openCreate()

    await within(form).findByRole('checkbox', { name: /Fire/ })
    await userEvent.click(within(form).getByRole('button', { name: 'Pilih semua' }))
    expect(within(form).getByText('2 dari 2 dipilih')).toBeInTheDocument()
    await userEvent.click(within(form).getByRole('checkbox', { name: /Fire/ }))
    expect(within(form).getByText('1 dari 2 dipilih')).toBeInTheDocument()
    await userEvent.click(within(form).getByRole('button', { name: 'Kosongkan' }))
    expect(within(form).getByText('0 dari 2 dipilih')).toBeInTheDocument()
    await userEvent.click(within(form).getByRole('checkbox', { name: /Marine Cargo/ }))

    expect(within(form).queryByRole('button', { name: 'Buang baris' })).not.toBeInTheDocument()
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah baris' }))
    expect(within(form).getAllByRole('button', { name: 'Buang baris' })).toHaveLength(2)
    expect(within(form).getByText(/1 bisnis × 2 dokumen/)).toBeInTheDocument()
    await userEvent.click(within(form).getAllByRole('button', { name: 'Buang baris' })[1]!)
    expect(within(form).getByText(/1 bisnis × 1 dokumen/)).toBeInTheDocument()

    await userEvent.type(within(form).getByLabelText('Object Dokumen'), 'Bangunan')
    await userEvent.click(within(form).getByRole('checkbox', { name: 'Status Wajib' }))
    const minimum = within(form).getByLabelText('Minimum Dokumen')
    await userEvent.clear(minimum)
    await userEvent.type(minimum, '4x')
    expect(minimum).toHaveValue('4')

    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'POST')).toBe(true))
    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({
      bisnis: ['004'],
      dokumen: [
        {
          id_tipe_dokumen: '',
          id_object_dokumen: '30001',
          id_detail_dokumen: '',
          detail_dokumen: '',
          status_wajib: true,
          minimum_dokumen: 4,
        },
      ],
    })
    await waitFor(() =>
      expect(screen.queryByRole('form', { name: 'Tambah Data' })).not.toBeInTheDocument(),
    )
  })

  it('menyembunyikan Pilih semua dan menyebut daftar bisnis yang tidak termuat', async () => {
    currentAnswer = (call) =>
      call.url.startsWith('/api/master/bisnis-pilihan') ? json(500, { kode: 'x', pesan: 'x' }) : undefined
    const form = await openCreate()

    expect(
      await within(form).findByText(/Daftar bisnis tidak dapat dimuat\./),
    ).toBeInTheDocument()
    expect(within(form).queryByRole('button', { name: 'Pilih semua' })).not.toBeInTheDocument()
  })

  it.each([
    {
      name: 'bisnis belum dipilih',
      answer: () => json(422, { kode: 'nama_bisnis_belum_diisi', pesan: 'x' }),
      title: 'Nama Bisnis belum di isi',
    },
    ...ERRORS,
    {
      name: 'galat API lain',
      answer: () => json(500, { kode: 'galat_internal', pesan: 'Simpan ditolak.' }),
      title: 'Penyimpanan gagal',
    },
  ])('menampilkan galat simpan: $name', async ({ answer, title }) => {
    currentAnswer = (call) => (call.method === 'POST' ? answer() : undefined)
    const form = await openCreate()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('tidak menampilkan pesan untuk galat bukan API lalu menutup lewat Batal', async () => {
    currentAnswer = (call) => (call.method === 'POST' ? 'rusak' : undefined)
    const form = await openCreate()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(within(form).getByRole('button', { name: 'Simpan' })).toBeEnabled())
    expect(calls.some((c) => c.method === 'POST')).toBe(true)
    expect(within(form).queryByText('Penyimpanan gagal')).not.toBeInTheDocument()

    await userEvent.click(within(form).getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('form', { name: 'Tambah Data' })).not.toBeInTheDocument()
  })
})

describe('form ubah', () => {
  it('menampilkan memuat lalu galat bila baris gagal dibaca', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((call) =>
      isRuleDetail(call)
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    await openRules()
    await userEvent.click(await screen.findByRole('button', { name: 'Ubah aturan 10001' }))

    expect(await screen.findByText('Memuat aturan dokumen…')).toBeInTheDocument()
    release(json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }))
    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
  })

  it('menampilkan nama dari join atau kodenya bila master tidak mengenalnya', async () => {
    installFetch()
    show()
    const form = await openEdit()

    expect(within(form).getByLabelText('Tipe Dokumen')).toHaveValue('TAHAP LAMA')
    expect(within(form).getByLabelText('Object Dokumen')).toHaveValue('39999')
    expect(within(form).getByLabelText('Detail Dokumen')).toHaveValue('Laporan Kerugian (40001)')
    expect(within(form).getByText('(tidak dapat dipindah)').parentElement).toHaveTextContent('—')
    expect(screen.getByText('Belum ada jenis klaim.')).toBeInTheDocument()
  })

  it('membuang huruf pada Minimum Dokumen dan mengirim nol bila kosong', async () => {
    installFetch()
    show()
    const form = await openEdit()

    await userEvent.clear(within(form).getByLabelText('Tipe Dokumen'))
    await userEvent.type(within(form).getByLabelText('Tipe Dokumen'), 'REGISTER')
    await userEvent.clear(within(form).getByLabelText('Minimum Dokumen'))
    await userEvent.type(within(form).getByLabelText('Minimum Dokumen'), 'abc')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    expect(calls.find((c) => c.method === 'PUT')?.body).toEqual({
      id_tipe_dokumen: '20001',
      id_object_dokumen: '',
      id_detail_dokumen: '40001',
      detail_dokumen: '',
      status_wajib: false,
      minimum_dokumen: 0,
    })
    await waitFor(() =>
      expect(screen.queryByRole('form', { name: 'Update Data' })).not.toBeInTheDocument(),
    )
  })

  it.each([
    {
      name: 'baris sudah tidak ada',
      answer: () => json(404, { kode: 'tipe_dokumen_bisnis_tidak_ditemukan', pesan: 'x' }),
      title: 'Aturan dokumen sudah tidak ada',
    },
    ...ERRORS,
    {
      name: 'galat API lain',
      answer: () => json(500, { kode: 'galat_internal', pesan: 'x' }),
      title: 'Penyimpanan gagal',
    },
  ])('menampilkan galat simpan: $name', async ({ answer, title }) => {
    installFetch((call) => (call.method === 'PUT' ? answer() : undefined))
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('tidak menampilkan pesan untuk galat simpan bukan API lalu menutup lewat Batal', async () => {
    installFetch((call) => (call.method === 'PUT' ? 'rusak' : undefined))
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(within(form).getByRole('button', { name: 'Simpan' })).toBeEnabled())
    expect(within(form).queryByText('Penyimpanan gagal')).not.toBeInTheDocument()
    await userEvent.click(within(form).getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('form', { name: 'Update Data' })).not.toBeInTheDocument()
  })

  it('mengosongkan kode jenis klaim setelah tersimpan dan menampilkan galatnya bila gagal', async () => {
    let fail = true
    installFetch((call) => {
      if (!call.url.endsWith('/jenis-klaim')) return undefined
      if (fail) return json(503, { kode: 'portal_belum_siap', pesan: 'x' })
      return json(200, { portal: 'ASM', tipe_dokumen_bisnis: { ...RULE, jenis_klaim: ['10015'] } })
    })
    show()
    await openEdit()

    const input = screen.getByLabelText('Kode Jenis Klaim')
    await userEvent.type(input, '10015')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah jenis klaim' }))
    expect(await screen.findByText('Basis data entitas ini belum tersedia')).toBeInTheDocument()
    expect(input).toHaveValue('10015')

    fail = false
    await userEvent.click(screen.getByRole('button', { name: 'Tambah jenis klaim' }))
    await waitFor(() => expect(input).toHaveValue(''))
  })
})

describe('pengurutan', () => {
  it('mengurutkan kedua tabel menurut setiap kolomnya', async () => {
    installFetch((call) =>
      isRuleList(call)
        ? json(200, {
            portal: 'ASM',
            total: 2,
            tipe_dokumen_bisnis: [
              RULE,
              { ...RULE, id: '10002', tipe_dokumen: 'A', detail_dokumen: '-', status_wajib: true },
            ],
          })
        : undefined,
    )
    show()
    await openRules()
    await screen.findByText('— disembunyikan')

    const [businessTable, ruleTable] = screen.getAllByRole('table')
    for (const title of ['ID', 'Nama Bisnis']) {
      const column = within(businessTable!).getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(column).getByRole('button'))
      expect(column).toHaveAttribute('aria-sort', 'ascending')
    }
    for (const title of [
      'Tipe Dokumen',
      'Object Dokumen',
      'Detail Dokumen',
      'Status Wajib',
      'Minimum Dokumen',
    ]) {
      const column = within(ruleTable!).getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(column).getByRole('button'))
      expect(column).toHaveAttribute('aria-sort', 'ascending')
    }
    // Baris "A" naik ke atas saat diurutkan menurut Tipe Dokumen.
    const tipe = within(ruleTable!).getByRole('columnheader', { name: /^Tipe Dokumen/ })
    await userEvent.click(within(tipe).getByRole('button'))
    expect(within(ruleTable!).getAllByRole('row')[1]).toHaveTextContent('— disembunyikan')
    expect(within(ruleTable!).getAllByRole('row')[2]).toHaveTextContent('TAHAP LAMA')
  })
})
