import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxManagerPage } from './InboxManagerPage'
import type { Counter, ListResponse, MetadataResponse, Tab } from './types'

/**
 * Uji tambahan Inbox Manager: cabang galat, penyaring periode, ekspor, dan bentuk sel.
 *
 * Layarnya digambar langsung (tanpa AppRoute) supaya yang diuji hanya layar ini.
 * Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-manager'

const SAMPLE_PROFILE = {
  identitas: '90000058',
  nama: 'Contoh Penyelia',
  jenis: 'KARYAWAN',
  login: 'penyelia.manager',
  email: 'contoh.penyelia@example.invalid',
  perusahaan: 'ASM',
}

/** Dashboard yang PUNYA penyaring periode — kode 2 dibandingkan dengan tahun lalu. */
const TAB_PRODUKTIVITAS: Tab = {
  kode: '2',
  nama: 'Produktivitas Klaim',
  keterangan: 'Produktivitas per periode.',
  jenis: 'dashboard',
  panel: [],
  keputusan: { dapat_diputuskan: false, dapat_massal: false, alasan_wajib_saat_menolak: false },
  punya_penyaring_periode: true,
  label_terapkan_periode: 'Cari',
}

/**
 * Dashboard lain yang punya penyaring periode, tanpa perbandingan tahun lalu.
 *
 * Label tombolnya BERBEDA, dan itu memang begitu di Pega: `pyButtonLabel Cari` pada
 * Produktivitas, `pyButtonLabel Lihat Data` pada Klaim.
 */
const TAB_KLAIM: Tab = {
  ...TAB_PRODUKTIVITAS,
  kode: '3',
  nama: 'Klaim',
  label_terapkan_periode: 'Lihat Data',

  // Bentuk grid ikut dikirim keterangan layar, dan itulah yang menggambar kepala kolom
  // ketika isinya belum ada — termasuk saat permintaannya ditolak.
  panel: [
    {
      kunci: 'bisnis',
      judul: 'Total Klaim Bisnis',
      kolom: [
        { kunci: 'nama_bisnis', judul: 'COB' },
        { kunci: 'total_klaim', judul: 'Total Klaim' },
      ],
    },
  ],
}

const TAB_RINGKASAN: Tab = {
  kode: '4',
  nama: 'Approval Master',
  keterangan: 'Ringkasan.',
  jenis: 'ringkasan',
  keputusan: { dapat_diputuskan: false, dapat_massal: false, alasan_wajib_saat_menolak: false },
  punya_penyaring_periode: false,
}

/** Antrean dengan alasan yang TIDAK wajib saat menolak. */
const TAB_ANTREAN: Tab = {
  kode: '5',
  nama: 'Master Bengkel',
  keterangan: 'Pengajuan bengkel.',
  jenis: 'antrean',
  induk: '4',
  dalam_bilah_induk: true,
  kolom: [
    { kunci: 'id', judul: 'ID Bengkel' },
    { kunci: 'nama', judul: 'Nama Bengkel' },
  ],
  keputusan: {
    dapat_diputuskan: true,
    dapat_massal: true,
    alasan_wajib_saat_menolak: false,
    label_alasan: 'Catatan',
  },
  punya_penyaring_periode: false,
}

/** Antrean baca-saja: tanpa kolom, tanpa keputusan. */
const TAB_BACA: Tab = {
  kode: '9',
  nama: 'Antrean Baca',
  keterangan: 'Hanya dilihat.',
  jenis: 'antrean',
  induk: '4',
  dalam_bilah_induk: true,
  keputusan: { dapat_diputuskan: false, dapat_massal: false, alasan_wajib_saat_menolak: false },
  punya_penyaring_periode: false,
}

/** Antrean yang dapat diputuskan TANPA kolom alasan. */
const TAB_TANPA_ALASAN: Tab = {
  kode: '10',
  nama: 'Antrean Tanpa Alasan',
  keterangan: 'Tanpa alasan.',
  jenis: 'antrean',
  induk: '4',
  dalam_bilah_induk: true,
  kolom: [{ kunci: 'id', judul: 'ID' }],
  keputusan: { dapat_diputuskan: true, dapat_massal: true, alasan_wajib_saat_menolak: false },
  punya_penyaring_periode: false,
}

function metadata(tabs: Tab[], bawaan: string): MetadataResponse {
  return { tab: tabs, tab_bawaan: bawaan, lini_bisnis_anda: 'NONMBU' }
}

const ROWS = [
  { kunci: 'BGK-001', sel: { id: 'BGK-001', nama: 'Bengkel Contoh Satu' } },
  { kunci: 'BGK-002', sel: { id: 'BGK-002', nama: 'Bengkel Contoh Dua' } },
]

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown, headers?: Record<string, string>): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  })
}

type Answer = (url: string, init?: RequestInit) => Response | undefined

/** Peladen tiruan: `answer` boleh mengembalikan undefined untuk memakai jawaban bawaan. */
function stubFetch(options: {
  meta?: MetadataResponse
  counters?: Counter[]
  list?: (tab: string, params: URLSearchParams) => ListResponse
  answer?: Answer
}) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    const custom = options.answer?.(url, init)
    if (custom) return Promise.resolve(custom)
    if (url === `${PATH}/tab`) return Promise.resolve(jsonResponse(200, options.meta))
    if (url === `${PATH}/ringkasan`) {
      return Promise.resolve(jsonResponse(200, { pencacah: options.counters ?? [] }))
    }
    const params = new URL(url, 'https://uji.invalid').searchParams
    const tab = params.get('tab') ?? ''
    return Promise.resolve(jsonResponse(200, options.list?.(tab, params)))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <InboxManagerPage />
    </QueryClientProvider>,
  )
}

function listCalls(): URLSearchParams[] {
  return calls
    .filter((call) => call.url.startsWith(`${PATH}?`) || call.url === PATH)
    .map((call) => new URL(call.url, 'https://uji.invalid').searchParams)
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
  vi.restoreAllMocks()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('cabang galat', () => {
  it('menampilkan galat bentuk layar beserta pesan server dan mematikan Export', async () => {
    stubFetch({
      answer: (url) =>
        url === `${PATH}/tab`
          ? jsonResponse(500, { kode: 'galat_internal', pesan: 'Basis data sedang sibuk.' })
          : undefined,
    })
    renderPage()

    expect(await screen.findByText('Layar tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Basis data sedang sibuk.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export' })).toBeDisabled()
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
  })

  // Pesan kosong dari server jatuh ke pesan cadangan yang ditulis layar.
  it('memakai pesan cadangan bila server tidak menyertakan pesan', async () => {
    stubFetch({
      answer: (url) =>
        url === `${PATH}/tab` ? jsonResponse(500, { kode: 'galat_internal', pesan: '' }) : undefined,
    })
    renderPage()

    expect(await screen.findByText('Keterangan layar tidak dapat diambil.')).toBeInTheDocument()
  })

  /*
    Antrean yang SUMBERNYA belum ada digambar KOSONG, bukan sebagai layar rusak.

    Server tetap menjawab 503 ber-kode `sumber_antrean_tidak_terbaca` dan tetap mencatat nama
    objeknya di log — yang berubah hanya apa yang digambar. Pesannya ditujukan ke tim teknis,
    dan sebabnya keadaan pengembangan: tabelnya memang belum dibuat (Work Owner, 2026-10-08).
  */
  it('menggambar antrean KOSONG saat sumbernya belum ada, tanpa pesan galat', async () => {
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      answer: (url) =>
        url.startsWith(`${PATH}?`) || url === PATH
          ? jsonResponse(503, {
              kode: 'sumber_antrean_tidak_terbaca',
              pesan:
                'Antrean itu belum dapat ditampilkan: objek sumbernya di basis data sedang ' +
                'tidak dapat dibaca.',
            })
          : undefined,
    })
    renderPage()

    expect(
      await screen.findByText('Tidak ada pengajuan yang menunggu persetujuan di antrean ini.'),
    ).toBeInTheDocument()

    expect(screen.queryByText(/tidak dapat dimuat/)).not.toBeInTheDocument()
    expect(screen.queryByText(/objek sumbernya/)).not.toBeInTheDocument()

    // Kepala kolomnya TETAP digambar — bentuknya diketahui dari keterangan layar.
    expect(screen.getByText('ID Bengkel')).toBeInTheDocument()
  })

  it('menampilkan galat isi tab dengan nama tabnya', async () => {
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      answer: (url) =>
        url.startsWith(`${PATH}?`)
          ? jsonResponse(500, { kode: 'galat_internal', pesan: 'Antrean tidak terbaca.' })
          : undefined,
    })
    renderPage()

    expect(await screen.findByText('"Master Bengkel" tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Antrean tidak terbaca.')).toBeInTheDocument()
  })

  it('menampilkan galat keputusan dari server', async () => {
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: ROWS }),
      answer: (url) =>
        url === `${PATH}/keputusan`
          ? jsonResponse(422, { kode: 'validasi', pesan: 'Alasan wajib diisi saat menolak.' })
          : undefined,
    })
    renderPage()

    await userEvent.click(await screen.findByLabelText('Pilih baris BGK-001'))
    await userEvent.click(screen.getByRole('button', { name: 'Tolak' }))

    expect(await screen.findByText('Keputusan tidak dapat disimpan')).toBeInTheDocument()
    expect(screen.getByText('Alasan wajib diisi saat menolak.')).toBeInTheDocument()
  })
})

describe('antrean — pemilihan dan keputusan', () => {
  it('mengirim keputusan tolak beserta alasan yang diketik', async () => {
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: ROWS }),
      answer: (url) =>
        url === `${PATH}/keputusan`
          ? jsonResponse(200, { diminta: 1, berubah: 1, tidak_berubah: 0, pesan: 'Ditolak.' })
          : undefined,
    })
    renderPage()

    // Alasan tidak wajib pada antrean ini, sehingga keterangan "(wajib …)" tidak tampil.
    expect(await screen.findByText('Catatan')).toBeInTheDocument()
    expect(screen.queryByText('(wajib saat menolak)')).not.toBeInTheDocument()

    await userEvent.click(await screen.findByLabelText('Pilih baris BGK-002'))
    await userEvent.type(screen.getByRole('textbox'), 'Data tidak lengkap')
    await userEvent.click(screen.getByRole('button', { name: 'Tolak' }))

    expect(await screen.findByRole('status')).toHaveTextContent('Ditolak.')
    const sent = calls.find((call) => call.url === `${PATH}/keputusan`)
    expect(sent?.init?.method).toBe('POST')
    expect(JSON.parse(String(sent?.init?.body))).toEqual({
      tab: '5',
      keputusan: 'tolak',
      kunci: ['BGK-002'],
      alasan: 'Data tidak lengkap',
    })
  })

  it('memilih semua lalu membatalkan pilihan semua dan satu per satu', async () => {
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: ROWS }),
    })
    renderPage()

    const all = await screen.findByLabelText('Pilih semua baris di halaman ini')
    await userEvent.click(all)
    expect(screen.getByText('2 baris dipilih')).toBeInTheDocument()
    expect(all).toBeChecked()

    await userEvent.click(all)
    expect(screen.getByText('0 baris dipilih')).toBeInTheDocument()

    const one = screen.getByLabelText('Pilih baris BGK-001')
    await userEvent.click(one)
    expect(screen.getByText('1 baris dipilih')).toBeInTheDocument()
    await userEvent.click(one)
    expect(screen.getByText('0 baris dipilih')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Setujui' })).toBeDisabled()
  })

  /*
    Enam dari sembilan antrean TIDAK punya jalur massal di Pega: sectionnya tidak menggambar
    `Select All`, dan tombol keputusannya memanggil activity yang menyentuh satu baris. Layar
    karena itu tidak boleh menawarkan pilih-semua di sana, dan memilih baris kedua harus
    MENGGANTI pilihan pertama — bukan menumpuknya menjadi permintaan yang pasti ditolak
    server.
  */
  it('tidak menawarkan pilih semua, dan memilih satu baris saja, pada antrean tanpa jalur massal', async () => {
    const satuan: Tab = {
      ...TAB_ANTREAN,
      keputusan: { ...TAB_ANTREAN.keputusan, dapat_massal: false },
    }

    stubFetch({
      meta: metadata([satuan], '5'),
      list: () => ({ tab: satuan, baris: ROWS }),
    })
    renderPage()

    expect(await screen.findByLabelText('Pilih baris BGK-001')).toBeInTheDocument()
    expect(screen.queryByLabelText('Pilih semua baris di halaman ini')).not.toBeInTheDocument()

    await userEvent.click(screen.getByLabelText('Pilih baris BGK-001'))
    await userEvent.click(screen.getByLabelText('Pilih baris BGK-002'))

    expect(screen.getByLabelText('Pilih baris BGK-001')).not.toBeChecked()
    expect(screen.getByLabelText('Pilih baris BGK-002')).toBeChecked()
  })

  /*
    Label tombol disalin dari Pega, termasuk huruf besarnya yang memang tidak seragam di
    sana. Yang tidak punya literal Pega jatuh ke kata aplikasi — lihat tes-tes lain yang masih
    mencari "Setujui".
  */
  it('memakai label tombol Pega bila antreannya punya', async () => {
    const rangka: Tab = {
      ...TAB_ANTREAN,
      keputusan: {
        ...TAB_ANTREAN.keputusan,
        dapat_massal: false,
        label_setujui: 'Setuju',
        label_tolak: 'Tidak Setuju',
      },
    }

    stubFetch({
      meta: metadata([rangka], '5'),
      list: () => ({ tab: rangka, baris: ROWS }),
    })
    renderPage()

    expect(await screen.findByRole('button', { name: 'Setuju' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tidak Setuju' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Setujui' })).not.toBeInTheDocument()
  })

  /*
    Tab yang di Pega isinya bilah sub-tab menggambar judul sub-tabnya.

    `Sec_PaymentAkseptasiKlaimCase1` memuat dua `<pyTitle>`, tetapi yang kedua bersyarat
    `1==2` — tidak pernah tampil. Yang digambar karena itu satu.
  */
  it('menggambar judul sub-tab pada tab yang punya, dan tidak pada yang lain', async () => {
    const payment: Tab = { ...TAB_ANTREAN, judul_sub_tab: 'Approval Payment Akseptasi' }
    stubFetch({
      meta: metadata([payment], '5'),
      list: () => ({ tab: payment, baris: ROWS }),
    })
    const { unmount } = renderPage()

    expect(
      await screen.findByRole('tab', { name: 'Approval Payment Akseptasi' }),
    ).toHaveAttribute('aria-selected', 'true')
    unmount()

    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: ROWS }),
    })
    renderPage()

    expect(await screen.findByLabelText('Pilih baris BGK-001')).toBeInTheDocument()
    expect(
      screen.queryByRole('tab', { name: 'Approval Payment Akseptasi' }),
    ).not.toBeInTheDocument()
  })

  it('mematikan pilih semua pada antrean kosong', async () => {
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: [] }),
    })
    renderPage()

    expect(
      await screen.findByText('Tidak ada pengajuan yang menunggu persetujuan di antrean ini.'),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Pilih semua baris di halaman ini')).toBeDisabled()
  })

  it('tidak menggambar bilah keputusan pada antrean baca-saja', async () => {
    stubFetch({
      meta: metadata([TAB_BACA], '9'),
      list: () => ({ tab: TAB_BACA }),
    })
    renderPage()

    expect(
      await screen.findByText('Tidak ada pengajuan yang menunggu persetujuan di antrean ini.'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Setujui' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Pilih semua baris di halaman ini')).not.toBeInTheDocument()
  })

  it('tidak menggambar isian alasan pada antrean tanpa kolom alasan', async () => {
    stubFetch({
      meta: metadata([TAB_TANPA_ALASAN], '10'),
      list: () => ({ tab: TAB_TANPA_ALASAN, baris: [{ kunci: 'X-1', sel: {} }] }),
    })
    renderPage()

    expect(await screen.findByLabelText('Pilih baris X-1')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Setujui' })).toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  })

  // Pilihan dibuang saat halaman berganti supaya kunci halaman 1 tidak ikut terkirim.
  it('meminta halaman berikutnya dan membuang pilihan halaman sebelumnya', async () => {
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: (_tab, params) => ({
        tab: TAB_ANTREAN,
        baris:
          params.get('halaman') === '2'
            ? [{ kunci: 'BGK-026', sel: { id: 'BGK-026', nama: 'Bengkel Contoh Halaman Dua' } }]
            : ROWS,
        paginasi: {
          halaman: Number(params.get('halaman') ?? '1'),
          ukuran: 2,
          total: 3,
          total_halaman: 2,
        },
      }),
    })
    renderPage()

    await userEvent.click(await screen.findByLabelText('Pilih baris BGK-001'))
    expect(screen.getByText('1 baris dipilih')).toBeInTheDocument()

    await userEvent.click(screen.getAllByRole('button', { name: 'Halaman berikutnya' })[0]!)

    expect(await screen.findByText('Bengkel Contoh Halaman Dua')).toBeInTheDocument()
    expect(screen.getByText('0 baris dipilih')).toBeInTheDocument()
    expect(listCalls().some((params) => params.get('halaman') === '2')).toBe(true)
  })
})

describe('bilah tab', () => {
  it('TIDAK menggambar angka pencacah pada tab mana pun', async () => {
    /*
      Pega menghitung antreannya, tetapi angkanya ditampilkan sebagai tabel `Status`/`Jumlah`
      tersendiri — bukan sebagai lencana pada tombol tabnya. Lencana itu dicabut atas
      permintaan Work Owner (2026-10-08).

      Keterangan sumber yang tidak terbaca TETAP ada, sebagai tooltip, karena ia bukan angka.
    */
    const extra: Tab = { ...TAB_ANTREAN, kode: '6', nama: 'Master Panel' }
    stubFetch({
      meta: metadata([TAB_RINGKASAN, TAB_ANTREAN, extra, TAB_BACA], '5'),
      counters: [
        { tab: '5', label: 'Master Bengkel', jumlah: 7, induk: '4' },
        { tab: '6', label: 'Master Panel', jumlah: 0, induk: '4' },
        {
          tab: '9',
          label: 'Antrean Baca',
          jumlah: 0,
          induk: '4',
          tidak_tersedia: 'Sumbernya tidak dapat dibaca.',
        },
      ],
      list: (tab) => ({ tab: tab === '9' ? TAB_BACA : TAB_ANTREAN, baris: [] }),
    })
    renderPage()

    const selected = await screen.findByRole('tab', { name: /Master Bengkel/ })
    expect(within(selected).queryByText('7')).not.toBeInTheDocument()
    expect(
      within(screen.getByRole('tab', { name: /Master Panel/ })).queryByText('0'),
    ).not.toBeInTheDocument()

    // Keterangan "sumber tidak terbaca" juga tidak lagi menempel sebagai tooltip: ia pesan
    // untuk tim teknis, bukan untuk penyelia.
    const unreadable = screen.getByRole('tab', { name: /Antrean Baca/ })
    expect(unreadable).toHaveAttribute('title', TAB_BACA.keterangan)
    expect(
      within(unreadable).queryByLabelText('sumber antrean ini sedang tidak dapat dibaca'),
    ).not.toBeInTheDocument()
  })
})

describe('dashboard dengan penyaring periode', () => {
  /*
    Galat pemuatan TIDAK boleh menghapus penyaring periodenya.

    Galat yang paling sering muncul di tab berperiode justru BERASAL dari isian periode —
    rentang lintas tahun yang ditolak tab Klaim, meniru `Activity/DashboardKlaim_act`.
    Mengganti seluruh badan tab membuat pesan "perbaiki yang ditandai" muncul tanpa ada yang
    dapat diperbaiki.
  */
  it('tetap menggambar penyaring periode DAN gridnya saat pemuatan GAGAL', async () => {
    // Tab Klaim — tab yang memang menolak rentang lintas tahun, dan yang bentuk gridnya
    // ikut dikirim keterangan layar.
    stubFetch({
      meta: metadata([TAB_KLAIM], '3'),
      counters: [],
      answer: (url) =>
        url.startsWith(`${PATH}?`) || url === PATH
          ? // Bentuk jawaban galat validasi yang SEBENARNYA: pesan amplopnya umum, dan yang
            // menyebut sebabnya ada di `detail`.
            jsonResponse(422, {
              kode: 'permintaan_tidak_valid',
              pesan: 'Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.',
              detail: [
                {
                  field: 'periode',
                  pesan: 'Periode Up To hanya untuk periode tahun yang sama',
                },
              ],
            })
          : undefined,
    })
    renderPage()

    expect(await screen.findByText(/tidak dapat dimuat/)).toBeInTheDocument()

    // Pesan Pega-lah yang tampil, BUKAN pesan amplop yang tidak menyebut apa pun.
    expect(
      screen.getByText('Periode Up To hanya untuk periode tahun yang sama'),
    ).toBeInTheDocument()
    expect(screen.queryByText(/Perbaiki yang ditandai lalu coba lagi/)).not.toBeInTheDocument()

    // Isiannya tetap tergambar supaya tanggalnya dapat dibetulkan di tempat.
    expect(screen.getByLabelText('Periode')).toBeInTheDocument()

    // Dan gridnya TIDAK dihapus — Pega menampilkan alert tanpa membuang tabelnya.
    expect(screen.getByRole('columnheader', { name: 'COB' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Total Klaim' })).toBeInTheDocument()
  })


  it('menyatakan tanpa penyaring lalu mengirim bulan yang dipilih', async () => {
    stubFetch({
      meta: metadata([TAB_PRODUKTIVITAS], '2'),
      list: (_tab, params) => ({
        tab: TAB_PRODUKTIVITAS,
        panel: [],
        ...(params.get('bulan')
          ? { periode: { dari: '2026-09-01', sampai: '2026-09-30' } }
          : {}),
      }),
    })
    renderPage()

    expect(
      await screen.findByText('Tanpa penyaring periode: angka di bawah mencakup seluruh periode.'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Hapus penyaring' })).not.toBeInTheDocument()

    const month = screen.getByLabelText('Bulan & Tahun')
    await userEvent.type(month, '2026-09')

    // Mengisi saja TIDAK memuat ulang — di Pega periode berlaku saat tombolnya ditekan.
    expect(listCalls().some((params) => params.get('bulan') === '2026-09')).toBe(false)
    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

    expect(await screen.findByText('2026-09-01 sampai 2026-09-30')).toBeInTheDocument()
    expect(
      screen.getByText(/dibandingkan dengan rentang yang sama tahun lalu/),
    ).toBeInTheDocument()
    const sent = listCalls().find((params) => params.get('bulan') === '2026-09')
    expect(sent?.get('bentuk_periode')).toBe('bulan')
    expect(sent?.get('tab')).toBe('2')
  })

  it('mengirim rentang tanggal lalu menghapus penyaringnya', async () => {
    stubFetch({
      meta: metadata([TAB_KLAIM], '3'),
      list: (_tab, params) => ({
        tab: TAB_KLAIM,
        panel: [],
        ...(params.get('sampai')
          ? { periode: { dari: params.get('dari') ?? '', sampai: params.get('sampai') ?? '' } }
          : {}),
      }),
    })
    renderPage()

    const mode = await screen.findByLabelText('Periode')
    await userEvent.selectOptions(mode, 'rentang')
    expect(screen.queryByLabelText('Bulan & Tahun')).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Dari'), '2026-08-01')
    await userEvent.type(screen.getByLabelText('Sampai'), '2026-08-31')
    await userEvent.click(screen.getByRole('button', { name: 'Lihat Data' }))

    // Tab selain kode 2 diakhiri titik, tanpa perbandingan tahun lalu.
    expect(await screen.findByText('2026-08-01 sampai 2026-08-31')).toBeInTheDocument()
    expect(screen.queryByText(/tahun lalu/)).not.toBeInTheDocument()
    const sent = listCalls().find((params) => params.get('sampai') === '2026-08-31')
    expect(sent?.get('bentuk_periode')).toBe('rentang')
    expect(sent?.get('dari')).toBe('2026-08-01')
    expect(sent?.has('bulan')).toBe(false)

    // Penyaring dihapus dengan MENGOSONGKAN isiannya lalu menekan tombol yang sama — tidak
    // ada tombol "Hapus penyaring" di Pega.
    await userEvent.clear(screen.getByLabelText('Dari'))
    await userEvent.clear(screen.getByLabelText('Sampai'))
    await userEvent.click(screen.getByRole('button', { name: 'Lihat Data' }))

    expect(
      await screen.findByText('Tanpa penyaring periode: angka di bawah mencakup seluruh periode.'),
    ).toBeInTheDocument()
    // Penyaring dihapus mengembalikan bentuk bawaan Bulan & Tahun.
    expect(screen.getByLabelText('Bulan & Tahun')).toBeInTheDocument()
  })

  /*
    Uji inti laporan Work Owner: mengetik tahun pada isian tanggal melewati keadaan setengah
    jadi ("0002", lalu "0020"), dan sebelumnya setiap keadaan itu menjadi permintaan
    tersendiri yang pasti ditolak server.
  */
  it('TIDAK mengirim permintaan apa pun selama tanggal masih diketik', async () => {
    stubFetch({
      meta: metadata([TAB_KLAIM], '3'),
      list: () => ({ tab: TAB_KLAIM, panel: [] }),
    })
    renderPage()

    await userEvent.selectOptions(await screen.findByLabelText('Periode'), 'rentang')
    const sebelum = listCalls().length

    await userEvent.type(screen.getByLabelText('Dari'), '2026-01-01')
    expect(listCalls()).toHaveLength(sebelum)

    // Rentang yang baru separuh terisi tidak dapat diterapkan sama sekali.
    expect(screen.getByRole('button', { name: 'Lihat Data' })).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Sampai'), '2026-12-31')
    expect(screen.getByRole('button', { name: 'Lihat Data' })).toBeEnabled()
    expect(listCalls()).toHaveLength(sebelum)
  })

  // Berganti bentuk periode saja TIDAK memuat ulang — ia bagian dari draf, sama seperti
  // isian tanggalnya.
  it('tidak memuat ulang hanya karena bentuk periode diganti', async () => {
    stubFetch({
      meta: metadata([TAB_KLAIM], '3'),
      list: () => ({ tab: TAB_KLAIM, panel: [] }),
    })
    renderPage()

    const mode = await screen.findByLabelText('Periode')
    const sebelum = listCalls().length
    await userEvent.selectOptions(mode, 'rentang')

    expect(listCalls()).toHaveLength(sebelum)
    expect(listCalls().some((params) => params.get('bentuk_periode') === 'rentang')).toBe(false)
  })
})

describe('panel dashboard', () => {
  const TAB_UANG: Tab = {
    kode: '1',
    nama: 'Outstanding',
    keterangan: 'Klaim berjalan.',
    jenis: 'dashboard',
    keputusan: { dapat_diputuskan: false, dapat_massal: false, alasan_wajib_saat_menolak: false },
    punya_penyaring_periode: false,
  }

  it('memformat nilai uang sebagai teks dan menyusun sel menurut jenisnya', async () => {
    stubFetch({
      meta: metadata([TAB_UANG], '1'),
      list: () => ({
        tab: TAB_UANG,
        disegarkan_pada: '2026-09-28T03:00:00Z',
        panel: [
          {
            kunci: 'nilai',
            judul: 'Nilai per Cabang',
            kolom: [
              { kunci: 'cabang', judul: 'Cabang' },
              { kunci: 'jumlah', judul: 'Jumlah' },
              { kunci: 'nilai', judul: 'Nilai', uang: true },
            ],
            baris: [
              {
                cabang: { teks: 'Contoh Pusat' },
                jumlah: { jumlah: 5 },
                nilai: { nilai: '1234567.50' },
              },
              { cabang: { teks: 'Contoh Barat' }, jumlah: { teks: '' }, nilai: { nilai: '-2500' } },
              { cabang: { teks: 'Contoh Timur' }, nilai: { nilai: 'n/a' } },
              { cabang: { teks: 'Contoh Utara' }, jumlah: {}, nilai: {} },
            ],
          },
        ],
      }),
    })
    renderPage()

    expect(await screen.findByText('1.234.567,50')).toBeInTheDocument()
    expect(screen.getByText('-2.500')).toBeInTheDocument()
    // Nilai yang bukan angka ditampilkan apa adanya, tidak diubah menjadi NaN.
    expect(screen.getByText('n/a')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText(/berasal dari cuplikan yang terakhir disegarkan/)).toBeInTheDocument()
    expect(screen.queryByText('2026-09-28T03:00:00Z')).not.toBeInTheDocument()

    const utara = screen.getByText('Contoh Utara').closest('tr')!
    const cells = within(utara).getAllByRole('cell')
    // Sel tanpa isi digambar DataTable sebagai tanda hubung.
    expect(cells[1]).toHaveTextContent('—')
    expect(cells[2]).toHaveTextContent('—')
    const barat = screen.getByText('Contoh Barat').closest('tr')!
    expect(within(barat).getAllByRole('cell')[1]).toHaveTextContent('—')
  })

  it('menampilkan waktu penyegaran apa adanya bila tidak dapat dibaca sebagai tanggal', async () => {
    stubFetch({
      meta: metadata([TAB_UANG], '1'),
      list: () => ({ tab: TAB_UANG, disegarkan_pada: 'kemarin sore', panel: [] }),
    })
    renderPage()

    expect(await screen.findByText('kemarin sore')).toBeInTheDocument()
  })
})

const originalCreate = URL.createObjectURL
const originalRevoke = URL.revokeObjectURL

/** Menggantikan URL objek peramban; dikembalikan di afterEach blok ini. */
function stubObjectURL(create: (blob: Blob) => string, revoke: (url: string) => void) {
  URL.createObjectURL = create
  URL.revokeObjectURL = revoke
}

describe('ekspor', () => {
  afterEach(() => {
    URL.createObjectURL = originalCreate
    URL.revokeObjectURL = originalRevoke
  })

  it('mengunduh CSV antrean dengan nama dari Content-Disposition', async () => {
    const createURL = vi.fn(() => 'blob:uji')
    const revokeURL = vi.fn()
    stubObjectURL(createURL, revokeURL)
    const clicked: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      clicked.push(this.download)
    })
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: ROWS }),
      answer: (url) =>
        url.startsWith(`${PATH}/ekspor`)
          ? new Response('id,nama\n', {
              status: 200,
              headers: { 'Content-Disposition': 'attachment; filename="bengkel.csv"' },
            })
          : undefined,
    })
    renderPage()

    const button = await screen.findByRole('button', { name: 'Export' })
    await waitFor(() => expect(button).toBeEnabled())
    await userEvent.click(button)

    await waitFor(() => expect(clicked).toEqual(['bengkel.csv']))
    expect(revokeURL).toHaveBeenCalledWith('blob:uji')
    const sent = calls.find((call) => call.url.startsWith(`${PATH}/ekspor`))
    expect(sent?.url).toBe(`${PATH}/ekspor?tab=5`)
    const header = sent?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
    expect(header['X-Portal']).toBe('ASM')
  })

  it('memakai nama bawaan bila server tidak menyebut nama berkas', async () => {
    stubObjectURL(
      vi.fn(() => 'blob:uji'),
      vi.fn(),
    )
    const clicked: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      clicked.push(this.download)
    })
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: ROWS }),
      answer: (url) =>
        url.startsWith(`${PATH}/ekspor`)
          ? new Response('id\n', {
              status: 200,
              headers: { 'Content-Disposition': 'attachment' },
            })
          : undefined,
    })
    renderPage()

    const button = await screen.findByRole('button', { name: 'Export' })
    await waitFor(() => expect(button).toBeEnabled())
    await userEvent.click(button)

    await waitFor(() => expect(clicked).toEqual(['inbox-manager.csv']))
  })

  it('tidak mengunduh apa pun saat server menolak ekspor', async () => {
    const createURL = vi.fn(() => 'blob:uji')
    stubObjectURL(createURL, vi.fn())
    stubFetch({
      meta: metadata([TAB_ANTREAN], '5'),
      list: () => ({ tab: TAB_ANTREAN, baris: ROWS }),
      answer: (url) =>
        url.startsWith(`${PATH}/ekspor`) ? new Response('bukan json', { status: 500 }) : undefined,
    })
    renderPage()

    const button = await screen.findByRole('button', { name: 'Export' })
    await waitFor(() => expect(button).toBeEnabled())
    await userEvent.click(button)

    await waitFor(() =>
      expect(calls.some((call) => call.url.startsWith(`${PATH}/ekspor`))).toBe(true),
    )
    // Tombol kembali aktif setelah permintaan selesai, tanpa berkas yang diunduh.
    await waitFor(() => expect(screen.getByRole('button', { name: 'Export' })).toBeEnabled())
    expect(createURL).not.toHaveBeenCalled()
  })
})
