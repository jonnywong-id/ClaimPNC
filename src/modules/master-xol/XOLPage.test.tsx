import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/xol'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Tiga induk contoh yang MENIRU BENTUK produksi, termasuk keanehannya.
 *
 *   - Nomor BERLUBANG (10001, 10002, 10004) — 10003 pernah dihapus di produksi.
 *   - Satu induk ber-Type XOL dan Remark Komite KOSONG.
 */
const SAMPLE = [
  {
    id: '10001',
    nama: 'Section 1',
    tahun: '2018',
    kurs: 13500,
    tipe: '1',
    tipe_label: 'Property / Motor / Engineering',
    remark_pic: 'Revisi',
    pic: 'MARIATRIELSA',
    status_komite: '0',
    komite: 'NOVERHALOMOAN',
    remark_komite: 'OK',
    bisnis: [],
    layer: [],
  },
  {
    id: '10002',
    nama: '',
    tahun: '2017',
    kurs: 14500,
    tipe: '',
    tipe_label: '',
    remark_pic: '',
    pic: '',
    status_komite: '',
    komite: '',
    remark_komite: '',
    bisnis: [],
    layer: [],
  },
  {
    id: '10004',
    nama: 'TESTING',
    tahun: '2022',
    kurs: 14000,
    tipe: '',
    tipe_label: '',
    remark_pic: 'Pengajuan Master XOL',
    pic: 'NOVERHALOMOAN',
    status_komite: '1',
    komite: 'NOVERHALOMOAN',
    remark_komite: 'OK',
    bisnis: [],
    layer: [],
  },
]

/** Detail satu induk — daftar sengaja tidak membawanya, sehingga ia diambil terpisah. */
const DETAIL_10001 = {
  ...SAMPLE[0],
  bisnis: [{ id: '10013', nama: 'FIRE' }],
  layer: [
    {
      id: '10001',
      nama: 'Sub Layer',
      limit: 1095000,
      excess: 1155000,
      limit_idr: 14782500000,
      reas: [{ id: '10036322', nama: 'SWISS RE', share: 100 }],
    },
  ],
}

const FORM_OPTION = {
  tahun: ['2026', '2022', '2018', '2017'],
  tipe: [
    { kode: '1', label: 'Property / Motor / Engineering' },
    { kode: '2', label: 'PA / GA' },
    { kode: '3', label: 'Marine / Heavy Equipment' },
  ],
  portal: 'ASM',
}

const BUSINESS_OPTION = {
  bisnis: [
    { id: '10013', nama: 'FIRE' },
    { id: '10009', nama: 'ENGINEERING' },
    { id: '', nama: 'TREATY INWARD' },
  ],
  total: 3,
  portal: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(reply: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    if (url === `${ROUTES}/form`) return Promise.resolve(jsonResponse(200, FORM_OPTION))
    if (url.startsWith(`${ROUTES}/bisnis`)) {
      return Promise.resolve(jsonResponse(200, BUSINESS_OPTION))
    }
    return Promise.resolve(reply(url, init))
  })
}

/** Peladen tiruan yang menjawab daftar, detail, simpan, dan hapus. */
function installDefaultFetch(options: { peringatan?: string[] } = {}) {
  installFetch((url, init) => {
    if (url === ROUTES && init?.method === 'POST') {
      return jsonResponse(201, {
        // Server menerbitkan nomor induk DAN nomor layer; keduanya belum diketahui klien.
        xol: { ...SAMPLE[0], id: '10009', layer: [{ ...DETAIL_10001.layer[0], id: '10018' }] },
        portal: 'ASM',
        ...(options.peringatan ? { peringatan: options.peringatan } : {}),
      })
    }
    if (init?.method === 'PUT') {
      return jsonResponse(200, {
        xol: DETAIL_10001,
        portal: 'ASM',
        ...(options.peringatan ? { peringatan: options.peringatan } : {}),
      })
    }
    if (init?.method === 'DELETE') {
      return new Response(null, { status: 204 })
    }
    if (url === `${ROUTES}/10001`) {
      return jsonResponse(200, { xol: DETAIL_10001, portal: 'ASM' })
    }
    return jsonResponse(200, { xol: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/xol']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

// ── Kolom grid: harus SAMA PERSIS dengan layar Pega ──────────────────────────────

describe('kolom grid mengikuti layar Pega', () => {
  it('memuat tepat empat kolom data: ID, Tahun, Kurs, Remark Komite', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')

    const judul = screen.getAllByRole('columnheader').map((h) => h.textContent?.trim() ?? '')
    // Kolom kelima adalah kolom aksi dan sengaja tanpa judul, sama seperti di Pega.
    expect(judul.filter((t) => t !== '')).toEqual(['ID', 'Tahun', 'Kurs', 'Remark Komite'])
  })

  it('TIDAK memuat Nama, Type XOL, maupun Status Komite', async () => {
    // Ketiganya ada di tabel tetapi TIDAK ada di grid Pega. Versi pertama layar ini
    // menambahkannya; uji ini yang menjaganya tidak kembali.
    installDefaultFetch()
    show()
    await screen.findByText('10001')

    const judul = screen.getAllByRole('columnheader').map((h) => h.textContent?.trim())
    expect(judul).not.toContain('Nama')
    expect(judul).not.toContain('Type XOL')
    expect(judul).not.toContain('Status Komite')
  })

  it('aksi per baris hanya Update, tanpa Hapus', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')

    expect(screen.getAllByRole('button', { name: /^Update master XOL/ })).toHaveLength(3)
    expect(screen.queryByRole('button', { name: /^Hapus master XOL/ })).not.toBeInTheDocument()
  })

  it('tombol halaman bernama Tambah dan Refresh', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')

    expect(screen.getByRole('button', { name: 'Tambah' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument()
  })

  it('menampilkan isi keempat kolom apa adanya', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')

    expect(screen.getByText('2018')).toBeInTheDocument()
    expect(screen.getByText('13.500')).toBeInTheDocument()
    expect(screen.getAllByText('OK').length).toBeGreaterThan(0)
  })
})

// ── Form: isian dan judul panel mengikuti layar Pega ─────────────────────────────

describe('form mengikuti layar Pega', () => {
  it('berjudul UPDATE DATA XOL dan memuat tepat empat isian', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(await screen.findByText('UPDATE DATA XOL')).toBeInTheDocument()
    expect(screen.getByLabelText('Tahun')).toBeInTheDocument()
    expect(screen.getByLabelText('Kurs IDR')).toBeInTheDocument()
    expect(screen.getByLabelText('Type XOL')).toBeInTheDocument()
    expect(screen.getByLabelText('Remark PIC')).toBeInTheDocument()

    // Nama dan ID TIDAK punya isian — layar Pega pun tidak punya.
    expect(screen.queryByLabelText('Nama')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('ID')).not.toBeInTheDocument()
  })

  it('memakai dua panel bernama Detail Group Bisnis dan Detail Layer', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(await screen.findByText('Detail Group Bisnis')).toBeInTheDocument()
    expect(screen.getByText('Detail Layer')).toBeInTheDocument()
    // Teks kosongnya pun diambil apa adanya dari Pega.
    expect(screen.getAllByText('Data Tidak Ada').length).toBeGreaterThanOrEqual(2)
  })

  it('grid Layer memakai keempat judul kolom Pega', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('Detail Layer')

    for (const judul of ['ID Layer', 'Nama Layer', 'Limit (USD)', 'Excess (USD)']) {
      expect(screen.getByText(judul)).toBeInTheDocument()
    }
    // Limit (IDR) TIDAK ada di grid Pega.
    expect(screen.queryByText('Limit (IDR)')).not.toBeInTheDocument()
  })

  it('kedua dropdown memakai teks kosong --Pilih--', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const tahun = await screen.findByLabelText('Tahun')
    expect(within(tahun).getByRole('option', { name: '--Pilih--' })).toBeInTheDocument()
    expect(
      within(screen.getByLabelText('Type XOL')).getByRole('option', { name: '--Pilih--' }),
    ).toBeInTheDocument()
  })

  it('memuat isi induk lewat permintaan detail, bukan dari baris grid', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))

    await waitFor(() => {
      expect(calls.some((c) => c.url === `${ROUTES}/10001`)).toBe(true)
    })
    expect(await screen.findByDisplayValue('Sub Layer')).toBeInTheDocument()
    expect(screen.getByDisplayValue('SWISS RE')).toBeInTheDocument()
  })
})

// ── Fungsi Tambah dan Ubah benar-benar bekerja ───────────────────────────────────

describe('tambah dan ubah', () => {
  it('menambah mengirim POST dan menampilkan nomor yang diterbitkan server', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('UPDATE DATA XOL')

    await userEvent.selectOptions(screen.getByLabelText('Tahun'), '2026')
    await userEvent.clear(screen.getByLabelText('Kurs IDR'))
    await userEvent.type(screen.getByLabelText('Kurs IDR'), '16000')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(/Tersimpan dan diajukan ke komite/)).toBeInTheDocument()
    const posted = calls.find((c) => c.url === ROUTES && c.init?.method === 'POST')
    expect(posted).toBeDefined()
    expect(String(posted?.init?.body)).toContain('"tahun":"2026"')
    expect(String(posted?.init?.body)).toContain('"kurs":16000')
  })

  // Menyimpan tanpa catatan MENUTUP form. Kabarnya karena itu wajib muncul di halaman —
  // bila ia dipasang di dalam form, ia hilang bersama form pada saat yang sama.
  it('menyimpan tanpa catatan menutup form dan memberi kabar di halaman', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('UPDATE DATA XOL')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(/Tersimpan dan diajukan ke komite/)).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText('UPDATE DATA XOL')).not.toBeInTheDocument())
  })

  // Sebaliknya: ada catatan berarti form TETAP terbuka, supaya catatannya terbaca dan
  // pengguna dapat memperbaiki share lalu menyimpan lagi.
  it('menyimpan dengan catatan membiarkan form tetap terbuka', async () => {
    installDefaultFetch({ peringatan: ['Total share pada Sub Layer belum 100% (sekarang 60%).'] })
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('UPDATE DATA XOL')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Tersimpan, dengan catatan:')).toBeInTheDocument()
    expect(screen.getByText('UPDATE DATA XOL')).toBeInTheDocument()
  })

  // Tombol Tutup mengirim objek klik bila tidak dibungkus, dan halaman akan menampilkan
  // objek itu sebagai pesan berhasil.
  it('tombol Tutup tidak memunculkan pesan berhasil', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('UPDATE DATA XOL')

    await userEvent.click(screen.getByRole('button', { name: 'Tutup' }))

    await waitFor(() => expect(screen.queryByText('UPDATE DATA XOL')).not.toBeInTheDocument())
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('menyimpan dua kali setelah menambah TIDAK membuat induk kembar', async () => {
    // Cacat yang nyata bila nomornya tidak dicatat: penyimpanan kedua akan mengirim POST
    // untuk kedua kalinya, dan server menerbitkan induk baru lagi.
    //
    // Jawaban server diberi CATATAN dengan sengaja, karena itulah satu-satunya jalur yang
    // menyisakan tombol Simpan kedua: penyimpanan tanpa catatan menutup form.
    installDefaultFetch({ peringatan: ['Total share pada Sub Layer belum 100% (sekarang 60%).'] })
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('UPDATE DATA XOL')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))
    await screen.findByText('Tersimpan, dengan catatan:')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.filter((c) => c.init?.method === 'PUT')).toHaveLength(1)
    })
    expect(calls.filter((c) => c.url === ROUTES && c.init?.method === 'POST')).toHaveLength(1)
  })

  it('mengubah mengirim PUT ke nomor induknya', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))
    await screen.findByDisplayValue('Sub Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.find((c) => c.init?.method === 'PUT')
      expect(saved?.url).toBe(`${ROUTES}/10001`)
    })
  })

  it('membawa Nama yang sudah tersimpan meski tidak ada isiannya di layar', async () => {
    // Kolom NAMA terisi di produksi tetapi tidak punya isian di form Pega. Bila ia tidak
    // ikut dikirim, menyimpan akan MENGOSONGKAN nama yang sudah ada.
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))
    await screen.findByDisplayValue('Sub Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.init?.method === 'PUT')).toBe(true)
    })
    const saved = calls.find((c) => c.init?.method === 'PUT')
    expect(String(saved?.init?.body)).toContain('"nama":"Section 1"')
  })

  it('tidak pernah mengirim limit_idr maupun penanda layar', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))
    await screen.findByDisplayValue('Sub Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.init?.method === 'PUT')).toBe(true)
    })
    const body = String(calls.find((c) => c.init?.method === 'PUT')?.init?.body ?? '')
    expect(body).not.toContain('limit_idr')
    expect(body).not.toContain('tersimpan')
    expect(body).not.toContain('status_komite')
  })

  it('menampilkan peringatan total share sebagai catatan, bukan kegagalan', async () => {
    installDefaultFetch({ peringatan: ['Total share pada Sub Layer belum 100% (sekarang 60%).'] })
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))
    await screen.findByDisplayValue('Sub Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Tersimpan, dengan catatan:')).toBeInTheDocument()
    expect(screen.getByText(/belum 100%/)).toBeInTheDocument()
  })
})

// ── Baris anak ──────────────────────────────────────────────────────────────────

describe('baris anak', () => {
  it('menambah baris layer menampilkan grid Reas-nya', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('Detail Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))

    expect(await screen.findByText('Reas')).toBeInTheDocument()
    expect(screen.getByText('Share (%)')).toBeInTheDocument()
    expect(screen.getByText('(baru)')).toBeInTheDocument()
  })

  it('menghapus baris anak yang sudah tersimpan menembak server seketika', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Update master XOL 10001' }))
    await screen.findByDisplayValue('Sub Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Hapus reas baris 1' }))

    await waitFor(() => {
      const removed = calls.find((c) => c.init?.method === 'DELETE')
      expect(removed?.url).toBe(`${ROUTES}/10001/layer/10001/reas/10036322`)
    })
  })

  it('menghapus baris anak yang BELUM tersimpan tidak menembak server', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('Detail Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah Layer' }))
    await screen.findByText('(baru)')
    await userEvent.click(screen.getByRole('button', { name: 'Hapus Layer baris 1' }))

    expect(calls.some((c) => c.init?.method === 'DELETE')).toBe(false)
  })
})

// ── Portal ──────────────────────────────────────────────────────────────────────

describe('portal', () => {
  it('menyebut entitas yang menjawab permintaan', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('10001')
    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })
})
