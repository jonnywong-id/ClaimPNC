import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-admin'
const TAB_PATH = `${PATH}/tab`

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/** Tab contoh — mewakili ketiga bentuk kemampuan penyaring yang mungkin. */
const TAB_ALL_CASE: Tab = {
  kode: '11',
  nama: 'All Case Admin',
  keterangan: 'Klaim yang Anda buat dan masih berjalan.',
  kolom: [
    { kunci: 'case_id', judul: 'No Klaim' },
    { kunci: 'no_polis', judul: 'Policy no' },
    { kunci: 'nama_tertanggung', judul: 'Insured name' },
    { kunci: 'tanggal_kejadian', judul: 'Date of loss' },
  ],
  hanya_milik_saya: true,
  pakai_pencarian: true,
  pakai_lini_bisnis: false,
}

const TAB_ALL: Tab = {
  kode: '3',
  nama: 'ALL',
  keterangan: 'Seluruh klaim PNC yang masih berjalan.',
  kolom: [
    { kunci: 'case_id', judul: 'Case ID' },
    { kunci: 'no_polis', judul: 'Policy no' },
    { kunci: 'aging_total', judul: 'Total Aging' },
  ],
  hanya_milik_saya: false,
  pakai_pencarian: true,
  pakai_lini_bisnis: true,
}

const TAB_RCL_PUCL: Tab = {
  kode: '13',
  nama: 'Status RCL/PUCL',
  keterangan: 'Klaim yang suratnya sudah dicetak dan menunggu persetujuan PUCL.',
  kolom: [
    { kunci: 'case_id', judul: 'Case ID' },
    { kunci: 'status_rcl_pucl', judul: 'Status RCL/PUCL' },
  ],
  hanya_milik_saya: false,
  pakai_pencarian: false,
  pakai_lini_bisnis: false,
}

const METADATA: MetadataResponse = {
  tab: [TAB_ALL_CASE, TAB_ALL, TAB_RCL_PUCL],
  tab_bawaan: '11',
  lini_bisnis: [
    { kode: 'ALL', label: 'Semua Lini Bisnis' },
    { kode: 'PA', label: 'Personal Accident' },
  ],
  tab_dinonaktifkan: [
    { kode: '4', nama: 'Not Answered', alasan: 'tidak dipakai — sudah di-remark di Pega' },
  ],
  portal: 'ASM',
  keterbatasan: ['Kolom Aging dihitung dalam hari kalender, bukan hari kerja.'],
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const ROW: WorkItem = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-9001',
  case_id: 'PNC-9001',
  no_polis: 'POL-CONTOH-0001',
  nama_tertanggung: 'PT SUMBER CONTOH SENTOSA',
  nama_bisnis: 'Aneka',
  sumber_bisnis: 'Broker',
  nama_cabang: 'Cabang Contoh Pusat',
  cabang_klaim: 'Cabang Contoh Pusat',
  pembuat: 'ADMINPNC',
  tanggal_kejadian: '2026-03-12',
  tanggal_lapor: '2026-03-14',
  tanggal_input: '2026-03-15',
  catatan: '',
  posisi_klaim: 'On Progress',
  status_klaim: 'Register',
  status_lod: '',
  tanggal_request: null,
  cabang_polis: '',
  cabang_survei: '',
  pic_klaim: '',
  surveyor: '',
  no_survei: '',
  tanggal_masuk_inbox: null,
  deskripsi_analis: '',
  status_rcl_pucl: '',
  tanggal_cetak_surat: null,
  lama_klaim: '',
  status_kadaluarsa: '',
  aging_lapor: 6,
  aging_total: 5,
  aging_lod: null,
  aging_request: null,
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal dan menu, sehingga SETIAP layar di balik sesi ikut
    // memanggil keduanya. Dijawab otomatis supaya uji layar ini menguji layarnya.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url, init))
  })
}

/** Peladen tiruan yang menjawab bentuk layar dan satu baris antrean. */
function stubDefaultFetch(rows: WorkItem[] = [ROW], tab: Tab = TAB_ALL_CASE) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    return jsonResponse(200, {
      tab,
      baris: rows,
      paginasi: { halaman: 1, ukuran: 25, total: rows.length, total_halaman: 1 },
      penyaring: { bisnis: '', cari: '' },
      portal: 'ASM',
    })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-admin']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada salah satu tab, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara bilah tab baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('tab', { name: 'All Case Admin' })
}

function lastListCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url === PATH || c.url.startsWith(`${PATH}?`))
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

describe('bentuk layar', () => {
  it('membuka tab bawaan yang ditetapkan server, bukan tab pertama yang kebetulan ada', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Keputusan Work Owner 2026-09-20: layar terbuka pada All Case Admin.
    expect(screen.getByRole('tab', { name: 'All Case Admin' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const judul of ['No Klaim', 'Policy no', 'Insured name', 'Date of loss']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  it('tidak menggambar tab yang tidak dibangun, dan menjelaskan ketiadaannya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.queryByRole('tab', { name: 'Not Answered' })).not.toBeInTheDocument()
    expect(screen.getByText(/"Not Answered"/)).toBeInTheDocument()
    expect(screen.getByText(/sudah di-remark di Pega/)).toBeInTheDocument()
  })

  it('menyatakan keterbatasan yang dikirim server', async () => {
    // Tanpa ini, angka Aging yang lebih besar daripada di Pega akan dilaporkan berulang
    // kali sebagai kerusakan modul.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/hari kalender, bukan hari kerja/)).toBeInTheDocument()
  })
})

describe('penyaring', () => {
  it('menyembunyikan kotak cari pada tab yang tidak menyaringnya', async () => {
    // Tab Status RCL/PUCL tidak punya `{ASIS:TempContents.Keyword}` di kuerinya. Kotak
    // cari yang tidak menyaring apa pun membuat pengguna menyimpulkan antreannya kosong.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      return jsonResponse(200, {
        tab: TAB_RCL_PUCL,
        baris: [],
        paginasi: { halaman: 1, ukuran: 25, total: 0, total_halaman: 1 },
        penyaring: { bisnis: '', cari: '' },
        portal: 'ASM',
      })
    })
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'Status RCL/PUCL' }))

    expect(await screen.findByText(/menunggu persetujuan PUCL/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Cari')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Business')).not.toBeInTheDocument()
  })

  it('menggambar dropdown lini bisnis hanya pada tab yang mendukungnya', async () => {
    stubDefaultFetch([ROW], TAB_ALL)
    await renderLoaded()

    // Tab bawaan tidak mendukungnya.
    expect(screen.queryByLabelText('Business')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: 'ALL' }))
    expect(await screen.findByLabelText('Business')).toBeInTheDocument()
  })

  it('mengirim kata kunci ke server, bukan menyaring di peramban', async () => {
    // Antreannya dibaca dari tabel berpuluh juta baris (`D-10`); menyaring di peramban
    // berarti mengirim seluruh antrean pada setiap ketikan.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('Cari'), 'PNC-9001')

    await vi.waitFor(() => {
      expect(lastListCall()?.url).toContain('cari=PNC-9001')
    })
  })

  it('membersihkan penyaring saat berpindah tab', async () => {
    // Membawa kata kunci lama ke tab baru menampilkan antrean yang tampak kosong padahal
    // isinya ada, tanpa cara bagi pengguna melihat sebabnya.
    stubDefaultFetch([ROW], TAB_ALL)
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('Cari'), 'PNC-9001')
    await userEvent.click(screen.getByRole('tab', { name: 'ALL' }))

    expect(await screen.findByLabelText('Business')).toBeInTheDocument()
    expect(screen.getByLabelText('Cari')).toHaveValue('')

    await vi.waitFor(() => {
      expect(lastListCall()?.url).not.toContain('cari=')
    })
  })
})

describe('isi tabel', () => {
  it('memberi satuan hari pada kolom Aging', async () => {
    stubDefaultFetch([ROW], TAB_ALL)
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'ALL' }))

    expect(await screen.findByText('5 hari')).toBeInTheDocument()
  })

  it('menampilkan tanda pisah pada isian kosong, bukan sel kosong', async () => {
    // Sel kosong tidak dapat dibedakan dari kolom yang gagal dimuat.
    const empty: WorkItem = { ...ROW, nama_tertanggung: '', tanggal_kejadian: null }
    stubDefaultFetch([empty])
    await renderLoaded()

    const baris = await screen.findByRole('row', { name: /PNC-9001/ })
    expect(within(baris).getAllByText('—').length).toBeGreaterThanOrEqual(2)
  })

  it('menjelaskan antrean kosong menurut sebabnya', async () => {
    stubDefaultFetch([])
    await renderLoaded()

    // Tab bawaan hanya berisi pekerjaan milik pemanggil, dan pesannya menyebut itu —
    // bukan "tidak ada data" yang membuat pengguna menduga sistemnya rusak.
    expect(
      await screen.findByText('Tidak ada pekerjaan milik Anda di antrean ini.'),
    ).toBeInTheDocument()
  })
})

describe('tombol Lihat Detail Klaim', () => {
  it('membawa kunci klaim ke layar rincian', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const baris = await screen.findByRole('row', { name: /PNC-9001/ })
    await userEvent.click(within(baris).getByRole('button', { name: 'Lihat Detail Klaim' }))

    // Layar tujuannya adalah `MENU_ID 75` "View Claim" yang belum dibangun. Yang diuji di
    // sini: kuncinya benar-benar sampai, sehingga menyalakan layar itu kelak tidak
    // menuntut perubahan kontrak.
    expect(await screen.findByText('Layar rincian klaim belum dibangun')).toBeInTheDocument()
    expect(screen.getByText('ASM-FW-GCNMFW-WORK PNC-9001')).toBeInTheDocument()
  })
})

describe('portal', () => {
  it('tidak menembak server sebelum entitas dipilih', async () => {
    // Antrean kerja milik satu badan hukum. Menembak tanpa portal berarti meminta
    // jawaban yang sudah pasti ditolak server (`TKT-F6-002`).
    useSelectedPortal.getState().clear()
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(calls.filter((c) => c.url.startsWith(PATH))).toHaveLength(0)
  })
})
