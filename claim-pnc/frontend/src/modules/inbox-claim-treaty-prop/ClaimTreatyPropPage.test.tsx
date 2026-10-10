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

/**
 * Antrean Admin.
 *
 * `hanya_milik_saya` dan `pakai_lihat_semua` KEDUANYA false, mengikuti server: Report
 * Definition `InboxKlaimPropAdmin` tidak menyaring menurut petugas, sehingga tidak ada
 * penyaring yang dapat dilepas checkbox "See All Claim".
 */
const TAB_WORKLIST: Tab = {
  kode: '1',
  nama: 'Prop Treaty-in Admin',
  judul_grid: 'Work List Treatyin Propotional',
  keterangan: 'Seluruh penugasan klaim treaty proporsional di entitas ini.',
  kolom: [
    { kunci: 'claim_id', judul: 'Claim ID' },
    { kunci: 'id_master', judul: 'ID Master' },
    { kunci: 'no_polis', judul: 'Policy No' },
    { kunci: 'tanggal_kejadian', judul: 'Date Of Loss' },
    { kunci: 'ceding_co', judul: 'Ceding Co Name' },
    { kunci: 'nama_tertanggung', judul: 'Insured Name' },
    { kunci: 'operator_pengubah', judul: 'Last update' },
    { kunci: 'status_kerja', judul: 'Status Claim ID' },
  ],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  terhalang: false,
}

/** Tab antrean bersama — satu-satunya yang punya kolom Subjectivity. */
const TAB_TEKNIK: Tab = {
  kode: '2',
  nama: 'Prop Treaty-in Teknik',
  judul_grid: 'Work Teknik Treatyin',
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
  operator_pengubah: 'ADMINTREATY1',
  status_kerja: 'New',
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
 * Penantiannya pada dropdown pemilih antrean, bukan pada judul layar: judulnya sudah ada
 * sejak penggambaran pertama, sementara isi dropdown baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('option', { name: 'Prop Treaty-in Admin' })
}

/**
 * queueSelect mengambil dropdown pemilih antrean.
 *
 * Dicari menurut NAMANYA, bukan menurut urutan combobox di halaman: bilah atas memuat
 * pemilih portal yang juga combobox, dan mengambil "yang pertama" akan menguji kontrol yang
 * salah begitu urutannya berubah.
 */
function queueSelect(): HTMLSelectElement {
  return screen.getByRole('combobox', {
    name: 'Antrean klaim treaty proporsional',
  }) as HTMLSelectElement
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
  it('memakai judul kontainer Pega, bukan nama menunya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // "Claim Treatyin In Progress" adalah judul kontainer di section Pega dan itu pula
    // yang terbaca di layar produksi (`D-13`). Nama menunya tetap di menu kiri.
    expect(
      screen.getByRole('heading', { name: 'Claim Treatyin In Progress' }),
    ).toBeInTheDocument()
  })

  it('memilih antrean lewat dropdown, bukan bilah tab', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Bentuk kontrolnya mengikuti Pega produksi (keputusan Work Owner 2026-09-29).
    // Bilah tab yang dipakai versi sebelumnya tidak boleh tersisa.
    expect(queueSelect()).toBeInTheDocument()
    expect(screen.queryAllByRole('tab')).toHaveLength(0)
  })

  it('membuka antrean bawaan yang ditetapkan server, yaitu milik pemanggil', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(queueSelect().value).toBe('1')
  })

  it('memberi label dropdown dari FilterWorkBasket_Act, bukan judul grid', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Label pilihannya datang dari `Data Transform/FilterWorkBasket_Act-DT.xml`
    // (`CARI2`), BUKAN dari judul kontainer grid. Keduanya sempat tertukar di sini.
    expect(screen.getByRole('option', { name: 'Prop Treaty-in Admin' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Prop Treaty-in Teknik' })).toBeInTheDocument()
    expect(
      screen.queryByRole('option', { name: /Work List Treatyin/ }),
    ).not.toBeInTheDocument()
  })

  it('menggambar entri kosong "Choose" seperti dropdown Pega', async () => {
    // `pyHasNoSelection=true`, `pyNoSelectionText="Choose"`. Ia bukan sebuah antrean.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('option', { name: 'Choose' })).toBeInTheDocument()
  })

  it('memperlakukan "Choose" sama dengan antrean admin, bukan layar kosong', async () => {
    // Keduanya bernilai `CARI1` kosong di Pega, sehingga pemeriksaan yang memilih antrean
    // teknik gagal pada keduanya dan yang tampil adalah antrean admin. Kotaknya karena itu
    // kembali menunjuk "Prop Treaty-in Admin" alih-alih menggantung.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.selectOptions(queueSelect(), '')

    expect(queueSelect().value).toBe('1')
    expect(await screen.findByText('CLMP-1001')).toBeInTheDocument()
  })

  it('mempertahankan salah eja judul GRID seperti di Pega (D-13)', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // "Propotional", bukan "Proportional" — dan ia judul GRID, bukan teks pilihan.
    expect(
      await screen.findByRole('heading', { name: /Work List Treatyin Propotional/ }),
    ).toBeInTheDocument()
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
      'Last update',
      'Status Claim ID',
    ]) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  // Keputusan Work Owner 2026-10-06: panel selisih terencana DIHAPUS dari seluruh layar.
  //
  // Daftarnya tetap hidup di kode Go untuk uji kesetaraan gerbang 1 (`D-54`); yang berubah
  // adalah ia berhenti menjadi isi layar.
  it('tidak lagi menggambar panel selisih terencana', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.queryByText(/Yang berbeda dari layar lama, dan itu disengaja/),
    ).not.toBeInTheDocument()
  })
})

describe('antrean yang belum dapat diisi', () => {
  it('tetap dapat dipilih dengan penanda, bukan disembunyikan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Penandanya masuk ke TEKS pilihannya: isi `<option>` hanya boleh berupa teks,
    // sehingga lencana di sampingnya bukan pilihan.
    expect(
      screen.getByRole('option', { name: /Komite Treaty ASM — belum tersedia/ }),
    ).toBeInTheDocument()
  })

  it('menjelaskan alasan dan pemilik penghalangnya, bukan tabel kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.selectOptions(queueSelect(), '3')

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
    await userEvent.selectOptions(queueSelect(), '3')
    await screen.findByText(/belum dapat dibaca sistem baru/)

    const after = calls.filter((c) => c.url.startsWith(PATH) && c.url !== TAB_PATH).length
    expect(after).toBe(before)
  })
})

describe('See All Claim', () => {
  it('tidak digambar sama sekali', async () => {
    // Report Definition yang memasok kedua antrean tidak menyaring menurut petugas,
    // sehingga checkbox itu tidak punya apa pun untuk dilepas. Kontrol yang tidak mengubah
    // apa pun lebih buruk daripada kontrol yang tidak ada.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.queryByLabelText('See All Claim')).not.toBeInTheDocument()

    stubDefaultFetch([ROW_TEKNIK], TAB_TEKNIK)
    await userEvent.selectOptions(queueSelect(), '2')

    expect(await screen.findByText(/antrean bersama/i)).toBeInTheDocument()
    expect(screen.queryByLabelText('See All Claim')).not.toBeInTheDocument()
  })

  it('tidak dikirim ke server', async () => {
    // Parameter yang tidak lagi berpengaruh tetap tidak dikirim: dua permintaan yang
    // hasilnya pasti sama tidak boleh punya URL berbeda, karena URL berbeda berarti cache
    // terpisah untuk hasil yang sama.
    stubDefaultFetch()
    await renderLoaded()

    await vi.waitFor(() => {
      expect(lastListCall()?.url ?? '').not.toContain('lihat_semua')
    })
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
    // Pesannya kini yang UMUM, bukan yang menyebut "milik Anda": antrean ini tidak lagi
    // disaring menurut petugas, sehingga kosong benar-benar berarti tidak ada pekerjaan —
    // bukan "pekerjaannya milik orang lain".
    stubDefaultFetch([])
    await renderLoaded()

    expect(await screen.findByText('Antrean ini sedang kosong.')).toBeInTheDocument()
    expect(screen.queryByText(/milik Anda/)).not.toBeInTheDocument()
  })
})

describe('nomor klaim sebagai tautan', () => {
  it('menunjuk layar Outstanding Claim', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Di Pega, mengklik Claim ID menjalankan Flow Action OutstandingClaim dan menggambar
    // Section OutstandingClaim. Tautannya menunjuk rute yang melayani layar itu.
    const link = await screen.findByRole('link', { name: 'CLMP-1001' })
    expect(link).toHaveAttribute('href', '/outstanding-claim/CLMP-1001')
  })

  it('menggantikan tombol "Lihat Detail Klaim" di kolom terakhir', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Tombol lama menunjuk penampung /view-claim yang memang belum punya isi. Dua jalan
    // dari satu baris ke dua layar berbeda, yang satu buntu, lebih buruk daripada satu.
    expect(
      screen.queryByRole('button', { name: 'Lihat Detail Klaim' }),
    ).not.toBeInTheDocument()
  })

  it('tidak menjadi tautan bila nomor klaimnya kosong', async () => {
    // Tautan beralamat kosong tetap dapat diklik dan membawa pengguna ke layar yang pasti
    // gagal. Yang digambar adalah tanda pisah, sama seperti sel kosong lain.
    stubDefaultFetch([{ ...ROW, claim_id: '' }])
    await renderLoaded()

    await screen.findByRole('table')
    expect(screen.queryAllByRole('link', { name: /CLMP/ })).toHaveLength(0)
  })
})

describe('dua kolom yang tidak bersumber dari export', () => {
  it('menggambar operator pengubah dan status objek kerja', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const row = (await screen.findByText('CLMP-1001')).closest('tr')
    expect(row).not.toBeNull()
    expect(within(row as HTMLElement).getByText('ADMINTREATY1')).toBeInTheDocument()
    expect(within(row as HTMLElement).getByText('New')).toBeInTheDocument()
  })

  it('menggambar tanda pisah bila objek kerjanya tidak terbaca', async () => {
    // Keduanya dibaca lewat LEFT JOIN ke tabel objek kerja, sehingga baris yang objek
    // kerjanya tidak terbaca memang mengirimkannya kosong — bukan sel yang tampak rusak.
    stubDefaultFetch([{ ...ROW, operator_pengubah: '', status_kerja: '' }])
    await renderLoaded()

    const row = (await screen.findByText('CLMP-1001')).closest('tr')
    expect(within(row as HTMLElement).getAllByText('—')).toHaveLength(2)
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
