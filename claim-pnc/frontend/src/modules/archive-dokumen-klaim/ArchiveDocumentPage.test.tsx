import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { ArchiveFile, ClaimCandidate, OpenResponse } from './types'

const PATH = '/api/arsip-dokumen'
const OPEN_PATH = `${PATH}/buka`

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
 * Seluruh data uji KARANGAN — `D-69` melarang data nasabah ditulis di berkas yang
 * di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const OPENED: OpenResponse = {
  tipe_input: [
    { kode: 'no_klaim', label: 'No Klaim' },
    { kode: 'no_polis', label: 'No Polis' },
    { kode: 'nama_tertanggung', label: 'Nama Tertanggung' },
  ],
  tipe_dokumen: [
    { kode: '0001', label: 'Dokumen Klaim' },
    { kode: '0002', label: 'Dokumen Survey' },
  ],
  jenis_dokumen: [
    { kode: '000101', label: 'Laporan Kerugian', kode_tipe_dokumen: '0001' },
    { kode: '000201', label: 'Berita Acara Survey', kode_tipe_dokumen: '0002' },
  ],
  cakupan_cabang: { lini_disembunyikan: ['002', '005'] },
  portal: 'ASM',
}

const ARCHIVE_FILE: ArchiveFile = {
  id: 1,
  nomor_klaim: 'PNC-100001',
  nomor_polis: 'POL-CONTOH-0001',
  nama_tertanggung: 'PT SUMBER CONTOH SENTOSA',
  tanggal_kejadian: '2026-03-12',
  pic_teknis: 'PIC Teknik Contoh A',
  tanggal_terima_dokumen: '2026-03-20',
  tanggal_input: '2026-03-21',
  jumlah_lembar: 24,
  kode_tipe_dokumen: '0001',
  tipe_dokumen: 'Dokumen Klaim',
  kode_jenis_dokumen: '000101',
  jenis_dokumen: 'Laporan Kerugian',
  nama_box: 'BOX-A-01',
  kode_filling: 'FIL-2026-001',
  user_input: 'adminpnc',
  tanggal_kirim_dokumen: null,
  group_panel: '006',
  sudah_dikirim: false,
  kode_layanan: '',
  catatan_layanan: '',
}

const CLAIM: ClaimCandidate = {
  nomor_klaim: 'PNC-100006',
  nomor_polis: 'POL-CONTOH-0006',
  nama_tertanggung: 'PT CONTOH KEDUA',
  tanggal_kejadian: '2026-07-21',
  bisnis: 'Fire / Property',
  cabang: 'Cabang Contoh Pusat',
  status: 'Resolved-Completed',
  posisi_klaim: 'Close Claim',
  tanggal_close: '2026-08-30',
  catatan_close: 'Selesai dibayarkan',
  pic_teknis: 'PIC Teknik Contoh A',
  group_panel: '006',
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

/** Peladen tiruan yang menjawab seluruh rute modul dengan hasil berisi. */
function stubDefaultFetch() {
  stubFetch((url) => {
    if (url === OPEN_PATH) return jsonResponse(200, OPENED)

    if (url.startsWith(`${PATH}/klaim`)) {
      return jsonResponse(200, { klaim: [CLAIM], portal: 'ASM' })
    }
    if (url.startsWith(`${PATH}/kode-filling`)) {
      return jsonResponse(200, {
        kode: [{ kode: 'FIL-2026-001', nama_box: 'BOX-A-01', jumlah_pemakaian: 2 }],
        dari_pemakaian: true,
        portal: 'ASM',
      })
    }
    if (url.startsWith(`${PATH}/kirim-cabang`)) {
      return jsonResponse(200, {
        berkas: [ARCHIVE_FILE],
        halaman: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
        cakupan_cabang: OPENED.cakupan_cabang,
        portal: 'ASM',
      })
    }
    if (url.startsWith(`${PATH}?`)) {
      return jsonResponse(200, {
        berkas: [ARCHIVE_FILE],
        halaman: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    }
    return jsonResponse(200, {
      id: 6,
      baru: true,
      pesan: 'Berkas arsip tersimpan. Berkas juga dikirim ke sistem Arsip.',
      terkirim: true,
      kode_layanan: '200',
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
      <MemoryRouter initialEntries={['/archive-dokumen-klaim']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** renderOpened menggambar layar lalu MENUNGGU isi dropdown tiba. */
async function renderOpened() {
  renderPage()
  await screen.findByRole('heading', { name: 'ARCHIVE FILE KLAIM' })
}

function lastCallStartingWith(prefix: string): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(prefix))
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

describe('pencarian berkas arsip', () => {
  it('tidak menembak server sebelum tombol Cari ditekan', async () => {
    stubDefaultFetch()
    await renderOpened()

    expect(lastCallStartingWith(`${PATH}?`)).toBeUndefined()
  })

  it('mengirim mode kata kunci beserta nilainya', async () => {
    stubDefaultFetch()
    await renderOpened()

    await userEvent.type(screen.getByLabelText('Keyword'), 'BOX-A-01')
    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

    expect(await screen.findByText('PNC-100001')).toBeInTheDocument()

    const call = lastCallStartingWith(`${PATH}?`)
    expect(call?.url).toContain('mode=kata_kunci')
    expect(call?.url).toContain('kata_kunci=BOX-A-01')
  })

  /**
   * Mengganti mode MEMBERSIHKAN isian lain.
   *
   * Nilai sisa dari mode sebelumnya yang ikut terkirim adalah kelas cacat yang tidak ada
   * di sistem lama — di sana tiap isian punya properti klipboardnya sendiri.
   */
  it('mengganti mode ke Tgl Input menukar isiannya dan tidak membawa kata kunci', async () => {
    stubDefaultFetch()
    await renderOpened()

    await userEvent.type(screen.getByLabelText('Keyword'), 'BOX-A-01')
    await userEvent.selectOptions(
      screen.getByLabelText('Tipe Pencarian Archive'),
      'tanggal_input',
    )

    expect(screen.queryByLabelText('Keyword')).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Tgl Input Dari'), '2026-03-01')
    await userEvent.type(screen.getByLabelText('Tgl Input Sampai'), '2026-03-31')
    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

    await screen.findByText('PNC-100001')

    const call = lastCallStartingWith(`${PATH}?`)
    expect(call?.url).toContain('mode=tanggal_input')
    expect(call?.url).toContain('tanggal_dari=2026-03-01')
    expect(call?.url).not.toContain('kata_kunci')
  })

  /**
   * Galat validasi ditandai PADA ISIANNYA, bukan sebagai satu kotak di atas formulir.
   *
   * Backend mengirim seluruh pelanggaran sekaligus beserta nama isiannya; membuangnya di
   * layar akan membuat pengguna menebak kalimat mana milik isian mana.
   */
  it('menandai isian yang ditolak server', async () => {
    stubFetch((url) => {
      if (url === OPEN_PATH) return jsonResponse(200, OPENED)
      return jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Isian belum benar.',
        detail: [{ field: 'kata_kunci', pesan: 'Isi Keyword yang dicari.' }],
      })
    })
    await renderOpened()

    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

    expect(await screen.findByText('Isi Keyword yang dicari.')).toBeInTheDocument()
  })
})

describe('input data archive', () => {
  async function openInputTab() {
    await renderOpened()
    await userEvent.click(screen.getByRole('button', { name: 'Input Data Archive' }))
  }

  it('memilih klaim membuka formulir berisi keterangan klaim itu', async () => {
    stubDefaultFetch()
    await openInputTab()

    await userEvent.type(screen.getByLabelText('Keyword'), 'PNC-100006')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Klaim' }))

    await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

    expect(screen.getByRole('heading', { name: 'Berkas Archive' })).toBeInTheDocument()

    // Dicari pada keterangan baca-saja formulir, BUKAN sekadar "ada di halaman": nama
    // tertanggung memang muncul dua kali — sekali di baris grid, sekali di formulir — dan
    // pencarian yang tidak membedakannya akan tetap lulus meski formulirnya kosong.
    const term = screen.getByText('Nama Tertanggung', { selector: 'dt' })
    expect(term.parentElement).toHaveTextContent('PT CONTOH KEDUA')

    expect(screen.getByLabelText('Jumlah Lembar')).toBeInTheDocument()
  })

  /**
   * Jenis Dokumen SELALU milik satu Tipe Dokumen.
   *
   * Kueri lama menggabungkannya dengan INNER JOIN pada `DOC_TYPE_ID`. Pasangan yang tidak
   * ada di master akan tersimpan dan kemudian tampil kosong di grid, tanpa ada yang tahu
   * sebabnya.
   */
  it('jenis dokumen hanya berisi jenis milik tipe yang dipilih', async () => {
    stubDefaultFetch()
    await openInputTab()

    await userEvent.type(screen.getByLabelText('Keyword'), 'PNC-100006')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Klaim' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

    const kind = screen.getByLabelText('Jenis Dokumen')
    expect(kind).toBeDisabled()

    await userEvent.selectOptions(screen.getByLabelText('Tipe Dokumen'), '0002')

    expect(within(kind).getByRole('option', { name: 'Berita Acara Survey' })).toBeInTheDocument()
    expect(within(kind).queryByRole('option', { name: 'Laporan Kerugian' })).toBeNull()
  })

  it('menyimpan berkas mengirim isian klaim beserta group panel-nya', async () => {
    stubDefaultFetch()
    await openInputTab()

    await userEvent.type(screen.getByLabelText('Keyword'), 'PNC-100006')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Klaim' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

    await userEvent.type(screen.getByLabelText('Tgl Terima Dokumen'), '2026-08-01')
    await userEvent.type(screen.getByLabelText('Jumlah Lembar'), '12')
    await userEvent.selectOptions(screen.getByLabelText('Tipe Dokumen'), '0001')
    await userEvent.selectOptions(screen.getByLabelText('Jenis Dokumen'), '000101')
    await userEvent.type(screen.getByLabelText('Nama BOX'), 'BOX-C-09')
    await userEvent.type(screen.getByLabelText('Kode Filling'), 'FIL-2026-009')

    await userEvent.click(screen.getByRole('button', { name: 'Save To Archive' }))

    const call = [...calls].reverse().find((c) => c.url === PATH && c.init?.method === 'POST')
    expect(call).toBeDefined()

    const body = JSON.parse(String(call?.init?.body))
    expect(body.nomor_klaim).toBe('PNC-100006')
    expect(body.jumlah_lembar).toBe(12)
    expect(body.kode_filling).toBe('FIL-2026-009')
    expect(body.group_panel).toBe('006')
    expect(body.id).toBe(0)

    // Menyimpan ikut mengirim ke sistem Arsip — perilaku sistem lama yang direplikasi.
    // Pesannya harus menyebutkannya, karena berkas yang tersimpan TETAPI belum terkirim
    // menuntut tindakan lanjutan dari tab Kirim ke Cabang.
    expect(await screen.findByText(/dikirim ke sistem Arsip/)).toBeInTheDocument()
  })

  /**
   * Pemilih Kode Filling MENYATAKAN bahwa daftarnya disusun dari pemakaian, bukan master.
   *
   * Masternya tidak ada di export (`R-16`). Tanpa keterangan itu, pengguna akan
   * menyimpulkan kode barunya ditolak.
   */
  it('pemilih kode menyatakan daftarnya disusun dari pemakaian', async () => {
    stubDefaultFetch()
    await openInputTab()

    await userEvent.type(screen.getByLabelText('Keyword'), 'PNC-100006')
    await userEvent.click(screen.getByRole('button', { name: 'Cari Klaim' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

    await userEvent.click(screen.getByRole('button', { name: 'Pilih Kode' }))

    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText(/bukan\s+dari master/i)).toBeInTheDocument()

    await userEvent.click(await screen.findByRole('button', { name: 'Pilih' }))

    expect(screen.getByLabelText('Kode Filling')).toHaveValue('FIL-2026-001')
    expect(screen.getByLabelText('Nama BOX')).toHaveValue('BOX-A-01')
  })
})

/**
 * Grid arsip BACA-SAJA — tidak ada tombol Ubah.
 *
 * Di seluruh export, penanda `flags` pada prosedur simpan hanya pernah disetel `"insert"`;
 * cabang `update`-nya tidak pernah dipanggil dari layar ini. Keputusan Work Owner
 * 2026-09-25: samakan dengan Pega.
 */
describe('grid arsip baca-saja', () => {
  it('tidak menggambar tombol ubah pada baris mana pun', async () => {
    stubDefaultFetch()
    await renderOpened()

    await userEvent.type(screen.getByLabelText('Keyword'), 'BOX-A-01')
    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

    expect(await screen.findByText('PNC-100001')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Ubah' })).toBeNull()
  })
})

describe('export to excel', () => {
  it('mengirim penyaring yang sama dengan daftar yang tampil', async () => {
    stubFetch((url) => {
      if (url === OPEN_PATH) return jsonResponse(200, OPENED)
      if (url.startsWith(`${PATH}/ekspor`)) {
        return new Response(['NO KLAIM', 'PNC-100001', ''].join('\n'), {
          status: 200,
          headers: {
            'Content-Type': 'text/csv; charset=utf-8',
            'Content-Disposition': 'attachment; filename="archive-dokumen-klaim-20260925.csv"',
          },
        })
      }
      return jsonResponse(200, {
        berkas: [ARCHIVE_FILE],
        halaman: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
        portal: 'ASM',
      })
    })

    // `createObjectURL` tidak ada di jsdom; tanpa ini unduhannya melempar.
    const createURL = vi.fn(() => 'blob:uji')
    const revokeURL = vi.fn()
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: createURL, revokeObjectURL: revokeURL }))

    await renderOpened()
    await userEvent.type(screen.getByLabelText('Keyword'), 'BOX-A-01')
    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))
    await screen.findByText('PNC-100001')

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    const call = lastCallStartingWith(`${PATH}/ekspor`)
    expect(call?.url).toContain('mode=kata_kunci')
    expect(call?.url).toContain('kata_kunci=BOX-A-01')

    // Blob dilepas setelah dipakai; tanpa itu setiap ekspor menyisakan satu di memori.
    expect(revokeURL).toHaveBeenCalled()
  })
})

describe('kirim ke cabang', () => {
  async function openBranchTab() {
    await renderOpened()
    await userEvent.click(screen.getByRole('button', { name: 'Kirim ke Cabang' }))
  }

  /**
   * Cakupan lini bisnis DIUMUMKAN di layar.
   *
   * Saringannya tampak terbalik dan direplikasi dengan sengaja; tanpa keterangan, berkas
   * yang hilang dari daftar akan dilaporkan berulang kali sebagai kerusakan modul.
   */
  it('menerangkan lini bisnis yang disembunyikan', async () => {
    stubDefaultFetch()
    await openBranchTab()

    expect(await screen.findByText(/tidak ditampilkan/)).toBeInTheDocument()
    expect(screen.getByText(/Personal Accident dan Travel/)).toBeInTheDocument()
  })

  it('mengirim satu berkas dan menampilkan jawaban sistem Arsip', async () => {
    stubFetch((url, init) => {
      if (url === OPEN_PATH) return jsonResponse(200, OPENED)
      if (url.startsWith(`${PATH}/kirim-cabang`)) {
        return jsonResponse(200, {
          berkas: [ARCHIVE_FILE],
          halaman: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
          cakupan_cabang: OPENED.cakupan_cabang,
          portal: 'ASM',
        })
      }
      if (url === `${PATH}/1/kirim-cabang` && init?.method === 'POST') {
        return jsonResponse(200, {
          id: 1,
          kode_layanan: '200',
          catatan_layanan: 'OK',
          pesan: 'Berkas dikirim ke sistem Arsip.',
          portal: 'ASM',
        })
      }
      return jsonResponse(200, {})
    })
    await openBranchTab()

    await userEvent.click(await screen.findByRole('button', { name: 'Kirim' }))

    expect(await screen.findByText(/Berkas dikirim ke sistem Arsip/)).toBeInTheDocument()
  })

  /**
   * Alamat layanan yang belum terdaftar disampaikan apa adanya.
   *
   * Barisnya memang belum ada di POOLDATA.GCNM_CONNECT_REST, dan yang memasangnya DBA.
   * Pesan umum "terjadi kesalahan" akan membuat kegagalan ini dilaporkan sebagai bug
   * aplikasi.
   */
  it('menyampaikan alamat layanan Arsip yang belum terdaftar', async () => {
    stubFetch((url, init) => {
      if (url === OPEN_PATH) return jsonResponse(200, OPENED)
      if (url.startsWith(`${PATH}/kirim-cabang`)) {
        return jsonResponse(200, {
          berkas: [ARCHIVE_FILE],
          halaman: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
          cakupan_cabang: OPENED.cakupan_cabang,
          portal: 'ASM',
        })
      }
      if (init?.method === 'POST') {
        return jsonResponse(503, {
          kode: 'alamat_layanan_arsip_kosong',
          pesan:
            'Alamat layanan Arsip belum terdaftar untuk portal ini. Mintakan penambahan barisnya pada tabel alamat layanan kepada DBA.',
        })
      }
      return jsonResponse(200, {})
    })
    await openBranchTab()

    await userEvent.click(await screen.findByRole('button', { name: 'Kirim' }))

    expect(await screen.findByText(/belum terdaftar untuk portal ini/)).toBeInTheDocument()
  })
})
