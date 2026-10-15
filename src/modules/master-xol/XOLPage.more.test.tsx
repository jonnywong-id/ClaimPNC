import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { XOLPage } from './XOLPage'
import { committeeLabel, committeeTone, formatMoney } from './format'

/**
 * Uji tambahan Master XOL: cabang galat muat dan simpan, form tambah, dan baris anak.
 * Seluruh data KARANGAN (`D-69`).
 *
 * Berkas ini melengkapi `XOLPage.test.tsx`, yang menjaga agar **kolom dan teks layar sama
 * persis dengan Pega**. Yang dijaga di sini adalah PERILAKU pada jalur yang jarang
 * ditempuh — galat jaringan, galat validasi, dan penyuntingan baris anak.
 *
 * Seluruh uji "hapus induk" dibuang pada 2026-10-05: grid Pega tidak punya tombol Hapus,
 * dan layar ini mengikutinya. Endpoint hapus induk tetap ada di server.
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
      return Promise.resolve(
        jsonResponse(201, { xol: { ...SAMPLE[0], id: '10009' }, portal: 'ASM' }),
      )
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

/** Membuka form ubah. Grid hanya memuat ID, jadi barisnya dikenali dari nomornya. */
async function openEdit() {
  await screen.findByText('10001')
  await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))
  await screen.findByDisplayValue('Sub Layer')
}

async function openAdd() {
  await screen.findByText('10001')
  await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
  await screen.findByText('UPDATE DATA XOL')
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

// ── pembantu tampilan ───────────────────────────────────────────────────────────

describe('pembantu tampilan', () => {
  it('memformat angka dengan pemisah ribuan Indonesia', () => {
    expect(formatMoney(13500)).toBe('13.500')
    expect(formatMoney(0)).toBe('0')
    expect(formatMoney(Number.NaN)).toBe('0')
  })

  it('menerjemahkan status komite, dan menampilkan nilai asing apa adanya', () => {
    expect(committeeLabel('')).toBe('Belum diajukan')
    expect(committeeLabel('0')).toBe('Menunggu komite')
    expect(committeeLabel('1')).toBe('Disetujui')
    // Kolomnya VARCHAR2(100) tanpa constraint; nilai tak terduga tidak disembunyikan.
    expect(committeeLabel('X')).toBe('X')
  })

  it('memberi warna berbeda untuk tiap status', () => {
    expect(committeeTone('0')).toContain('amber')
    expect(committeeTone('1')).toContain('emerald')
    expect(committeeTone('')).toContain('slate')
    expect(committeeTone('X')).toContain('slate')
  })
})

// ── daftar ──────────────────────────────────────────────────────────────────────

describe('daftar', () => {
  it('menyaring baris lewat kotak cari', async () => {
    installFetch()
    show()
    await screen.findByText('10001')

    await userEvent.type(screen.getByLabelText(/Cari ID/), '10002')

    await waitFor(() => expect(screen.queryByText('10001')).not.toBeInTheDocument())
    expect(screen.getByText('10002')).toBeInTheDocument()
  })

  it('memuat ulang daftar saat tombol Refresh ditekan', async () => {
    installFetch()
    show()
    await screen.findByText('10001')
    const sebelum = calls.filter((c) => isList(c.url, c.init)).length

    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => {
      expect(calls.filter((c) => isList(c.url, c.init)).length).toBeGreaterThan(sebelum)
    })
    expect(await screen.findByRole('button', { name: 'Refresh' })).toBeEnabled()
  })

  it('menampilkan tanda hubung bila Remark Komite kosong', async () => {
    installFetch()
    show()
    await screen.findByText('10002')
    expect(screen.getByText('—')).toBeInTheDocument()
  })
})

// ── galat muat ──────────────────────────────────────────────────────────────────

describe('galat muat', () => {
  it('menjelaskan portal belum dipilih', async () => {
    installFetch((url, init) =>
      isList(url, init)
        ? jsonResponse(400, { kode: 'portal_tidak_disebut', pesan: 'Portal belum disebut.' })
        : undefined,
    )
    show()
    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
  })

  it('menjelaskan basis data entitas belum tersedia', async () => {
    installFetch((url, init) =>
      isList(url, init)
        ? jsonResponse(503, { kode: 'portal_belum_siap', pesan: 'Belum siap.' })
        : undefined,
    )
    show()
    expect(await screen.findByText('Basis data entitas ini belum tersedia')).toBeInTheDocument()
  })

  it('meneruskan pesan server pada galat lain', async () => {
    installFetch((url, init) =>
      isList(url, init)
        ? jsonResponse(500, { kode: 'galat_internal', pesan: 'Pesan dari server.' })
        : undefined,
    )
    show()
    expect(await screen.findByText('Pesan dari server.')).toBeInTheDocument()
  })

  it('menjelaskan jaringan putus', async () => {
    installFetch((url, init) => {
      if (!isList(url, init)) return undefined
      throw new TypeError('gagal')
    })
    show()
    expect(await screen.findByText('Tidak dapat menghubungi server')).toBeInTheDocument()
  })

  it('menjelaskan galat yang bukan APIError maupun NetworkError', async () => {
    installFetch((url, init) => (isList(url, init) ? brokenResponse() : undefined))
    show()
    expect(await screen.findByText('Daftar master XOL gagal dimuat')).toBeInTheDocument()
  })
})

// ── form tambah ─────────────────────────────────────────────────────────────────

describe('form tambah', () => {
  it('mengisi keempat isian beserta baris anak, lalu mengirim POST', async () => {
    installFetch()
    show()
    await openAdd()

    await userEvent.selectOptions(screen.getByLabelText('Tahun'), '2026')
    await userEvent.selectOptions(screen.getByLabelText('Type XOL'), '1')
    await userEvent.clear(screen.getByLabelText('Kurs IDR'))
    await userEvent.type(screen.getByLabelText('Kurs IDR'), '15000')
    await userEvent.type(screen.getByLabelText('Remark PIC'), 'catatan uji')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Nama Bisnis' }))
    await userEvent.selectOptions(screen.getByLabelText('Nama Bisnis baris 1'), '10013\u0000FIRE')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
    await userEvent.type(screen.getByLabelText('Nama Layer baris 1'), 'L baru')
    await userEvent.clear(screen.getByLabelText('Limit (USD) baris 1'))
    await userEvent.type(screen.getByLabelText('Limit (USD) baris 1'), '2')
    await userEvent.clear(screen.getByLabelText('Excess (USD) baris 1'))
    await userEvent.type(screen.getByLabelText('Excess (USD) baris 1'), '3')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Reas layer baris 1' }))
    await userEvent.type(screen.getByLabelText('ID reas baris 1'), 'R9')
    await userEvent.type(screen.getByLabelText('Reasuransi baris 1'), 'REAS BARU')
    await userEvent.clear(screen.getByLabelText('Share baris 1'))
    await userEvent.type(screen.getByLabelText('Share baris 1'), '100')

    // Total share ditampilkan hidup, sebelum menyimpan.
    expect(screen.getByText('Total share 100%')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.url === ROUTES && c.init?.method === 'POST')).toBe(true)
    })
    const body = String(calls.find((c) => c.init?.method === 'POST')?.init?.body ?? '')
    expect(body).toContain('"tahun":"2026"')
    expect(body).toContain('"kurs":15000')
    expect(body).toContain('"nama":"FIRE"')
    expect(body).toContain('"nama":"L baru"')
    expect(body).toContain('"nama":"REAS BARU"')
  })

  it('menandai total share yang belum 100% tanpa memblokir', async () => {
    installFetch()
    show()
    await openAdd()

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah Reas layer baris 1' }))
    await userEvent.clear(screen.getByLabelText('Share baris 1'))
    await userEvent.type(screen.getByLabelText('Share baris 1'), '60')

    expect(screen.getByText('Total share 60%')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Simpan' })).toBeEnabled()
  })

  it('membuang baris anak yang belum tersimpan tanpa memanggil server', async () => {
    installFetch()
    show()
    await openAdd()

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Nama Bisnis' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah Reas layer baris 1' }))

    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))
    await userEvent.click(screen.getByRole('button', { name: 'Hapus Layer baris 1' }))
    await userEvent.click(screen.getByRole('button', { name: 'Hapus Nama Bisnis baris 1' }))

    expect(calls.some((c) => c.init?.method === 'DELETE')).toBe(false)
    expect(screen.getAllByText('Data Tidak Ada').length).toBeGreaterThanOrEqual(2)
  })

  // Isian angka berangkat KOSONG, bukan nol.
  //
  // Versi pertama memberinya nilai awal `0`, dan uji ini dahulu menuntut kosong DITOLAK.
  // Keduanya karangan: layar Pega menampilkan kotak kosong dan menyimpannya tanpa keluhan
  // (kolomnya nullable, isiannya `pyRequired=false`). Nol yang terlanjur ada juga merusak
  // pengetikan — mengetik `1` menghasilkan `01`.
  it('memulai isian angka dalam keadaan kosong, bukan berisi nol', async () => {
    installFetch()
    show()
    await openAdd()

    expect(screen.getByLabelText('Kurs IDR')).toHaveValue('')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
    expect(screen.getByLabelText('Limit (USD) baris 1')).toHaveValue('')
    expect(screen.getByLabelText('Excess (USD) baris 1')).toHaveValue('')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Reas layer baris 1' }))
    expect(screen.getByLabelText('Share baris 1')).toHaveValue('')
  })

  // Inilah cacat yang dilaporkan Work Owner pada 2026-10-05: kotak berisi `0`, mengetik
  // `1` menghasilkan `01`. Uji ini mengetik SATU angka tanpa menghapus apa pun lebih dulu.
  it('mengetik satu angka menghasilkan angka itu, bukan didahului nol', async () => {
    installFetch()
    show()
    await openAdd()

    await userEvent.type(screen.getByLabelText('Kurs IDR'), '1')
    expect(screen.getByLabelText('Kurs IDR')).toHaveValue('1')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
    await userEvent.type(screen.getByLabelText('Limit (USD) baris 1'), '7')
    expect(screen.getByLabelText('Limit (USD) baris 1')).toHaveValue('7')
  })

  // Kosong diterima dan dikirim sebagai nol — bukan ditolak, bukan dibuang dari muatan.
  it('menyimpan isian angka yang dikosongkan sebagai nol', async () => {
    installFetch()
    show()
    await openAdd()

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.init?.method === 'POST')).toBe(true))
    const body = String(calls.find((c) => c.init?.method === 'POST')?.init?.body ?? '')
    expect(JSON.parse(body).kurs).toBe(0)
  })

  it('menolak angka yang tidak bulat atau negatif tanpa memanggil server', async () => {
    installFetch()
    show()
    await openAdd()

    await userEvent.type(screen.getByLabelText('Kurs IDR'), '-5')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText('Kurs IDR harus bilangan bulat tidak negatif.'),
    ).toBeInTheDocument()
    expect(calls.some((c) => c.init?.method === 'POST')).toBe(false)
  })

  it('menutup form tanpa menyimpan lewat tombol Tutup', async () => {
    installFetch()
    show()
    await openAdd()

    await userEvent.click(screen.getByRole('button', { name: 'Tutup' }))

    await waitFor(() => expect(screen.queryByText('UPDATE DATA XOL')).not.toBeInTheDocument())
    expect(calls.some((c) => c.init?.method === 'POST')).toBe(false)
  })
})

// ── form ubah ───────────────────────────────────────────────────────────────────

describe('form ubah', () => {
  it('menampilkan keadaan memuat lalu galat bila detail gagal dimuat', async () => {
    installFetch((url) =>
      url === `${ROUTES}/10001`
        ? jsonResponse(500, { kode: 'galat_internal', pesan: 'rusak' })
        : undefined,
    )
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))

    expect(await screen.findByText('Isi master XOL gagal dimuat')).toBeInTheDocument()
  })

  it('menyatakan tersimpan tanpa catatan bila server tidak mengirim peringatan', async () => {
    installFetch()
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(/Tersimpan dan diajukan ke komite/)).toBeInTheDocument()
  })

  it('menampilkan galat simpan: validasi berisi detail', async () => {
    installFetch((_url, init) =>
      init?.method === 'PUT'
        ? jsonResponse(422, {
            kode: 'validasi_gagal',
            pesan: 'Isian belum benar.',
            detail: [{ field: 'tahun', pesan: 'Tahun keliru.' }],
          })
        : undefined,
    )
    show()
    await openEdit()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Isian belum benar.')).toBeInTheDocument()
    expect(screen.getByText('Tahun keliru.')).toBeInTheDocument()
  })

  it('menampilkan galat simpan: validasi tanpa detail', async () => {
    installFetch((_url, init) =>
      init?.method === 'PUT'
        ? jsonResponse(422, { kode: 'validasi_gagal', pesan: 'Isian belum benar.' })
        : undefined,
    )
    show()
    await openEdit()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Gagal menyimpan')).toBeInTheDocument()
  })

  it('menampilkan galat simpan: nomor bentrok', async () => {
    installFetch((_url, init) =>
      init?.method === 'PUT'
        ? jsonResponse(409, { kode: 'id_xol_sudah_dipakai', pesan: 'bentrok' })
        : undefined,
    )
    show()
    await openEdit()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nomor bentrok')).toBeInTheDocument()
  })

  it('menampilkan galat simpan: jaringan putus', async () => {
    installFetch((_url, init) => {
      if (init?.method !== 'PUT') return undefined
      throw new TypeError('gagal')
    })
    show()
    await openEdit()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Tidak dapat menghubungi server')).toBeInTheDocument()
  })

  it('menampilkan galat simpan: galat bukan API', async () => {
    installFetch((_url, init) => (init?.method === 'PUT' ? brokenResponse() : undefined))
    show()
    await openEdit()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Gagal menyimpan')).toBeInTheDocument()
  })

  it('menghapus baris anak tersimpan seketika lewat server, dengan nomor yang benar', async () => {
    // Nomor layer harus `10001` — nomor dari server — bukan kunci internal useFieldArray.
    installFetch()
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))

    await waitFor(() => {
      const removed = calls.find((c) => c.init?.method === 'DELETE')
      expect(removed?.url).toBe(`${ROUTES}/10001/layer/10001/reas/R1`)
    })
  })

  it('menghapus layer tersimpan memakai nomor dari server', async () => {
    installFetch()
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus Layer baris 1' }))

    await waitFor(() => {
      const removed = calls.find((c) => c.init?.method === 'DELETE')
      expect(removed?.url).toBe(`${ROUTES}/10001/layer/10001`)
    })
  })

  it('menghapus baris bisnis tersimpan memakai ID dari server', async () => {
    installFetch()
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus Nama Bisnis baris 1' }))

    await waitFor(() => {
      const removed = calls.find((c) => c.init?.method === 'DELETE')
      expect(removed?.url).toBe(`${ROUTES}/10001/bisnis/10013`)
    })
  })

  it('memberi tahu bila baris anak gagal dihapus di server', async () => {
    installFetch((_url, init) =>
      init?.method === 'DELETE'
        ? jsonResponse(500, { kode: 'galat_internal', pesan: 'rusak' })
        : undefined,
    )
    show()
    await openEdit()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))

    expect(await screen.findByText(/Baris gagal dihapus di server/)).toBeInTheDocument()
  })

  it('menampilkan nomor layer yang sudah tersimpan, dan "(baru)" untuk yang belum', async () => {
    installFetch()
    show()
    await openEdit()

    // Nomornya dicari DI DALAM panel Detail Layer: angka yang sama juga muncul sebagai
    // ID induk di grid, sehingga pencarian seluruh layar akan ambigu.
    const panelLayer = screen.getByRole('heading', { name: 'Detail Layer' }).closest('section')
    expect(panelLayer).not.toBeNull()
    expect(within(panelLayer as HTMLElement).getByText('10001')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
    expect(within(panelLayer as HTMLElement).getByText('(baru)')).toBeInTheDocument()
  })
})

// ── pemisahan entitas ───────────────────────────────────────────────────────────

describe('portal', () => {
  it('menuntun memilih portal bila belum dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch()
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
  })

  it('menyebut portal pada setiap permintaan', async () => {
    installFetch()
    show()
    await screen.findByText('10001')

    const header = calls.find((c) => isList(c.url, c.init))?.init?.headers as
      | Record<string, string>
      | undefined
    expect(header?.['X-Portal']).toBe('ASM')
  })
})


// ── Kotak NAMA REASURANSI ────────────────────────────────────────────────────────
//
// Section pencariannya TIDAK ADA di export (`R-16`), sehingga bentuk layarnya tidak dapat
// disalin. Yang dijaga uji ini adalah hal-hal yang dapat salah secara nyata: tombol tidak
// menyimpan form, hasil mengisi KEDUA kolom, dan baris kembar tidak ikut masuk.
describe('cari reasuransi', () => {
  const HASIL_REAS = {
    reas: [
      { id: '10001', nama: 'REASURADUR ALFA INTERNASIONAL' },
      { id: '10002', nama: 'REASURADUR ALFA NUSANTARA' },
    ],
    total: 2,
    portal: 'ASM',
  }

  function installWithSearch() {
    installFetch((url) => {
      if (url.startsWith(`${ROUTES}/reas`)) return jsonResponse(200, HASIL_REAS)
      return undefined
    })
  }

  async function bukaLayerBaru() {
    await openAdd()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
  }

  it('hasil pencarian mengisi ID dan nama sekaligus', async () => {
    installWithSearch()
    show()
    await bukaLayerBaru()

    await userEvent.type(screen.getByLabelText('Nama Reasuransi layer baris 1'), 'alfa')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Reasuransi layer baris 1' }))

    await userEvent.click(await screen.findByText('REASURADUR ALFA INTERNASIONAL'))

    expect(screen.getByLabelText('ID reas baris 1')).toHaveValue('10001')
    expect(screen.getByLabelText('Reasuransi baris 1')).toHaveValue(
      'REASURADUR ALFA INTERNASIONAL',
    )
  })

  it('menekan Cari TIDAK menyimpan form', async () => {
    installWithSearch()
    show()
    await bukaLayerBaru()

    await userEvent.type(screen.getByLabelText('Nama Reasuransi layer baris 1'), 'alfa')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Reasuransi layer baris 1' }))
    await screen.findByText('REASURADUR ALFA NUSANTARA')

    expect(calls.some((c) => c.init?.method === 'POST')).toBe(false)
  })

  // Enter di dalam kotak cari akan menembus ke form induk bila tidak dicegah, dan di sana
  // Enter berarti Simpan — mencari reasuradur akan diam-diam menyimpan seluruh master.
  it('menekan Enter di kotak cari TIDAK menyimpan form', async () => {
    installWithSearch()
    show()
    await bukaLayerBaru()

    await userEvent.type(screen.getByLabelText('Nama Reasuransi layer baris 1'), 'alfa{Enter}')
    await screen.findByText('REASURADUR ALFA NUSANTARA')

    expect(calls.some((c) => c.init?.method === 'POST')).toBe(false)
  })

  // MST_XOL_REAS tidak punya kunci utama, jadi baris kembar benar-benar dapat tersimpan.
  it('memilih reasuradur yang sama dua kali tidak menambah baris kembar', async () => {
    installWithSearch()
    show()
    await bukaLayerBaru()

    await userEvent.type(screen.getByLabelText('Nama Reasuransi layer baris 1'), 'alfa')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Reasuransi layer baris 1' }))

    await userEvent.click(await screen.findByText('REASURADUR ALFA INTERNASIONAL'))
    await userEvent.click(screen.getByText('REASURADUR ALFA INTERNASIONAL'))

    expect(screen.getByLabelText('ID reas baris 1')).toBeInTheDocument()
    expect(screen.queryByLabelText('ID reas baris 2')).not.toBeInTheDocument()
  })

  it('pencarian tanpa hasil mengatakan Data Tidak Ada, bukan diam saja', async () => {
    installFetch((url) => {
      if (url.startsWith(`${ROUTES}/reas`)) {
        return jsonResponse(200, { reas: [], total: 0, portal: 'ASM' })
      }
      return undefined
    })
    show()
    await bukaLayerBaru()

    await userEvent.type(screen.getByLabelText('Nama Reasuransi layer baris 1'), 'zzz')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Reasuransi layer baris 1' }))

    await waitFor(() => {
      expect(screen.getAllByText('Data Tidak Ada').length).toBeGreaterThanOrEqual(1)
    })
  })

  // Pencarian gagal tidak boleh menutup jalan: kolom ID dan Reasuransi tetap dapat diketik.
  it('pencarian gagal memberi tahu bahwa isian manual masih bisa dipakai', async () => {
    installFetch((url) => {
      if (url.startsWith(`${ROUTES}/reas`)) return jsonResponse(500, { kode: 'galat_internal' })
      return undefined
    })
    show()
    await bukaLayerBaru()

    await userEvent.type(screen.getByLabelText('Nama Reasuransi layer baris 1'), 'alfa')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Reasuransi layer baris 1' }))

    expect(await screen.findByText(/masih dapat diketik langsung di grid/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah Reas layer baris 1' }))
    expect(screen.getByLabelText('Reasuransi baris 1')).toBeInTheDocument()
  })
})
