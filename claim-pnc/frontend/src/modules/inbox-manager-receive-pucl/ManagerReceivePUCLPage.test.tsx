import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-manager-receive-pucl'
const TAB_PATH = `${PATH}/tab`
const EXPORT_PATH = `${PATH}/ekspor`

const SAMPLE_PROFILE = {
  identitas: '90000003',
  nama: 'Contoh Penyelia',
  jenis: 'KARYAWAN',
  login: 'penyeliacontoh',
  email: 'contoh.penyelia@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kolom tab Receive — sembilan, persis seperti pada kedua grid Pega.
 *
 * Keduanya digambar Report Definition yang sama (`ManagementRecieveView`), dijalankan
 * dengan parameter yang berbeda, sehingga kolomnya memang identik. Sejak keduanya digabung
 * menjadi satu tab, yang membedakan lini bisnisnya adalah ISI kolom "Jenis Klaim".
 */
const KOLOM_RECEIVE = [
  { kunci: 'no_case', judul: 'CaseID' },
  { kunci: 'no_polis', judul: 'No Polis' },
  { kunci: 'no_klaim_pnc', judul: 'PNC CaseID' },
  { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
  { kunci: 'tanggal_kejadian', judul: 'Tanggal Kejadian' },
  { kunci: 'jenis_klaim', judul: 'Jenis Klaim' },
  { kunci: 'nama_pengirim', judul: 'Nama Pengirim' },
  { kunci: 'tanggal_terima_dokumen', judul: 'Tanggal Terima Dokumen' },
  { kunci: 'jumlah_lembar_dokumen', judul: 'Jumlah Lembar Dokumen' },
] as Tab['kolom']

/** Tab Receive — satu-satunya yang nomor case-nya membuka layar kerja. */
const TAB_RECEIVE: Tab = {
  kode: '1',
  nama: 'Receive',
  keterangan:
    'Berkas penerimaan dokumen klaim yang masih punya penugasan terbuka.',
  kolom: KOLOM_RECEIVE,
  antrean_bersama: false,
  buka_layar_kerja: true,
  terhalang: false,
}

/** Tab RCL/PUCL — kolomnya BERBEDA seluruhnya, dan sumbernya antrean bersama. */
const TAB_RCLPUCL: Tab = {
  kode: '2',
  nama: 'RCL/PUCL',
  keterangan: 'Klaim yang ditolak (RCL) atau diproses ulang (PUCL).',
  kolom: [
    { kunci: 'no_case', judul: 'Nomor Case' },
    { kunci: 'no_polis', judul: 'No Polis' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
    { kunci: 'tanggal_masuk_inbox', judul: 'Tanggal Masuk Inbox' },
    { kunci: 'deskripsi_analyst', judul: 'Deskripsi Analyst' },
    { kunci: 'rcl_pucl', judul: 'RCL/PUCL' },
    { kunci: 'status_rcl_pucl', judul: 'Status RCL/PUCL' },
    { kunci: 'tanggal_cetak_surat', judul: 'Tanggal Cetak Surat' },
    { kunci: 'lama_klaim', judul: 'Lama Klaim' },
    { kunci: 'status_kadaluarsa', judul: 'Status Kadaluarsa' },
  ],
  antrean_bersama: true,

  // Nomor case di tab ini BUKAN tautan — di layar lama pun perilaku klik hanya dipasang
  // pada kedua grid Receive.
  buka_layar_kerja: false,

  terhalang: false,
}

const METADATA: MetadataResponse = {
  tab: [TAB_RECEIVE, TAB_RCLPUCL],
  tab_bawaan: '1',
  portal: 'ASM',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * `jumlah_lembar_dokumen` sengaja KOSONG: begitulah yang selalu dikirim server, karena
 * tidak ada kolom basis data untuknya. Uji di bawah memastikan layar menggambarnya sebagai
 * tanda pisah, bukan membiarkan selnya kosong.
 */
const BARIS_RECEIVE: WorkItem = {
  referensi: 'ASM-FW-GCNMFW-WORK RCV-900001',
  no_case: 'RCV-900001',
  no_polis: 'CONTOH-PA-0001',
  no_klaim_pnc: 'PNCN.26.0001',
  nama_tertanggung: 'Tertanggung Contoh Satu',
  tanggal_kejadian: '2026-09-01',
  jenis_klaim: 'PA',
  nama_pengirim: 'Pengirim Contoh Satu',
  tanggal_terima_dokumen: '03/09/2026',
  jumlah_lembar_dokumen: '',
  tanggal_masuk_inbox: '2026-09-03 09:14:00',
  deskripsi_analyst: '',
  rcl_pucl: '',
  status_rcl_pucl: '',
  tanggal_cetak_surat: '',
  lama_klaim: '',
  status_kadaluarsa: '',
}

const BARIS_RCLPUCL: WorkItem = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-800002',
  no_case: 'PNC-800002',
  no_polis: 'CONTOH-KLAIM-8002',
  no_klaim_pnc: '',
  nama_tertanggung: 'Tertanggung Contoh Tujuh',
  tanggal_kejadian: '',
  jenis_klaim: '',
  nama_pengirim: '',
  tanggal_terima_dokumen: '',
  jumlah_lembar_dokumen: '',
  tanggal_masuk_inbox: '2026-09-14 13:05:00',
  deskripsi_analyst: 'Dokumen pendukung tidak lengkap.',
  rcl_pucl: 'RCL',
  status_rcl_pucl: 'Menunggu Keputusan',
  tanggal_cetak_surat: '16/09/2026',
  lama_klaim: '8',
  status_kadaluarsa: 'Belum Kadaluarsa',
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

/**
 * Isi layar kerja seperlunya — hanya untuk menguji PERPINDAHAN dari antrean ke sana.
 *
 * Isinya sengaja minimal: bentuk layar kerja diuji lengkap di
 * `InputReceiveDocumentPage.test.tsx`, dan menyalinnya ke sini akan membuat dua berkas uji
 * menjaga hal yang sama.
 */
const DOKUMEN_RINGKAS = {
  referensi: BARIS_RECEIVE.referensi,
  no_case: BARIS_RECEIVE.no_case,
  no_klaim_pnc: BARIS_RECEIVE.no_klaim_pnc,
  jenis_klaim: 'PA',
  status_kerja: 'Open',
  kelompok: [
    {
      judul: 'Penerimaan Dokumen',
      isian: [{ kunci: 'nama_pengirim', judul: 'Nama Pengirim / Pelapor Dokumen' }],
    },
  ],
  nilai: { nama_pengirim: 'Pengirim Contoh Satu' },
  tindakan: [],
  portal: 'ASM',
}

/** Peladen tiruan yang menjawab bentuk layar dan satu baris antrean. */
function stubDefaultFetch(rows: WorkItem[] = [BARIS_RECEIVE], tab: Tab = TAB_RECEIVE) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url.startsWith(`${PATH}/dokumen/`)) return jsonResponse(200, DOKUMEN_RINGKAS)
    if (url.startsWith(EXPORT_PATH)) {
      return new Response('CaseID\nRCV-900001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="receive.csv"',
        },
      })
    }

    // Tab yang diminta menentukan bentuk jawabannya. Menjawab tab yang sama untuk setiap
    // permintaan akan membuat uji perpindahan tab lulus tanpa membuktikan apa pun.
    const wanted = new URL(url, 'https://uji.invalid').searchParams.get('tab')
    const answered = wanted === TAB_RCLPUCL.kode ? TAB_RCLPUCL : tab

    const baris = answered.kode === TAB_RCLPUCL.kode ? [BARIS_RCLPUCL] : rows

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
      <MemoryRouter initialEntries={['/inbox-manager-receive-pucl']}>
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
  await screen.findByRole('tab', { name: 'Receive' })
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
  it('menggambar DUA tab, sama dengan judul tab di Pega', async () => {
    // `Section/InboxManagerReceive_Section-Section.xml` memuat tepat dua `<pyTitle>`.
    // Versi pertama modul ini memecah "Receive" menjadi dua tab; pemecahan itu dicabut
    // 2026-09-30, dan uji ini yang menjaganya tidak diam-diam kembali menjadi tiga.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getAllByRole('tab')).toHaveLength(2)
    expect(screen.getByRole('tab', { name: 'Receive' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'RCL/PUCL' })).toBeInTheDocument()
  })

  it('membuka tab bawaan yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: 'Receive' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const column of TAB_RECEIVE.kolom) {
      expect(
        await screen.findByRole('columnheader', { name: column.judul }),
      ).toBeInTheDocument()
    }
  })

  it('menyatakan bahwa isinya pekerjaan seluruh petugas, bukan milik pemanggil', async () => {
    // Ini satu-satunya layar inbox yang TIDAK menyaring menurut pengguna yang login.
    // Tanpa keterangan itu di layar, petugas yang terbiasa dengan inbox lain akan mengira
    // daftarnya keliru karena memuat pekerjaan orang lain.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/seluruh/i)).toBeInTheDocument()
  })

  // Keputusan Work Owner 2026-10-06: panel selisih terencana DIHAPUS dari seluruh layar.
  //
  // Daftarnya tetap hidup di kode Go untuk uji kesetaraan gerbang 1 (`D-54`); yang berubah
  // adalah ia berhenti menjadi isi layar.
  it('tidak lagi menggambar panel selisih terencana', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.queryByText(/Jenis Klaim.*diturunkan dari Group Panel/),
    ).not.toBeInTheDocument()
  })
})

describe('perpindahan tab', () => {
  it('meminta kolom dan baris tab yang dipilih, bukan tab sebelumnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'RCL/PUCL' }))

    // Kolom tab RCL/PUCL berbeda seluruhnya dari tab Receive. Bila permintaannya tidak
    // membawa kode tab, layar akan tetap menggambar kolom tab sebelumnya.
    expect(
      await screen.findByRole('columnheader', { name: 'Deskripsi Analyst' }),
    ).toBeInTheDocument()

    expect(lastListCall()?.url).toContain('tab=2')
  })

  it('kembali ke halaman pertama saat tab berpindah', async () => {
    // Tanpa itu, berpindah dari halaman 4 sebuah tab ke tab yang hanya punya 2 halaman
    // akan menampilkan tabel kosong yang terbaca seperti antrean yang memang kosong.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'RCL/PUCL' }))
    await screen.findByRole('columnheader', { name: 'Deskripsi Analyst' })

    expect(lastListCall()?.url).not.toContain('halaman=')
  })
})

describe('penggambaran sel', () => {
  it('menggambar kolom yang selalu kosong sebagai tanda pisah, bukan sel kosong', async () => {
    // "Jumlah Lembar Dokumen" tidak punya kolom basis data mana pun. Selnya harus terbaca
    // sebagai "tidak ada isinya", bukan sebagai kolom yang gagal dimuat.
    stubDefaultFetch()
    await renderLoaded()

    const baris = await screen.findByText('RCV-900001')
    const row = baris.closest('tr')
    expect(row).not.toBeNull()
    expect(row?.textContent).toContain('—')
  })

  it('menampilkan Lama Klaim apa adanya, tanpa menambahkan satuan', async () => {
    // Satuannya tidak diketahui: tidak satu pun kueri di export menghitungnya, dan tidak
    // ada DDL yang menyatakan tipenya. Menulis "8 hari" berarti menetapkan satuan yang
    // belum pernah dipastikan.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'RCL/PUCL' }))
    await screen.findByRole('columnheader', { name: 'Lama Klaim' })

    expect(await screen.findByText('8')).toBeInTheDocument()
    expect(screen.queryByText('8 hari')).not.toBeInTheDocument()
  })
})

describe('portal', () => {
  it('menolak menggambar antrean sebelum entitas dipilih', async () => {
    // Antrean satu badan hukum bukan antrean badan hukum lain (`R-20`). Layar tidak boleh
    // menampilkan apa pun sebelum portalnya jelas.
    useSelectedPortal.getState().clear()
    stubDefaultFetch()

    renderPage()

    expect(await screen.findByText(/Pilih entitas lebih dulu/)).toBeInTheDocument()
    expect(lastListCall()).toBeUndefined()
  })
})

describe('ekspor', () => {
  it('mengirim tab yang sedang dilihat ke endpoint ekspor', async () => {
    // Kedua tab punya kolom yang berbeda, sehingga berkasnya pun berbeda susunannya.
    // Ekspor yang mengabaikan tab akan mengeluarkan berkas yang tidak dapat dicocokkan
    // dengan apa pun di layar.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'RCL/PUCL' }))
    await screen.findByRole('columnheader', { name: 'Deskripsi Analyst' })

    await userEvent.click(screen.getByRole('button', { name: /Export Data/ }))

    const ekspor = [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
    expect(ekspor?.url).toContain('tab=2')
  })

  it('mematikan tombol ekspor saat tidak ada yang dapat diekspor', async () => {
    // Berkas kosong yang tetap terunduh adalah jawaban yang membingungkan: pengguna tidak
    // dapat membedakannya dari ekspor yang gagal diam-diam.
    stubDefaultFetch([])
    await renderLoaded()

    expect(await screen.findByRole('button', { name: /Export Data/ })).toBeDisabled()
  })
})

describe('membuka layar kerja lewat nomor case', () => {
  it('menggambar nomor case tab Receive sebagai TAUTAN, bukan teks biasa', async () => {
    // Di Pega, sel nomor case pada grid Receive adalah `pyUIElement = link` yang menjalankan
    // `SetAssignmentInboxReceive_act` lalu Open Assignment.
    //
    // Versi pertama modul ini menggambarnya sebagai teks biasa dan menambahkan tombol
    // "Lihat Detail" di ujung baris — kolom yang tidak ada di Pega sama sekali.
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByRole('button', { name: 'RCV-900001' })).toBeInTheDocument()
  })

  it('TIDAK menambahkan kolom aksi yang tidak ada di Pega', async () => {
    // Kolom tambahan bukan sekadar beda tampilan: ia membuat pengguna mengira ada dua cara
    // berbeda membuka baris, dan menggeser lebar seluruh kolom lain.
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByRole('columnheader', { name: 'CaseID' })

    expect(screen.queryByRole('button', { name: /Lihat Detail/ })).not.toBeInTheDocument()
    expect(screen.getAllByRole('columnheader')).toHaveLength(TAB_RECEIVE.kolom.length)
  })

  it('TIDAK menjadikan nomor case tab RCL/PUCL sebuah tautan', async () => {
    // Di layar lama, perilaku klik hanya dipasang pada kedua grid Receive. Bila penandanya
    // bocor ke tab ini, nomor case menjadi tautan yang membuka layar kerja PENERIMAAN
    // DOKUMEN untuk sebuah KLAIM — dan kuerinya menyaring kelas objek kerja, sehingga
    // jawabannya "tidak ditemukan" pada setiap baris.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'RCL/PUCL' }))
    await screen.findByRole('columnheader', { name: 'Deskripsi Analyst' })

    expect(await screen.findByText('PNC-800002')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'PNC-800002' })).not.toBeInTheDocument()
  })

  it('menuju layar kerja dengan kunci terkodekan saat nomor case diklik', async () => {
    // `pzInsKey` memuat SPASI — `ASM-FW-GCNMFW-WORK RCV-900001` — dan spasi mentah di dalam
    // alamat bukan alamat yang sah.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByRole('button', { name: 'RCV-900001' }))

    // Layar kerjanya terbuka: judulnya menyebut berkas, bukan antrean.
    expect(await screen.findByText('Berkas RCV-900001')).toBeInTheDocument()

    const dokumen = [...calls].reverse().find((c) => c.url.startsWith(`${PATH}/dokumen/`))
    expect(dokumen?.url).toContain(encodeURIComponent(BARIS_RECEIVE.referensi))
  })

  it('membawa tab dan halaman ke alamat layar kerja supaya tombol kembali tepat', async () => {
    // Berpindah halaman melepas komponen antrean. Tanpa tab dan halaman di alamat, petugas
    // yang kembali dari sebuah berkas mendarat di tab pertama halaman pertama.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByRole('button', { name: 'RCV-900001' }))
    await screen.findByRole('button', { name: /Kembali ke antrean/ })

    await userEvent.click(screen.getByRole('button', { name: /Kembali ke antrean/ }))

    expect(await screen.findByRole('tab', { name: 'Receive' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })
})
