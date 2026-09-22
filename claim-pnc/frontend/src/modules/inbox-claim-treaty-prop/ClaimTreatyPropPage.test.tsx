import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-claim-treaty-prop'
const TAB_PATH = `${PATH}/tab`

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'admintreaty1',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/** Tab antrean milik pemanggil — satu-satunya yang mengenal "See All Claim". */
const TAB_WORKLIST: Tab = {
  kode: '1',
  nama: 'Work List Treatyin Propotional',
  keterangan: 'Klaim treaty proporsional yang ditugaskan kepada Anda.',
  kolom: [
    { kunci: 'claim_id', judul: 'Claim ID' },
    { kunci: 'id_master', judul: 'ID Master' },
    { kunci: 'no_polis', judul: 'Policy No' },
    { kunci: 'tanggal_kejadian', judul: 'Date Of Loss' },
    { kunci: 'ceding_co', judul: 'Ceding Co Name' },
    { kunci: 'nama_tertanggung', judul: 'Insured Name' },
  ],
  hanya_milik_saya: true,
  pakai_lihat_semua: true,
  terhalang: false,
}

/** Tab antrean bersama — satu-satunya yang punya kolom Subjectivity. */
const TAB_TEKNIK: Tab = {
  kode: '2',
  nama: 'Work Teknik Treatyin',
  keterangan: 'Antrean bersama PIC Teknik treaty — belum diambil siapa pun.',
  kolom: [
    { kunci: 'claim_id', judul: 'Claim ID' },
    { kunci: 'tanggal_kejadian', judul: 'Date Of Loss' },
    { kunci: 'subjectivity', judul: 'Subjectivity' },
  ],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  terhalang: false,
}

/** Tab yang digambar tetapi belum dapat diisi. */
const TAB_KOMITE: Tab = {
  kode: '3',
  nama: 'Komite Treaty ASM',
  keterangan: 'Antrean persetujuan komite treaty.',
  kolom: [],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  terhalang: true,
  alasan_terhalang:
    'Antrean komite treaty belum dapat dibaca sistem baru. Kedua sumbernya di Pega ' +
    'mengambil nilai dari properti di dalam BLOB objek kerja, bukan dari kolom tabel.',
  pemilik_penghalang: 'DBA — dibutuhkan DDL DATAPEGA.PC_ASM_FW_GCNMFW_WORK.',
}

const METADATA: MetadataResponse = {
  tab: [TAB_WORKLIST, TAB_TEKNIK, TAB_KOMITE],
  tab_bawaan: '1',
  selisih_terencana: [
    'Kolom "Date Of Loss" pada tab Work Teknik Treatyin kini TERISI.',
    'Tombol "Create Claim Treaty Prop" belum membuat klaim.',
  ],
  portal: 'ASM',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const ROW: WorkItem = {
  referensi: 'ASSIGN-WORKLIST CLMP-1001!FLOW',
  claim_id: 'CLMP-1001',
  id_master: 'TRP-2026-01',
  no_polis: '99.001.2026.00001',
  tanggal_kejadian: '2026-08-14',
  nama_bisnis: 'Property All Risk',
  sumber_bisnis: 'Treaty Inward',
  ceding_co: 'Asuransi Contoh Pertama',
  nama_tertanggung: 'PT Contoh Sejahtera',
  subjectivity: '',
}

const ROW_TEKNIK: WorkItem = {
  ...ROW,
  referensi: 'ASSIGN-WORKBASKET CLMP-2001!FLOW',
  claim_id: 'CLMP-2001',
  subjectivity: '1',
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
function stubDefaultFetch(rows: WorkItem[] = [ROW], tab: Tab = TAB_WORKLIST) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    return jsonResponse(200, {
      tab,
      baris: rows,
      paginasi: { halaman: 1, ukuran: 25, total: rows.length, total_halaman: 1 },
      penyaring: { lihat_semua: false },
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
      <MemoryRouter initialEntries={['/inbox-claim-treaty-prop']}>
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
  await screen.findByRole('tab', { name: /Work List Treatyin Propotional/ })
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
  it('membuka tab bawaan yang ditetapkan server, yaitu antrean milik pemanggil', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('tab', { name: /Work List Treatyin Propotional/ }),
    ).toHaveAttribute('aria-selected', 'true')
  })

  it('mempertahankan salah eja judul tab seperti di Pega (D-13)', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // "Propotional", bukan "Proportional". Membetulkannya akan membuat tab ini tidak
    // dikenali pengguna yang mencarinya.
    expect(screen.getByRole('tab', { name: /Propotional/ })).toBeInTheDocument()
  })

  it('menggambar kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const judul of [
      'Claim ID',
      'ID Master',
      'Policy No',
      'Date Of Loss',
      'Ceding Co Name',
      'Insured Name',
    ]) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  it('menampilkan selisih terencana supaya tidak dilaporkan sebagai kerusakan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByText(/Yang berbeda dari layar lama, dan itu disengaja/),
    ).toBeInTheDocument()
    expect(screen.getByText(/Date Of Loss.*kini TERISI/)).toBeInTheDocument()
  })
})

describe('tab yang belum dapat diisi', () => {
  it('tetap digambar dengan penanda, bukan disembunyikan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const komite = screen.getByRole('tab', { name: /Komite Treaty ASM/ })
    expect(komite).toBeInTheDocument()
    expect(within(komite).getByText('belum tersedia')).toBeInTheDocument()
  })

  it('menjelaskan alasan dan pemilik penghalangnya, bukan tabel kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Komite Treaty ASM/ }))

    expect(await screen.findByText(/belum dapat dibaca sistem baru/)).toBeInTheDocument()
    expect(screen.getByText(/DDL DATAPEGA.PC_ASM_FW_GCNMFW_WORK/)).toBeInTheDocument()

    // Tidak ada tabel sama sekali: tabel kosong terbaca sebagai "tidak ada pekerjaan",
    // padahal yang benar adalah "belum dapat dibaca".
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('tidak meminta isinya ke server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const before = calls.filter((c) => c.url.startsWith(PATH) && c.url !== TAB_PATH).length
    await userEvent.click(screen.getByRole('tab', { name: /Komite Treaty ASM/ }))
    await screen.findByText(/belum dapat dibaca sistem baru/)

    const after = calls.filter((c) => c.url.startsWith(PATH) && c.url !== TAB_PATH).length
    expect(after).toBe(before)
  })
})

describe('See All Claim', () => {
  it('hanya digambar pada tab yang mengenalnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByLabelText('See All Claim')).toBeInTheDocument()

    stubDefaultFetch([ROW_TEKNIK], TAB_TEKNIK)
    await userEvent.click(screen.getByRole('tab', { name: /Work Teknik Treatyin/ }))

    expect(await screen.findByText(/Antrean bersama PIC Teknik/)).toBeInTheDocument()
    expect(screen.queryByLabelText('See All Claim')).not.toBeInTheDocument()
  })

  it('mengirim penyaringnya ke server, bukan menyaring di peramban', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByLabelText('See All Claim'))

    // Menyaring di peramban hanya menyentuh halaman yang sedang terbuka, sehingga
    // hasilnya menyesatkan pada data berhalaman.
    await vi.waitFor(() => {
      expect(lastListCall()?.url).toContain('lihat_semua=1')
    })
  })

  it('memperingatkan bahwa yang tampil bukan lagi milik pemanggil saja', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/hanya berisi pekerjaan milik Anda/)).toBeInTheDocument()

    await userEvent.click(screen.getByLabelText('See All Claim'))

    expect(
      await screen.findByText(/termasuk yang bukan milik Anda/),
    ).toBeInTheDocument()
  })
})

describe('isi grid', () => {
  it('memformat Tanggal Kejadian yang berbentuk YYYY-MM-DD', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const row = (await screen.findByText('CLMP-1001')).closest('tr')
    expect(row).not.toBeNull()
    expect(within(row as HTMLElement).getByText('14 Agustus 2026')).toBeInTheDocument()
  })

  it('menampilkan Tanggal Kejadian yang bentuknya tidak dikenali APA ADANYA', async () => {
    // Nilainya dibaca dari blob JSON yang bentuknya tidak dapat diperiksa (`R-08`).
    // Memaksakan pemformatan akan mengubahnya menjadi teks yang salah.
    stubDefaultFetch([{ ...ROW, tanggal_kejadian: '14/08/2026' }])
    await renderLoaded()

    const row = (await screen.findByText('CLMP-1001')).closest('tr')
    expect(within(row as HTMLElement).getByText('14/08/2026')).toBeInTheDocument()
  })

  it('menggambar tanda pisah untuk isian kosong, bukan sel kosong', async () => {
    stubDefaultFetch([{ ...ROW, id_master: '' }])
    await renderLoaded()

    const row = (await screen.findByText('CLMP-1001')).closest('tr')
    expect(within(row as HTMLElement).getByText('—')).toBeInTheDocument()
  })

  it('menjelaskan antrean kosong menurut sebabnya', async () => {
    stubDefaultFetch([])
    await renderLoaded()

    expect(
      await screen.findByText(/Tidak ada pekerjaan klaim treaty milik Anda/),
    ).toBeInTheDocument()
  })
})

describe('tombol pembuat klaim', () => {
  it('digambar meski belum membuat apa pun', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('button', { name: 'Create Claim Treaty Prop' }),
    ).toBeInTheDocument()
  })

  it('menampilkan alasan dari server, bukan gagal diam-diam', async () => {
    stubFetch((url, init) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (init?.method === 'POST') {
        return jsonResponse(501, {
          kode: 'belum_tersedia',
          pesan:
            'Pembuatan klaim treaty belum tersedia di sistem baru. Buat klaim treaty ' +
            'baru lewat Pega.',
        })
      }
      return jsonResponse(200, {
        tab: TAB_WORKLIST,
        baris: [ROW],
        paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
        penyaring: { lihat_semua: false },
        portal: 'ASM',
      })
    })

    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: 'Create Claim Treaty Prop' }))

    expect(await screen.findByText(/Buat klaim treaty baru lewat Pega/)).toBeInTheDocument()
  })
})

describe('portal', () => {
  it('menolak membuka layar sebelum entitas dipilih', async () => {
    stubDefaultFetch()
    useSelectedPortal.getState().clear()

    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  })

  it('menyebut portal pada setiap permintaan antrean', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await vi.waitFor(() => {
      const header = lastListCall()?.init?.headers as Record<string, string> | undefined
      expect(header?.['X-Portal']).toBe('ASM')
    })
  })
})
