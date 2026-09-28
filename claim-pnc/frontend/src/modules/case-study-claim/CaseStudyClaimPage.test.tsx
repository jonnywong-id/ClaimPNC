import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { CaseStudyRow, MetadataResponse } from './types'

const PATH = '/api/case-study-claim'
const META_PATH = `${PATH}/penyaring`

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

/**
 * Keterangan layar, sama bentuknya dengan yang disusun backend.
 *
 * Keduapuluh empat kolomnya ditulis lengkap dengan sengaja: urutan dan judulnya adalah
 * KONTRAK dengan grid Pega, dan uji yang hanya memakai tiga kolom tidak akan menangkap
 * kolom yang tertukar.
 */
const METADATA: MetadataResponse = {
  bisnis: [
    { kode: '002', label: 'PA' },
    { kode: '005', label: 'TRAVEL' },
    { kode: '346', label: 'NONMBU' },
    { kode: '003', label: 'BONDING' },
  ],
  status: [
    { kode: 'progress-accept', label: 'CLAIM ON PROGRESS/ ACCEPT' },
    { kode: 'reject', label: 'REJECT' },
  ],
  kolom: [
    { kunci: 'nomor_klaim', judul: 'PNC Case ID', jenis: 'teks' },
    { kunci: 'nomor_polis', judul: 'No Polis', jenis: 'teks' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung', jenis: 'teks' },
    { kunci: 'cob', judul: 'COB', jenis: 'teks' },
    { kunci: 'periode_polis', judul: 'Periode Polis', jenis: 'teks' },
    { kunci: 'bulan_klaim', judul: 'Bulan Klaim', jenis: 'teks' },
    { kunci: 'tanggal_kejadian', judul: 'Date of Loss', jenis: 'tanggal' },
    { kunci: 'sob', judul: 'SOB', jenis: 'teks' },
    { kunci: 'posisi_reasuransi', judul: 'Leader/Member/Fac In', jenis: 'teks' },
    { kunci: 'nature_of_loss', judul: 'Nature of Loss', jenis: 'teks' },
    { kunci: 'cause_of_loss', judul: 'Cause of Loss', jenis: 'teks' },
    { kunci: 'tsi_100', judul: 'TSI (100%)', jenis: 'uang' },
    { kunci: 'share_asm', judul: 'ASM SHARE', jenis: 'persen' },
    { kunci: 'deductible', judul: 'Deductible', jenis: 'uang' },
    { kunci: 'nilai_share_asm', judul: 'Nilai share ASM', jenis: 'uang' },
    { kunci: 'nilai_klaim_100', judul: 'NILAI KLAIM 100%', jenis: 'uang' },
    { kunci: 'adjuster_fee_100', judul: 'ADJUSTER FEE 100% SHARE', jenis: 'uang' },
    { kunci: 'nilai_klaim_net_100', judul: 'NILAI KLAIM NET 100%', jenis: 'uang' },
    { kunci: 'nilai_klaim_net_share_asm', judul: 'NILAI KLAIM NET ASM SHARE', jenis: 'uang' },
    {
      kunci: 'lack_of_doc',
      judul: 'LACK OF DOC/SALVAGE / RECOVERY / SUBROGARATION',
      jenis: 'uang',
    },
    { kunci: 'cabang', judul: 'Cabang', jenis: 'teks' },
    { kunci: 'status_klaim', judul: 'Status', jenis: 'teks' },
    { kunci: 'kronologi', judul: 'Kronologi', jenis: 'teks' },
    { kunci: 'remark', judul: 'Remark', jenis: 'catatan' },
  ],
  ambang_nilai_klaim: 500_000_000_000,
  catatan_periode:
    'Dari kedua tanggal yang dipilih, yang dipakai menyaring hanya TAHUNNYA.',
  catatan_kolom_kembar:
    'Kolom "Nature of Loss" dan "Cause of Loss" selalu berisi teks yang sama.',
}

/**
 * Satu baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di
 * berkas yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk
 * dokumen.
 *
 * Tiga nilainya sengaja `null`: ia yang membuktikan "belum ada nilainya" digambar sebagai
 * tanda hubung, bukan sebagai Rp 0,00.
 */
const ROW: CaseStudyRow = {
  nomor_klaim: 'STD-0001',
  nomor_polis: 'POL-CONTOH-0001',
  nama_tertanggung: 'PT Contoh Marine Sejahtera',
  cob: 'Marine Cargo',
  periode_polis: '2024',
  bulan_klaim: '03',
  tanggal_kejadian: '2024-03-11',
  sob: 'Broker Contoh',
  posisi_reasuransi: 'Leader',
  nature_of_loss: 'Kerusakan muatan dalam pengangkutan',
  cause_of_loss: 'Kerusakan muatan dalam pengangkutan',
  tsi_100: 2_500_000_000_000,
  share_asm: 400_000,
  deductible: null,
  nilai_share_asm: 320_000_000_000,
  nilai_klaim_100: 800_000_000_000,
  adjuster_fee_100: null,
  nilai_klaim_net_100: 738_000_000_000,
  nilai_klaim_net_share_asm: 295_200_000_000,
  lack_of_doc: null,
  cabang: 'Kantor Pusat',
  status_klaim: 'Claim On Progress',
  kronologi: 'Kontainer terguling saat bongkar muat di pelabuhan.',
  remark: '',
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

/** Peladen tiruan yang menjawab keterangan layar dan isi grid. */
function stubDefaultFetch(rows: CaseStudyRow[] = [ROW], total = rows.length) {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    return jsonResponse(200, {
      baris: rows,
      total,
      periode: { tahun_awal: '2024', tahun_akhir: '2024' },
    })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/case-study-claim']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** renderLoaded menggambar layar lalu MENUNGGU keterangan layarnya tiba. */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('option', { name: 'NONMBU' })
}

/** lihatData mengisi periode lalu menekan tombolnya. */
async function lihatData(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Awal'), '2024-01-01')
  await user.type(screen.getByLabelText('Akhir'), '2024-01-31')
  await user.click(screen.getByRole('button', { name: 'Lihat Data' }))
}

function lastListCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(`${PATH}?`))
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
  it('menggambar keempat penyaring seperti layar lama', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByLabelText('Awal')).toBeInTheDocument()
    expect(screen.getByLabelText('Akhir')).toBeInTheDocument()
    expect(screen.getByLabelText('Status')).toBeInTheDocument()
    expect(screen.getByLabelText('Bisnis')).toBeInTheDocument()

    expect(screen.getByRole('button', { name: 'Lihat Data' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export Data' })).toBeInTheDocument()
  })

  // Isi kedua dropdown datang dari SERVER, bukan disalin ke layar. Labelnya terbaca dari
  // `Property/StatusReceiver_property.xml` dan `Activity/FilterStudyClaim_act`.
  it('mengisi dropdown dari keterangan yang dikirim server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const business = screen.getByLabelText('Bisnis')
    expect(within(business).getByRole('option', { name: 'PA' })).toBeInTheDocument()
    expect(within(business).getByRole('option', { name: 'BONDING' })).toBeInTheDocument()

    const status = screen.getByLabelText('Status')
    // Spasi setelah garis miring ADA di export Pega dan dipertahankan (`D-13`).
    expect(
      within(status).getByRole('option', { name: 'CLAIM ON PROGRESS/ ACCEPT' }),
    ).toBeInTheDocument()
  })

  // Grid KOSONG sampai tombol ditekan — layar lama pun begitu.
  it('tidak memanggil daftar sebelum Lihat Data ditekan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(lastListCall()).toBeUndefined()
    expect(screen.getByText(/tekan/i)).toBeInTheDocument()
  })
})

describe('penyaring', () => {
  it('mengirim tanggal, bisnis, dan status ke server', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()

    await user.selectOptions(screen.getByLabelText('Bisnis'), '346')
    await user.selectOptions(screen.getByLabelText('Status'), 'reject')
    await lihatData(user)

    await waitFor(() => expect(lastListCall()).toBeDefined())

    const url = lastListCall()!.url
    expect(url).toContain('dari=2024-01-01')
    expect(url).toContain('sampai=2024-01-31')
    expect(url).toContain('bisnis=346')
    expect(url).toContain('status=reject')
  })

  // Perilaku yang paling mudah disangka cacat: HARI dan BULAN dibuang.
  //
  // Layar menyatakan tahun yang benar-benar dipakai, supaya daftar yang memuat klaim di
  // luar bulan yang dipilih tidak dilaporkan sebagai kerusakan.
  it('menyatakan bahwa hanya tahun yang dipakai menyaring', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()

    // Kalimatnya muncul TEPAT SEKALI, di bawah kedua isian tanggal.
    //
    // Uji ini pula yang menangkap duplikasinya: kalimat yang sama sempat digambar lagi di
    // bawah tabel, dan catatan yang berulang justru berhenti dibaca.
    expect(screen.getAllByText(/hanya TAHUNNYA/)).toHaveLength(1)

    await lihatData(user)
    expect(await screen.findByText(/tahun registrasi 2024/)).toBeInTheDocument()
  })
})

describe('grid', () => {
  it('menggambar keduapuluh empat kolom persis seperti yang ditetapkan server', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    const table = await screen.findByRole('table')
    const headers = within(table).getAllByRole('columnheader').map((h) => h.textContent)

    for (const column of METADATA.kolom) {
      expect(headers.some((h) => h?.includes(column.judul))).toBe(true)
    }
  })

  // Dua kolom bersebelahan yang memang KEMBAR — keduanya terikat kolom sumber yang sama.
  it('menggambar Nature of Loss dan Cause of Loss berisi teks yang sama', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    const cells = await screen.findAllByText(ROW.cause_of_loss)
    expect(cells.length).toBeGreaterThanOrEqual(2)
  })

  it('memformat uang sebagai rupiah dan persentase empat desimal', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    // Rp 8.000.000.000,00 — 800.000.000.000 sen.
    expect(await screen.findByText('Rp 8.000.000.000,00')).toBeInTheDocument()
    expect(screen.getByText('40%')).toBeInTheDocument()
  })

  // `null` digambar sebagai tanda hubung, BUKAN sebagai Rp 0,00.
  //
  // Perbedaannya menentukan: nol adalah angka yang ikut terhitung, "belum ada nilainya"
  // tidak.
  it('menggambar nilai kosong sebagai tanda hubung, bukan nol', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    await screen.findByRole('table')
    expect(screen.queryByText('Rp 0,00')).not.toBeInTheDocument()
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(3)
  })

  // Pesan grid kosong MENJELASKAN sebabnya, bukan sekadar "tidak ada data".
  //
  // Dicari DI DALAM tabel, bukan di seluruh halaman: catatan di bawah tabel menyebut ambang
  // yang sama secara permanen, dan keduanya memang sengaja ada. Yang diuji di sini adalah
  // pesan gridnya.
  it('menyebut ambang nilai saat hasilnya kosong', async () => {
    stubDefaultFetch([], 0)
    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    const table = await screen.findByRole('table')
    expect(
      within(table).getByText(/melampaui Rp 5\.000\.000\.000,00/),
    ).toBeInTheDocument()
  })
})

describe('simpan catatan', () => {
  it('menyimpan catatan baris lewat PUT dan menyegarkan daftarnya', async () => {
    let saved: RequestInit | undefined
    stubFetch((url, init) => {
      if (url === META_PATH) return jsonResponse(200, METADATA)
      if (url.startsWith(`${PATH}/STD-0001/catatan`)) {
        saved = init
        return jsonResponse(200, { nomor_klaim: 'STD-0001', catatan: 'sudah ditelaah' })
      }
      return jsonResponse(200, {
        baris: [saved ? { ...ROW, remark: 'sudah ditelaah' } : ROW],
        total: 1,
        periode: { tahun_awal: '2024', tahun_akhir: '2024' },
      })
    })

    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    const field = await screen.findByLabelText(/Remark untuk klaim STD-0001/)
    await user.type(field, 'sudah ditelaah')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(saved).toBeDefined())
    expect(saved?.method).toBe('PUT')
    expect(String(saved?.body)).toContain('sudah ditelaah')
  })

  // Tombol mati saat tidak ada perubahan: penyimpanan ini MENULIS ke tabel klaim milik
  // sistem lama, dan menekan Save tanpa perubahan adalah penulisan yang tidak dibutuhkan
  // siapa pun.
  it('menonaktifkan Save selama catatan belum berubah', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    await screen.findByLabelText(/Remark untuk klaim STD-0001/)
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })

  it('menampilkan pesan server saat penyimpanan ditolak', async () => {
    stubFetch((url) => {
      if (url === META_PATH) return jsonResponse(200, METADATA)
      if (url.startsWith(`${PATH}/STD-0001/catatan`)) {
        return jsonResponse(404, {
          kode: 'tidak_ditemukan',
          pesan: 'Klaim ini sudah tidak ada di daftar. Muat ulang daftarnya, lalu coba lagi.',
        })
      }
      return jsonResponse(200, {
        baris: [ROW],
        total: 1,
        periode: { tahun_awal: '2024', tahun_akhir: '2024' },
      })
    })

    const user = userEvent.setup()
    await renderLoaded()
    await lihatData(user)

    const field = await screen.findByLabelText(/Remark untuk klaim STD-0001/)
    await user.type(field, 'apa pun')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/Muat ulang daftarnya/)
  })
})

describe('portal', () => {
  // Klaim telaah milik satu badan hukum; tanpa portal, layar menolak membuka diri.
  it('meminta pengguna memilih entitas lebih dulu', async () => {
    stubDefaultFetch()
    useSelectedPortal.getState().clear()

    renderPage()

    expect(await screen.findByText(/Pilih entitas lebih dulu/)).toBeInTheDocument()
  })
})
