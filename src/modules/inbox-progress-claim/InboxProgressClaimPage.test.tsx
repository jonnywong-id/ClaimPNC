import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { ClaimRow, MetadataResponse, PICRow, Section } from './types'

const PATH = '/api/inbox-progress-claim'
const SECTION_PATH = `${PATH}/bagian`

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
 * Bagian Outstanding — memuat kolom ber-judul menyesatkan yang menjadi ciri layar ini,
 * termasuk `DateForAging` yang memang digambar DUA KALI di sistem lama.
 */
const OUTSTANDING: Section = {
  kode: 'outstanding',
  nama: 'Outstanding',
  keterangan: 'Seluruh klaim yang masih berjalan.',
  bentuk: 'klaim',
  kolom: [
    { kunci: 'no_klaim', isian: 'no_klaim', judul: 'CaseID', keterangan: 'Nomor Klaim' },
    { kunci: 'no_polis', isian: 'no_polis', judul: 'ClaimNo', keterangan: 'Nomor Polis' },
    {
      kunci: 'nama_tertanggung',
      isian: 'nama_tertanggung',
      judul: 'District',
      keterangan: 'Nama Tertanggung',
    },
    {
      kunci: 'tanggal_registrasi',
      isian: 'tanggal_registrasi',
      judul: 'DateForAging',
      keterangan: 'Tanggal Registrasi',
    },
    { kunci: 'posisi', isian: 'posisi', judul: 'City', keterangan: 'Posisi klaim' },
    {
      kunci: 'tanggal_registrasi_2',
      isian: 'tanggal_registrasi',
      judul: 'DateForAging',
      keterangan: 'Tanggal Registrasi — kolom kedua',
    },
  ],
  pakai_paginasi: true,
  pakai_pencarian: true,
  pakai_rentang_tanggal: false,
  pakai_lini_bisnis: false,
  hanya_milik_saya: false,
  kontrol_mati: [
    { nama: 'Tanggal Kejadian', alasan: 'tidak menyaring apa pun di sistem lama' },
    { nama: 'Business', alasan: 'hanya berlaku pada Progress Klaim per PIC' },
  ],
}

const NEXT_FU: Section = {
  ...OUTSTANDING,
  kode: 'next-fu',
  nama: 'Next Follow Up',
  keterangan: 'Klaim yang tindak lanjutnya jatuh tempo hari ini atau sudah terlewat.',
  kontrol_mati: [
    { nama: 'Business', alasan: 'hanya berlaku pada Progress Klaim per PIC' },
  ],
}

const PER_PIC: Section = {
  kode: 'per-pic',
  nama: 'Progress Klaim per PIC',
  keterangan: 'Rekap beban dan ketepatan tindak lanjut.',
  bentuk: 'pic',
  kolom: [
    { kunci: 'pic', isian: 'pic', judul: 'PIC', keterangan: 'Nama petugas' },
    {
      kunci: 'jumlah_klaim',
      isian: 'jumlah_klaim',
      judul: 'NOKLAIM',
      keterangan: 'Jumlah klaim yang ditangani',
    },
    {
      kunci: 'terlambat',
      isian: 'terlambat',
      judul: 'NOPOLIS',
      keterangan: 'Tindak lanjut yang terlambat',
    },
  ],
  pakai_paginasi: false,
  pakai_pencarian: false,
  pakai_rentang_tanggal: true,
  pakai_lini_bisnis: true,
  hanya_milik_saya: true,
  kontrol_mati: [],
}

const EVALUATION: Section = {
  kode: 'evaluasi',
  nama: 'Evaluasi Progress Klaim',
  keterangan: 'Bagian ini kosong di sistem lama.',
  bentuk: 'kosong',
  kolom: [],
  pakai_paginasi: false,
  pakai_pencarian: false,
  pakai_rentang_tanggal: false,
  pakai_lini_bisnis: false,
  hanya_milik_saya: false,
  kontrol_mati: [],
}

const METADATA: MetadataResponse = {
  bagian: [OUTSTANDING, NEXT_FU, PER_PIC, EVALUATION],
  bagian_bawaan: 'outstanding',
  lini_bisnis: [
    { kode: 'NONMBU', label: 'Non-MBU' },
    { kode: 'PA', label: 'Personal Accident' },
  ],
  ukuran_halaman: 15,
  portal: 'ASM',
  keterbatasan: ['Penyaring Cabang belum aktif.'],
}

const CLAIM: ClaimRow = {
  no_klaim: 'PNCN.26.0101',
  no_polis: '16.001.2026.00101',
  nama_tertanggung: 'PT SUMBER CONTOH SENTOSA',
  tanggal_registrasi: '2026-08-03',
  tanggal_kejadian: '2026-07-28',
  catatan_lgb: 'Kerugian kebakaran gudang',
  pic_klaim: 'ADMINPNC',
  posisi: 'SURVEY, KOMITE',
  status_progres_1: 'Survey Berjalan, Menunggu Komite',
  status_progres_2: 'Menunggu laporan, Berkas lengkap',
  next_follow_up: '2026-09-18, 2026-09-19',
  jumlah_posisi: 2,
  follow_up_terawal: '2026-09-18',
  tanggal_proses: '2026-08-04',
  prod_ke: '2',
}

const SUMMARY: PICRow = {
  pic: 'ADMINPNC',
  jumlah_klaim: 12,
  jumlah_pembaruan: 37,
  jatuh_tempo_hari_ini: 2,
  tepat_waktu: 30,
  terlambat: 5,
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

/** Peladen tiruan yang menjawab bentuk layar dan isi setiap bagian. */
function stubDefaultFetch(claims: ClaimRow[] = [CLAIM], summaries: PICRow[] = [SUMMARY]) {
  stubFetch((url) => {
    if (url === SECTION_PATH) return jsonResponse(200, METADATA)

    const perPIC = url.includes('bagian=per-pic')
    const evaluation = url.includes('bagian=evaluasi')
    const section = perPIC ? PER_PIC : evaluation ? EVALUATION : OUTSTANDING
    const rows = perPIC ? summaries : evaluation ? [] : claims

    return jsonResponse(200, {
      bagian: section,
      baris: rows,
      paginasi: {
        halaman: 1,
        ukuran: 15,
        total: rows.length,
        total_halaman: 1,
      },
      penyaring: { cari: '', bisnis: '', dari: null, sampai: null },
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
      <MemoryRouter initialEntries={['/inbox-progress-claim']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada judul salah satu bagian, bukan pada judul layar: judul layar sudah ada
 * sejak penggambaran pertama, sementara daftar bagian baru tiba bersama jawaban `/bagian`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('button', { name: /Progress Klaim per PIC/ })
}

/** listCalls mengembalikan seluruh permintaan isi bagian. */
function listCalls(): Call[] {
  return calls.filter((c) => c.url.startsWith(`${PATH}?`))
}

/** sectionPanel mengambil elemen bagian menurut judulnya. */
function sectionPanel(name: string): HTMLElement {
  const heading = screen.getByRole('button', { name: new RegExp(name) })
  const panel = heading.closest('section')
  if (panel === null) throw new Error(`bagian ${name} tidak ditemukan`)
  return panel
}

/**
 * openSection membuka satu bagian lalu menunggu isinya tiba.
 *
 * Penantiannya perlu: tiap bagian memuat datanya sendiri saat dibuka, sehingga menegaskan
 * isinya tepat setelah klik akan menguji tabel yang masih kosong.
 */
async function openSection(name: string): Promise<HTMLElement> {
  await userEvent.click(screen.getByRole('button', { name: new RegExp(name) }))

  const panel = sectionPanel(name)
  await within(panel).findByRole('table')
  return panel
}

/**
 * headerTitles membaca judul kolom dari `<th>` saja.
 *
 * `DataTable` menggambar judul DUA KALI: sekali di kepala tabel, dan sekali lagi di dalam
 * tiap sel sebagai label tampilan kartu untuk layar sempit. Mencarinya lewat teks akan
 * menghitung keduanya, sehingga yang dibaca di sini hanyalah kepala tabelnya.
 */
function headerTitles(panel: HTMLElement): string[] {
  return within(panel)
    .getAllByRole('columnheader')
    .map((cell) => cell.textContent?.trim() ?? '')
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
  it('menggambar keempat bagian sebagai bagian bertumpuk, bukan sebagai tab', async () => {
    // Bentuknya diambil dari Pega: kontainer berjudul yang ditumpuk, bukan bilah tab
    // seperti Inbox Admin.
    stubDefaultFetch()
    await renderLoaded()

    for (const name of [
      'Outstanding',
      'Next Follow Up',
      'Progress Klaim per PIC',
      'Evaluasi Progress Klaim',
    ]) {
      expect(screen.getByRole('button', { name: new RegExp(name) })).toBeInTheDocument()
    }

    expect(screen.queryAllByRole('tab')).toHaveLength(0)
  })

  it('membuka bagian bawaan yang ditetapkan server dan menutup sisanya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('PNCN.26.0101')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /Progress Klaim per PIC/ }),
    ).toHaveAttribute('aria-expanded', 'false')
  })

  it('hanya meminta isi bagian yang terbuka', async () => {
    // Meniru `pyDeferLoadRetrievalActivity` pada setiap kontainer: sistem lama pun tidak
    // menjalankan ketiga kuerinya bersamaan. Rekap per PIC menghitung lima subkueri
    // agregat, dan membayarnya untuk bagian yang tidak dilihat adalah pemborosan nyata.
    stubDefaultFetch()
    await renderLoaded()

    expect(listCalls()).toHaveLength(1)
    expect(listCalls()[0]?.url).toContain('bagian=outstanding')
  })

  it('memuat isi sebuah bagian saat bagian itu dibuka', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('button', { name: /Next Follow Up/ }))

    await vi.waitFor(() => {
      expect(listCalls().some((c) => c.url.includes('bagian=next-fu'))).toBe(true)
    })
  })
})

describe('kolom dan judulnya', () => {
  it('memakai alias Pega sebagai judul, apa adanya', async () => {
    // Keputusan Work Owner 2026-09-21. Judul `District` berisi nama tertanggung dan
    // `ClaimNo` berisi nomor polis — keduanya memang begitu di sistem lama.
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('PNCN.26.0101')

    expect(headerTitles(sectionPanel('Outstanding'))).toEqual([
      'CaseID',
      'ClaimNo',
      'District',
      'DateForAging',
      'City',
      'DateForAging',
      '',
    ])
  })

  it('menggambar kolom yang isiannya kembar tanpa saling menimpa', async () => {
    // `DateForAging` digambar dua kali di sistem lama, keduanya terikat kolom yang sama.
    // Bila identitas kolom diambil dari nama isiannya, salah satunya akan hilang.
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('PNCN.26.0101')

    const panel = sectionPanel('Outstanding')
    expect(within(panel).getAllByText('3 Agustus 2026')).toHaveLength(2)
  })

  it('menyebut jumlah posisi saat satu klaim berada di beberapa posisi', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('SURVEY, KOMITE')).toBeInTheDocument()
    expect(screen.getByText('2 posisi berjalan')).toBeInTheDocument()
  })

  it('menampilkan nomor perpanjangan di bawah nomor polis', async () => {
    // Di Pega keduanya berada di baris rincian yang terbuka saat baris diklik; komponen
    // tabel baku belum punya baris rincian, dan menempelkannya di sini menjaga
    // keterangannya tetap terbaca alih-alih hilang.
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('Prod ke-2')).toBeInTheDocument()
  })
})

describe('penyaring', () => {
  it('menggambar kotak cari hanya pada bagian yang memang menyaringnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(within(sectionPanel('Outstanding')).getByLabelText('Cari')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /Progress Klaim per PIC/ }))
    const perPIC = sectionPanel('Progress Klaim per PIC')

    expect(within(perPIC).queryByLabelText('Cari')).not.toBeInTheDocument()
    expect(within(perPIC).getByLabelText('Lini Bisnis')).toBeInTheDocument()
    expect(within(perPIC).getByLabelText('Tanggal Registrasi dari')).toBeInTheDocument()
  })

  it('mengirim kata kunci hanya setelah tombol Cari ditekan', async () => {
    // Setiap ketikan menembak basis data. Kotak cari di sistem lama pun baru menyaring
    // setelah tombol "Cari Data" ditekan.
    stubDefaultFetch()
    await renderLoaded()

    const panel = sectionPanel('Outstanding')
    await userEvent.type(within(panel).getByLabelText('Cari'), '0101')

    expect(listCalls().some((c) => c.url.includes('cari='))).toBe(false)

    await userEvent.click(within(panel).getByRole('button', { name: 'Cari Data' }))

    await vi.waitFor(() => {
      expect(listCalls().some((c) => c.url.includes('cari=0101'))).toBe(true)
    })
  })

  it('membersihkan seluruh penyaring lewat Clear Filter', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const panel = sectionPanel('Outstanding')
    const search = within(panel).getByLabelText('Cari')

    await userEvent.type(search, '0101')
    await userEvent.click(within(panel).getByRole('button', { name: 'Clear Filter' }))

    expect(search).toHaveValue('')
  })

  it('menjelaskan penyaring yang tampil di sistem lama tetapi tidak menyaring', async () => {
    // Tanpa keterangan ini, pengguna yang mencari penyaring Tanggal Kejadian akan
    // melaporkannya sebagai kerusakan, berulang kali.
    stubDefaultFetch()
    await renderLoaded()

    const panel = sectionPanel('Outstanding')
    expect(within(panel).getByText(/Tanggal Kejadian/)).toBeInTheDocument()
    expect(within(panel).getByText(/tidak menyaring apa pun di sistem lama/)).toBeInTheDocument()
  })
})

describe('rekap per PIC', () => {
  it('menggambar pencacahnya dengan judul alias Pega', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const panel = await openSection('Progress Klaim per PIC')

    expect(headerTitles(panel)).toEqual(['PIC', 'NOKLAIM', 'NOPOLIS'])
    expect(within(panel).getByText('12')).toBeInTheDocument()
  })

  it('tidak menggambar tombol Lihat Detail Klaim, karena barisnya bukan klaim', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const panel = await openSection('Progress Klaim per PIC')

    expect(
      within(panel).queryByRole('button', { name: 'Lihat Detail Klaim' }),
    ).not.toBeInTheDocument()
  })

  it('mengirim lini bisnis yang dipilih', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const panel = await openSection('Progress Klaim per PIC')

    await userEvent.selectOptions(within(panel).getByLabelText('Lini Bisnis'), 'PA')

    await vi.waitFor(() => {
      expect(listCalls().some((c) => c.url.includes('bisnis=PA'))).toBe(true)
    })
  })
})

describe('bagian yang kosong di sistem lama', () => {
  it('menjelaskan kekosongannya alih-alih menggambar tabel kosong', async () => {
    // Bagian Evaluasi punya judul dan kerangka tabel di Pega, tetapi nol properti terikat
    // dan nol activity pengisi. Mengarang isinya berarti mengarang layar yang tidak
    // pernah ada.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('button', { name: /Evaluasi Progress Klaim/ }))
    const panel = sectionPanel('Evaluasi Progress Klaim')

    expect(
      within(panel).getByText(/Tidak ada yang dapat dipindahkan/),
    ).toBeInTheDocument()
    expect(within(panel).queryByRole('table')).not.toBeInTheDocument()
  })

  it('tidak meminta apa pun ke server untuk bagian itu', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('button', { name: /Evaluasi Progress Klaim/ }))
    await screen.findByText(/Tidak ada yang dapat dipindahkan/)

    // Bagiannya memang punya kode, tetapi tidak punya kueri. Memintanya berarti
    // membebani basis data untuk jawaban yang sudah pasti kosong.
    expect(listCalls().some((c) => c.url.includes('bagian=evaluasi'))).toBe(false)
  })
})

describe('keadaan tepi', () => {
  it('menjelaskan hasil kosong menurut sebabnya', async () => {
    stubDefaultFetch([])
    await renderLoaded()

    expect(
      await screen.findByText('Tidak ada klaim yang sedang berjalan.'),
    ).toBeInTheDocument()
  })

  it('menampilkan keterbatasan yang dikirim server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText('Penyaring Cabang belum aktif.')).toBeInTheDocument()
  })

  it('meminta portal dipilih lebih dulu', async () => {
    // Progres klaim milik satu badan hukum. Tanpa portal, permintaannya tidak boleh
    // dikirim sama sekali (`R-20`).
    useSelectedPortal.getState().clear()
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(listCalls()).toHaveLength(0)
  })

  it('menjelaskan kegagalan memuat bentuk layar', async () => {
    stubFetch((url) => {
      if (url === SECTION_PATH) {
        return jsonResponse(500, { kode: 'galat_internal', pesan: 'Sistem sedang bermasalah.' })
      }
      return jsonResponse(200, {})
    })
    renderPage()

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
  })
})
