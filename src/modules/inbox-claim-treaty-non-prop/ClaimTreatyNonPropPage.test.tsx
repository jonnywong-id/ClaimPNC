import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-claim-treaty-non-prop'
const TAB_PATH = `${PATH}/tab`
const EXPORT_PATH = `${PATH}/ekspor`

/** Nama dropdown pemilih antrean — labelnya sr-only, lihat NonPropViewSelect. */
const SELECT_NAME = /Antrean klaim treaty non-proporsional/

const SAMPLE_PROFILE = {
  identitas: '90000002',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminnonprop1',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Antrean milik pemanggil — satu-satunya yang mengenal KEDUA checkbox.
 *
 * Kolomnya disalin dari grid Admin di `Section/InboxClaimNonProp_Harness-Section.xml`,
 * termasuk urutannya: nomor polis berada SESUDAH Ceding Co, dan tidak ada kolom "Status".
 */
const TAB_ADMIN: Tab = {
  kode: '1',
  nama: 'Treaty-In Admin',
  judul_grid: 'Work Treatyin Non Propotional Admin',
  keterangan: 'Klaim treaty non-proporsional yang ditugaskan kepada Anda.',
  kolom: [
    { kunci: 'no_klaim', judul: 'Claim.ID' },
    // Grid Admin memakai master id dari blob JSON, berjudul tanpa spasi.
    { kunci: 'id_master', judul: 'MasterID' },
    { kunci: 'nama_tertanggung', judul: 'Insured Name' },
    { kunci: 'tanggal_kejadian', judul: 'Date of Loss' },
    { kunci: 'nama_bisnis', judul: 'Business Name' },
    { kunci: 'sumber_bisnis', judul: 'Source of Business' },
    { kunci: 'ceding_co', judul: 'Ceding Co Name' },
    { kunci: 'no_polis', judul: 'Policy No' },
    { kunci: 'aging', judul: 'Aging' },
    { kunci: 'operator_pembuat', judul: 'Create Operator' },
    { kunci: 'operator_pengubah', judul: 'List Update Operator' },
  ],
  hanya_milik_saya: true,
  pakai_lihat_semua: true,
  pakai_lihat_tba: true,
  terhalang: false,
}

/**
 * Antrean bersama — tidak mengenal satu pun checkbox.
 *
 * Ia punya DUA belas kolom: master id-nya dari kolom objek kerja (berjudul "ID Master",
 * dengan spasi), dan ada kolom ke-12 berjudul "Status" yang berisi waktu pembuatan.
 */
const TAB_TEKNIK: Tab = {
  kode: '2',
  nama: 'Treaty-In Teknik',
  judul_grid: 'Work Treatyin Non Propotional Teknik',
  keterangan: 'Antrean bersama PIC Teknik treaty — belum diambil siapa pun.',
  kolom: [
    { kunci: 'no_klaim', judul: 'Claim.ID' },
    { kunci: 'master_id', judul: 'ID Master' },
    { kunci: 'nama_tertanggung', judul: 'Insured Name' },
    { kunci: 'tanggal_kejadian', judul: 'Date of Loss' },
    { kunci: 'nama_bisnis', judul: 'Business Name' },
    { kunci: 'sumber_bisnis', judul: 'Source of Business' },
    { kunci: 'ceding_co', judul: 'Ceding Co Name' },
    { kunci: 'no_polis', judul: 'Policy No' },
    { kunci: 'aging', judul: 'Aging' },
    { kunci: 'operator_pembuat', judul: 'Create Operator' },
    { kunci: 'operator_pengubah', judul: 'List Update Operator' },
    { kunci: 'dibuat_pada', judul: 'Status' },
  ],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  pakai_lihat_tba: false,
  terhalang: false,
}

/**
 * Antrean yang digambar tetapi belum dapat diisi.
 *
 * Di Pega ia BUKAN pilihan pada dropdown ini — gridnya muncul lewat pemeriksaan keanggotaan
 * komite. Ia dibawa sebagai pilihan yang terhalang, dan karena tidak punya teks pilihan yang
 * dapat dibaca, judul kontainernya yang dipakai.
 */
const TAB_KOMITE: Tab = {
  kode: '3',
  nama: 'Work List Treatyin Non Proportional Komite',
  keterangan: 'Antrean persetujuan komite klaim treaty non-proporsional.',
  kolom: [],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  pakai_lihat_tba: false,
  terhalang: true,
  alasan_terhalang:
    'Antrean komite treaty non-proporsional belum dapat dibaca sistem baru. Kuerinya ' +
    'dipanggil GetWorkCNP_Act tetapi tidak ada di export rule.',
  pemilik_penghalang: 'Tim Pega — dibutuhkan export rule KmtGetInboxListCNP_SQL.',
}

const METADATA: MetadataResponse = {
  tab: [TAB_ADMIN, TAB_TEKNIK, TAB_KOMITE],
  tab_bawaan: '1',
  portal: 'ASM',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * `aging` sengaja NOL: nol hari adalah nilai yang sah dan berarti "masuk hari ini", dan
 * itulah nilai yang paling mudah keliru digambar sebagai tanda pisah.
 */
const ROW: WorkItem = {
  referensi: 'ASSIGN-WORKLIST CLMNP-1001!FLOW',
  no_klaim: 'CLMNP-1001',
  master_id: 'TNP-2026-01',
  id_master: 'TNP-2026-01-REV1',
  no_polis: '99.002.2026.00001',
  tanggal_kejadian: '2026-08-14',
  nama_bisnis: 'Property All Risk',
  sumber_bisnis: 'Treaty Inward',
  ceding_co: 'Asuransi Contoh Pertama',
  nama_tertanggung: 'PT Contoh Sejahtera',
  dibuat_pada: '20260918T030000.000 GMT',
  aging: 0,
  operator_pembuat: 'ADMINNONPROP1',
  operator_pengubah: 'PICNONPROP1',
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
function stubDefaultFetch(rows: WorkItem[] = [ROW], tab: Tab = TAB_ADMIN) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url.startsWith(EXPORT_PATH)) {
      return new Response('Claim.ID\nCLMNP-1001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="klaim-treaty-non-prop-tab1.csv"',
        },
      })
    }
    return jsonResponse(200, {
      tab,
      baris: rows,
      paginasi: { halaman: 1, ukuran: 25, total: rows.length, total_halaman: 1 },
      penyaring: { lihat_semua: false, lihat_tba: false },
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
      <MemoryRouter initialEntries={['/inbox-claim-treaty-non-prop']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada pilihan dropdown, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara daftar antrean baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('option', { name: 'Treaty-In Admin' })
}

/** antrean mengembalikan dropdown pemilih antrean. */
function antrean(): HTMLSelectElement {
  return screen.getByRole('combobox', { name: SELECT_NAME }) as HTMLSelectElement
}

/** pilihAntrean berpindah antrean lewat dropdown, sama seperti pengguna. */
async function pilihAntrean(code: string) {
  await userEvent.selectOptions(antrean(), code)
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
    // "Claims In Progress" adalah `<pyValue>` judul kontainer di section. Nama menunya
    // tetap terlihat di menu kiri, tempat pengguna memilihnya (`D-13`).
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('heading', { name: 'Claims In Progress' }),
    ).toBeInTheDocument()
  })

  it('memilih antrean lewat DROPDOWN, bukan bilah tab', async () => {
    // Itulah bentuknya di Pega: satu `<select>` di antara judul dan grid.
    stubDefaultFetch()
    await renderLoaded()

    expect(antrean()).toBeInTheDocument()
    expect(screen.queryAllByRole('tab')).toHaveLength(0)
  })

  it('menyediakan entri kosong "Choose" bawaan dropdown Pega', async () => {
    // `pyHasNoSelection=true`, `pyNoSelectionText="Choose"` — ia bukan sebuah antrean.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('option', { name: 'Choose' })).toBeInTheDocument()
  })

  it('membuka antrean bawaan yang ditetapkan server, yaitu milik pemanggil', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(antrean().value).toBe('1')
  })

  it('memakai teks dropdown dan judul grid yang BERBEDA, seperti di Pega', async () => {
    // Dropdown bertuliskan "Treaty-In Admin"; grid di bawahnya berjudul "Work Treatyin Non
    // Propotional Admin". Menyamakan keduanya membuat judul grid hilang dan pilihan
    // dropdown tidak dikenali pengguna.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('option', { name: 'Treaty-In Admin' })).toBeInTheDocument()
    expect(
      await screen.findByRole('heading', { name: 'Work Treatyin Non Propotional Admin' }),
    ).toBeInTheDocument()
  })

  it('mempertahankan kedua ejaan judul grid seperti di Pega (D-13)', async () => {
    // Section yang SAMA memakai dua ejaan: "Propotional" pada dua grid pertama dan
    // "Proportional" pada grid komite. Keduanya dibawa apa adanya — menyeragamkannya
    // berarti memilih salah satu yang benar tanpa dasar.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByRole('heading', { name: 'Work Treatyin Non Propotional Admin' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('option', { name: /Non Proportional Komite/ }),
    ).toBeInTheDocument()
  })

  it('menggambar kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const column of TAB_ADMIN.kolom) {
      expect(
        await screen.findByRole('columnheader', { name: column.judul }),
      ).toBeInTheDocument()
    }
  })

  it('tidak menambah kolom aksi di ujung — Pega hanya punya kolom yang disebut server', async () => {
    // Tombol "Lihat Detail Klaim" dulu menempati kolom ke-12 yang tidak ada di Pega.
    // Penggantinya adalah tautan pada kolom Claim.ID yang memang sudah ada.
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByRole('columnheader', { name: 'Claim.ID' })
    expect(screen.getAllByRole('columnheader')).toHaveLength(TAB_ADMIN.kolom.length)
    expect(
      screen.queryByRole('button', { name: 'Lihat Detail Klaim' }),
    ).not.toBeInTheDocument()
  })
})

describe('kolom master id', () => {
  it('menggambar SATU kolom pada antrean Admin, dari blob JSON', async () => {
    // Grid Admin terikat CARI23 — `c.data_json.IDMaster` — dan judulnya dirangkai tanpa
    // spasi. Versi sebelumnya menggambar KEDUA sumbernya berdampingan; Pega tidak.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByRole('columnheader', { name: 'MasterID' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('columnheader', { name: 'ID Master' }),
    ).not.toBeInTheDocument()
    expect(screen.getByText('TNP-2026-01-REV1')).toBeInTheDocument()
  })

  it('menggambar SATU kolom pada antrean Teknik, dari kolom objek kerja', async () => {
    // Grid Teknik terikat CARI19 — `b.MASTERID` — dan judulnya memakai spasi. Kedua isian
    // sengaja berbeda pada baris contoh supaya tertukarnya ketahuan.
    stubDefaultFetch([ROW], TAB_TEKNIK)
    await renderLoaded()

    await pilihAntrean('2')

    expect(
      await screen.findByRole('columnheader', { name: 'ID Master' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('columnheader', { name: 'MasterID' }),
    ).not.toBeInTheDocument()
    expect(screen.getByText('TNP-2026-01')).toBeInTheDocument()
  })
})

describe('kolom berjudul "Status"', () => {
  it('hanya ada pada antrean Teknik, dan berisi waktu pembuatan apa adanya', async () => {
    // Judulnya menyesatkan sejak di Pega: sel ke-12 grid Teknik berjudul "Status" tetapi
    // terikat CARI21 = `b.PXCREATEDATETIME`. Bentuk teksnya pun dipertahankan.
    stubDefaultFetch([ROW], TAB_TEKNIK)
    await renderLoaded()

    await pilihAntrean('2')

    expect(await screen.findByRole('columnheader', { name: 'Status' })).toBeInTheDocument()
    expect(screen.getByText('20260918T030000.000 GMT')).toBeInTheDocument()
  })

  it('tidak digambar pada antrean Admin', async () => {
    // Ketiga kueri Admin tidak memilih `b.PXCREATEDATETIME` sama sekali, dan grid Admin di
    // section memang berhenti di sel ke-11.
    stubDefaultFetch()
    await renderLoaded()

    await screen.findByRole('columnheader', { name: 'Claim.ID' })
    expect(screen.queryByRole('columnheader', { name: 'Status' })).not.toBeInTheDocument()
  })
})

describe('kedua checkbox', () => {
  it('menggambar keduanya hanya pada antrean yang mengenalnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByLabelText('See All Claim')).toBeInTheDocument()
    expect(screen.getByLabelText('See TBA Claim')).toBeInTheDocument()
  })

  it('mengirim lihat_tba TANPA lihat_semua — selisih terencana P-5', async () => {
    // Inilah perbaikan yang disetujui Work Owner 2026-09-22. Di layar Pega, mencentang
    // "See TBA Claim" sendirian tidak mengubah apa pun: kueri TBA-nya dijaga dua
    // prakondisi ber-AND. Di sini ia berdiri sendiri.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See TBA Claim'))

    const call = lastListCall()
    expect(call?.url).toContain('lihat_tba=1')
    expect(call?.url).not.toContain('lihat_semua')
  })

  it('mengirim keduanya saat dicentang bersama', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See All Claim'))
    await userEvent.click(screen.getByLabelText('See TBA Claim'))

    const call = lastListCall()
    expect(call?.url).toContain('lihat_semua=1')
    expect(call?.url).toContain('lihat_tba=1')
  })

  it('membersihkan kedua centang saat berpindah antrean', async () => {
    // Membawa centang ke antrean yang tidak mengenalnya akan membuat centang yang tampak
    // aktif padahal tidak mengubah apa pun.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See All Claim'))
    await pilihAntrean('2')

    const call = lastListCall()
    expect(call?.url).not.toContain('lihat_semua')
    expect(call?.url).not.toContain('lihat_tba')
  })

  it('tidak menggambar satu pun checkbox pada antrean yang tidak mengenalnya', async () => {
    stubDefaultFetch([ROW], TAB_TEKNIK)
    await renderLoaded()

    await pilihAntrean('2')

    expect(screen.queryByLabelText('See All Claim')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('See TBA Claim')).not.toBeInTheDocument()
  })
})

describe('antrean yang terhalang', () => {
  it('menandai keadaannya di TEKS pilihan, sebelum dipilih', async () => {
    // Isi `<option>` hanya boleh berupa teks, dan keadaan "belum tersedia" harus diketahui
    // SEBELUM dipilih — bukan sesudahnya.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('option', { name: /Non Proportional Komite — belum tersedia/ }),
    ).toBeInTheDocument()
  })

  it('menampilkan alasan dan pemiliknya, bukan tabel kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await pilihAntrean('3')

    expect(await screen.findByText(/tidak ada di export rule/)).toBeInTheDocument()
    expect(screen.getByText(/KmtGetInboxListCNP_SQL/)).toBeInTheDocument()
  })

  it('TIDAK meminta isinya ke server', async () => {
    // Permintaannya pasti ditolak 422, dan galat yang sudah diketahui pasti terjadi bukan
    // galat yang layak ditampilkan.
    stubDefaultFetch()
    await renderLoaded()

    calls = []
    await pilihAntrean('3')
    await screen.findByText(/tidak ada di export rule/)

    expect(lastListCall()).toBeUndefined()
  })
})

describe('tombol ekspor', () => {
  it('mengirim penyaring yang SAMA dengan yang sedang tampil', async () => {
    // Ekspor yang mengabaikan penyaring akan mengeluarkan berkas yang isinya tidak dapat
    // dicocokkan dengan apa pun di layar.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See TBA Claim'))
    await userEvent.click(screen.getByRole('button', { name: 'Export Data' }))

    const ekspor = [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
    expect(ekspor?.url).toContain('lihat_tba=1')
  })

  it('membawa token dan portal sebagai header, bukan di dalam alamat', async () => {
    // `<a href>` tidak membawa header, dan menaruh token di URL akan meninggalkan
    // jejaknya di riwayat peramban, log proxy, dan header Referer.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByRole('button', { name: 'Export Data' }))

    const ekspor = [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
    const header = ekspor?.init?.headers as Record<string, string> | undefined

    expect(header?.['Authorization']).toBe('Bearer token-uji')
    expect(ekspor?.url).not.toContain('token-uji')
  })

  it('dimatikan saat antreannya kosong', async () => {
    // Berkas kosong yang tetap terunduh tidak dapat dibedakan pengguna dari ekspor yang
    // gagal diam-diam.
    stubDefaultFetch([])
    await renderLoaded()

    expect(await screen.findByRole('button', { name: 'Export Data' })).toBeDisabled()
  })
})

describe('penggambaran sel', () => {
  it('menjadikan nomor klaim TAUTAN ke layar Acceptation Claim', async () => {
    // Sel Claim.ID di Pega ber-`pyAction openAssignment`, dan Flow Action yang digambar
    // untuk objek kerja itu adalah `InputAcceptation` — satu-satunya yang terdaftar pada
    // kelas ASM-FW-GCNMFW-Work-ClaimTreatyNonProp.
    //
    // Alamatnya memakai NOMOR KLAIM, bukan kunci teknis Pega: ia terbaca orang dan dapat
    // disalin ke percakapan, mengikuti layar saudaranya Outstanding Claim.
    stubDefaultFetch()
    await renderLoaded()

    const tautan = await screen.findByRole('link', { name: 'CLMNP-1001' })
    expect(tautan).toHaveAttribute('href', '/input-acceptation/CLMNP-1001')
  })

  it('menampilkan aging nol sebagai angka polos, bukan sebagai tanda pisah', async () => {
    // Nol hari adalah nilai yang SAH dan berarti "masuk hari ini". Mengubahnya menjadi
    // "—" akan menyamakannya dengan kolom yang gagal dimuat. Satuannya tidak diulang di
    // setiap baris — Pega pun menggambarnya sebagai angka polos.
    stubDefaultFetch()
    await renderLoaded()

    const baris = (await screen.findByRole('link', { name: 'CLMNP-1001' })).closest('tr')
    expect(baris).not.toBeNull()
    expect(within(baris as HTMLElement).getByText('0')).toBeInTheDocument()
  })

  // Keputusan Work Owner 2026-10-06: panel selisih terencana DIHAPUS dari seluruh layar.
  //
  // Daftarnya tetap hidup di kode Go untuk uji kesetaraan gerbang 1 (`D-54`); yang berubah
  // adalah ia berhenti menjadi isi layar.
  it('tidak lagi menggambar panel selisih terencana', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.queryByText(/See TBA Claim" kini BERDIRI SENDIRI/)).not.toBeInTheDocument()
  })
})

describe('tombol buat klaim', () => {
  it('menolak dengan alasan, bukan dengan "halaman tidak ditemukan"', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url === `${PATH}/klaim`) {
        return jsonResponse(501, {
          kode: 'belum_tersedia',
          pesan:
            'Pembuatan klaim treaty non-prop belum tersedia di sistem baru. Buat klaim ' +
            'treaty non-prop baru lewat Pega.',
        })
      }
      return jsonResponse(200, {
        tab: TAB_ADMIN,
        baris: [ROW],
        paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
        penyaring: { lihat_semua: false, lihat_tba: false },
        portal: 'ASM',
      })
    })
    await renderLoaded()

    await userEvent.click(
      screen.getByRole('button', { name: 'Create Claim Treaty Non Prop' }),
    )

    expect(await screen.findByText(/lewat Pega/)).toBeInTheDocument()
  })
})
