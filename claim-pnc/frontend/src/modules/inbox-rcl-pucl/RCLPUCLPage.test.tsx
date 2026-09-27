import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-rcl-pucl'
const TAB_PATH = `${PATH}/tab`
const EXPORT_PATH = `${PATH}/ekspor`
const CLAIM_PATH = `${PATH}/klaim/`

const SAMPLE_PROFILE = {
  identitas: '90000004',
  nama: 'Contoh Petugas RCL',
  jenis: 'KARYAWAN',
  login: 'petugasrclcontoh',
  email: 'contoh.rcl@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kolom grid — SEMBILAN, dan identik di ketiga tab kecuali judul kolom keenam.
 *
 * Kesamaan itu bukan kebetulan: ketiga section menggambar sel yang sama persis, dan yang
 * membedakan tab hanyalah penyaringnya. Justru karena itu layar ini menuntut keterangan
 * per tab — tidak ada petunjuk visual apa pun yang membedakan ketiganya.
 */
function kolom(judulJalur: string): Tab['kolom'] {
  return [
    { kunci: 'no_case', judul: 'Nomor Case' },
    { kunci: 'no_polis', judul: 'No Polis' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
    { kunci: 'tanggal_masuk_inbox', judul: 'Tanggal Masuk Inbox' },
    { kunci: 'deskripsi_analyst', judul: 'Deskripsi Analyst' },
    { kunci: 'status_rcl_pucl', judul: judulJalur },
    { kunci: 'tanggal_cetak_surat', judul: 'Tanggal Cetak Surat' },
    { kunci: 'lama_klaim', judul: 'Lama Klaim' },
    { kunci: 'status_kadaluarsa', judul: 'Status Kadaluarsa' },
  ] as Tab['kolom']
}

const TAB_CETAK: Tab = {
  kode: '1',
  nama: 'Cetak Surat',
  keterangan: 'Klaim RCL/PUCL yang suratnya BELUM dicetak.',
  kolom: kolom('Status RCL/PUCL'),
  punya_laporan_rentang_tanggal: true,
  catatan: 'Kolom "Tanggal Cetak Surat" selalu kosong di tab ini.',
  terhalang: false,
}

const TAB_LENGKAP: Tab = {
  kode: '2',
  nama: 'Kelengkapan Dokumen',
  keterangan: 'Klaim yang suratnya SUDAH dicetak tetapi belum disetujui.',
  kolom: kolom('Status RCL/PUCL'),
  punya_laporan_rentang_tanggal: false,
  terhalang: false,
}

const TAB_MSIG: Tab = {
  kode: '3',
  nama: 'Klaim MSIG',
  keterangan: 'Klaim RCL/PUCL jalur MSIG.',
  kolom: kolom('Status'),
  punya_laporan_rentang_tanggal: false,
  catatan: 'Tab ini kemungkinan besar kosong. Menunggu pemastian DBA.',
  terhalang: false,
}

const METADATA: MetadataResponse = {
  tab: [TAB_CETAK, TAB_LENGKAP, TAB_MSIG],
  tab_bawaan: '1',
  kolom_laporan: [
    { kunci: 'no_case', judul: 'Nomor Case' },
    { kunci: 'status_klaim', judul: 'Status Klaim' },
  ],
  selisih_terencana: [
    'Tab "Klaim MSIG" kemungkinan besar kosong, dan itu bukan kerusakan.',
    'Isian tanggal di tab "Cetak Surat" TIDAK menyaring tabel di bawahnya.',
  ],
  portal: 'ASM',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * `tanggal_cetak_surat` sengaja KOSONG: begitulah yang selalu dikirim server pada tab
 * "Cetak Surat", karena penyaring tab itu justru "surat belum dicetak". Uji di bawah
 * memastikan layar menggambarnya sebagai tanda pisah, bukan membiarkan selnya kosong.
 */
const BARIS: WorkItem = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-700001',
  no_case: 'PNC-700001',
  no_polis: 'CONTOH-RCL-0001',
  nama_tertanggung: 'Tertanggung Contoh Satu',
  tanggal_masuk_inbox: '2026-09-10 09:30:00',
  deskripsi_analyst: 'Dokumen pendukung tidak lengkap.',
  status_rcl_pucl: 'RCL',
  tanggal_cetak_surat: '',
  lama_klaim: '12',
  status_kadaluarsa: 'Belum Kadaluarsa',
}

type Call = { url: string; init?: RequestInit | undefined }

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

/**
 * Jawaban layar kerja satu klaim.
 *
 * `up` sengaja SAMA dengan `nama_peserta`: begitulah Pega mengisinya — kedua penetapan di
 * `SetDataLampiranSuratRCLPUCL_Act` menunjuk ekspresi yang sama persis. Uji di bawah
 * memastikan layar menjelaskannya alih-alih membiarkannya terbaca sebagai kerusakan.
 */
const DETAIL = {
  referensi: BARIS.referensi,
  no_case: BARIS.no_case,
  lampiran_surat: {
    rcl_pucl: 'RCL',
    kode_rcl_pucl: '1',
    deskripsi_analyst: 'Dokumen pendukung tidak lengkap.',
    no_polis: 'CONTOH-RCL-0001',
    tanggal_kejadian: '2026-09-01',
    nama_peserta: 'Objek Contoh Satu',
    up: '250000000',
    jumlah_tagihan: '15000000',
    tanggal_cetak_surat: '',
    tanggal_kirim_rcl_pucl: '2026-09-10 09:30:00',
  },
  penerimaan_dokumen: { komentar_pucl: 'Menunggu kelengkapan dari cabang.' },
  isian_belum_terpetakan: ['NIK', 'Perihal', 'Email LOD'],
  tindakan_masih_di_pega: true,
  portal: 'ASM',
}

/** Peladen tiruan yang menjawab bentuk layar, satu baris antrean, dan layar kerjanya. */
function stubDefaultFetch(rows: WorkItem[] = [BARIS]) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL)
    if (url.startsWith(EXPORT_PATH)) {
      return new Response('Nomor Case\nPNC-700001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="rcl-pucl.csv"',
        },
      })
    }

    // Tab yang diminta menentukan bentuk jawabannya. Menjawab tab yang sama untuk setiap
    // permintaan akan membuat uji perpindahan tab lulus tanpa membuktikan apa pun — dan di
    // layar ini bahayanya lebih besar, karena ketiga tab menggambar kolom yang identik.
    const wanted = new URL(url, 'http://uji.invalid').searchParams.get('tab')
    const answered =
      wanted === TAB_MSIG.kode
        ? TAB_MSIG
        : wanted === TAB_LENGKAP.kode
          ? TAB_LENGKAP
          : TAB_CETAK

    const baris = answered.kode === TAB_MSIG.kode ? [] : rows

    return jsonResponse(200, {
      tab: answered,
      baris,
      paginasi: { halaman: 1, ukuran: 50, total: baris.length, total_halaman: 1 },
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
      <MemoryRouter initialEntries={['/inbox-rcl-pucl']}>
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
  await screen.findByRole('tab', { name: /Cetak Surat/ })
}

function lastListCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url === PATH || c.url.startsWith(`${PATH}?`))
}

function lastExportCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
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
  it('menggambar ketiga tab', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Cetak Surat/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Kelengkapan Dokumen/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Klaim MSIG/ })).toBeInTheDocument()
  })

  it('membuka tab bawaan yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Cetak Surat/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar kolom dari server, bukan dari daftar yang ditulis di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const judul of [
      'Nomor Case',
      'No Polis',
      'Nama Tertanggung',
      'Tanggal Masuk Inbox',
      'Deskripsi Analyst',
      'Tanggal Cetak Surat',
      'Lama Klaim',
      'Status Kadaluarsa',
    ]) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  it('menyatakan layar ini antrean bersama, bukan pekerjaan pemanggil', async () => {
    // Tanpa keterangan ini, petugas yang terbiasa dengan inbox lain akan mengira daftarnya
    // keliru karena memuat pekerjaan orang lain.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/antrean bersama/i)).toBeInTheDocument()
  })

  it('menampilkan selisih terencana dari server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByText(/Tab "Klaim MSIG" kemungkinan besar kosong/),
    ).toBeInTheDocument()
  })

  it('menyatakan kedua tindakan yang masih dikerjakan lewat Pega', async () => {
    // Keduanya pekerjaan NYATA pengguna layar ini setiap hari. Layar yang kehilangan
    // tombolnya tanpa penjelasan akan dilaporkan sebagai kerusakan.
    stubDefaultFetch()
    await renderLoaded()

    const notice = screen.getByText(/Yang masih dikerjakan lewat Pega/)
    expect(notice).toBeInTheDocument()
    expect(screen.getByText(/Reminder PUCL/)).toBeInTheDocument()
  })
})

describe('catatan per tab', () => {
  it('menampilkan catatan tab Cetak Surat tentang kolom yang selalu kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByText(/"Tanggal Cetak Surat" selalu kosong di tab ini/),
    ).toBeInTheDocument()
  })

  it('menampilkan catatan tab Klaim MSIG tentang kemungkinan kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Klaim MSIG/ }))

    expect(
      await screen.findByText(/kemungkinan besar kosong. Menunggu pemastian DBA/),
    ).toBeInTheDocument()
  })
})

describe('rentang tanggal', () => {
  it('hanya muncul di tab yang punya laporan rentang tanggal', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByLabelText('FROM RCL/PUCL')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: /Kelengkapan Dokumen/ }))
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/)

    expect(screen.queryByLabelText('FROM RCL/PUCL')).not.toBeInTheDocument()
  })

  it('menyatakan bahwa tanggalnya TIDAK menyaring tabel', async () => {
    // Ini peringatan yang mencegah laporan kerusakan palsu: pengguna mengisi tanggal,
    // tabel tidak berubah, dan tanpa kalimat ini ia akan mengira layarnya rusak.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByText(/Kedua tanggal ini hanya dipakai tombol unduh/),
    ).toBeInTheDocument()
  })

  it('tidak ikut dikirim pada permintaan daftar', async () => {
    // Grid memang tidak tersaring tanggal. Mengirimkannya akan membuat orang mengira
    // sebaliknya saat menelusuri lalu lintas jaringan.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('FROM RCL/PUCL'), '2026-09-01')

    const call = lastListCall()
    expect(call?.url ?? '').not.toContain('dari=')
  })
})

describe('perpindahan tab', () => {
  it('meminta tab yang dipilih ke server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Kelengkapan Dokumen/ }))
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/)

    expect(lastListCall()?.url).toContain('tab=2')
  })

  it('mengganti judul kolom jalur pada tab Klaim MSIG', async () => {
    // Satu-satunya perbedaan visual antartab. Ia dibawa apa adanya dari section
    // (`D-13`), dan datang dari server — bukan ditebak layar.
    //
    // Stub-nya DITULIS SENDIRI di sini, bukan memakai stubDefaultFetch: di sana tab MSIG
    // sengaja dijawab kosong untuk menguji pesan antrean kosong, dan tabel kosong tidak
    // menggambar satu pun kepala kolom. Uji judul kolom karena itu menuntut tabnya berisi.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)

      const wanted = new URL(url, 'http://uji.invalid').searchParams.get('tab')
      const answered = wanted === TAB_MSIG.kode ? TAB_MSIG : TAB_CETAK

      return jsonResponse(200, {
        tab: answered,
        baris: [BARIS],
        paginasi: { halaman: 1, ukuran: 50, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    })
    await renderLoaded()

    expect(
      await screen.findByRole('columnheader', { name: 'Status RCL/PUCL' }),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: /Klaim MSIG/ }))

    expect(await screen.findByRole('columnheader', { name: 'Status' })).toBeInTheDocument()
    expect(
      screen.queryByRole('columnheader', { name: 'Status RCL/PUCL' }),
    ).not.toBeInTheDocument()
  })
})

describe('isi tabel', () => {
  it('menggambar baris dari server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('PNC-700001')).toBeInTheDocument()
    expect(screen.getByText('Tertanggung Contoh Satu')).toBeInTheDocument()
  })

  it('menggambar sel kosong sebagai tanda pisah, bukan dibiarkan hampa', async () => {
    // "Tanggal Cetak Surat" memang SELALU kosong di tab ini. Tanda pisah menyatakan
    // "tidak ada isinya" alih-alih "gagal dimuat".
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByText('PNC-700001')
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)
  })

  it('menggambar Nomor Case sebagai TAUTAN, bukan teks biasa', async () => {
    // Begitulah bentuknya di ketiga section RCL/PUCL: sel "Nomor Case" adalah
    // `pyUIElement = link` ber-`pyLabel = .pyID` (`D-13`).
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByRole('button', { name: /PNC-700001/ })).toBeInTheDocument()
  })

  it('TIDAK menggambar kolom tombol "Lihat Detail"', async () => {
    // Versi pertama modul ini menambahkan kolom tombol yang TIDAK ADA di Pega sama sekali.
    // Uji ini menjaga agar ia tidak kembali: kolom yang tidak pernah ada membuat pengguna
    // mengira ada dua cara berbeda membuka baris, dan menggeser lebar kolom lain.
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByText('Tertanggung Contoh Satu')
    expect(screen.queryByRole('button', { name: /Lihat Detail/ })).not.toBeInTheDocument()
  })

  it('membuka LAYAR KERJA klaim, bukan panel di halaman antrean', async () => {
    // Di Pega, tautannya menjalankan `SetAssignmentInboxPUCL_act` dengan satu parameter:
    // `inskey = .pzInsKey`, dan Open Assignment membuka klaimnya pada tahap alur kerjanya.
    // Yang dituju karena itu layar TERSENDIRI — section `SendtoRCLPUCL` — bukan pratinjau
    // baris di bawah tabel.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByRole('button', { name: /PNC-700001/ }))

    expect(await screen.findByText('Lampiran Surat')).toBeInTheDocument()
    expect(screen.getByText('Penerimaan Dokumen')).toBeInTheDocument()

    // Antreannya BENAR-BENAR ditinggalkan: bilah tab tidak lagi tergambar.
    expect(screen.queryByRole('tab', { name: /Cetak Surat/ })).not.toBeInTheDocument()
  })

  it('membawa kunci klaim yang TERKODEKAN ke alamat layar kerja', async () => {
    // `pzInsKey` memuat SPASI (`ASM-FW-GCNMFW-WORK PNC-700001`). Spasi mentah di dalam
    // alamat bukan alamat yang sah, dan permintaannya akan berangkat dengan kunci yang
    // rusak — tanpa satu pun galat di sisi layar.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByRole('button', { name: /PNC-700001/ }))
    await screen.findByText('Lampiran Surat')

    const claimCall = [...calls].reverse().find((c) => c.url.startsWith(CLAIM_PATH))
    expect(claimCall?.url).toContain('ASM-FW-GCNMFW-WORK%20PNC-700001')
  })

  it('mengembalikan tab dan halaman yang sama saat kembali dari layar kerja', async () => {
    // Tanpa ini, petugas yang membuka satu klaim dari tab kedua mendarat kembali di tab
    // pertama — dan antrean ini dikerjakan berpuluh baris sehari.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Kelengkapan Dokumen/ }))
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/)

    await userEvent.click(await screen.findByRole('button', { name: /PNC-700001/ }))
    await screen.findByText('Lampiran Surat')

    await userEvent.click(screen.getByRole('button', { name: /Kembali ke antrean/ }))

    expect(
      await screen.findByRole('tab', { name: /Kelengkapan Dokumen/, selected: true }),
    ).toBeInTheDocument()
  })

  it('menjelaskan antrean kosong menurut sebabnya', async () => {
    // "Tidak ada data" tidak cukup: ketiga tab punya kolom yang identik, sehingga pengguna
    // tidak punya petunjuk apakah ia berada di tab yang salah atau antreannya memang sepi.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Klaim MSIG/ }))

    expect(
      await screen.findByText(/tab ini memang diperkirakan kosong/i),
    ).toBeInTheDocument()
  })
})

describe('ekspor', () => {
  it('mengirim rentang tanggal pada tab Cetak Surat', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('FROM RCL/PUCL'), '2026-09-01')
    await userEvent.type(screen.getByLabelText('TO RCL/PUCL'), '2026-09-30')
    await userEvent.click(screen.getByRole('button', { name: /Unduh Laporan Harian/ }))

    const call = lastExportCall()
    expect(call?.url).toContain('dari=2026-09-01')
    expect(call?.url).toContain('sampai=2026-09-30')
  })

  it('TIDAK mengirim rentang tanggal pada tab yang mengekspor tabel', async () => {
    // Parameter yang tidak berarti apa-apa di alamat unduhan membuat orang menduga ia
    // berpengaruh.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Kelengkapan Dokumen/ }))
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/)

    await userEvent.click(screen.getByRole('button', { name: /Export To Excel/ }))

    const call = lastExportCall()
    expect(call?.url ?? '').not.toContain('dari=')
    expect(call?.url).toContain('tab=2')
  })

  it('menamai tombolnya berbeda supaya isinya diketahui sebelum diunduh', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('button', { name: /Unduh Laporan Harian/ })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: /Kelengkapan Dokumen/ }))
    await screen.findByText(/Klaim yang suratnya SUDAH dicetak/)

    expect(screen.getByRole('button', { name: /Export To Excel/ })).toBeInTheDocument()
  })

  it('menyebutkan kolom berkas laporan sebelum diunduh', async () => {
    // Isinya berbeda dari tabel yang sedang dilihat, dan perbedaan itu tidak boleh baru
    // ketahuan setelah berkasnya dibuka.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/Berisi:.*Status Klaim/)).toBeInTheDocument()
  })

  it('menampilkan pesan per isian saat rentangnya ditolak server', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url.startsWith(EXPORT_PATH)) {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Permintaan belum benar.',
          detail: [
            { field: 'dari', pesan: 'Tanggal "FROM RCL/PUCL" wajib diisi.' },
            { field: 'sampai', pesan: 'Tanggal "TO RCL/PUCL" wajib diisi.' },
          ],
        })
      }
      return jsonResponse(200, {
        tab: TAB_CETAK,
        baris: [BARIS],
        paginasi: { halaman: 1, ukuran: 50, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    })

    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: /Unduh Laporan Harian/ }))

    // Pesan per isian lebih berguna daripada pesan umum: ia menyebut isian MANA yang harus
    // diperbaiki, dan di layar ini isiannya ada dua.
    expect(await screen.findByRole('alert')).toHaveTextContent(/FROM RCL\/PUCL/)
  })
})

describe('portal', () => {
  it('meminta pengguna memilih entitas lebih dulu', async () => {
    useSelectedPortal.getState().clear()
    stubDefaultFetch()

    renderPage()

    expect(await screen.findByText(/Pilih entitas lebih dulu/)).toBeInTheDocument()
  })
})
