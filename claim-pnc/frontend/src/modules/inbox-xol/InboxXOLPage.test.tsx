import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  Advice,
  ApprovalResponse,
  Breakdown,
  CauseOfLoss,
  ClaimSummaryResponse,
  MasterXOL,
} from './types'

const PATH = '/api/inbox-xol'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh PIC Teknik',
  jenis: 'KARYAWAN',
  login: 'picteknik01',
  email: 'contoh.pic@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Seluruh data uji KARANGAN — `D-69` melarang data nasabah ditulis di berkas yang
 * di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const MASTERS: MasterXOL[] = [
  {
    id: 'XOL-001',
    nama: 'XOL Property Treaty',
    tahun: '2024',
    kurs: 15500,
    min_limit: 2000,
    min_limit_idr: 31000000,
    tipe: 'PROPERTY',
    group_business: 'Fire, Marine Cargo',
    jumlah_group_business: 2,
    status_komite: '1',
    menunggu_komite: false,
    catatan_komite: '',
    pic: 'PICTEKNIK01',
    catatan_pic: '',
  },
  {
    id: 'XOL-003',
    nama: 'XOL Engineering Treaty',
    tahun: '2025',
    kurs: 16200,
    min_limit: 0,
    min_limit_idr: 0,
    tipe: 'ENGINEERING',
    group_business: '',
    jumlah_group_business: 0,
    status_komite: '0',
    menunggu_komite: true,
    catatan_komite: '',
    pic: 'PICTEKNIK01',
    catatan_pic: 'Menunggu persetujuan komite.',
  },
]

/**
 * Grid menggabungkan SELURUH perjanjian, jadi contohnya pun memuat baris dari dua
 * perjanjian berbeda — masing-masing dengan group business dan kursnya sendiri.
 */
const SUMMARY: ClaimSummaryResponse = {
  perjanjian: MASTERS[0]!,
  baris: [
    {
      id_master: 'XOL-001',
      tanggal_kejadian: '12/03/2024',
      sebab_kerugian: 'BANJIR',
      group_business: 'Fire, Marine Cargo',
      nilai_outstanding: 300000,
      nilai_akseptasi: 200000,
    },
    {
      id_master: 'XOL-009',
      tanggal_kejadian: '27/07/2018',
      sebab_kerugian: 'ACCIDENTAL DAMAGE - FIRE',
      group_business: 'Heavy Equipment',
      nilai_outstanding: 106535.05,
      nilai_akseptasi: 106535.05,
    },
  ],
}

const BREAKDOWN: Breakdown[] = [
  {
    group_business: 'Fire',
    kode_group_business: '10',
    jumlah_klaim: 12,
    nilai_outstanding: 200000,
    nilai_akseptasi: 140000,
    sumber: 'bisnis',
    kurs_tidak_tersedia: false,
  },
  {
    group_business: 'Treaty Inward',
    kode_group_business: '',
    jumlah_klaim: 2,
    nilai_outstanding: 0,
    nilai_akseptasi: 0,
    sumber: 'treaty',
    kurs_tidak_tersedia: true,
  },
]

const CAUSES: CauseOfLoss[] = [
  { id: '12001', deskripsi: 'BANJIR' },
  { id: '12003', deskripsi: 'KEBAKARAN' },
]

const ADVICES: Advice[] = [
  {
    nomor: 'PLA/XOL/2024/0002 / 1',
    nomor_asli: 'PLA/XOL/2024/0002',
    revisi: '1',
    nama_insurance: 'Reasuransi Contoh Kedua',
    nama_layer: 'Layer 2',
    tahun: '2024',
    sebab_kerugian: 'BANJIR',
    kurs: 15500,
    share_percent: 64.5,
    email: 'reas.kedua@contoh.invalid',
    remark: 'Sudah dikonfirmasi.',
    status_persetujuan: '1',
    sudah_disetujui: true,
    catatan_persetujuan: 'Disetujui komite.',
    catatan_pic: '',
    batas_layer: '25.000.000.000',
    negara: 'MALAYSIA',
    tanggal_terbit: '18/03/2024',
    tipe: 'PLA',
  },
]

const APPROVALS: ApprovalResponse = {
  pemberitahuan: [
    { tahun: '2024', sebab_kerugian: 'KEBAKARAN', tipe: 'DLA', tanggal_insert: '02/08/2024' },
  ],
  perjanjian: [MASTERS[1]!],
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

/** Peladen tiruan yang menjawab seluruh rute modul ini dengan data contoh. */
function stubDefaultFetch() {
  stubFetch((url) => {
    if (url === `${PATH}/perjanjian`) return jsonResponse(200, { perjanjian: MASTERS })
    if (url === `${PATH}/sebab-kerugian`) return jsonResponse(200, { sebab_kerugian: CAUSES })
    if (url === `${PATH}/persetujuan`) return jsonResponse(200, APPROVALS)
    if (url.startsWith(`${PATH}/klaim/summary`)) {
      return jsonResponse(200, {
        baris: [
          { kode_group_business: '10', group_business: 'Fire' },
          { kode_group_business: '', group_business: 'Treaty Inward' },
        ],
      })
    }
    if (url.startsWith(`${PATH}/klaim/rincian`)) return jsonResponse(200, { baris: BREAKDOWN })
    if (url.startsWith(`${PATH}/klaim`)) {
      return jsonResponse(200, SUMMARY)
    }
    if (url.startsWith(`${PATH}/pla-dla`)) {
      return jsonResponse(200, { pemberitahuan: ADVICES })
    }
    return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'rute tidak dikenal' })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-xol']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * band mengambil satu deret tombol berdasarkan labelnya.
 *
 * Diperlukan karena "Cari Data DLA PLA XOL" ADA DUA — satu di tiap deret, persis seperti
 * di `Section/InboxClaimXOL-Section.xml` (`:1940` dan `:3052`). Mencarinya di seluruh
 * layar akan menemukan keduanya dan gagal.
 */
function band(label: 'Inbox XOL' | 'Inbox XOL Komite'): HTMLElement {
  return screen.getByRole('region', { name: label })
}

/**
 * isi mengambil wadah isi — padanan container ber-`pyContainerVisibleWhen` di Pega.
 *
 * Perlu dilingkupi karena beberapa judul grid memakai teks yang sama dengan nama tombol
 * pembukanya, "Approval XOL" misalnya.
 */
function isi(): HTMLElement {
  return screen.getByRole('region', { name: 'Isi Inbox XOL' })
}

/** klik menekan satu tombol di dalam deret yang disebut. */
async function klik(label: 'Inbox XOL' | 'Inbox XOL Komite', tombol: string) {
  await userEvent.click(within(band(label)).getByRole('button', { name: tombol }))
}

/**
 * renderWithMasters menggambar layar, MEMBUKA wadah "Generated DLA PLA XOL", lalu menunggu
 * daftar perjanjian tiba.
 *
 * Tombolnya ditekan lebih dulu karena layar ini terbuka kosong — sama seperti di Pega,
 * tempat ketiga wadah isi bergantung pada `FlagDataForSerachingXOL.source` yang masih
 * kosong sampai satu tombol ditekan.
 *
 * Penantiannya pada salah satu pilihan dropdown, bukan pada labelnya: label "Perjanjian
 * XOL" sudah ada sejak panelnya dipasang, sementara isinya baru tiba bersama jawaban
 * server. Menunggu label saja membuat uji menekan dropdown yang masih kosong.
 */
async function renderWithMasters(): Promise<void> {
  renderPage()
  await klik('Inbox XOL', 'Generated DLA PLA XOL')
  await screen.findByRole('button', { name: 'INSERT DOL DAN COL' })
}

/**
 * bukaRincian mengeklik BARISNYA, bukan tautan di kolom tersendiri.
 *
 * Begitulah layar lama bekerja — barisnya tersorot lalu isinya terbentang di bawahnya —
 * dan `DataTable` meniru itu dengan memberi baris yang dapat dibuka `role="button"`.
 */
async function bukaRincian(tanggal: string) {
  await userEvent.click(screen.getByText(tanggal).closest('tr')!)
}

/**
 * rincianTerbuka menunggu isi baris terbentang.
 *
 * Penandanya kepala kolom "Number of Claim", bukan judul grid: grid rincian di layar lama
 * TIDAK punya judul, dan judul yang sempat kami tambahkan sudah dihapus.
 */
async function rincianTerbuka() {
  await screen.findAllByText('Number of Claim')
}

/**
 * renderCari menggambar layar lalu membuka wadah pencarian (`ParFlags = 2`).
 *
 * Panel PLA/DLA hanya ada di wadah ITU. Wadah `ParFlags = 1` memuat grid akumulasi klaim
 * beserta DATA XOL KLAIM dan DATA MASTER XOL, bukan formulir pencarian.
 */
async function renderCari(): Promise<void> {
  renderPage()
  await klik('Inbox XOL', 'Cari Data DLA PLA XOL')
  await screen.findByLabelText('Date Of Loss')
}

function callsTo(prefix: string): Call[] {
  return calls.filter((c) => c.url.startsWith(prefix))
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

describe('kerangka layar', () => {
  /**
   * Tampilan awal layar lama: DUA deret tombol, dan tidak satu pun grid.
   *
   * Ketiga wadah isi di `Section/InboxClaimXOL-Section.xml` bergantung pada
   * `FlagDataForSerachingXOL.source`, yang baru terisi saat salah satu tombol memanggil
   * `ToFlaggingDataXOLByRequest`. Sebelum itu tidak ada yang tampil.
   */
  it('membuka layar hanya dengan deret tombol, tanpa satu pun grid', async () => {
    stubDefaultFetch()
    renderPage()

    const utama = await screen.findByRole('region', { name: 'Inbox XOL' })
    expect(within(utama).getByRole('button', { name: 'Generated DLA PLA XOL' })).toBeInTheDocument()
    expect(within(utama).getByRole('button', { name: 'Cari Data DLA PLA XOL' })).toBeInTheDocument()

    const komite = band('Inbox XOL Komite')
    expect(within(komite).getByRole('button', { name: 'Approval XOL' })).toBeInTheDocument()
    expect(within(komite).getByRole('button', { name: 'Cari Data DLA PLA XOL' })).toBeInTheDocument()

    expect(screen.queryByRole('button', { name: 'INSERT DOL DAN COL' })).not.toBeInTheDocument()
    expect(screen.queryByText('DATA XOL BASED ON DOL AND COL')).not.toBeInTheDocument()
    expect(screen.getByText('Pilih salah satu tombol di atas untuk menampilkan datanya.'))
      .toBeInTheDocument()
  })

  /**
   * Janji tampilan awal itu ditepati sampai ke jaringan: layar yang belum menampilkan apa
   * pun tidak boleh meminta apa pun. Memuat ketiga wadah di muka menggandakan beban setiap
   * kali layar dibuka, dan dua pertiganya tidak pernah dilihat.
   */
  it('tidak memanggil satu pun rute sebelum sebuah tombol ditekan', async () => {
    stubDefaultFetch()
    renderPage()

    await screen.findByRole('region', { name: 'Inbox XOL' })

    expect(calls.filter((c) => c.url.startsWith(PATH))).toHaveLength(0)
  })

  /**
   * "Generated DLA PLA XOL" dan "Approval XOL" sama-sama mengirim `ParFlags = 1`, dan
   * kedua wadah `source=='1'` memperbolehkan `GCNMFW:Administrators`. Akun selengkap itu
   * melihat keempat blok dari tombol mana pun — dan hari ini setiap pengguna begitu,
   * karena peran belum dapat ditegakkan (`TKT-F3-004`).
   */
  it.each([
    ['Inbox XOL', 'Generated DLA PLA XOL'],
    ['Inbox XOL Komite', 'Approval XOL'],
  ] as const)('membuka keempat blok lewat tombol %s › %s', async (deret, tombol) => {
    stubDefaultFetch()
    renderPage()
    await screen.findByRole('region', { name: 'Inbox XOL Komite' })

    expect(callsTo(`${PATH}/persetujuan`)).toHaveLength(0)

    await klik(deret, tombol)

    expect(await screen.findByText('DATA MASTER XOL')).toBeInTheDocument()
    // Judul grid pertama adalah "DATA XOL KLAIM" (`:23598`); "Approval XOL" hanyalah nama
    // tombol pembukanya (`pyButtonLabel`, `:30913`), bukan judul apa pun di dalam layar.
    expect(within(isi()).getByText('DATA XOL KLAIM')).toBeInTheDocument()
    expect(within(isi()).getByText('DATA XOL BASED ON DOL AND COL')).toBeInTheDocument()
    expect(within(isi()).getByRole('button', { name: 'INSERT DOL DAN COL' })).toBeInTheDocument()
    expect(callsTo(`${PATH}/persetujuan`)).toHaveLength(1)
  })

  /**
   * Kedua tombol "Cari Data DLA PLA XOL" mengirim `ParFlags = 2` dan membuka wadah yang
   * SAMA (`:21730`). Ia ada dua kali karena tiap peran hanya melihat satu deret.
   */
  it('membuka panel pencarian dari deret mana pun', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByRole('region', { name: 'Inbox XOL' })

    await klik('Inbox XOL Komite', 'Cari Data DLA PLA XOL')
    expect(await screen.findByLabelText('Date Of Loss')).toBeInTheDocument()
    // Wadah pencarian berdiri sendiri — grid klaim milik wadah "Generated" tidak ikut.
    expect(screen.queryByRole('button', { name: 'INSERT DOL DAN COL' })).not.toBeInTheDocument()

    await klik('Inbox XOL', 'Cari Data DLA PLA XOL')
    expect(await screen.findByLabelText('Date Of Loss')).toBeInTheDocument()
  })

  /** Satu penanda, satu wadah: membuka yang lain menutup yang sedang tampil. */
  it('menampilkan satu wadah saja pada satu waktu', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    expect(screen.getByRole('button', { name: 'INSERT DOL DAN COL' })).toBeInTheDocument()

    await klik('Inbox XOL', 'Cari Data DLA PLA XOL')

    expect(await screen.findByLabelText('Date Of Loss')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'INSERT DOL DAN COL' })).not.toBeInTheDocument()
    expect(screen.queryByText('DATA XOL KLAIM')).not.toBeInTheDocument()
  })

  it('menolak membuka layar sebelum portal dipilih', async () => {
    stubDefaultFetch()
    useSelectedPortal.getState().clear()
    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  })
})

describe('wadah Generated DLA PLA XOL', () => {
  /**
   * Grid terisi SENDIRI saat wadah dibuka, tanpa pengguna memilih apa pun — dan isinya
   * gabungan SELURUH perjanjian.
   *
   * `Activity/GetClaimXOL-Act.xml` step 4 me-loop `MstXOL.pxResults` tanpa batas, lalu
   * meng-APPEND hasil tiap perjanjian ke satu daftar. Itu sebabnya layar lama tidak punya
   * pemilih perjanjian, dan gridnya sudah terisi begitu dibuka.
   */
  it('memuat akumulasi seluruh perjanjian tanpa menyebut satu pun', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    expect(await screen.findByText('12/03/2024')).toBeInTheDocument()
    expect(screen.getByText('27/07/2018')).toBeInTheDocument()

    const call = callsTo(`${PATH}/klaim`)[0]!
    expect(call.url).not.toContain('id_master')
  })

  it('menampilkan nilai tiap baris dengan pemisah ribuan Indonesia', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    // Dilingkupi ke BARISNYA, bukan ke seluruh layar: "BANJIR" juga muncul sebagai
    // pilihan pada dropdown panel PLA/DLA, dan pencarian tanpa lingkup menemukan keduanya.
    const row = (await screen.findByText('12/03/2024')).closest('tr')!

    expect(within(row).getByText('BANJIR')).toBeInTheDocument()
    // 300000 digambar dengan pemisah ribuan Indonesia, BUKAN sebagai rupiah: satuannya
    // mata uang perjanjian, dan menempelkan "Rp" akan menyatakan hal yang salah.
    expect(within(row).getByText('300.000')).toBeInTheDocument()
  })

  /**
   * Tiap baris membawa perjanjian asalnya, dan rincian memakai perjanjian BARIS ITU —
   * bukan satu perjanjian yang berlaku untuk seluruh grid.
   */
  it('membuka rincian memakai perjanjian asal barisnya', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('27/07/2018')

    await bukaRincian('27/07/2018')

    await rincianTerbuka()
    expect(callsTo(`${PATH}/klaim/rincian`)[0]!.url).toContain('id_master=XOL-009')
  })

  it('memuat rincian hanya saat sebuah baris dibuka', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')

    expect(callsTo(`${PATH}/klaim/rincian`)).toHaveLength(0)

    await bukaRincian('12/03/2024')

    await rincianTerbuka()
    expect(callsTo(`${PATH}/klaim/rincian`)).toHaveLength(1)
  })

  /**
   * Inilah pengganti `RETURN 1` pada fungsi kurs sistem lama.
   *
   * Yang diuji bukan angkanya melainkan KETIADAANNYA: baris yang kursnya tidak ditemukan
   * tidak boleh menampilkan angka yang tampak benar.
   */
  it('menolak menampilkan nilai baris treaty inward yang kursnya tidak ada', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')
    await bukaRincian('12/03/2024')

    await rincianTerbuka()

    // Barisnya ADA — yang tidak ada adalah angkanya.
    // Dilingkupi ke grid RINCIAN: "Treaty Inward" kini muncul juga di grid
    // "Summary Data XOL" di bawahnya.
    const rincian = (await screen.findAllByText('Number of Claim'))[0]!.closest('table')!
    const treatyRow = within(rincian).getByText('Treaty Inward').closest('tr')!
    expect(within(treatyRow).getByText('2')).toBeInTheDocument()
    expect(within(treatyRow).queryByText('0')).not.toBeInTheDocument()

    expect(
      await screen.findByText(/tidak dapat dihitung karena kurs mata uangnya/),
    ).toBeInTheDocument()
  })

  /**
   * Separuh bawah layar rincian: grid "Master tahun XOL".
   *
   * Sectionnya tidak ada di export (`R-16`), tetapi sumber datanya ada dan cocok kolom
   * per kolom dengan `RDB List/GetDataMasterXOL-SQL.xml:11` — termasuk `min(limit)` dan
   * `min(limit)*KURSVALUE` yang menjadi "Min Limit" dan "Min Limit IDR".
   */
  it('menggambar grid Master tahun XOL beserta Min Limit di layar rincian', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')
    await bukaRincian('12/03/2024')

    expect(await screen.findByLabelText('Master tahun XOL')).toBeInTheDocument()

    // Kepala kolom terlihat SEJAK gridnya masih kosong — seperti layar lama, yang
    // menampilkan kepala di atas tulisan "Data Tidak Ada".
    //
    // getAllByText: DataTable menggambar judul kolom dua kali — tabel untuk layar lebar
    // dan kartu untuk layar sempit.
    expect(screen.getAllByText('Min Limit').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Min Limit IDR').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Nama Master').length).toBeGreaterThan(0)

    // Grid mulai kosong; isinya baru muncul setelah Show All Data ditekan.
    expect(screen.queryByText('XOL-001')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Show All Data' }))

    // 2000 × 15.500 = 31.000.000 — dihitung backend, digambar dengan pemisah Indonesia.
    expect((await screen.findAllByText('31.000.000')).length).toBeGreaterThan(0)
  })

  /** Tahun menyaring, Show All Data menampilkan semua, Remove All Data mengosongkan. */
  it('menyaring grid Master tahun XOL, lalu mengosongkannya kembali', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')
    await bukaRincian('12/03/2024')

    const picker = await screen.findByLabelText('Master tahun XOL')
    await userEvent.selectOptions(picker, '2025')

    expect((await screen.findAllByText('XOL-003')).length).toBeGreaterThan(0)
    expect(screen.queryByText('XOL-001')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Show All Data' }))
    expect((await screen.findAllByText('XOL-001')).length).toBeGreaterThan(0)

    /*
      "Remove All Data" MENGHAPUS — `DeleteDataInXOLSummarybasedondol-SQL.xml` membuang
      baris dari `POOLDATA.XOL_TABLE_ALL_KLAIM`. Jadi ia mengirim permintaan ke server,
      bukan mengosongkan tampilan; penolakannya datang dari sana beserta sebabnya.
    */
    await userEvent.click(screen.getByRole('button', { name: 'Remove All Data' }))

    await waitFor(() => expect(callsTo(`${PATH}/dol-col`)).toHaveLength(1))
    expect(callsTo(`${PATH}/dol-col`)[0]!.init?.method).toBe('POST')
    // Gridnya TIDAK dikosongkan sendiri oleh layar.
    expect((await screen.findAllByText('XOL-001')).length).toBeGreaterThan(0)
  })

  /**
   * Tombol "INSERT DOL DAN COL" DIGAMBAR, dan modalnya memuat keempat isi modal lama:
   * Date Of Loss, Cause Of Loss, grid PILIH MASTER XOL, dan Simpan.
   *
   * Menyembunyikannya — seperti versi sebelumnya, yang hanya menyisakan keterangan —
   * membuat pengguna melaporkan fitur yang hilang. Itu pula alasan rute
   * `POST /inbox-xol/dol-col` ada di backend meski ia selalu menolak.
   */
  it('menggambar tombol INSERT DOL DAN COL beserta isi modalnya', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    await userEvent.click(screen.getByRole('button', { name: 'INSERT DOL DAN COL' }))

    const modal = await screen.findByRole('dialog', { name: 'Insert DOL dan COL' })
    expect(within(modal).getByLabelText('Date Of Loss')).toBeInTheDocument()
    expect(within(modal).getByLabelText('Cause Of Loss')).toBeInTheDocument()
    expect(within(modal).getByText('PILIH MASTER XOL')).toBeInTheDocument()
    expect(within(modal).getByRole('button', { name: 'Simpan' })).toBeInTheDocument()
  })

  /**
   * "Pilih" menentukan perjanjian mana yang akan DITULISI, bukan mana yang ditampilkan.
   *
   * Grid di belakang modal tidak boleh berubah karenanya — ia gabungan seluruh
   * perjanjian, dan layar lama tidak punya penyaring perjanjian sama sekali. Versi
   * sebelumnya memperlakukannya sebagai penyaring, dan itulah sebab grid tampil kosong.
   */
  it('tidak menyaring grid saat sebuah perjanjian dipilih di modal', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')

    const sebelum = callsTo(`${PATH}/klaim`).length

    await userEvent.click(screen.getByRole('button', { name: 'INSERT DOL DAN COL' }))
    const modal = await screen.findByRole('dialog', { name: 'Insert DOL dan COL' })
    await userEvent.click(
      within(modal).getByRole('button', { name: 'Pilih perjanjian 2025' }),
    )

    // Modal TETAP terbuka — Date Of Loss dan Cause Of Loss masih harus diisi.
    expect(screen.getByRole('dialog', { name: 'Insert DOL dan COL' })).toBeInTheDocument()
    expect(callsTo(`${PATH}/klaim`)).toHaveLength(sebelum)
  })

  /**
   * Grid "Summary Data XOL" — `RDB List/GetBusinessnameXOLForSummerry-SQL.xml`.
   *
   * Isinya group business yang menanggung klaim pada tanggal kejadian dan penyebab
   * kerugian itu, termasuk satu baris "Treaty Inward" dari `T_CLAIM_INWARD_XOL`.
   */
  it('menggambar Summary Data XOL beserta baris treaty inward', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')
    await bukaRincian('12/03/2024')

    expect(await screen.findByText('Summary Data XOL')).toBeInTheDocument()
    expect(screen.getAllByText('Business Name').length).toBeGreaterThan(0)
    expect((await screen.findAllByText('Treaty Inward')).length).toBeGreaterThan(0)

    const call = callsTo(`${PATH}/klaim/summary`)[0]!
    expect(call.url).toContain('tanggal_kejadian=12%2F03%2F2024')
    expect(call.url).toContain('sebab_kerugian=BANJIR')
  })

  /**
   * Grid "No Klaim" digambar dengan kolom yang benar tetapi KOSONG: activity pengisinya
   * (`ShowDataKlaimXOLKlaimBeforeGenerated`) tidak ada di export Pega.
   */
  it('menggambar kolom grid No Klaim dan menyatakan isinya belum tersedia', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')
    await bukaRincian('12/03/2024')

    expect((await screen.findAllByText('No Klaim')).length).toBeGreaterThan(0)
    expect(screen.getAllByText('Os Value').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Accept Value').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Currency').length).toBeGreaterThan(0)
    expect(
      screen.getByText(/ShowDataKlaimXOLKlaimBeforeGenerated/),
    ).toBeInTheDocument()
  })
})

describe('panel PLA/DLA', () => {
  it('tidak mencari sebelum tombol ditekan', async () => {
    stubDefaultFetch()
    await renderCari()

    expect(callsTo(`${PATH}/pla-dla`)).toHaveLength(0)
  })

  it('mencari dan menampilkan nomor beserta revisinya', async () => {
    stubDefaultFetch()
    await renderCari()

    await userEvent.type(screen.getByLabelText('Date Of Loss'), '2024')
    await userEvent.selectOptions(screen.getByLabelText('Cause Of Loss'), 'BANJIR')
    await userEvent.click(screen.getByRole('checkbox', { name: 'PLA' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data' }))

    expect(await screen.findByText('PLA/XOL/2024/0002 / 1')).toBeInTheDocument()
    // Tanda persen ditambahkan di layar, bukan dirangkai di SQL.
    expect(screen.getByText('64,5 %')).toBeInTheDocument()
  })

  /**
   * Penyebab kerugian dikirim sebagai DESKRIPSI, bukan ID.
   *
   * Itulah yang tersimpan di kolom CAUSEOFLOSS pada tabel PLA/DLA; mengirim ID akan
   * menghasilkan daftar kosong tanpa satu pun pesan galat.
   */
  it('mengirim deskripsi penyebab kerugian, bukan kodenya', async () => {
    stubDefaultFetch()
    await renderCari()

    await userEvent.type(screen.getByLabelText('Date Of Loss'), '2024')
    await userEvent.selectOptions(screen.getByLabelText('Cause Of Loss'), 'BANJIR')
    await userEvent.click(screen.getByRole('checkbox', { name: 'DLA' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data' }))

    await screen.findByText(/Pemberitahuan DLA/)

    const call = callsTo(`${PATH}/pla-dla`)[0]!
    expect(call.url).toContain('sebab_kerugian=BANJIR')
    expect(call.url).not.toContain('sebab_kerugian=12001')
    expect(call.url).toContain('tipe=DLA')
  })

  it('menandai pelanggaran validasi pada isiannya masing-masing', async () => {
    stubFetch((url) => {
      if (url === `${PATH}/perjanjian`) return jsonResponse(200, { perjanjian: MASTERS })
      if (url === `${PATH}/sebab-kerugian`) {
        return jsonResponse(200, { sebab_kerugian: CAUSES })
      }
      if (url.startsWith(`${PATH}/pla-dla`)) {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [{ field: 'tahun', pesan: 'Tahun XOL wajib diisi.' }],
        })
      }
      return jsonResponse(200, {})
    })
    await renderCari()

    await userEvent.selectOptions(screen.getByLabelText('Cause Of Loss'), 'BANJIR')
    await userEvent.click(screen.getByRole('checkbox', { name: 'PLA' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data' }))

    expect(await screen.findByText('Tahun XOL wajib diisi.')).toBeInTheDocument()
  })

  /**
   * Tombol unduh TIDAK berbunyi "Print Perhitungan".
   *
   * Yang diunduh adalah isi perhitungannya, bukan dokumen PLA/DLA resmi — dan dokumen
   * itu dikirim kepada reasuradur. Berkas yang tampak resmi padahal bukan adalah
   * kekeliruan yang berakibat ke luar perusahaan.
   */
  it('menawarkan unduhan hanya setelah ada hasil, dan tidak menyebutnya dokumen resmi', async () => {
    stubDefaultFetch()
    await renderCari()

    expect(screen.queryByRole('link', { name: /Unduh perhitungan/ })).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Date Of Loss'), '2024')
    await userEvent.selectOptions(screen.getByLabelText('Cause Of Loss'), 'BANJIR')
    await userEvent.click(screen.getByRole('checkbox', { name: 'PLA' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data' }))

    const link = await screen.findByRole('link', { name: 'Unduh perhitungan (CSV)' })
    expect(link).toHaveAttribute('href', expect.stringContaining(`${PATH}/pla-dla/unduh`))
    expect(screen.queryByText('Print Perhitungan')).not.toBeInTheDocument()
  })
})

describe('DATA XOL KLAIM dan DATA MASTER XOL', () => {
  it('menampilkan kedua antrean dan menyatakan persetujuan belum tersedia', async () => {
    stubDefaultFetch()
    renderPage()

    await klik('Inbox XOL Komite', 'Approval XOL')

    const master = (await screen.findByText('DATA MASTER XOL')).closest('section')!
    expect(within(isi()).getByText('DATA XOL KLAIM')).toBeInTheDocument()
    expect(screen.getByText('KEBAKARAN')).toBeInTheDocument()
    // Dilingkupi ke grid DATA MASTER XOL: "XOL-003" juga muncul sebagai ID Master pada
    // kepala DETAIL MASTER XOL, karena perjanjian itulah yang terpilih sendiri.
    expect(within(master).getByText('XOL-003')).toBeInTheDocument()
    expect(
      screen.getByText(/Menyetujui atau menolak belum tersedia/),
    ).toBeInTheDocument()
  })

  /**
   * Kolom "Date Of Loss" pada grid Approval berisi TAHUN, bukan tanggal — kuerinya
   * `select tahun as "City"`. Judulnya dipertahankan (`D-13`), dan keterangannya
   * dinyatakan supaya pengguna tidak salah membacanya.
   */
  it('menjelaskan bahwa kolom Date Of Loss berisi tahun perjanjian', async () => {
    stubDefaultFetch()
    renderPage()

    await klik('Inbox XOL Komite', 'Approval XOL')

    expect(
      await screen.findByText(/berisi tahun perjanjian, bukan tanggal kejadian/),
    ).toBeInTheDocument()
  })
})

/**
 * Ketiga kotak centang menyatu menjadi satu tipe lewat URUTAN PRIORITAS, bukan lewat
 * pilihan tunggal — `Activity/BrowseDataXOLPLADLAGenerated-Act.xml:976`:
 *
 *	Local.tipe = @if(Province=="true","PLA", @if(ProvinceID=="true","DLA",""))
 */
describe('kotak centang PLA · DLA · DATA', () => {
  async function cari() {
    await userEvent.type(screen.getByLabelText('Date Of Loss'), '2024')
    await userEvent.selectOptions(screen.getByLabelText('Cause Of Loss'), 'BANJIR')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data' }))
  }

  it('memenangkan PLA ketika PLA dan DLA sama-sama dicentang', async () => {
    stubDefaultFetch()
    await renderCari()

    await userEvent.click(screen.getByRole('checkbox', { name: 'DLA' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'PLA' }))
    await cari()

    await screen.findByText(/Pemberitahuan PLA/)
    expect(callsTo(`${PATH}/pla-dla`)[0]!.url).toContain('tipe=PLA')
  })

  /**
   * Mencentang DATA saja menghasilkan `Local.tipe == ""`, dan di aktivitas lama cabang itu
   * tidak menjalankan satu pun kueri PLA/DLA. Permintaannya ditahan di layar — bukan
   * dikirim untuk ditolak server dengan galat validasi yang menyalahkan pengguna.
   */
  it('menahan pencarian dan menjelaskan sebabnya ketika hanya DATA yang dicentang', async () => {
    stubDefaultFetch()
    await renderCari()

    await userEvent.click(screen.getByRole('checkbox', { name: 'DATA' }))
    await cari()

    expect(await screen.findByText(/tidak menjalankan pencarian/)).toBeInTheDocument()
    expect(callsTo(`${PATH}/pla-dla`)).toHaveLength(0)
  })
})

describe('pemisahan entitas', () => {
  /**
   * `R-20`: perjanjian XOL dan nilai klaimnya milik satu badan hukum. Setiap permintaan
   * WAJIB menyebut portalnya — backend memang menolak yang tidak menyebutkannya, tetapi
   * penolakan itu hanya terjadi bila layar benar-benar mengirimkannya.
   */
  it('menyertakan header portal pada setiap permintaan', async () => {
    stubDefaultFetch()
    await renderWithMasters()
    await screen.findByText('12/03/2024')
    await bukaRincian('12/03/2024')
    await rincianTerbuka()

    const moduleCalls = calls.filter((c) => c.url.startsWith(PATH))
    expect(moduleCalls.length).toBeGreaterThan(0)

    for (const call of moduleCalls) {
      const headers = new Headers(call.init?.headers)
      expect(headers.get('X-Portal')).toBe('ASM')
    }
  })
})
