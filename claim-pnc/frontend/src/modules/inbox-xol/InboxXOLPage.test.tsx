import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
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

const SUMMARY: ClaimSummaryResponse = {
  perjanjian: MASTERS[0]!,
  baris: [
    {
      tanggal_kejadian: '12/03/2024',
      sebab_kerugian: 'BANJIR',
      group_business: 'Fire, Marine Cargo',
      nilai_outstanding: 300000,
      nilai_akseptasi: 200000,
    },
  ],
}

const EMPTY_SUMMARY: ClaimSummaryResponse = { perjanjian: MASTERS[1]!, baris: [] }

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
    if (url.startsWith(`${PATH}/klaim/rincian`)) return jsonResponse(200, { baris: BREAKDOWN })
    if (url.startsWith(`${PATH}/klaim`)) {
      return jsonResponse(200, url.includes('XOL-003') ? EMPTY_SUMMARY : SUMMARY)
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
 * renderWithMasters menggambar layar lalu MENUNGGU daftar perjanjian tiba.
 *
 * Penantiannya pada salah satu pilihan dropdown, bukan pada labelnya: label "Perjanjian
 * XOL" sudah ada sejak penggambaran pertama, sementara isinya baru tiba bersama jawaban
 * server. Menunggu label saja membuat uji menekan dropdown yang masih kosong.
 */
async function renderWithMasters(): Promise<HTMLElement> {
  renderPage()
  await screen.findByRole('option', { name: /2024 — Fire, Marine Cargo/ })
  return screen.getByLabelText('Perjanjian XOL')
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
  it('menggambar kedua tab, dan membuka tab Inbox XOL lebih dulu', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    expect(screen.getByRole('tab', { name: 'Inbox XOL' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(screen.getByRole('tab', { name: 'Inbox XOL Komite' })).toHaveAttribute(
      'aria-selected',
      'false',
    )
  })

  // Tab yang belum dibuka tidak boleh menembak permintaannya. Memuat keduanya di muka
  // menggandakan beban setiap kali layar dibuka, dan separuhnya tidak pernah dilihat.
  it('tidak memuat antrean komite sebelum tabnya dibuka', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    expect(callsTo(`${PATH}/persetujuan`)).toHaveLength(0)

    await userEvent.click(screen.getByRole('tab', { name: 'Inbox XOL Komite' }))
    expect(await screen.findByText('Approval XOL')).toBeInTheDocument()
    expect(callsTo(`${PATH}/persetujuan`)).toHaveLength(1)
  })

  it('menolak membuka layar sebelum portal dipilih', async () => {
    stubDefaultFetch()
    useSelectedPortal.getState().clear()
    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  })
})

describe('tab Inbox XOL', () => {
  it('tidak memuat klaim sebelum perjanjian dipilih', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    expect(
      screen.getByText('Pilih perjanjian XOL untuk melihat akumulasi klaimnya.'),
    ).toBeInTheDocument()
    expect(callsTo(`${PATH}/klaim`)).toHaveLength(0)
  })

  it('menampilkan akumulasi klaim setelah perjanjian dipilih', async () => {
    stubDefaultFetch()
    const picker = await renderWithMasters()

    await userEvent.selectOptions(picker, 'XOL-001')

    // Dilingkupi ke BARISNYA, bukan ke seluruh layar: "BANJIR" juga muncul sebagai
    // pilihan pada dropdown panel PLA/DLA di bawah, dan pencarian tanpa lingkup akan
    // menemukan keduanya.
    const row = (await screen.findByText('12/03/2024')).closest('tr')!

    expect(within(row).getByText('BANJIR')).toBeInTheDocument()
    // 300000 digambar dengan pemisah ribuan Indonesia, BUKAN sebagai rupiah: satuannya
    // mata uang perjanjian, dan menempelkan "Rp" akan menyatakan hal yang salah.
    expect(within(row).getByText('300.000')).toBeInTheDocument()
  })

  /**
   * Perjanjian tanpa group business tidak akan PERNAH punya baris, dan itu bukan
   * "belum ada klaim" melainkan master yang belum lengkap. Menyatukan kedua pesan
   * membuat pengguna menunggu data yang tidak akan datang.
   */
  it('membedakan perjanjian yang belum diisi dari perjanjian tanpa klaim', async () => {
    stubDefaultFetch()
    const picker = await renderWithMasters()

    await userEvent.selectOptions(picker, 'XOL-003')

    expect(
      await screen.findByText(/Perjanjian ini belum punya group business/),
    ).toBeInTheDocument()
  })

  it('memuat rincian hanya saat sebuah baris dibuka', async () => {
    stubDefaultFetch()
    const picker = await renderWithMasters()
    await userEvent.selectOptions(picker, 'XOL-001')
    await screen.findByText('12/03/2024')

    expect(callsTo(`${PATH}/klaim/rincian`)).toHaveLength(0)

    await userEvent.click(screen.getByRole('button', { name: 'Lihat rincian' }))

    expect(await screen.findByText(/Rincian — 12\/03\/2024/)).toBeInTheDocument()
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
    const picker = await renderWithMasters()
    await userEvent.selectOptions(picker, 'XOL-001')
    await screen.findByText('12/03/2024')
    await userEvent.click(screen.getByRole('button', { name: 'Lihat rincian' }))

    await screen.findByText(/Rincian — 12\/03\/2024/)

    // Barisnya ADA — yang tidak ada adalah angkanya.
    const treatyRow = (await screen.findByText('Treaty Inward')).closest('tr')!
    expect(within(treatyRow).getByText('2')).toBeInTheDocument()
    expect(within(treatyRow).queryByText('0')).not.toBeInTheDocument()

    expect(
      await screen.findByText(/tidak dapat dihitung karena kurs mata uangnya/),
    ).toBeInTheDocument()
  })

  // Tombol "INSERT DOL DAN COL" menulis ke tabel yang masih dimiliki Pega (`P-1`).
  // Ketiadaannya dinyatakan, bukan disembunyikan — tombol yang hilang tanpa keterangan
  // dilaporkan pengguna sebagai kerusakan.
  it('menyatakan penambahan DOL dan COL belum tersedia', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    expect(
      screen.getByText(/Menambah data DOL dan Cause Of Loss belum tersedia/),
    ).toBeInTheDocument()
  })
})

describe('panel PLA/DLA', () => {
  it('tidak mencari sebelum tombol ditekan', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    expect(callsTo(`${PATH}/pla-dla`)).toHaveLength(0)
  })

  it('mencari dan menampilkan nomor beserta revisinya', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    await userEvent.type(screen.getByLabelText('Tahun XOL'), '2024')
    await userEvent.selectOptions(screen.getByLabelText('Penyebab Kerugian'), 'BANJIR')
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pemberitahuan'), 'PLA')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data DLA PLA XOL' }))

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
    await renderWithMasters()

    await userEvent.type(screen.getByLabelText('Tahun XOL'), '2024')
    await userEvent.selectOptions(screen.getByLabelText('Penyebab Kerugian'), 'BANJIR')
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pemberitahuan'), 'DLA')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data DLA PLA XOL' }))

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
    await renderWithMasters()

    await userEvent.selectOptions(screen.getByLabelText('Penyebab Kerugian'), 'BANJIR')
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pemberitahuan'), 'PLA')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data DLA PLA XOL' }))

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
    await renderWithMasters()

    expect(screen.queryByRole('link', { name: /Unduh perhitungan/ })).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Tahun XOL'), '2024')
    await userEvent.selectOptions(screen.getByLabelText('Penyebab Kerugian'), 'BANJIR')
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pemberitahuan'), 'PLA')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Data DLA PLA XOL' }))

    const link = await screen.findByRole('link', { name: 'Unduh perhitungan (CSV)' })
    expect(link).toHaveAttribute('href', expect.stringContaining(`${PATH}/pla-dla/unduh`))
    expect(screen.queryByText('Print Perhitungan')).not.toBeInTheDocument()
  })
})

describe('tab Inbox XOL Komite', () => {
  it('menampilkan kedua antrean dan menyatakan persetujuan belum tersedia', async () => {
    stubDefaultFetch()
    await renderWithMasters()

    await userEvent.click(screen.getByRole('tab', { name: 'Inbox XOL Komite' }))

    expect(await screen.findByText('Approval XOL')).toBeInTheDocument()
    expect(screen.getByText('DATA MASTER XOL')).toBeInTheDocument()
    expect(screen.getByText('KEBAKARAN')).toBeInTheDocument()
    expect(screen.getByText('XOL-003')).toBeInTheDocument()
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
    await renderWithMasters()

    await userEvent.click(screen.getByRole('tab', { name: 'Inbox XOL Komite' }))

    expect(
      await screen.findByText(/berisi tahun perjanjian, bukan tanggal kejadian/),
    ).toBeInTheDocument()
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
    const picker = await renderWithMasters()
    await userEvent.selectOptions(picker, 'XOL-001')
    await screen.findByText('12/03/2024')

    const moduleCalls = calls.filter((c) => c.url.startsWith(PATH))
    expect(moduleCalls.length).toBeGreaterThan(0)

    for (const call of moduleCalls) {
      const headers = new Headers(call.init?.headers)
      expect(headers.get('X-Portal')).toBe('ASM')
    }
  })
})
