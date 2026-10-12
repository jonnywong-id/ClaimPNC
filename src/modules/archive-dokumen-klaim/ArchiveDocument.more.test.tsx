import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ArchiveDocumentPage } from './ArchiveDocumentPage'
import { ArchiveFormPanel } from './ArchiveFormPanel'
import { BranchQueue } from './BranchQueue'
import { ClaimPicker } from './ClaimPicker'
import { FillingCodePicker } from './FillingCodePicker'
import {
  EMPTY_ARCHIVE_FORM,
  EMPTY_CLAIM_SEARCH,
  type ArchiveFile,
  type ClaimCandidate,
  type OpenResponse,
} from './types'

const PATH = '/api/arsip-dokumen'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/** Seluruh data uji KARANGAN — `D-69`. */
const OPENED: OpenResponse = {
  tipe_pencarian: [
    { kode: 'no_klaim', label: 'No Klaim' },
    { kode: 'nama_box', label: 'Nama BOX' },
    { kode: 'tertanggung', label: 'Tertanggung' },
  ],
  tipe_input: [
    { kode: 'no_klaim', label: 'No Klaim' },
    { kode: 'no_polis', label: 'No Polis' },
  ],
  tipe_dokumen: [{ kode: '0001', label: 'Dokumen Klaim' }],
  jenis_dokumen: [{ kode: '000101', label: 'Laporan Kerugian', kode_tipe_dokumen: '0001' }],
  cakupan_cabang: { lini_disembunyikan: ['002', '777'] },
  portal: 'ASM',
}

function file(overrides: Partial<ArchiveFile> = {}): ArchiveFile {
  return {
    id: 1,
    nomor_klaim: 'PNC-100001',
    nomor_polis: 'POL-CONTOH-0001',
    nama_tertanggung: 'PT CONTOH',
    tanggal_kejadian: '2026-03-12',
    pic_teknis: 'PIC Contoh',
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
    ...overrides,
  }
}

const CLAIM: ClaimCandidate = {
  nomor_klaim: 'PNC-100006',
  nomor_polis: 'POL-CONTOH-0006',
  nama_tertanggung: 'PT CONTOH KEDUA',
  tanggal_kejadian: null,
  bisnis: 'Fire / Property',
  cabang: 'Cabang Contoh',
  status: 'Resolved-Completed',
  posisi_klaim: '',
  tanggal_close: null,
  catatan_close: '',
  pic_teknis: 'PIC Contoh',
  group_panel: '',
}

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

type Call = { url: string; init: RequestInit | undefined }
type Answer = Response | Error | null

let calls: Call[] = []
let clicked: string[] = []

function page(berkas: ArchiveFile[], halaman = 1, total = berkas.length, totalPage = 1) {
  return {
    berkas,
    halaman: { halaman, ukuran: 20, total, total_halaman: totalPage },
    cakupan_cabang: OPENED.cakupan_cabang,
    portal: 'ASM',
  }
}

/** stub menjawab permintaan; `null` memakai jawaban bawaan, `Error` menolak permintaan. */
function stub(answer: (url: string, init?: RequestInit) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    const chosen = answer(url, init)
    if (chosen instanceof Error) return Promise.reject(chosen)
    if (chosen) return Promise.resolve(chosen)
    if (url === `${PATH}/buka`) return Promise.resolve(json(200, OPENED))
    if (url.startsWith(`${PATH}/klaim`)) {
      return Promise.resolve(json(200, { klaim: [CLAIM], portal: 'ASM' }))
    }
    return Promise.resolve(json(200, page([file()])))
  })
}

function wrapper(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

async function renderOpened() {
  render(wrapper(<ArchiveDocumentPage />))
  await waitFor(() => expect(calls.some((c) => c.url === `${PATH}/buka`)).toBe(true))
}

function lastCall(prefix: string): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(prefix))
}

beforeEach(() => {
  calls = []
  clicked = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
  vi.stubGlobal(
    'URL',
    Object.assign(URL, { createObjectURL: () => 'blob:uji', revokeObjectURL: () => undefined }),
  )
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push(this.download)
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('ArchiveDocumentPage — keadaan awal', () => {
  it('meminta pengguna memilih entitas lebih dulu', () => {
    useSelectedPortal.getState().clear()
    stub(() => null)
    render(wrapper(<ArchiveDocumentPage />))

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(calls).toEqual([])
  })

  it('menyatakan layar tidak dapat dibuka dengan pesan dari server', async () => {
    stub((url) => (url === `${PATH}/buka` ? json(500, { kode: 'x', pesan: 'Buka gagal.' }) : null))
    render(wrapper(<ArchiveDocumentPage />))

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Buka gagal.')).toBeInTheDocument()
  })
})

describe('ArchiveDocumentPage — pencarian dan ekspor', () => {
  async function searchKeyword(keyword = '') {
    await renderOpened()
    const user = userEvent.setup()
    if (keyword) await user.type(screen.getByLabelText('Keyword'), keyword)
    await user.click(screen.getByRole('button', { name: 'Cari' }))
    return user
  }

  it('menampilkan galat non-validasi di tabel', async () => {
    stub((url) =>
      url.startsWith(`${PATH}?`) ? json(500, { kode: 'x', pesan: 'Cari gagal.' }) : null,
    )
    await searchKeyword('BOX')

    expect(await screen.findByText('Cari gagal.')).toBeInTheDocument()
  })

  it('memakai kode atau tanda pisah bila nama tipe, jenis, dan tanggal kosong', async () => {
    stub((url) =>
      url.startsWith(`${PATH}?`)
        ? json(
            200,
            page([
              file({
                id: 1,
                tipe_dokumen: '',
                jenis_dokumen: '',
                tanggal_kejadian: null,
                tanggal_terima_dokumen: null,
                tanggal_input: null,
              }),
              file({
                id: 2,
                tipe_dokumen: '',
                kode_tipe_dokumen: '',
                jenis_dokumen: '',
                kode_jenis_dokumen: '',
              }),
            ]),
          )
        : null,
    )
    await searchKeyword()

    const table = await screen.findByRole('table')
    const rows = await within(table).findAllByRole('row')
    expect(within(rows[1]!).getByText('0001')).toBeInTheDocument()
    expect(within(rows[1]!).getByText('000101')).toBeInTheDocument()
    expect(within(rows[1]!).getAllByText('—')).toHaveLength(4)
    expect(within(rows[2]!).getAllByText('—')).toHaveLength(3)
    // Kata kunci kosong tidak dikirim — DAN kolomnya ikut tidak dikirim, karena kolom tanpa
    // kata kunci tidak menyaring apa pun. Dulu parameter `mode` tetap terkirim; penyaring
    // mode itu sudah tidak ada (2026-10-03).
    expect(lastCall(`${PATH}?`)?.url).toBe(`${PATH}?`)
  })

  it('meminta halaman kedua lewat bilah halaman', async () => {
    stub((url) => {
      if (!url.startsWith(`${PATH}?`)) return null
      const second = url.includes('halaman=2')
      return json(
        200,
        page(
          [file({ id: second ? 2 : 1, nomor_klaim: second ? 'PNC-2' : 'PNC-1' })],
          second ? 2 : 1,
          40,
          2,
        ),
      )
    })
    const user = await searchKeyword()

    await screen.findByText('PNC-1')
    await user.click(screen.getAllByRole('button', { name: 'Halaman berikutnya' })[0]!)
    expect(await screen.findByText('PNC-2')).toBeInTheDocument()
    expect(lastCall(`${PATH}?`)?.url).toContain('halaman=2')
  })

  it('mengekspor tanpa penyaring dengan nama berkas cadangan', async () => {
    stub((url) => (url.startsWith(`${PATH}/ekspor`) ? new Response('x', { status: 200 }) : null))
    await renderOpened()
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Cari' }))
    // Tanpa satu pun isian, tidak ada parameter yang dikirim.
    expect(lastCall(`${PATH}?`)?.url).toBe(`${PATH}?`)

    await screen.findByText('PNC-100001')
    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    await waitFor(() => expect(clicked).toEqual(['archive-dokumen-klaim.csv']))
    expect(lastCall(`${PATH}/ekspor`)?.url).toBe(`${PATH}/ekspor?`)
  })

  it('mengirim kedua tanggal pada ekspor', async () => {
    stub((url) => (url.startsWith(`${PATH}/ekspor`) ? new Response('x', { status: 200 }) : null))
    await renderOpened()
    const user = userEvent.setup()

    await user.type(screen.getByLabelText('Tgl Input Dari'), '2026-03-01')
    await user.type(screen.getByLabelText('Tgl Input Sampai'), '2026-03-31')
    await user.click(screen.getByRole('button', { name: 'Cari' }))
    await screen.findByText('PNC-100001')
    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    await waitFor(() => expect(clicked).toHaveLength(1))
    expect(lastCall(`${PATH}/ekspor`)?.url).toBe(
      `${PATH}/ekspor?tanggal_dari=2026-03-01&tanggal_sampai=2026-03-31`,
    )
  })

  it.each([
    ['pesan dari server', json(403, { pesan: 'Ekspor dilarang.' }), 'Ekspor dilarang.'],
    [
      'pesan bawaan bila galatnya bukan JSON',
      new Response('x', { status: 500 }),
      'Berkas ekspor tidak dapat diambil.',
    ],
    ['pesan umum bila galatnya tanpa pesan', new Error(''), 'Terjadi kesalahan pada sistem.'],
  ])('menampilkan galat ekspor dengan %s', async (_label, answer, message) => {
    stub((url) => (url.startsWith(`${PATH}/ekspor`) ? answer : null))
    const user = await searchKeyword()

    await screen.findByText('PNC-100001')
    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    expect(await screen.findByText('Berkas ekspor tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getAllByText(message).length).toBeGreaterThanOrEqual(1)
    expect(clicked).toEqual([])
  })
})

describe('ArchiveDocumentPage — input data archive', () => {
  async function pickClaim() {
    await renderOpened()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Cari Klaim' }))
    await user.click(await screen.findByRole('button', { name: 'Detail' }))
    return user
  }

  it('menampilkan galat pencarian klaim yang bukan galat validasi', async () => {
    stub((url) =>
      url.startsWith(`${PATH}/klaim`) ? json(500, { kode: 'x', pesan: 'Klaim gagal.' }) : null,
    )
    await renderOpened()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Cari Klaim' }))

    expect(await screen.findByText('Klaim gagal.')).toBeInTheDocument()
  })

  it('menandai isian dari detail berkunci kolom dan mengabaikan detail tanpa nama', async () => {
    stub((url) =>
      url.startsWith(`${PATH}/klaim`)
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'Isian belum benar.',
            detail: [
              { kolom: 'keyword_klaim', pesan: 'Isi Keyword klaim.' },
              { pesan: 'Tanpa nama.' },
            ],
          })
        : null,
    )
    await renderOpened()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Cari Klaim' }))

    expect(await screen.findByText('Isi Keyword klaim.')).toBeInTheDocument()
    expect(screen.queryByText('Tanpa nama.')).not.toBeInTheDocument()
    // Galat validasi tidak digambar sebagai galat tabel.
    expect(screen.queryByText('Isian belum benar.')).not.toBeInTheDocument()
  })

  it('mengirim tanggal kosong sebagai null dan lembar kosong sebagai nol', async () => {
    stub((url, init) =>
      url === PATH && init?.method === 'POST'
        ? json(200, {
            id: 9,
            baru: true,
            pesan: 'Berkas tersimpan.',
            terkirim: false,
            kode_layanan: '',
            portal: 'ASM',
          })
        : null,
    )
    const user = await pickClaim()

    // Klaim tanpa tanggal kejadian dan group panel digambar dengan tanda pisah.
    expect(screen.getByText('Tgl Kejadian', { selector: 'dt' }).parentElement).toHaveTextContent(
      '—',
    )
    expect(screen.getByText('Group Panel', { selector: 'dt' }).parentElement).toHaveTextContent('—')

    await user.click(screen.getByRole('button', { name: 'Save To Archive' }))

    expect(await screen.findByText('Berkas tersimpan.')).toBeInTheDocument()
    const body = JSON.parse(
      String(calls.find((c) => c.url === PATH && c.init?.method === 'POST')?.init?.body),
    )
    expect(body.tanggal_terima_dokumen).toBeNull()
    expect(body.jumlah_lembar).toBe(0)
  })

  it('menandai isian yang ditolak saat menyimpan', async () => {
    stub((url, init) =>
      url === PATH && init?.method === 'POST'
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'Isian belum benar.',
            detail: [{ field: 'jumlah_lembar', pesan: 'Jumlah Lembar wajib diisi.' }],
          })
        : null,
    )
    const user = await pickClaim()
    await user.click(screen.getByRole('button', { name: 'Save To Archive' }))

    expect(await screen.findByText('Jumlah Lembar wajib diisi.')).toBeInTheDocument()
    expect(screen.queryByText('Berkas tidak tersimpan')).not.toBeInTheDocument()
  })

  it('menampilkan penolakan umum yang tidak menunjuk isian', async () => {
    stub((url, init) =>
      url === PATH && init?.method === 'POST'
        ? json(409, { kode: 'x', pesan: 'Berkas sudah ada.' })
        : null,
    )
    const user = await pickClaim()
    await user.click(screen.getByRole('button', { name: 'Save To Archive' }))

    expect(await screen.findByText('Berkas tidak tersimpan')).toBeInTheDocument()
    expect(screen.getByText('Berkas sudah ada.')).toBeInTheDocument()
  })

  it('menutup formulir lewat Batal dan kembali ke petunjuk awal', async () => {
    stub(() => null)
    const user = await pickClaim()

    expect(screen.getByRole('button', { name: 'Terpilih' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('heading', { name: 'Berkas Archive' })).not.toBeInTheDocument()
    expect(screen.getByText(/tekan Detail pada barisnya/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Detail' })).toBeInTheDocument()
  })
})

describe('ArchiveDocumentPage — kirim ke cabang', () => {
  async function openBranch() {
    await renderOpened()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Dokumen Cabang' }))
    return user
  }

  it('menampilkan galat daftar kirim cabang', async () => {
    stub((url) =>
      url.startsWith(`${PATH}/kirim-cabang`)
        ? json(500, { kode: 'x', pesan: 'Daftar gagal.' })
        : null,
    )
    await openBranch()

    expect(await screen.findByText('Daftar gagal.')).toBeInTheDocument()
  })

  it('menulis tanda pisah saat sistem Arsip tidak mengirim kode jawaban', async () => {
    stub((_url, init) =>
      init?.method === 'POST'
        ? json(200, {
            id: 1,
            kode_layanan: '',
            catatan_layanan: '',
            pesan: 'Terkirim.',
            portal: 'ASM',
          })
        : null,
    )
    const user = await openBranch()

    await user.click(await screen.findByRole('button', { name: 'Kirim' }))
    expect(await screen.findByText('Terkirim. Jawaban sistem Arsip: —.')).toBeInTheDocument()
  })

  it('menampilkan galat pengiriman', async () => {
    stub((_url, init) => (init?.method === 'POST' ? new TypeError('Failed to fetch') : null))
    const user = await openBranch()

    await user.click(await screen.findByRole('button', { name: 'Kirim' }))
    expect(await screen.findByText('Berkas tidak terkirim')).toBeInTheDocument()
    expect(screen.getByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
  })

  it('meminta halaman kedua antrean kirim cabang', async () => {
    stub((url) => {
      if (!url.startsWith(`${PATH}/kirim-cabang`)) return null
      const second = url.includes('halaman=2')
      return json(
        200,
        page(
          [file({ id: second ? 2 : 1, nomor_klaim: second ? 'PNC-B2' : 'PNC-B1' })],
          second ? 2 : 1,
          40,
          2,
        ),
      )
    })
    const user = await openBranch()

    await screen.findByText('PNC-B1')
    await user.click(screen.getAllByRole('button', { name: 'Halaman berikutnya' })[0]!)
    expect(await screen.findByText('PNC-B2')).toBeInTheDocument()
  })
})

describe('komponen bagian layar', () => {
  it('BranchQueue memakai kode lini apa adanya bila namanya tidak dikenal', () => {
    render(
      wrapper(
        <BranchQueue
          files={[file({ group_panel: '777' })]}
          page={undefined}
          currentPage={1}
          onPageChange={() => undefined}
          scope={undefined}
          isLoading={false}
          error=""
          onSend={() => undefined}
          sendingID={1}
          sendError=""
          sendMessage=""
        />,
      ),
    )

    const table = screen.getByRole('table')
    expect(within(table).getByText('777')).toBeInTheDocument()
    expect(within(table).getByRole('button', { name: 'Mengirim…' })).toBeDisabled()
    expect(screen.queryByText(/disaring menurut jabatan/)).not.toBeInTheDocument()
  })

  it('BranchQueue menyebut kode lini tersembunyi yang tidak dikenal apa adanya', () => {
    render(
      wrapper(
        <BranchQueue
          files={[]}
          page={undefined}
          currentPage={1}
          onPageChange={() => undefined}
          scope={{ lini_disembunyikan: ['002', '777'] }}
          isLoading={false}
          error=""
          onSend={() => undefined}
          sendingID={null}
          sendError=""
          sendMessage=""
        />,
      ),
    )

    expect(screen.getByText('Personal Accident dan 777')).toBeInTheDocument()
  })

  it('ClaimPicker memberi petunjuk khusus pencarian No Polis', () => {
    render(
      wrapper(
        <ClaimPicker
          types={OPENED.tipe_input}
          form={{ ...EMPTY_CLAIM_SEARCH, tipe: 'no_polis' }}
          onChange={() => undefined}
          onSubmit={() => undefined}
          claims={[]}
          isLoading
          error=""
          onPick={() => undefined}
          pickedNumber=""
          fieldError={{}}
          searched={false}
        />,
      ),
    )

    expect(screen.getByText('Hanya No Polis yang dicocokkan.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Mencari…' })).toBeDisabled()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('ClaimPicker memformat tanggal close yang terisi', () => {
    render(
      wrapper(
        <ClaimPicker
          types={OPENED.tipe_input}
          form={EMPTY_CLAIM_SEARCH}
          onChange={() => undefined}
          onSubmit={() => undefined}
          claims={[
            {
              ...CLAIM,
              tanggal_close: '2026-08-30',
              posisi_klaim: 'Close',
              catatan_close: 'Selesai',
            },
          ]}
          isLoading={false}
          error=""
          onPick={() => undefined}
          pickedNumber=""
          fieldError={{}}
          searched
        />,
      ),
    )

    expect(screen.getByText('30 Agustus 2026')).toBeInTheDocument()
    expect(screen.getByText('Selesai')).toBeInTheDocument()
  })

  it('ArchiveFormPanel menandai mode ubah dan menampilkan pesan berhasil', () => {
    render(
      wrapper(
        <ArchiveFormPanel
          form={{
            ...EMPTY_ARCHIVE_FORM,
            id: 7,
            nomor_klaim: 'PNC-7',
            tanggal_kejadian: '2026-03-12',
          }}
          onChange={() => undefined}
          onSubmit={() => undefined}
          onCancel={() => undefined}
          documentTypes={[]}
          documentKinds={[]}
          busy
          fieldError={{}}
          successMessage="Tersimpan."
        />,
      ),
    )

    expect(screen.getByRole('heading', { name: 'Ubah Berkas Archive' })).toBeInTheDocument()
    expect(screen.getByText('Mengubah berkas nomor 7')).toBeInTheDocument()
    expect(screen.getByText('Tersimpan.')).toBeInTheDocument()
    expect(screen.getByText('12 Maret 2026')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Menyimpan…' })).toBeDisabled()
  })

  it('ArchiveFormPanel mempertahankan Nama BOX bila kode terpilih tidak punya nama box', async () => {
    stub((url) =>
      url.startsWith(`${PATH}/kode-filling`)
        ? json(200, {
            kode: [{ kode: 'FIL-9', nama_box: '', jumlah_pemakaian: 1 }],
            dari_pemakaian: true,
            portal: 'ASM',
          })
        : null,
    )
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(
      wrapper(
        <ArchiveFormPanel
          form={{
            ...EMPTY_ARCHIVE_FORM,
            nomor_klaim: 'PNC-1',
            nama_box: 'BOX-LAMA',
            kode_tipe_dokumen: '0001',
          }}
          onChange={onChange}
          onSubmit={() => undefined}
          onCancel={() => undefined}
          documentTypes={OPENED.tipe_dokumen}
          documentKinds={OPENED.jenis_dokumen}
          busy={false}
          fieldError={{}}
          successMessage=""
        />,
      ),
    )

    expect(screen.getByLabelText('Jenis Dokumen')).toBeEnabled()
    await user.click(screen.getByRole('button', { name: 'Pilih Kode' }))
    const dialog = await screen.findByRole('dialog')
    expect(await within(dialog).findByText('—')).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Pilih' }))

    expect(onChange).toHaveBeenCalledWith({ kode_filling: 'FIL-9', nama_box: 'BOX-LAMA' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('FillingCodePicker menjelaskan daftar kosong dan dapat ditutup dari kedua tombol', async () => {
    stub((url) =>
      url.startsWith(`${PATH}/kode-filling`)
        ? json(200, { kode: [], dari_pemakaian: true, portal: 'ASM' })
        : null,
    )
    const onClose = vi.fn()
    const user = userEvent.setup()
    render(wrapper(<FillingCodePicker open onClose={onClose} onPick={() => undefined} />))

    expect(screen.getByText('Memuat kode…')).toBeInTheDocument()
    expect(await screen.findByText(/Belum ada kode yang cocok/)).toBeInTheDocument()

    await user.type(screen.getByLabelText('Cari Kode'), 'X')
    await waitFor(() => expect(lastCall(`${PATH}/kode-filling`)?.url).toContain('kata_kunci=X'))

    await user.click(screen.getByRole('button', { name: 'Tutup pemilih kode' }))
    await user.click(screen.getByRole('button', { name: 'Tutup' }))
    expect(onClose).toHaveBeenCalledTimes(2)
  })
})
