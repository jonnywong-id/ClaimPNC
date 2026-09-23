import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
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
 * Ketiganya dipilih dengan sengaja, dan masing-masing menguji perilaku nyata:
 *
 *   - Nomor BERLUBANG (10001, 10002, 10004) — 10003 pernah dihapus di produksi, sehingga
 *     layar tidak boleh mengandaikan nomornya berurutan.
 *   - Satu induk ber-Type XOL KOSONG, karena dua dari delapan induk produksi memang NULL.
 *   - Status komite yang berbeda-beda, supaya lencananya teruji pada ketiga keadaan.
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
    remark_komite: 'ok',
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
    remark_komite: 'ok',
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
    // Kerangka layar memuat menunya sendiri sejak menu dibaca dari basis data. Ia dijawab
    // di sini supaya uji layar ini menguji layarnya, bukan jalur galat menu.
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
        xol: { ...SAMPLE[0], id: '10009' },
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
  // Layar berada di balik sesi. Tanpa ini SessionGuard melempar ke layar masuk.
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

describe('daftar', () => {
  it('menampilkan induk beserta nomor yang berlubang', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Section 1')).toBeInTheDocument()
    expect(screen.getByText('TESTING')).toBeInTheDocument()
    // 10003 tidak ada — nomor produksi memang berlubang.
    expect(screen.getByText('10004')).toBeInTheDocument()

    // Total datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/3 master XOL terdaftar/)).toBeInTheDocument()
  })

  it('menandai induk tanpa nama dan tanpa Type XOL, bukan membiarkannya kosong', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('(tanpa nama)')).toBeInTheDocument()
    expect(screen.getAllByText('(belum dipilih)').length).toBeGreaterThan(0)
  })

  it('menerjemahkan status komite menjadi kalimat', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Menunggu komite')).toBeInTheDocument()
    expect(screen.getByText('Disetujui')).toBeInTheDocument()
    expect(screen.getByText('Belum diajukan')).toBeInTheDocument()
  })

  it('menyebut entitas yang menjawab permintaan', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })
})

describe('form', () => {
  it('memberi tahu bahwa menyimpan sekaligus mengajukan ke komite', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(await screen.findByText('Menambah Data')).toBeInTheDocument()
    expect(screen.getByText(/mengajukan ke komite/)).toBeInTheDocument()
  })

  it('memuat isi induk lewat permintaan detail, bukan dari baris grid', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah master XOL 10001' }))

    // Baris grid tidak membawa anaknya; layar wajib menariknya sendiri.
    await waitFor(() => {
      expect(calls.some((c) => c.url === `${ROUTES}/10001`)).toBe(true)
    })
    expect(await screen.findByDisplayValue('Sub Layer')).toBeInTheDocument()
  })

  it('menghitung Limit (IDR) dari Limit dolar dikali kurs', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah master XOL 10001' }))

    // 1.095.000 × 13.500 = 14.782.500.000
    expect(await screen.findByText('14.782.500.000')).toBeInTheDocument()
  })

  it('menampilkan peringatan total share sebagai catatan, bukan sebagai kegagalan', async () => {
    installDefaultFetch({ peringatan: ['Total share pada Sub Layer belum 100% (sekarang 60%).'] })
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah master XOL 10001' }))
    await screen.findByDisplayValue('Sub Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Tersimpan, dengan catatan:')).toBeInTheDocument()
    expect(screen.getByText(/belum 100%/)).toBeInTheDocument()
  })

  it('tidak pernah mengirim limit_idr maupun kolom komite ke server', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah master XOL 10001' }))
    await screen.findByDisplayValue('Sub Layer')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.init?.method === 'PUT')).toBe(true)
    })
    const saved = calls.find((c) => c.init?.method === 'PUT')
    const body = String(saved?.init?.body ?? '')

    // Server menolak field tak dikenal; ketiganya wajib dibuang di layar.
    expect(body).not.toContain('limit_idr')
    expect(body).not.toContain('tersimpan')
    expect(body).not.toContain('status_komite')
  })
})

describe('hapus', () => {
  it('meminta konfirmasi dan menyebut bahwa hapus berkaskade', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Hapus master XOL 10001' }))

    const dialog = await screen.findByRole('alertdialog')
    expect(dialog).toHaveTextContent(/ikut terhapus/)
    // Tidak ada permintaan hapus sebelum dikonfirmasi.
    expect(calls.some((c) => c.init?.method === 'DELETE')).toBe(false)
  })

  it('menghapus setelah dikonfirmasi', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Hapus master XOL 10001' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Ya, hapus' }))

    await waitFor(() => {
      const removed = calls.find((c) => c.init?.method === 'DELETE')
      expect(removed?.url).toBe(`${ROUTES}/10001`)
    })
  })

  it('membatalkan tanpa mengirim apa pun', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('Section 1')
    await userEvent.click(screen.getByRole('button', { name: 'Hapus master XOL 10001' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Batal' }))

    await waitFor(() => {
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    })
    expect(calls.some((c) => c.init?.method === 'DELETE')).toBe(false)
  })
})
