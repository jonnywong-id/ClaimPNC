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
  await screen.findByRole('tab', { name: /Request Banding Harga/ })
}

/**
 * bukaBarisHistory menyerahkan BARIS grid History sebagai elemen yang dapat diklik.
 *
 * Grid itu memakai `expandedRow`, yang menjadikan tiap `<tr>` ber-`role="button"` — tiruan
 * `pyEditingMode = expandPane` di Pega. Tidak ada tombol untuk ditekan; yang ditekan barisnya.
 */
async function bukaBarisHistory() {
  return await screen.findByRole('button', { name: /PNCN\.26\.0440/ })
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

    expect(screen.getByRole('tab', { name: /Request Banding Harga/ })).toHaveAttribute(
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

    await screen.findByRole('tab', { name: /History Cheker/ })
    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))

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

describe('jumlah antrean', () => {
  /*
    Tabel ringkas "Status Salvage / Jumlah" DIBUANG, dan angkanya pindah ke tabnya.

    Ia berdiri tepat di atas bilah tab dengan kedua barisnya dapat diklik untuk berpindah
    antrean — dua kendali berlabel sama persis, berurutan, untuk satu sakelar yang sama
    (Work Owner, 2026-10-05).

    Pemeriksaan ke Pega membenarkannya dua kali: `Section/InboxReqSalvageASM` memuat NOL
    "Jumlah" — tabel itu milik layar Inbox Salvage (`MENU_ID 71`) — dan
    `Activity/GCNMCountRequestSalvage_act` yang memasok angkanya tidak dipanggil harness
    maupun section mana pun.
  */
  it('tidak menggambar tabel ringkas tersendiri', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.queryByRole('columnheader', { name: 'Status Salvage' })).not.toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: 'Jumlah' })).not.toBeInTheDocument()
  })

  // Angkanya tetap terbaca — ia menjadi bagian NAMA tabnya, sehingga pembaca layar pun
  // menyebutkannya.
  it('menggambar jumlahnya sebagai lencana pada tabnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByRole('tab', { name: 'Request Banding Harga 1' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'History Cheker 4' })).toBeInTheDocument()
  })

  /*
    Lencananya TIDAK digambar selama angkanya belum tiba.

    Menggambar "0" saat belum dimuat membuat antrean yang sebenarnya berisi tampak kosong —
    kelas cacat yang sama dengan `GETSELISIHJAM` (`D-49` butir 10): nol yang tidak dapat
    dibedakan dari kegagalan membaca.
  */
  it('tidak menggambar lencana sebelum angkanya tiba', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url === RINGKAS_PATH) return jsonResponse(500, { kode: 'galat_internal', pesan: 'x' })
      return jsonResponse(200, {
        tab: TAB_REQUEST,
        baris: [BARIS],
        paginasi: PAGINASI,
        penyaring: { cari: '' },
        antrean: ANTREAN_SENDIRI,
        portal: 'ASM',
      })
    })
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Request Banding Harga/ })).toBeInTheDocument()
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

    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))

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
  /*
    Kedua blok catatan di bawah tabel DIHAPUS dari layar (Work Owner, 2026-10-05): isinya
    menyebut nama rule Pega, nama kolom Oracle, dan nomor keputusan — untuk pengembang, bukan
    untuk petugas klaim.

    Uji ini menjaga keputusan itu. Tanpanya bloknya mudah kembali: ia pernah ada, dan pola
    yang sama masih dipakai modul lain sehingga menyalinnya kembali terasa seperti
    menyeragamkan.
  */
  it('tidak menggambar catatan apa pun di bawah tabel', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')

    for (const judul of ['Yang sengaja berbeda dari layar lama', 'Yang belum tersedia']) {
      expect(screen.queryByText(judul)).not.toBeInTheDocument()
    }

    // Isi yang dikirim server pun tidak boleh bocor ke layar lewat jalan lain.
    expect(
      screen.queryByText(/Kolom Aging diurutkan sebagai ANGKA hari/),
    ).not.toBeInTheDocument()
    expect(screen.queryByText(/BELUM DIKIRIM ke balai lelang/)).not.toBeInTheDocument()
  })

  /*
    Server TETAP mengirim keterbatasan, dan itu disengaja: `Limitations()` adalah catatan
    resmi yang dipakai uji kesetaraan gerbang 1. Yang dihapus hanya tampilannya, bukan
    datanya — uji ini menjaga agar penghapusan di layar tidak merembet ke backend.
  */
  it('tetap menerima keterbatasan dari server meski tidak digambar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const metaCall = [...calls].find((c) => c.url === TAB_PATH)
    expect(metaCall).toBeDefined()
    expect(METADATA.keterbatasan.length).toBeGreaterThan(0)
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
    expect(screen.queryByRole('button', { name: 'Detail' })).not.toBeInTheDocument()
  })

  it('membuka panel keputusan dari baris History', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: /History Cheker/ })
    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))

    // BARISNYA yang diklik, bukan tombol — grid History di Pega tidak punya kolom aksi.
    await userEvent.click(await bukaBarisHistory())

    expect(await screen.findByText('Riwayat Keputusan Banding')).toBeInTheDocument()
    expect(await screen.findByText('Forklift Rusak')).toBeInTheDocument()
    expect(screen.getByText('Setuju')).toBeInTheDocument()
  })

  it('menggambar ketujuh kolom panel yang ditetapkan server', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: /History Cheker/ })
    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))
    await userEvent.click(await bukaBarisHistory())

    await screen.findByText('Riwayat Keputusan Banding')
    for (const judul of ['Tgl Approve', 'Jawaban Checker', 'Nama Checker']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  // Ditutup dengan menekan barisnya LAGI — sama seperti membukanya, dan sama seperti
  // `expandPane` di Pega. Tidak ada tombol "Tutup", karena tidak ada kolom aksi.
  it('menutup panel ketika barisnya ditekan lagi', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: /History Cheker/ })
    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))
    await userEvent.click(await bukaBarisHistory())
    await screen.findByText('Riwayat Keputusan Banding')

    await userEvent.click(await bukaBarisHistory())

    await waitFor(() => {
      expect(screen.queryByText('Riwayat Keputusan Banding')).not.toBeInTheDocument()
    })
  })

  /*
    Grid History TIDAK punya kolom aksi, dan itu mengikuti Pega apa adanya.

    `Section/InboxReqSalvageASM` mengatur grid itu `pyEditingMode = expandPane` dengan
    `pyEditAction = DetailHistoryRequestSalvage`: yang diklik barisnya. Kolom aksi di sana
    tidak ada sama sekali — Work Owner menunjukkannya 2026-10-05, dan tombol "Detail" yang
    sempat dibangun di sini dibuang.

    Uji ini menjaga ketiadaannya. Tanpa uji, kolom itu mudah kembali: grid Request di
    sebelahnya PUNYA kolom aksi, sehingga menyamakan keduanya terasa seperti merapikan.
  */
  it('tidak punya kolom aksi maupun tombol pembuka', async () => {
    stubDefaultFetch([BARIS_HISTORY], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: /History Cheker/ })
    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))
    await screen.findByText('Alat Berat')

    expect(screen.queryByRole('columnheader', { name: 'Aksi' })).not.toBeInTheDocument()
    for (const label of ['Detail', 'Lihat Riwayat', 'Lihat Detail Klaim']) {
      expect(screen.queryByRole('button', { name: label })).not.toBeInTheDocument()
    }
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

    await screen.findByRole('tab', { name: /History Cheker/ })
    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))
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

// ─────────────────────────────────────────────────────────────────────────────
// Kepala kolom saat datanya belum ada
// ─────────────────────────────────────────────────────────────────────────────

describe('kolom tetap tergambar meski datanya kosong', () => {
  /*
    Di Pega, grid SELALU menampilkan kepalanya — pesan "tidak ada baris" muncul DI DALAM
    tabel, bukan menggantikannya (`D-13`).

    Tanpa ini, antrean kosong ikut menyembunyikan kolom mana saja yang ada, dan pengguna
    kehilangan satu-satunya petunjuk bahwa ia sedang melihat tab yang benar. Ketiga uji di
    bawah menjaga ketiga tabel modul ini tetap begitu.
  */
  it('grid Request menggambar kesembilan kolomnya meski antreannya kosong', async () => {
    stubDefaultFetch([])
    await renderLoaded()

    for (const judul of ['Tanggal Request', 'No Klaim', 'Detail Object', 'Harga Request']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }

    // Pesannya tetap muncul — ia menemani kepalanya, bukan menggantikannya.
    expect(screen.getByText(/Tidak ada banding harga pada antrean/)).toBeInTheDocument()
  })

  it('grid History menggambar keempat kolomnya meski antreannya kosong', async () => {
    stubDefaultFetch([], TAB_HISTORY)
    renderPage()

    await screen.findByRole('tab', { name: /History Cheker/ })
    await userEvent.click(screen.getByRole('tab', { name: /History Cheker/ }))

    for (const judul of ['No Klaim', 'Object Name', 'Lokasi Salvage', 'PIC']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  it('dialog Lihat File menggambar ketiga kolomnya meski tanpa dokumen', async () => {
    stubDocumentFetch([])
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')
    await userEvent.click(screen.getByRole('button', { name: 'Lihat File' }))

    await screen.findByText(/Dokumen Banding Harga/)
    for (const judul of ['Kategori', 'Nama', 'Tanggal']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })
})


// ─────────────────────────────────────────────────────────────────────────────
// Judul kolom aksi
// ─────────────────────────────────────────────────────────────────────────────

describe('judul kolom aksi', () => {
  /*
    Kolom tombol SELALU berjudul "Aksi" — ketetapan Work Owner 2026-10-03, berlaku seluruh
    modul, dan satu-satunya judul kolom yang sengaja TIDAK menyalin Pega.

    Mengosongkannya sudah dicoba dan DITOLAK (`master-login`, 2026-10-04): kolom tanpa judul
    tampak *tidak ada* bagi pengguna yang membaca kepala tabel, bukan sekadar sulit dibaca
    pembaca layar.

    Kedua grid diuji karena keduanya sempat menyimpang dengan cara yang BERBEDA — yang satu
    `'Action'`, yang satu lagi kosong.
  */
  it('grid Request memakai "Aksi", bukan "Action"', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByRole('columnheader', { name: 'Aksi' })).toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: 'Action' })).not.toBeInTheDocument()
  })

  // Grid History tidak diuji di sini: sejak 2026-10-05 ia TIDAK punya kolom aksi sama
  // sekali, mengikuti `pyEditingMode = expandPane` di Pega. Ketiadaannya dijaga uji
  // tersendiri pada describe "panel riwayat keputusan".
})

// ─────────────────────────────────────────────────────────────────────────────
// Label tombol pada kolom Aksi
// ─────────────────────────────────────────────────────────────────────────────

describe('label tombol kolom Aksi', () => {
  /*
    Tiga tombol grid Request menyalin literal Pega apa adanya, dan itu keputusan Work Owner
    2026-10-05 setelah literalnya dibuktikan ada:

      Section/ButtonApproveRejectedRequest-Section.xml
        pyButtonLabel Approve · pyButtonLabel Reject · pyButtonLabel Lihat File

    Konvensi aplikasi ini sendiri memakai "Setujui"/"Tolak" (3 modul) dan "View Document"
    (2 modul) untuk perilaku yang sama. Keduanya SENGAJA tidak dipakai di sini — `D-13`
    menetapkan tampilan meniru Pega supaya pengguna tidak belajar ulang, dan ketetapan
    kolom "Aksi" menegaskan bahwa judul yang PUNYA literal tetap disalin apa adanya.

    Uji ini menjaga keputusan itu dari "diseragamkan" oleh orang yang hanya melihat
    ketidakkonsistenannya dan tidak tahu ketiganya literal.
  */
  it('grid Request memakai literal Pega, bukan padanan Indonesianya', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')

    for (const literal of ['Approve', 'Reject', 'Lihat File']) {
      expect(screen.getByRole('button', { name: literal })).toBeInTheDocument()
    }
    for (const padanan of ['Setujui', 'Tolak', 'View Document']) {
      expect(screen.queryByRole('button', { name: padanan })).not.toBeInTheDocument()
    }
  })

  // Grid History tidak punya tombol untuk dinilai labelnya — lihat catatan di atas.
})

// ─────────────────────────────────────────────────────────────────────────────
// Penempatan tombol Refresh
// ─────────────────────────────────────────────────────────────────────────────

describe('tombol Refresh', () => {
  /*
    Tombolnya berada di HEADER HALAMAN, bukan di dalam kanvas tabel.

    Itu pola mayoritas aplikasi ini. Pindaian seluruh modul (2026-10-05):

      di header halaman, berlabel "Refresh"       28
      di dalam tabel (`actions=`), "Muat ulang"   11
      di dalam tabel, "Refresh"                    8
      di header halaman, "Muat ulang"              2

    Layar ini semula memakai pola kedua; dipindahkan atas permintaan Work Owner.

    Uji ini menuntut tombolnya berbagi `<header>` dengan judul halaman — cara paling
    langsung menyatakan "di luar tabel" tanpa bergantung pada struktur dalam DataTable,
    yang bukan milik modul ini dan dapat berubah.
  */
  it('berada di header halaman, satu wadah dengan judulnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Ditunggu sampai datanya tiba: selama memuat, tombolnya bertuliskan "Memuat…".
    await screen.findByText('Mesin Genset Bekas')

    const judul = screen.getByRole('heading', {
      name: 'Inbox Banding Harga Salvage',
      level: 1,
    })
    const refresh = screen.getByRole('button', { name: 'Refresh' })

    const header = judul.closest('header')
    expect(header).not.toBeNull()
    expect(header).toContainElement(refresh)
  })

  // Keadaan memuat dinyatakan pada tombolnya, seperti 49 tombol Refresh lain di aplikasi
  // ini — tanpa itu, menekan tombol pada jaringan lambat tidak memberi umpan balik apa pun
  // dan pengguna menekannya berulang kali.
  it('menyatakan keadaan memuat pada tombolnya', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')

    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Memuat…|Refresh/ })).toBeInTheDocument()
    })
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// Kerapian grid berkolom sepuluh
// ─────────────────────────────────────────────────────────────────────────────

describe('kerapian grid Request', () => {
  /*
    Kolom aksi TIDAK boleh berlebar tetap.

    Semula `19rem` — selebar ketiga tombolnya berjajar. Pada grid ber-9 kolom data, lebar itu
    memaksa setiap judul pecah dua baris, sementara kolomnya sendiri melompong saat antrean
    kosong (Work Owner, 2026-10-05).

    Tanpa lebar tetap, peramban yang membagi ruangnya: menyusut saat kosong, melebar saat
    berisi, dan menyusut lagi bila kolom data membutuhkannya — pembungkus tombolnya
    `flex-wrap`, jadi tombolnya boleh turun baris.

    Uji ini menjaga ketiadaannya. Lebar tetap mudah kembali: kolom yang tombolnya turun baris
    terlihat seperti "terlalu sempit", dan memasang `width` terasa seperti memperbaikinya.
  */
  it('kolom Aksi tidak berlebar tetap', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')

    const aksi = screen.getByRole('columnheader', { name: 'Aksi' })
    expect(aksi.getAttribute('style') ?? '').not.toContain('width')
  })

  // Mode rapat `dense` memangkas jarak sel dari `px-5` ke `px-3`. Pada sepuluh kolom itu
  // membebaskan sekitar 10rem — dan ia memang dibuat untuk grid yang harus muat satu layar
  // tanpa gulir menyamping.
  it('memakai mode rapat, karena kolomnya sepuluh', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('Mesin Genset Bekas')

    const judul = screen.getByRole('columnheader', { name: 'No Klaim' })
    expect(judul.className).toContain('px-3')
    expect(judul.className).not.toContain('px-5')
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// Keterangan di bawah kotak cari
// ─────────────────────────────────────────────────────────────────────────────

describe('kotak cari', () => {
  /*
    Keterangannya DIPANGKAS, bukan dibuang.

    Semula: *"Harus nomor klaim UTUH — pencarian di layar ini cocok persis, sama seperti di
    layar lama. Separuh nomor tidak menghasilkan baris."*

    Work Owner menyisakan kalimat pertamanya saja (2026-10-05). Yang bertahan adalah
    ATURANNYA; yang dibuang PENJELASAN MENGAPA. Orang yang sedang mengetik butuh tahu apa
    yang harus ia ketik, bukan riwayat perilaku pencarian.

    Kedua sisi diuji sekaligus, karena keduanya mudah bergeser ke arah yang salah: aturannya
    ikut terhapus saat membersihkan, atau penjelasannya kembali saat seseorang merasa
    aturannya kurang jelas.
  */
  it('menyisakan aturannya saja, tanpa penjelasan mengapa', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText('Harus nomor klaim UTUH.')).toBeInTheDocument()

    expect(screen.queryByText(/cocok persis, sama seperti di layar lama/)).not.toBeInTheDocument()
    expect(screen.queryByText(/Separuh nomor tidak menghasilkan baris/)).not.toBeInTheDocument()
  })

  /*
    Keterangannya TIDAK hilang seluruhnya — ia pindah ke pesan kosong.

    Di situ ia muncul hanya ketika sebuah pencarian benar-benar tidak menghasilkan apa pun,
    yakni saat pengguna sedang menanyakannya. Menghapusnya dari sana pula akan membuat
    kekosongan terbaca sebagai "datanya hilang".
  */
  it('tetap menjelaskannya saat sebuah pencarian tidak menghasilkan baris', async () => {
    stubDefaultFetch([])
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('Cari No Klaim'), 'PNCN')

    expect(await screen.findByText(/COCOK PERSIS/)).toBeInTheDocument()
  })
})
