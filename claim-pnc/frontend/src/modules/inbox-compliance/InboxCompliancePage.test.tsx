import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-compliance'
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

/** Tab Compliance — kedelapan kolomnya seperti yang disusun backend. */
const TAB_COMPLIANCE: Tab = {
  kode: 'compliance',
  nama: 'Compliance',
  keterangan: 'Klaim yang menunggu pemeriksaan kepatuhan di antrean CompliancePNC.',
  kolom: [
    { kunci: 'nomor_case', judul: 'Nomor Case' },
    { kunci: 'no_polis', judul: 'No Polis' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
    { kunci: 'nama_bisnis', judul: 'Nama Bisnis' },
    { kunci: 'nama_cabang', judul: 'Nama Cabang' },
    { kunci: 'nama_admin', judul: 'Nama Admin' },
    { kunci: 'tanggal_kirim_compliance', judul: 'Tanggal Kirim Compliance' },
    { kunci: 'aging', judul: 'Aging' },
  ],
  tersedia: true,
}

/** Tab Post Audit — ketujuh kolomnya seperti yang disusun backend. */
const TAB_POST_AUDIT: Tab = {
  kode: 'post-audit',
  nama: 'Post Audit',
  keterangan: 'Pemeriksaan Post Audit yang baru dibuat dan belum ditindaklanjuti.',
  kolom: [
    { kunci: 'nomor_case', judul: 'Nomor Case' },
    { kunci: 'no_klaim', judul: 'No Klaim' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
    { kunci: 'no_polis', judul: 'No Polis' },
    { kunci: 'catatan_compliance', judul: 'Catatan' },
    { kunci: 'tanggal_kirim_post_audit', judul: 'Tanggal Kirim Audit Compliance' },
    { kunci: 'outstanding', judul: 'OutStanding' },
  ],
  tersedia: true,
}

/**
 * Tab buatan yang BELUM dapat dilayani.
 *
 * Ia tidak mewakili tab mana pun yang ada hari ini — Post Audit sudah hidup sejak DDL
 * `POOLDATA.T_CLAIM_COMPLIANCE_H` diterima. Ia ada supaya MEKANISME "tab belum siap" tetap
 * teruji, karena mekanismenya masih di dalam kode dan modul ini masih akan menumbuhkan tab
 * baru.
 */
const TAB_BELUM_SIAP: Tab = {
  kode: 'contoh-belum-siap',
  nama: 'Contoh Belum Siap',
  keterangan: 'Tab buatan untuk menguji panel penghalang.',
  kolom: [{ kunci: 'nomor_case', judul: 'Case ID' }],
  tersedia: false,
  penghalang: 'Menunggu artefak dari pihak lain.',
}

/** Jawaban pengiriman yang berhasil. */
const SEND_OK = {
  nomor_case: 'CPL-100001',
  no_klaim: 'ASM-FW-GCNMFW-WORK PNC-9001',
  nama_tertanggung: 'PT SUMBER CONTOH SENTOSA',
  no_polis: 'POL-CONTOH-0001',
  catatan: 'to compilance',
  tanggal_kirim_post_audit: '2026-09-24 09:57',
  portal: 'ASM',
}

const METADATA: MetadataResponse = {
  tab: [TAB_COMPLIANCE, TAB_POST_AUDIT],
  tab_bawaan: 'compliance',
  portal: 'ASM',
  keterbatasan: [
    'Kolom Aging memotong hari Sabtu dan Minggu, tetapi TIDAK memotong hari libur nasional.',
  ],
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const ROW: WorkItem = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-9001',
  nomor_case: 'PNC-9001',
  no_klaim: '',
  no_polis: 'POL-CONTOH-0001',
  nama_tertanggung: 'PT SUMBER CONTOH SENTOSA',
  nama_bisnis: 'Aneka',
  nama_cabang: 'Cabang Contoh Pusat',
  nama_admin: 'ADMINCONTOH1',
  tanggal_kirim_compliance: '2026-09-21',
  tanggal_kirim_post_audit: null,
  catatan_compliance: '',
  aging: '2 days 3 hours ago',
  aging_jam: 51,
  outstanding: '',
}

/** Satu baris tab Post Audit — kolomnya berbeda dari tab Compliance. */
const ROW_POST_AUDIT: WorkItem = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-9101',
  nomor_case: 'CPL-19',
  no_klaim: 'ASM-FW-GCNMFW-WORK PNC-2114',
  no_polis: 'POL-CONTOH-0009',
  nama_tertanggung: 'PT CONTOH SEMBILAN ABADI',
  nama_bisnis: '',
  nama_cabang: '',
  nama_admin: '',
  tanggal_kirim_compliance: null,
  tanggal_kirim_post_audit: '2026-09-18 13:46',
  catatan_compliance: 'Contoh catatan compliance',
  aging: '',
  aging_jam: null,
  outstanding: '6 days ago',
}

/** Baris tanpa Tanggal Kirim Compliance — kolom Aging-nya harus KOSONG, bukan nol jam. */
const ROW_TANPA_TANGGAL: WorkItem = {
  ...ROW,
  referensi: 'ASM-FW-GCNMFW-WORK PNC-9002',
  nomor_case: 'PNC-9002',
  tanggal_kirim_compliance: null,
  aging: '',
  aging_jam: null,
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

/** Peladen tiruan yang menjawab bentuk layar dan isi antrean. */
function stubDefaultFetch(rows: WorkItem[] = [ROW], total = rows.length, pages = 1) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    return jsonResponse(200, {
      tab: TAB_COMPLIANCE,
      baris: rows,
      paginasi: { halaman: 1, ukuran: 25, total, total_halaman: pages },
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
      <MemoryRouter initialEntries={['/inbox-compliance']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada bilah tab, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara bilah tab baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('tab', { name: /Compliance/ })
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
  it('membuka tab bawaan yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /^Compliance/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar kolom persis seperti yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const table = await screen.findByRole('table')
    for (const column of TAB_COMPLIANCE.kolom) {
      expect(
        within(table).getByRole('columnheader', { name: column.judul }),
      ).toBeInTheDocument()
    }
  })

  /**
   * Layar ini TIDAK punya kotak cari maupun penyaring, dan ketiadaannya diuji.
   *
   * `Section/InputCompliance_Section-Section.xml` tidak memuat satu pun field masukan.
   * Menambahkannya berarti mengarang kemampuan yang tidak pernah ada di sistem lama — dan
   * uji ini yang menahan penambahan itu lolos tanpa disadari.
   */
  it('tidak menggambar kotak cari maupun dropdown penyaring', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: /business/i })).not.toBeInTheDocument()
  })

  it('menampilkan keterbatasan yang dikirim server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText(/memotong hari Sabtu dan Minggu/)).toBeInTheDocument()
  })
})

describe('kolom Aging', () => {
  it('menampilkan teks Aging apa adanya dari server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('2 days 3 hours ago')).toBeInTheDocument()
  })

  /**
   * Baris tanpa Tanggal Kirim Compliance menampilkan tanda pisah, BUKAN "0 hours ago".
   *
   * Ini butir ke-13 daftar perbaikan eksplisit `P-5` (`D-49` butir 10) sebagaimana terlihat
   * pengguna: `GETSELISIHJAM` lama mengembalikan `0` pada setiap kegagalan, sehingga
   * kegagalan tak terbedakan dari klaim yang baru masuk antrean.
   */
  it('menampilkan tanda pisah, bukan nol jam, saat Aging tidak dapat dihitung', async () => {
    stubDefaultFetch([ROW_TANPA_TANGGAL])
    await renderLoaded()

    const table = await screen.findByRole('table')
    expect(within(table).queryByText(/0 hours ago/)).not.toBeInTheDocument()
    expect(within(table).getAllByText('—').length).toBeGreaterThan(0)
  })

  /**
   * Kolom Aging tidak dapat diurutkan, dan itu disengaja.
   *
   * `DataTable` mengurutkan berdasarkan teks, sedangkan isi kolom ini berbentuk
   * "2 days 3 hours ago" — mengurutkannya sebagai teks menaruh "10 hours ago" sebelum
   * "2 days ago".
   */
  it('tidak menawarkan pengurutan pada kolom Aging', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const table = await screen.findByRole('table')
    const aging = within(table).getByRole('columnheader', { name: 'Aging' })

    expect(within(aging).queryByRole('button')).not.toBeInTheDocument()
  })
})

describe('tab Post Audit', () => {
  it('dapat dibuka dan menggambar kolomnya sendiri', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      return jsonResponse(200, {
        tab: TAB_POST_AUDIT,
        baris: [ROW_POST_AUDIT],
        paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    })
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Post Audit/ }))

    const table = await screen.findByRole('table')
    for (const column of TAB_POST_AUDIT.kolom) {
      expect(
        within(table).getByRole('columnheader', { name: column.judul }),
      ).toBeInTheDocument()
    }

    expect(within(table).getByText('Contoh catatan compliance')).toBeInTheDocument()

    // "No Klaim" berisi kunci teknis Pega, bukan nomor klaim. Namanya menyesatkan dan
    // itu ditiru apa adanya (`D-13`) — uji ini yang menahan seseorang "memperbaikinya"
    // menjadi nomor klaim tanpa keputusan.
    expect(within(table).getByText('ASM-FW-GCNMFW-WORK PNC-2114')).toBeInTheDocument()
    expect(within(table).getByText('CPL-19')).toBeInTheDocument()

    // Kolom Aging tidak digambar di tab ini; yang ada OutStanding, dan keduanya berbeda
    // dasar hitungannya.
    expect(within(table).queryByRole('columnheader', { name: 'Aging' })).not.toBeInTheDocument()
    expect(within(table).getByText('6 days ago')).toBeInTheDocument()
  })

  /**
   * Tanggal Kirim Audit Compliance menampilkan JAM, berbeda dari kolom tanggal lain.
   *
   * Layar Pega menampilkannya sebagai `22/04/25 13:46`. Tanpa uji ini, penyeragaman
   * pemformatan tanggal kelak akan membuang jamnya tanpa ada yang menyadarinya.
   */
  it('menampilkan jam pada kolom Tanggal Kirim Audit Compliance', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      return jsonResponse(200, {
        tab: TAB_POST_AUDIT,
        baris: [ROW_POST_AUDIT],
        paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    })
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Post Audit/ }))

    const table = await screen.findByRole('table')
    expect(within(table).getByText('18 September 2026 13:46')).toBeInTheDocument()
  })

  it('meminta tab yang dipilih ke server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Post Audit/ }))

    expect(lastListCall()?.url).toContain('tab=post-audit')
  })
})

/**
 * Mekanisme "tab belum siap" TETAP diuji meski tidak ada tab yang memakainya hari ini.
 *
 * Ia sengaja dipertahankan, bukan dibuang bersama penghalang Post Audit: modul ini masih
 * akan menumbuhkan tab baru, dan panel penjelas inilah yang membedakan "menunggu pihak lain"
 * dari "layarnya rusak".
 */
describe('tab yang belum dapat dilayani', () => {
  function stubWithPendingTab() {
    stubFetch((url) => {
      if (url === TAB_PATH) {
        return jsonResponse(200, {
          ...METADATA,
          tab: [TAB_COMPLIANCE, TAB_BELUM_SIAP],
        })
      }
      return jsonResponse(200, {
        tab: TAB_COMPLIANCE,
        baris: [ROW],
        paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    })
  }

  it('tetap digambar beserta penandanya, bukan disembunyikan', async () => {
    stubWithPendingTab()
    await renderLoaded()

    const tab = screen.getByRole('tab', { name: /Contoh Belum Siap/ })
    expect(tab).toBeInTheDocument()
    expect(tab).toHaveAttribute('aria-disabled', 'true')
  })

  it('menjelaskan apa yang ditunggu saat dibuka, dan tidak memanggil server', async () => {
    stubWithPendingTab()
    await renderLoaded()

    await screen.findByRole('table')
    const before = calls.length

    await userEvent.click(screen.getByRole('tab', { name: /Contoh Belum Siap/ }))

    expect(await screen.findByText(/Menunggu artefak dari pihak lain/)).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()

    // Tidak ada permintaan baru: memanggil sesuatu yang sudah pasti gagal hanya menambah
    // galat di log tanpa menambah keterangan apa pun.
    expect(calls.length).toBe(before)
  })
})

describe('paginasi', () => {
  it('meminta halaman berikutnya ke server, bukan memotong di peramban', async () => {
    stubDefaultFetch([ROW], 40, 2)
    await renderLoaded()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))

    expect(lastListCall()?.url).toContain('halaman=2')
  })

  /**
   * Halaman pertama tidak mengirim parameter `halaman`, tetapi TETAP menyebut tabnya.
   *
   * Tab disebut eksplisit begitu bentuk layar tiba — bukan dibiarkan kosong supaya server
   * memilih bawaannya. Keduanya menghasilkan isi yang sama hari ini, tetapi yang eksplisit
   * membuat kunci cache TanStack Query dan jejak di log menyebut antrean mana yang
   * sebenarnya dibaca. Tanpa itu, dua permintaan yang berbeda tabnya dapat berbagi satu
   * kunci cache pada saat tab bawaan berubah.
   */
  it('menyebut tab tetapi tidak mengirim parameter halaman pada halaman pertama', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByRole('table')
    expect(lastListCall()?.url).toBe(`${PATH}?tab=compliance`)
    expect(lastListCall()?.url).not.toContain('halaman=')
  })
})

describe('pemisahan entitas', () => {
  it('meminta pengguna memilih portal lebih dulu', async () => {
    useSelectedPortal.getState().clear()
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText(/Pilih entitas lebih dulu/)).toBeInTheDocument()
    expect(lastListCall()).toBeUndefined()
  })

  it('mengirim portal yang sedang dipilih pada setiap permintaan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByRole('table')

    const headers = new Headers(lastListCall()?.init?.headers)
    expect(headers.get('X-Portal')).toBe('ASM')
  })
})

describe('galat', () => {
  it('menjelaskan saat bentuk layar tidak dapat dimuat', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(500, { kode: 'galat_internal', pesan: 'gagal' })
      return jsonResponse(200, {})
    })
    renderPage()

    expect(await screen.findByText(/Layar tidak dapat dibuka/)).toBeInTheDocument()
  })

  it('menjelaskan saat isi antrean tidak dapat dimuat', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      return jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' })
    })
    await renderLoaded()

    expect(await screen.findByText(/Antrean tidak dapat dimuat/)).toBeInTheDocument()
  })
})

describe('kirim ke Post Audit', () => {
  /** Menjawab metadata, daftar Compliance, dan pengiriman. */
  function stubWithSend(sendStatus = 201, sendBody: unknown = SEND_OK) {
    stubFetch((url, init) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url.endsWith('/post-audit') && init?.method === 'POST') {
        return jsonResponse(sendStatus, sendBody)
      }
      return jsonResponse(200, {
        tab: TAB_COMPLIANCE,
        baris: [ROW],
        paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    })
  }

  function sendCall(): Call | undefined {
    return calls.find((c) => c.url.endsWith('/post-audit'))
  }

  it('hanya menawarkan tombolnya di tab Compliance', async () => {
    stubWithSend()
    await renderLoaded()

    await screen.findByRole('table')
    expect(screen.getByRole('button', { name: 'Kirim ke Post Audit' })).toBeInTheDocument()
  })

  it('mengirim referensi klaim dan catatan, bukan isian yang diketik ulang', async () => {
    stubWithSend()
    await renderLoaded()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim ke Post Audit' }))

    // Klaimnya disebutkan ulang di form supaya salah klik terlihat sebelum dikirim.
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('PNC-9001')).toBeInTheDocument()
    expect(within(dialog).getByText('PT SUMBER CONTOH SENTOSA')).toBeInTheDocument()

    await userEvent.type(within(dialog).getByLabelText('Catatan'), 'to compilance')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Kirim' }))

    const sent = sendCall()
    expect(sent).toBeDefined()
    expect(JSON.parse(String(sent?.init?.body))).toEqual({
      referensi: 'ASM-FW-GCNMFW-WORK PNC-9001',
      catatan: 'to compilance',
    })
  })

  /**
   * Catatan boleh kosong — kolomnya nullable dan tidak ada bukti bahwa ia wajib.
   *
   * Uji ini menahan seseorang menambahkan kewajiban yang tidak ada sumbernya, yang akan
   * menahan pekerjaan petugas tanpa alasan.
   */
  it('mengizinkan pengiriman tanpa catatan', async () => {
    stubWithSend()
    await renderLoaded()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim ke Post Audit' }))

    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Kirim' }))

    expect(JSON.parse(String(sendCall()?.init?.body))).toEqual({
      referensi: 'ASM-FW-GCNMFW-WORK PNC-9001',
      catatan: '',
    })
  })

  it('menutup form setelah berhasil', async () => {
    stubWithSend()
    await renderLoaded()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim ke Post Audit' }))

    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Kirim' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  /**
   * Klaim yang sudah tidak ada di antrean dijawab 409, dan pesannya menjelaskan apa yang
   * perlu dilakukan — bukan sekadar "gagal".
   */
  it('menjelaskan saat klaimnya sudah tidak ada di antrean', async () => {
    stubWithSend(409, {
      kode: 'klaim_tidak_di_antrean',
      pesan: 'Klaim ini sudah tidak ada di antrean Compliance. Segarkan daftar.',
    })
    await renderLoaded()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim ke Post Audit' }))

    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Kirim' }))

    expect(await screen.findByText(/sudah tidak ada di antrean/)).toBeInTheDocument()

    // Formnya TETAP terbuka supaya catatan yang sudah diketik tidak hilang.
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('tidak mengirim apa pun saat dibatalkan', async () => {
    stubWithSend()
    await renderLoaded()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim ke Post Audit' }))

    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(sendCall()).toBeUndefined()
  })
})
