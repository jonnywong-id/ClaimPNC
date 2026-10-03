import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  AppealRow,
  Decision,
  DecisionColumn,
  MetadataResponse,
  PageInfo,
  QueueInfo,
  SummaryResponse,
  Tab,
} from './types'

const PATH = '/api/inbox-banding-harga-salvage'
const TAB_PATH = `${PATH}/tab`
const RINGKAS_PATH = `${PATH}/ringkas`

const SAMPLE_PROFILE = {
  identitas: '90000003',
  nama: 'Contoh Komite Salvage',
  jenis: 'KARYAWAN',
  login: 'komitesalvage',
  email: 'contoh.komite@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kesembilan kolom grid Request, berurutan seperti sel gridnya di Pega.
 *
 * Judulnya diambil dari `pyCaption` pada `Section/InboxReqSalvageASM-Section.xml` — tidak
 * diterjemahkan dan tidak dibetulkan (`D-13`).
 */
const KOLOM_REQUEST: Tab['kolom'] = [
  { kunci: 'tanggal_request', judul: 'Tanggal Request', angka: false },
  { kunci: 'no_klaim', judul: 'No Klaim', angka: false },
  { kunci: 'detail_object', judul: 'Detail Object', angka: false },
  { kunci: 'nama_barang', judul: 'Nama Barang', angka: false },
  { kunci: 'harga_barang', judul: 'Harga Barang', angka: true },
  { kunci: 'harga_request', judul: 'Harga Request', angka: true },
  { kunci: 'note_request', judul: 'Note Request', angka: false },
  { kunci: 'aging', judul: 'Aging', angka: false },
  { kunci: 'note_checker', judul: 'Note Checker', angka: false },
]

/** Keempat kolom grid History. "Object Name" sebenarnya berisi jenis salvage. */
const KOLOM_HISTORY: Tab['kolom'] = [
  { kunci: 'no_klaim', judul: 'No Klaim', angka: false },
  { kunci: 'object_name', judul: 'Object Name', angka: false },
  { kunci: 'lokasi_salvage', judul: 'Lokasi Salvage', angka: false },
  { kunci: 'pic', judul: 'PIC', angka: false },
]

const TAB_REQUEST: Tab = {
  kode: 'request-banding-harga',
  nama: 'Request Banding Harga',
  keterangan: 'Banding harga yang BELUM diputus.',
  parameter_pega: '1',
  kolom: KOLOM_REQUEST,
}

const TAB_HISTORY: Tab = {
  kode: 'history-cheker',
  nama: 'History Cheker',
  keterangan: 'Pengajuan salvage yang banding harganya SUDAH Anda putuskan.',
  parameter_pega: '2',
  kolom: KOLOM_HISTORY,
}

/** Ketujuh kolom panel rincian, berurutan seperti sel gridnya di Pega. */
const KOLOM_RINCIAN: DecisionColumn[] = [
  { kunci: 'tanggal_approve', judul: 'Tgl Approve', angka: false },
  { kunci: 'detail_object', judul: 'Detail Object', angka: false },
  { kunci: 'nama_barang', judul: 'Nama Barang', angka: false },
  { kunci: 'harga_barang', judul: 'Harga Barang', angka: true },
  { kunci: 'harga_request_keputusan', judul: 'Harga Request', angka: true },
  { kunci: 'jawaban_checker', judul: 'Jawaban Checker', angka: false },
  { kunci: 'nama_checker', judul: 'Nama Checker', angka: false },
]

const METADATA: MetadataResponse = {
  tab: [TAB_REQUEST, TAB_HISTORY],
  tab_bawaan: 'request-banding-harga',
  label_cari: 'Cari No Klaim',
  petunjuk_cari: 'Contoh : PNC-1234',
  selisih_terencana: ['Kolom Aging diurutkan sebagai ANGKA hari, bukan sebagai teks.'],
  keterbatasan: [
    'Keputusan Approve dan Reject TERSIMPAN, tetapi BELUM DIKIRIM ke balai lelang.',
  ],
  kolom_rincian: KOLOM_RINCIAN,
  portal: 'ASM',
}

/** Satu keputusan contoh. Seluruh isinya karangan (`D-69`). */
const KEPUTUSAN: Decision = {
  tanggal_approve: '2026-09-20',
  detail_object: 'PNC-0440/1',
  nama_barang: 'Forklift Rusak',
  harga_barang: '27000000.00',
  harga_request_keputusan: '21000000.00',
  jawaban_checker_kode: '1',
  jawaban_checker: 'Setuju',
  nama_checker: 'KOMITESALVAGE',
}

const ANTREAN_SENDIRI: QueueInfo = {
  milik: 'KOMITESALVAGE',
  diwakilkan: false,
  catatan_perwakilan: '',
  catatan_giliran: '',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas yang
 * di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const BARIS: AppealRow = {
  no_klaim: 'PNCN.26.0451',
  tanggal_request: '2026-08-30',
  detail_object: 'PNC-0451/1',
  nama_barang: 'Mesin Genset Bekas',
  harga_barang: '12500000.00',
  harga_request: '9750000.00',
  note_request: 'Kondisi mesin di bawah taksiran.',
  note_checker: '',
  aging: '30 days',
  object_name: '',
  lokasi_salvage: '',
  pic: '',
  id_salvage: '451',
  nama_komite: 'KOMITESALVAGE',
}

/**
 * Baris contoh grid History.
 *
 * Isian grid Request dibiarkan kosong, dan itu memang bentuknya: kedua grid membaca tabel
 * yang berbeda, sehingga isian di luar tabnya tidak pernah terisi.
 */
const BARIS_HISTORY: AppealRow = {
  no_klaim: 'PNCN.26.0440',
  tanggal_request: null,
  detail_object: '',
  nama_barang: '',
  harga_barang: '',
  harga_request: '',
  note_request: '',
  note_checker: '',
  aging: '',
  object_name: 'Alat Berat',
  lokasi_salvage: 'Gudang Cakung',
  pic: 'PICTEKNIKSATU',
  id_salvage: '440',
  nama_komite: 'KOMITESALVAGE',
}

const PAGINASI: PageInfo = { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 }

const RINGKAS: SummaryResponse = {
  baris: [
    { status_salvage: 'Request Banding Harga', tab: 'request-banding-harga', jumlah: 1 },
    { status_salvage: 'History Cheker', tab: 'history-cheker', jumlah: 4 },
  ],
  antrean: ANTREAN_SENDIRI,
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

/** Peladen tiruan yang menjawab bentuk layar, ringkasan, dan satu baris antrean. */
function stubDefaultFetch(
  baris: AppealRow[] = [BARIS],
  tab: Tab = TAB_REQUEST,
  antrean: QueueInfo = ANTREAN_SENDIRI,
) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url === RINGKAS_PATH) return jsonResponse(200, { ...RINGKAS, antrean })
    if (url.startsWith(`${PATH}/riwayat/`)) {
      return jsonResponse(200, {
        no_klaim: 'PNCN.26.0440',
        baris: [KEPUTUSAN],
        antrean,
        portal: 'ASM',
      })
    }
    return jsonResponse(200, {
      tab,
      baris,
      paginasi: { ...PAGINASI, total: baris.length },
      penyaring: { cari: '' },
      antrean,
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
      <MemoryRouter initialEntries={['/inbox-banding-harga-salvage']}>
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
  await screen.findByRole('tab', { name: 'Request Banding Harga' })
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

    expect(screen.getByRole('tab', { name: 'Request Banding Harga' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar kesembilan kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const judul of [
      'Tanggal Request',
      'No Klaim',
      'Detail Object',
      'Nama Barang',
      'Harga Barang',
      'Harga Request',
      'Note Request',
      'Aging',
      'Note Checker',
    ]) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  // Judul kolom ini menyesatkan di layar lama — isinya jenis salvage, bukan nama objek.
  // Ia dipertahankan apa adanya (`D-13`), dan uji ini menjaga agar tidak "diperbaiki".
  it('mempertahankan judul kolom "Object Name" pada tab History apa adanya', async () => {
    // Satu baris diberikan, bukan daftar kosong: DataTable menggambar kepala kolom hanya
    // bila ada isinya, sehingga daftar kosong akan menguji ketiadaan tabel — bukan judulnya.
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: 'History Cheker' })
    await userEvent.click(screen.getByRole('tab', { name: 'History Cheker' }))

    expect(
      await screen.findByRole('columnheader', { name: 'Object Name' }),
    ).toBeInTheDocument()
  })
})

describe('isi sel', () => {
  it('menggambar nilai uang dengan pemisah ribuan, bukan angka mentah', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('12.500.000')).toBeInTheDocument()
    expect(screen.getByText('9.750.000')).toBeInTheDocument()
  })

  // Teks `<n> days` mengikuti layar lama. Yang tidak diikuti adalah urutannya — itu
  // dikerjakan server, dan dinyatakan lewat selisih terencana.
  it('menggambar umur sebagai teks "<n> days" seperti layar lama', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('30 days')).toBeInTheDocument()
  })

  // Kolom "Note Checker" SELALU kosong di layar lama. Sel kosong tanpa tanda tidak dapat
  // dibedakan dari kolom yang gagal dimuat.
  it('menandai sel kosong dengan tanda hubung, bukan ruang kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByText('Mesin Genset Bekas')
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)
  })
})

describe('tabel ringkas', () => {
  it('menggambar kedua baris beserta jumlahnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByRole('button', { name: 'History Cheker' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Status Salvage' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Jumlah' })).toBeInTheDocument()
  })

  // Di layar lama, tabel ringkas inilah cara berpindah daftar. Menggambarnya sebagai angka
  // mati akan menghilangkan satu cara berpindah yang sudah ada.
  it('membuka tab ketika barisnya diklik', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByRole('button', { name: 'History Cheker' }))

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: 'History Cheker' })).toHaveAttribute(
        'aria-selected',
        'true',
      )
    })
  })
})

describe('pencarian', () => {
  it('mengirim kata kunci ke server setelah ketikan berhenti', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(
      screen.getByLabelText('Cari No Klaim'),
      'PNCN.26.0451',
    )

    await waitFor(() => {
      expect(lastListCall()?.url).toContain('cari=PNCN.26.0451')
    })
  })

  // Pencarian COCOK PERSIS. Tanpa keterangan itu, pengguna yang mengetik separuh nomor
  // klaim akan menyimpulkan antreannya kosong.
  it('menjelaskan bahwa pencariannya cocok persis saat hasilnya nol', async () => {
    stubDefaultFetch([])
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('Cari No Klaim'), 'PNCN')

    expect(await screen.findByText(/COCOK PERSIS/)).toBeInTheDocument()
  })

  // Membawa kata kunci lama ke tab baru akan menampilkan antrean yang tampak kosong padahal
  // isinya ada — dan pada pencarian yang cocok persis, itu hampir selalu terjadi.
  it('membersihkan kotak cari saat berpindah tab', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const kotak = screen.getByLabelText('Cari No Klaim')
    await userEvent.type(kotak, 'PNCN.26.0451')
    await waitFor(() => expect(lastListCall()?.url).toContain('cari='))

    await userEvent.click(screen.getByRole('tab', { name: 'History Cheker' }))

    await waitFor(() => expect(kotak).toHaveValue(''))
    await waitFor(() => expect(lastListCall()?.url).not.toContain('cari='))
  })
})

describe('antrean siapa yang dibaca', () => {
  // Petugas yang melihat antrean komite lain tanpa diberi tahu akan menyimpulkan antreannya
  // sendiri kosong — kesimpulan yang tidak akan ia laporkan sebagai kerusakan.
  it('menyatakan dengan jelas saat antrean yang tampil milik komite lain', async () => {
    stubDefaultFetch([BARIS], TAB_REQUEST, {
      milik: 'BAMBANGSETIADJIGUNAWAN',
      diwakilkan: true,
      catatan_perwakilan:
        'Anda sedang melihat antrean banding milik BAMBANGSETIADJIGUNAWAN, ' +
        'bukan antrean Anda sendiri.',
      catatan_giliran: '',
    })
    await renderLoaded()

    expect(
      await screen.findByText(/melihat antrean banding milik BAMBANGSETIADJIGUNAWAN/),
    ).toBeInTheDocument()
  })

  it('menyatakan saat sebagian baris ditahan menunggu komite lain', async () => {
    stubDefaultFetch([BARIS], TAB_REQUEST, {
      milik: 'DANIELLISWANDI',
      diwakilkan: false,
      catatan_perwakilan: '',
      catatan_giliran:
        'Banding yang belum diputus BAMBANGSETIADJIGUNAWAN tidak ditampilkan di sini.',
    })
    await renderLoaded()

    expect(await screen.findByText(/belum diputus BAMBANGSETIADJIGUNAWAN/)).toBeInTheDocument()
  })

  it('tidak menggambar keterangan apa pun saat antreannya milik sendiri', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByText('Mesin Genset Bekas')
    expect(screen.queryByText(/melihat antrean banding milik/)).not.toBeInTheDocument()
  })
})

describe('catatan bawah', () => {
  // Selisih yang tidak dinyatakan akan dilaporkan berulang kali sebagai kerusakan, dan pada
  // uji kesetaraan gerbang 1 ia akan diperlakukan sebagai bug (`D-54`).
  it('menggambar selisih terencana dan keterbatasan dari server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByText(/Kolom Aging diurutkan sebagai ANGKA hari/),
    ).toBeInTheDocument()
    expect(screen.getByText(/BELUM DIKIRIM ke balai lelang/)).toBeInTheDocument()
  })
})

describe('portal', () => {
  it('menolak membuka layar sebelum entitas dipilih', async () => {
    stubDefaultFetch()
    useSelectedPortal.getState().clear()

    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  })
})

describe('panel riwayat keputusan', () => {
  // Kedua grid punya kolom aksi, tetapi ISINYA berbeda — persis seperti di Pega. Tombol
  // riwayat milik History; grid Request memberi Approve dan Reject.
  it('tidak menggambar tombol riwayat pada tab Request', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByText('Mesin Genset Bekas')
    expect(screen.queryByRole('button', { name: 'Lihat Riwayat' })).not.toBeInTheDocument()
  })

  it('membuka panel keputusan dari baris History', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: 'History Cheker' })
    await userEvent.click(screen.getByRole('tab', { name: 'History Cheker' }))

    await userEvent.click(await screen.findByRole('button', { name: 'Lihat Riwayat' }))

    expect(await screen.findByText('Riwayat Keputusan Banding')).toBeInTheDocument()
    expect(await screen.findByText('Forklift Rusak')).toBeInTheDocument()
    expect(screen.getByText('Setuju')).toBeInTheDocument()
  })

  it('menggambar ketujuh kolom panel yang ditetapkan server', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: 'History Cheker' })
    await userEvent.click(screen.getByRole('tab', { name: 'History Cheker' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Lihat Riwayat' }))

    await screen.findByText('Riwayat Keputusan Banding')
    for (const judul of ['Tgl Approve', 'Jawaban Checker', 'Nama Checker']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  it('menutup panel ketika tombol Tutup ditekan', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: 'History Cheker' })
    await userEvent.click(screen.getByRole('tab', { name: 'History Cheker' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Lihat Riwayat' }))
    await screen.findByText('Riwayat Keputusan Banding')

    await userEvent.click(screen.getByRole('button', { name: 'Tutup riwayat keputusan' }))

    await waitFor(() => {
      expect(screen.queryByText('Riwayat Keputusan Banding')).not.toBeInTheDocument()
    })
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// Tombol Approve dan Reject
// ─────────────────────────────────────────────────────────────────────────────

const KEPUTUSAN_PATH = `${PATH}/keputusan`

const PESAN_TANPA_HARGA =
  'Persetujuan tersimpan dan banding ini berpindah ke History Cheker. Harga barang BELUM ' +
  'diubah: di layar lama, penerapan harga hanya dilakukan jenjang komite terakhir. Balai ' +
  'lelang TIDAK diberi tahu dari aplikasi ini — pengirimannya belum dibangun.'

/** Peladen tiruan yang ikut menjawab POST keputusan. */
function stubDecisionFetch(
  keputusan: () => Response = () =>
    jsonResponse(200, {
      tersimpan: true,
      harga_diterapkan: false,
      dokumen_ditandai: false,
      pesan: PESAN_TANPA_HARGA,
      portal: 'ASM',
    }),
) {
  stubFetch((url) => {
    if (url === KEPUTUSAN_PATH) return keputusan()
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url === RINGKAS_PATH) return jsonResponse(200, RINGKAS)
    return jsonResponse(200, {
      tab: TAB_REQUEST,
      baris: [BARIS],
      paginasi: PAGINASI,
      penyaring: { cari: '' },
      antrean: ANTREAN_SENDIRI,
      portal: 'ASM',
    })
  })
}

/** bukaPenegasan menggambar layar lalu menekan salah satu tombol keputusan. */
async function bukaPenegasan(nama: 'Approve' | 'Reject') {
  await renderLoaded()
  await screen.findByText('Mesin Genset Bekas')
  await userEvent.click(screen.getByRole('button', { name: nama }))
}

function decisionCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url === KEPUTUSAN_PATH)
}

describe('tombol Approve dan Reject', () => {
  it('menggambar kedua tombol pada tab Request', async () => {
    stubDecisionFetch()
    await renderLoaded()

    await screen.findByText('Mesin Genset Bekas')
    expect(screen.getByRole('button', { name: 'Approve' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reject' })).toBeInTheDocument()
  })

  it('tidak menggambar tombol keputusan pada tab History', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: 'History Cheker' })
    await userEvent.click(screen.getByRole('tab', { name: 'History Cheker' }))
    await screen.findByText('Alat Berat')

    expect(screen.queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument()
  })

  // Satu klik TIDAK langsung menyimpan. Putusan ini menyentuh nilai uang dan tidak dapat
  // ditarik kembali; kesempatan membacanya dulu adalah inti dari kotak penegasan.
  it('tidak mengirim apa pun sebelum penegasan ditekan', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Approve')

    expect(await screen.findByText(/Approve banding harga/)).toBeInTheDocument()
    expect(decisionCall()).toBeUndefined()
  })

  // Yang diputuskan adalah SELISIH dua angka, dan keduanya harus terlihat berdampingan.
  it('menggambar kedua harga beserta selisihnya', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Approve')

    await screen.findByText(/Approve banding harga/)
    expect(screen.getAllByText('12.500.000').length).toBeGreaterThan(0)
    expect(screen.getAllByText('9.750.000').length).toBeGreaterThan(0)
    expect(screen.getByText('-2.750.000')).toBeInTheDocument()
  })

  it('mengirim identitas baris dan catatan yang diketik', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Approve')

    await userEvent.type(screen.getByLabelText('Note Checker'), '  sudah dibandingkan  ')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan Approve' }))

    await waitFor(() => expect(decisionCall()).toBeDefined())

    const body = JSON.parse(String(decisionCall()?.init?.body))
    expect(body).toEqual({
      detail_object: 'PNC-0451/1',
      id_salvage: '451',
      harga_request: '9750000.00',
      catatan: 'sudah dibandingkan',
      setujui: true,
    })
  })

  it('mengirim setujui=false ketika yang ditekan Reject', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Reject')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan Reject' }))

    await waitFor(() => expect(decisionCall()).toBeDefined())
    expect(JSON.parse(String(decisionCall()?.init?.body)).setujui).toBe(false)
  })

  // Catatan TIDAK diwajibkan, meniru layar lama — di sana ia isian biasa tanpa penanda wajib.
  it('menyimpan meski catatan dikosongkan', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Approve')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan Approve' }))

    await waitFor(() => expect(decisionCall()).toBeDefined())
    expect(JSON.parse(String(decisionCall()?.init?.body)).catatan).toBe('')
  })

  /*
    Kabar hasilnya datang dari SERVER apa adanya.

    Ini yang paling mahal bila salah: menyetujui tidak selalu mengubah harga, dan tidak
    pernah memberi tahu balai lelang. Kalimat "berhasil disimpan" yang disusun layar akan
    menyiratkan keduanya sudah terjadi.
  */
  it('menampilkan pesan server apa adanya, bukan kalimat yang disusun layar', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Approve')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan Approve' }))

    expect(await screen.findByText(PESAN_TANPA_HARGA)).toBeInTheDocument()
    expect(screen.queryByText(/Approve banding harga/)).not.toBeInTheDocument()
  })

  it('menutup kotak penegasan tanpa mengirim apa pun ketika Batal ditekan', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Approve')

    await screen.findByText(/Approve banding harga/)
    await userEvent.click(screen.getAllByRole('button', { name: 'Batal' })[0]!)

    await waitFor(() => {
      expect(screen.queryByText(/Approve banding harga/)).not.toBeInTheDocument()
    })
    expect(decisionCall()).toBeUndefined()
  })

  // Catatan yang ditulis untuk satu keputusan tidak boleh terbawa ke keputusan berikutnya —
  // ia tersimpan sebagai alasan putusan atas barang yang salah.
  it('mengosongkan catatan ketika tombol lain ditekan', async () => {
    stubDecisionFetch()
    await bukaPenegasan('Approve')

    await userEvent.type(screen.getByLabelText('Note Checker'), 'catatan lama')
    await userEvent.click(screen.getAllByRole('button', { name: 'Batal' })[0]!)
    await userEvent.click(screen.getByRole('button', { name: 'Reject' }))

    expect(await screen.findByLabelText('Note Checker')).toHaveValue('')
  })

  /*
    "Sudah diputus" BUKAN kerusakan.

    Ia jawaban yang sah atas penekanan kedua, atau atas baris yang komite lain sudah
    tangani. Menampilkannya dengan nada yang sama seperti galat sistem akan membuat
    pengguna melaporkannya sebagai kerusakan.
  */
  it('membedakan "sudah diputus" dari galat sistem', async () => {
    stubDecisionFetch(() =>
      jsonResponse(409, {
        kode: 'banding_sudah_diputus',
        pesan: 'Banding ini sudah diputus, atau bukan banding yang Anda tangani.',
      }),
    )
    await bukaPenegasan('Approve')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan Approve' }))

    expect(await screen.findByText('Banding ini sudah diputus')).toBeInTheDocument()
  })

  it('menahan kotak penegasan tetap terbuka ketika penyimpanan gagal', async () => {
    stubDecisionFetch(() =>
      jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' }),
    )
    await bukaPenegasan('Reject')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan Reject' }))

    expect(await screen.findByText('Keputusan tidak tersimpan')).toBeInTheDocument()
    expect(screen.getByText(/Reject banding harga/)).toBeInTheDocument()
  })
})

/*
  Baris yang HARGA BARANG-nya kosong sementara harga request terisi.

  Bentuk ini tidak dikarang: ia ada di `T_CLAIM_CHEKER_SALVAGE` produksi, terlihat saat Work
  Owner memperlihatkan isi tabelnya pada 2026-09-30. Isinya tetap karangan (`D-69`) — yang
  ditiru hanya BENTUKNYA.
*/
const BARIS_TANPA_HARGA_BARANG: AppealRow = {
  ...BARIS,
  no_klaim: 'PNC-9001',
  detail_object: 'PNC-9001/1',
  id_salvage: '901',
  nama_barang: 'Kulkas Bekas',
  harga_barang: '',
  harga_request: '1350000.00',
}

function stubBarisTanpaHarga() {
  stubDecisionFetch()
  stubFetch((url) => {
    if (url === KEPUTUSAN_PATH) {
      return jsonResponse(200, {
        tersimpan: true,
        harga_diterapkan: false,
        dokumen_ditandai: false,
        pesan: PESAN_TANPA_HARGA,
        portal: 'ASM',
      })
    }
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url === RINGKAS_PATH) return jsonResponse(200, RINGKAS)
    return jsonResponse(200, {
      tab: TAB_REQUEST,
      baris: [BARIS_TANPA_HARGA_BARANG],
      paginasi: PAGINASI,
      penyaring: { cari: '' },
      antrean: ANTREAN_SENDIRI,
      portal: 'ASM',
    })
  })
}

describe('harga barang yang kosong', () => {
  /*
    `Number('')` menghasilkan **0**, dan `Number.isFinite(0)` bernilai benar.

    Tanpa penjagaan, selisihnya terhitung `request - 0` dan panel menampilkan SELURUH harga
    request sebagai "Selisih" — seolah harga dasarnya memang nol. Komite lalu membaca angka
    yang salah tepat sebelum memutuskan nilai uang.

    Kelas cacatnya sama dengan `GETSELISIHJAM` (`D-49` butir 10): nol yang tidak dapat
    dibedakan dari kegagalan membaca. Ditemukan dari data produksi, bukan dari membaca ulang
    kodenya.
  */
  it('tidak menghitung selisih ketika harga barang kosong', async () => {
    stubBarisTanpaHarga()

    await renderLoaded()
    await screen.findByText('Kulkas Bekas')
    await userEvent.click(screen.getByRole('button', { name: 'Approve' }))
    await screen.findByText(/Approve banding harga/)

    /*
      Sel "Selisih" ditunjuk LANGSUNG, bukan dengan mencari teks angkanya.

      Dengan bug, selisihnya terhitung `1.350.000 - 0` dan tergambar **`1.350.000`** —
      teks yang sama persis dengan sel "Harga Request" di sebelahnya. Uji yang mencari teks
      itu di seluruh layar karena itu akan LULUS meski bugnya ada; yang membedakan hanyalah
      sel mana yang memuatnya.
    */
    const selisih = screen.getByText('Selisih').nextElementSibling
    expect(selisih).toHaveTextContent('—')
    expect(selisih).not.toHaveTextContent('1.350.000')

    // Harga request sendiri TETAP tergambar — yang tidak terbaca hanyalah selisihnya.
    expect(screen.getAllByText('1.350.000').length).toBeGreaterThan(0)
  })

  // Harga barang yang kosong TIDAK menghalangi keputusan: yang disetujui adalah harga
  // request, dan itu terisi. Layar lama pun tidak menghalanginya.
  it('tetap dapat diputuskan meski harga barangnya kosong', async () => {
    stubBarisTanpaHarga()

    await renderLoaded()
    await screen.findByText('Kulkas Bekas')
    await userEvent.click(screen.getByRole('button', { name: 'Approve' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Simpan Approve' }))

    await waitFor(() => expect(decisionCall()).toBeDefined())
    expect(JSON.parse(String(decisionCall()?.init?.body)).harga_request).toBe('1350000.00')
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// Dialog "Lihat File"
// ─────────────────────────────────────────────────────────────────────────────

const DOKUMEN_PATH = `${PATH}/dokumen`

/** Dua dokumen contoh. Seluruh isinya karangan (`D-69`). */
const DOKUMEN = [
  {
    id: '9001',
    kategori: 'BandingHarga',
    nama: 'penawaran-balai-lelang.pdf',
    tanggal_unggah: '2026-09-02',
  },
  {
    id: '9002',
    kategori: 'BandingHarga',
    nama: 'foto-kondisi-barang.jpg',
    tanggal_unggah: null,
  },
]

function stubDocumentFetch(dokumen: unknown[] = DOKUMEN, status = 200) {
  stubFetch((url) => {
    if (url.startsWith(DOKUMEN_PATH)) {
      return status === 200
        ? jsonResponse(200, { baris: dokumen, portal: 'ASM' })
        : jsonResponse(status, {
            kode: 'dokumen_tidak_ditemukan',
            pesan: 'Dokumen ini tidak dapat dibuka.',
          })
    }
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url === RINGKAS_PATH) return jsonResponse(200, RINGKAS)
    return jsonResponse(200, {
      tab: TAB_REQUEST,
      baris: [BARIS],
      paginasi: PAGINASI,
      penyaring: { cari: '' },
      antrean: ANTREAN_SENDIRI,
      portal: 'ASM',
    })
  })
}

async function bukaDokumen() {
  await renderLoaded()
  await screen.findByText('Mesin Genset Bekas')
  await userEvent.click(screen.getByRole('button', { name: 'Lihat File' }))
}

function documentCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(DOKUMEN_PATH))
}

describe('dialog Lihat File', () => {
  it('menggambar tombolnya hanya pada tab Request', async () => {
    stubDocumentFetch()
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')

    expect(screen.getByRole('button', { name: 'Lihat File' })).toBeInTheDocument()
  })

  it('tidak meminta dokumen sebelum tombolnya ditekan', async () => {
    stubDocumentFetch()
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')

    expect(documentCall()).toBeUndefined()
  })

  /*
    Kedua id dikirim sebagai PARAMETER KUERI, bukan sebagai ruas jalur.

    `IDDETAILSALVAGE` memuat garis miring di dalamnya (`PNC-<n>/<m>`). Sebagai ruas jalur ia
    terpecah menjadi dua segmen dan rutenya tidak akan pernah cocok — kerusakan yang hanya
    muncul pada data sungguhan, bukan pada id karangan yang tidak bergaris miring.
  */
  it('mengirim kedua id sebagai parameter kueri, bukan ruas jalur', async () => {
    stubDocumentFetch()
    await bukaDokumen()

    await waitFor(() => expect(documentCall()).toBeDefined())

    const url = new URL(documentCall()!.url, 'https://uji.invalid')
    expect(url.pathname).toBe(DOKUMEN_PATH)
    expect(url.searchParams.get('detail_object')).toBe('PNC-0451/1')
    expect(url.searchParams.get('id_salvage')).toBe('451')
  })

  it('menggambar ketiga kolom section lama', async () => {
    stubDocumentFetch()
    await bukaDokumen()

    await screen.findByText('Dokumen Banding Harga — PNCN.26.0451')
    for (const judul of ['Kategori', 'Nama', 'Tanggal']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  // Nama berkas adalah TAUTAN UNDUH, bukan teks — di layar lama pun begitu (kontrol
  // `PNCDownloadFile`).
  it('menggambar nama berkas sebagai tautan unduh', async () => {
    stubDocumentFetch()
    await bukaDokumen()

    const tautan = await screen.findByRole('link', { name: 'penawaran-balai-lelang.pdf' })
    expect(tautan).toHaveAttribute('download', 'penawaran-balai-lelang.pdf')

    const href = new URL(tautan.getAttribute('href')!, 'https://uji.invalid')
    expect(href.pathname).toBe(`${DOKUMEN_PATH}/9001`)
    expect(href.searchParams.get('detail_object')).toBe('PNC-0451/1')
  })

  // Tanggal yang kosong digambar sebagai tanda hubung, bukan ruang kosong yang tidak dapat
  // dibedakan dari kolom yang gagal dimuat.
  it('menggambar tanggal kosong sebagai tanda hubung', async () => {
    stubDocumentFetch()
    await bukaDokumen()

    await screen.findByRole('link', { name: 'foto-kondisi-barang.jpg' })
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)
  })

  // Banding tanpa dokumen adalah keadaan yang SAH, bukan galat — dan pesannya menjelaskan
  // sebab yang paling mungkin.
  it('menjelaskan mengapa daftarnya kosong', async () => {
    stubDocumentFetch([])
    await bukaDokumen()

    expect(
      await screen.findByText(/tidak punya dokumen pendukung yang dapat dibuka/),
    ).toBeInTheDocument()
  })

  it('menutup dialog ketika tombol Tutup ditekan', async () => {
    stubDocumentFetch()
    await bukaDokumen()
    await screen.findByText('Dokumen Banding Harga — PNCN.26.0451')

    await userEvent.click(screen.getByRole('button', { name: 'Tutup dokumen banding' }))

    await waitFor(() => {
      expect(
        screen.queryByText('Dokumen Banding Harga — PNCN.26.0451'),
      ).not.toBeInTheDocument()
    })
  })

  it('menampilkan galat ketika dokumennya tidak dapat dimuat', async () => {
    stubDocumentFetch([], 404)
    await bukaDokumen()

    expect(await screen.findByText('Dokumen tidak dapat dimuat')).toBeInTheDocument()
  })
})
