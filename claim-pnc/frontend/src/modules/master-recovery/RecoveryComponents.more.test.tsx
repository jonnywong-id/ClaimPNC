import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState, type ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { RecoveryClaimLine, RecoveryPrincipal } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimLineUpload } from './ClaimLineUpload'
import { OutstandingTable } from './OutstandingTable'
import { RecoveryForm } from './RecoveryForm'
import { RecoveryPage } from './RecoveryPage'
import { VirtualAccountPanel } from './VirtualAccountPanel'

/**
 * Isian "Nama Principal" milik bagian Principal — BUKAN milik blok Generated New VA.
 *
 * Sejak blok VA duduk di dalam alur Tambah (2026-10-06), label itu muncul dua kali di
 * satu layar — persis seperti layar lama, yang juga memuatnya dua kali. Yang di bawah
 * adalah isian form.
 */
function labelTerakhir(nama: string): HTMLElement {
  const daftar = screen.getAllByLabelText(nama)
  return daftar[daftar.length - 1] as HTMLElement
}

function namaPrincipalTerakhir(): HTMLElement {
  const semua = screen.getAllByLabelText('Nama Principal')
  return semua[semua.length - 1] as HTMLElement
}

/**
 * Uji tambahan per komponen Master Recovery.
 *
 * Berbeda dari `RecoveryPage.test.tsx` yang merakit seluruh aplikasi, berkas ini merender
 * komponennya langsung supaya cabang galat dan keadaan tepi dapat dicapai satu per satu.
 * Seluruh nilai di sini KARANGAN (`D-69`).
 */

const ROUTE = '/api/master/recovery'

type Call = { url: string; init: RequestInit | undefined }
type Reply = Response | (() => Response | Promise<Response>)

let calls: Call[] = []
let replies: Record<string, Reply> = {}

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  })
}

/**
 * Jawaban yang badannya gagal dibaca — satu-satunya cara memunculkan galat yang BUKAN
 * APIError maupun NetworkError dari klien API, yaitu cabang "kesalahan tak terduga".
 */
function brokenResponse(): Response {
  return {
    status: 500,
    ok: false,
    text: () => Promise.reject(new Error('badan rusak')),
  } as unknown as Response
}

const NETWORK = 'jaringan'

/**
 * Fetch tiruan berkunci "METODE jalur". Jalur dicocokkan penuh lebih dulu, lalu awalannya
 * (untuk daftar Outstanding yang selalu membawa query string).
 */
function installFetch() {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    const method = init?.method ?? 'GET'
    const exact = replies[`${method} ${url}`]
    const prefix = Object.entries(replies).find(
      ([key]) => key.endsWith('*') && `${method} ${url}`.startsWith(key.slice(0, -1)),
    )?.[1]
    const reply = exact ?? prefix
    if (reply === undefined) {
      return Promise.resolve(jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak dilayani' }))
    }
    if (reply === (NETWORK as unknown as Reply)) return Promise.reject(new TypeError('offline'))
    return Promise.resolve(typeof reply === 'function' ? reply() : reply.clone())
  })
}

function apiError(status: number, kode: string, pesan = 'Pesan dari server.', extra = {}) {
  return () => jsonResponse(status, { kode, pesan, ...extra })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

function callsTo(method: string, url: string) {
  return calls.filter((c) => (c.init?.method ?? 'GET') === method && c.url === url)
}

const PRINCIPAL: RecoveryPrincipal[] = [
  {
    client_id: 'CONTOH-001',
    nama_principal: 'PT CONTOH SATU',
    nomor_virtual_account: '0000000000000001',
    email_inputor_va: 'contoh.satu@example.invalid',
  },
]

const FORM = { nomor_batch_perkiraan: 7, tahun: ['2027', '2026'], portal: 'ASM' }

function savedRecovery(nomorPolis = '') {
  return {
    recovery: {
      nomor_batch: 7,
      nama_principal: 'PT CONTOH SATU',
      client_id: '',
      nomor_virtual_account: '',
      tahun: '2027',
      nilai_klaim: 1000,
      pembayaran_sebelumnya: 0,
      pembayaran: 100,
      sisa: 900,
      keterangan: 'k',
      posisi_kasus: 'p',
      id_dokumen: '',
      dicatat_oleh: '90000001',
      nomor_polis: nomorPolis,
      id_lini_bisnis: '',
      id_cabang: '',
      id_agen: '',
      id_marketing: '',
      baris_klaim: [],
    },
    identitas_polis_terisi: false,
    portal: 'ASM',
  }
}

let clicked: { href: string; download: string }[] = []

beforeEach(() => {
  calls = []
  replies = {}
  clicked = []
  useSession.setState({
    token: 'token-uji',
    user: {
      identitas: '90000001',
      nama: 'Contoh Administrator',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: 'contoh.admin@example.invalid',
      perusahaan: 'ASM',
    },
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
  // jsdom tidak menyediakan URL objek maupun navigasi unduhan; keduanya ditiru.
  Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:contoh'), revokeObjectURL: vi.fn() })
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push({ href: this.href, download: this.download })
  })
  installFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.useRealTimers()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

// ── Panel Virtual Account ─────────────────────────────────────────────────────────

describe('VirtualAccountPanel', () => {
  async function fillVA(user: ReturnType<typeof userEvent.setup>, email = 'petugas@contoh.invalid') {
    await user.type(labelTerakhir('Client ID'), 'CL-9')
    await user.type(namaPrincipalTerakhir(), 'PT CONTOH VA')
    await user.type(screen.getByLabelText('Email Inputor VA'), email)
  }

  it('menandai ketiga isian wajib tanpa mengirim permintaan', async () => {
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(await screen.findByText('Client ID wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Nama principal wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Email inputor VA wajib diisi.')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('menolak surel yang tidak berbentuk alamat', async () => {
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user, 'bukan-surel')
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(
      await screen.findByText('Email inputor VA belum berupa alamat surel.'),
    ).toBeInTheDocument()
  })

  it('menampilkan nomor yang baru terbit dan meneruskannya ke form utama', async () => {
    replies[`POST ${ROUTE}/virtual-account`] = jsonResponse(201, {
      nomor_virtual_account: '8800000000000009',
      dipakai_ulang: false,
      pesan: 'Nomor baru diterbitkan.',
    })
    const onIssued = vi.fn()
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={onIssued} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(await screen.findByText('Virtual account berhasil diterbitkan')).toBeInTheDocument()
    expect(screen.getByText('8800000000000009')).toBeInTheDocument()
    expect(screen.getByText('Nomor baru diterbitkan.')).toBeInTheDocument()
    expect(onIssued).toHaveBeenCalledWith({
      client_id: 'CL-9',
      nama_principal: 'PT CONTOH VA',
      nomor: '8800000000000009',
    })
    expect(JSON.parse(String(calls[0]?.init?.body))).toEqual({
      client_id: 'CL-9',
      nama_principal: 'PT CONTOH VA',
      email_inputor_va: 'petugas@contoh.invalid',
    })
  })

  it('membedakan nomor yang dipakai ulang dari yang baru terbit', async () => {
    replies[`POST ${ROUTE}/virtual-account`] = jsonResponse(200, {
      nomor_virtual_account: '8800000000000001',
      dipakai_ulang: true,
      pesan: '',
    })
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(
      await screen.findByText('Principal ini sudah punya virtual account'),
    ).toBeInTheDocument()
    expect(screen.queryByText('Virtual account berhasil diterbitkan')).not.toBeInTheDocument()
  })

  it('menampilkan Memproses selama permintaan berjalan', async () => {
    let finish: (value: Response) => void = (_value) => undefined
    replies[`POST ${ROUTE}/virtual-account`] = () =>
      new Promise<Response>((resolve) => {
        finish = resolve
      }) as unknown as Response
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(await screen.findByRole('button', { name: 'Memproses…' })).toBeDisabled()
    expect(labelTerakhir('Client ID')).toBeDisabled()

    finish(jsonResponse(200, { nomor_virtual_account: '1', dipakai_ulang: false, pesan: '' }))
    expect(await screen.findByRole('button', { name: 'Get VA' })).toBeEnabled()
  })


  it.each([
    ['penerbit_va_belum_terdaftar', 'Layanan penerbit VA belum terdaftar'],
    ['penerbit_va_tidak_terhubung', 'Layanan penerbit VA sedang tidak dapat dihubungi'],
    ['penerbit_va_menolak', 'Permintaan ditolak layanan penerbit'],
    ['portal_tidak_disebut', 'Portal entitas belum dipilih'],
    ['portal_tidak_dikenal', 'Portal entitas belum dipilih'],
    ['portal_belum_siap', 'Basis data entitas ini belum tersedia'],
  ])('menerjemahkan kode %s menjadi judul yang dapat ditindaklanjuti', async (kode, judul) => {
    replies[`POST ${ROUTE}/virtual-account`] = apiError(502, kode)
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(await screen.findByText(judul)).toBeInTheDocument()
  })
  it('memakai pesan server untuk kode yang tidak dikenal', async () => {
    replies[`POST ${ROUTE}/virtual-account`] = apiError(500, 'galat_lain', 'Sesuatu rusak.')
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(await screen.findByText('Gagal menerbitkan virtual account')).toBeInTheDocument()
    expect(screen.getByText('Sesuatu rusak.')).toBeInTheDocument()
  })

  it('menyorot isian yang ditolak server tanpa kotak pesan ringkas', async () => {
    replies[`POST ${ROUTE}/virtual-account`] = apiError(422, 'validasi_gagal', 'Isian salah.', {
      detail: [{ field: 'client_id', pesan: 'Client ID sudah dipakai principal lain.' }],
    })
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(
      await screen.findByText('Client ID sudah dipakai principal lain.'),
    ).toBeInTheDocument()
    expect(screen.queryByText('Isian belum benar')).not.toBeInTheDocument()
  })

  it('meringkas galat validasi yang tidak menunjuk isian', async () => {
    replies[`POST ${ROUTE}/virtual-account`] = apiError(422, 'validasi_gagal', 'Badan cacat.')
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    expect(await screen.findByText('Isian belum benar')).toBeInTheDocument()
    expect(screen.getByText('Badan cacat.')).toBeInTheDocument()
  })

  it('membedakan gangguan jaringan dan kesalahan tak terduga', async () => {
    replies[`POST ${ROUTE}/virtual-account`] = NETWORK as unknown as Reply
    const user = userEvent.setup()
    const view = wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)

    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))
    expect(await screen.findByText('Tidak dapat menghubungi server')).toBeInTheDocument()
    view.unmount()

    replies[`POST ${ROUTE}/virtual-account`] = brokenResponse
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={vi.fn()} />)
    await fillVA(user)
    await user.click(screen.getByRole('button', { name: 'Get VA' }))
    expect(await screen.findByText(/kesalahan yang tidak terduga/)).toBeInTheDocument()
  })

  it('memanggil onClose saat Tutup ditekan', async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    wrap(<VirtualAccountPanel onIssued={vi.fn()} onClose={onClose} />)

    await user.click(screen.getByRole('button', { name: 'Tutup' }))

    expect(onClose).toHaveBeenCalledTimes(1)
  })
})

// ── Unggahan data klaim ───────────────────────────────────────────────────────────

describe('ClaimLineUpload', () => {
  /** Membungkus komponen dengan state sungguhan, seperti RecoveryForm memakainya. */
  function Harness({ disabled = false, initial = [] }: { disabled?: boolean; initial?: RecoveryClaimLine[] }) {
    const [line, setLine] = useState<RecoveryClaimLine[]>(initial)
    return <ClaimLineUpload claimLine={line} onChange={setLine} disabled={disabled} />
  }

  const csv = () => new File(['polis' + String.fromCharCode(10) + 'P-1'], 'klaim.csv', { type: 'text/csv' })

  it('menyebut batch tetap dapat disimpan tanpa daftar', () => {
    wrap(<Harness />)
    expect(screen.getByText(/Data Tidak Ada/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kosongkan' })).not.toBeInTheDocument()
  })

  it('membaca berkas lewat server, menjumlahkan nilai, dan menyebut baris yang ditolak', async () => {
    replies[`POST ${ROUTE}/baris-klaim`] = jsonResponse(200, {
      baris_klaim: [
        { nomor_polis: 'P-1', nilai_klaim: 1000 },
        { nomor_polis: 'P-2', nilai_klaim: 2500 },
      ],
      baris_ditolak: [{ pesan: 'Baris 4: nilai klaim bukan angka.' }],
      portal: 'ASM',
    })
    const user = userEvent.setup()
    wrap(<Harness />)

    await user.upload(screen.getByLabelText('Pilih berkas CSV data klaim'), csv())

    expect(await screen.findByText('P-1')).toBeInTheDocument()
    expect(screen.getByText('P-2')).toBeInTheDocument()
    expect(screen.getByText('Total Klaim')).toBeInTheDocument()
    expect(screen.getByText('3.500')).toBeInTheDocument()
    expect(screen.getByText('1 baris tidak dapat dibaca dan tidak ikut disimpan')).toBeInTheDocument()
    expect(screen.getByText(/Baris 4: nilai klaim bukan angka\./)).toBeInTheDocument()
    expect((calls[0]?.init?.body as FormData).get('berkas')).toBeInstanceOf(File)

    await user.click(screen.getByRole('button', { name: 'Kosongkan' }))
    expect(screen.getByText(/Data Tidak Ada/)).toBeInTheDocument()
  })

  it('tidak mengirim apa pun bila pemilih berkas ditutup tanpa memilih', () => {
    wrap(<Harness />)

    fireEvent.change(screen.getByLabelText('Pilih berkas CSV data klaim'), {
      target: { files: [] },
    })

    expect(calls).toHaveLength(0)
  })

  it('membuka pemilih berkas dari tombol unggah', async () => {
    const pick = vi.spyOn(HTMLInputElement.prototype, 'click')
    const user = userEvent.setup()
    wrap(<Harness />)

    await user.click(screen.getByRole('button', { name: 'Upload Data Klaim' }))

    expect(pick).toHaveBeenCalled()
  })

  it('mematikan tombol unggah dan kosongkan saat form sedang menyimpan', () => {
    wrap(<Harness disabled initial={[{ nomor_polis: 'P-9', nilai_klaim: 1 }]} />)

    expect(screen.getByRole('button', { name: 'Upload Data Klaim' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Kosongkan' })).toBeDisabled()
  })

  it('mengunduh berkas contoh dengan nama tetap', async () => {
    replies[`GET ${ROUTE}/format-unggahan`] = new Response('nomor_polis\n', { status: 200 })
    const user = userEvent.setup()
    wrap(<Harness />)

    await user.click(screen.getByRole('button', { name: 'Format File' }))

    await waitFor(() =>
      expect(clicked).toEqual([
        { href: 'blob:contoh', download: 'format-data-klaim-recovery.csv' },
      ]),
    )
    expect(await screen.findByRole('button', { name: 'Format File' })).toBeEnabled()
  })

  it('menampilkan Membaca dan Menyiapkan selama permintaan berjalan', async () => {
    const pending = () => new Promise<Response>(() => undefined) as unknown as Response
    replies[`POST ${ROUTE}/baris-klaim`] = pending
    replies[`GET ${ROUTE}/format-unggahan`] = pending
    const user = userEvent.setup()
    wrap(<Harness />)

    await user.upload(screen.getByLabelText('Pilih berkas CSV data klaim'), csv())
    await user.click(screen.getByRole('button', { name: 'Format File' }))

    expect(await screen.findByRole('button', { name: 'Membaca…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Menyiapkan…' })).toBeDisabled()
  })

  it.each([
    ['berkas_klaim_kosong', 'Berkas tidak memuat data'],
    ['berkas_klaim_terlalu_besar', 'Berkas terlalu besar'],
    ['berkas_klaim_tidak_terbaca', 'Bentuk berkas tidak dikenali'],
  ])('menerjemahkan kode %s', async (kode, judul) => {
    replies[`POST ${ROUTE}/baris-klaim`] = apiError(422, kode)
    const user = userEvent.setup()
    wrap(<Harness />)

    await user.upload(screen.getByLabelText('Pilih berkas CSV data klaim'), csv())

    expect(await screen.findByText(judul)).toBeInTheDocument()
  })

  it('memakai pesan server untuk kode lain, lalu membedakan jaringan dan galat tak terduga', async () => {
    replies[`POST ${ROUTE}/baris-klaim`] = apiError(500, 'galat_lain', 'Pembaca berkas mati.')
    const user = userEvent.setup()
    const first = wrap(<Harness />)
    await user.upload(screen.getByLabelText('Pilih berkas CSV data klaim'), csv())
    expect(await screen.findByText('Pembaca berkas mati.')).toBeInTheDocument()
    first.unmount()

    replies[`POST ${ROUTE}/baris-klaim`] = NETWORK as unknown as Reply
    const second = wrap(<Harness />)
    await user.upload(screen.getByLabelText('Pilih berkas CSV data klaim'), csv())
    expect(await screen.findByText(/Berkas belum terbaca/)).toBeInTheDocument()
    second.unmount()

    replies[`POST ${ROUTE}/baris-klaim`] = brokenResponse
    wrap(<Harness />)
    await user.upload(screen.getByLabelText('Pilih berkas CSV data klaim'), csv())
    expect(await screen.findByText(/kesalahan yang tidak terduga/)).toBeInTheDocument()
  })
})

// ── Form entri ─────────────────────────────────────────────────────────────────────

describe('RecoveryForm', () => {
  function showForm(onSaved = vi.fn(), year = ['2027', '2026']) {
    wrap(<RecoveryForm nextBatch={undefined} year={year} principal={PRINCIPAL} onSaved={onSaved} />)
    return onSaved
  }

  /**
   * Membuka sisa isian dengan mengunggah satu baris klaim.
   *
   * Sejak alurnya disamakan dengan Pega (2026-10-06), menekan Tambah hanya
   * memunculkan kotak Tambah Data; isian lain baru muncul setelah data klaim masuk.
   */
  async function bukaIsian(user: ReturnType<typeof userEvent.setup>) {
    replies[`POST ${ROUTE}/baris-klaim`] = jsonResponse(200, {
      baris_klaim: [{ nomor_polis: 'P-1', nilai_klaim: 1000 }],
      baris_ditolak: [],
      portal: 'ASM',
    })
    await user.upload(
      screen.getByLabelText('Pilih berkas CSV data klaim'),
      new File(['polis' + String.fromCharCode(10) + 'P-1'], 'klaim.csv', { type: 'text/csv' }),
    )
    // Isian nilai baru muncul SETELAH VA terbit — padanan TempRecovery.IBNR == 1.
    replies[`POST ${ROUTE}/virtual-account`] = jsonResponse(201, {
      nomor_virtual_account: '8800000000000009',
      dipakai_ulang: false,
      pesan: 'Nomor baru diterbitkan.',
    })
    // Isian PERTAMA: milik blok Generated New VA, yang berada di atas isian form.
    // Dipakai getAll supaya pembantu ini tetap benar saat dipanggil dua kali.
    await user.type(screen.getAllByLabelText('Client ID')[0]!, 'C-1')
    await user.type(screen.getAllByLabelText('Nama Principal')[0]!, 'PT CONTOH SATU')
    await user.type(screen.getByLabelText('Email Inputor VA'), 'a@b.invalid')
    await user.click(screen.getByRole('button', { name: 'Get VA' }))

    await screen.findByLabelText('Tahun')
  }
  async function fillRequired(user: ReturnType<typeof userEvent.setup>) {
    await user.type(namaPrincipalTerakhir(), 'PT CONTOH SATU')
    await user.type(screen.getByLabelText('Nilai Klaim'), '1000')
    await user.type(screen.getByLabelText('Pembayaran'), '100')
    await user.type(screen.getByLabelText('Keterangan'), 'k')
    await user.type(screen.getByLabelText('Posisi Kasus'), 'p')
  }

  function lastSaveBody() {
    const call = callsTo('POST', `${ROUTE}/`).at(-1)
    return JSON.parse(String(call?.init?.body ?? '{}')) as Record<string, unknown>
  }

    it('menolak nilai uang negatif', async () => {
    const user = userEvent.setup()
    showForm()
    await bukaIsian(user)

    await user.type(screen.getByLabelText('Pembayaran'), '-5')
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(await screen.findByText('Pembayaran tidak boleh negatif.')).toBeInTheDocument()
  })


  it('menyertakan baris klaim yang diunggah ke dalam permintaan simpan', async () => {
    replies[`POST ${ROUTE}/baris-klaim`] = jsonResponse(200, {
      baris_klaim: [{ nomor_polis: 'P-5', nilai_klaim: 50 }],
      portal: 'ASM',
    })
    replies[`POST ${ROUTE}/`] = jsonResponse(201, savedRecovery())
    const user = userEvent.setup()
    showForm()

    await user.upload(screen.getByLabelText('Pilih berkas CSV data klaim'), new File(['p'], 'k.csv'))
    expect(await screen.findByText('P-5')).toBeInTheDocument()

    // Isian nilai baru muncul setelah VA terbit.
    replies[`POST ${ROUTE}/virtual-account`] = jsonResponse(201, {
      nomor_virtual_account: '8800000000000009',
      dipakai_ulang: false,
      pesan: 'Nomor baru diterbitkan.',
    })
    // Isian PERTAMA: milik blok Generated New VA, yang berada di atas isian form.
    // Dipakai getAll supaya pembantu ini tetap benar saat dipanggil dua kali.
    await user.type(screen.getAllByLabelText('Client ID')[0]!, 'C-1')
    await user.type(screen.getAllByLabelText('Nama Principal')[0]!, 'PT CONTOH SATU')
    await user.type(screen.getByLabelText('Email Inputor VA'), 'a@b.invalid')
    await user.click(screen.getByRole('button', { name: 'Get VA' }))
    await screen.findByLabelText('Tahun')
    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    await waitFor(() =>
      expect(lastSaveBody()['baris_klaim']).toEqual([{ nomor_polis: 'P-5', nilai_klaim: 50 }]),
    )
    // Daftar klaim dikosongkan bersama isian setelah berhasil.
    expect(await screen.findByText(/Data Tidak Ada/)).toBeInTheDocument()
  })

  it('menampilkan Menyimpan dan mematikan isian selama penyimpanan berjalan', async () => {
    replies[`POST ${ROUTE}/`] = () => new Promise<Response>(() => undefined) as unknown as Response
    const user = userEvent.setup()
    showForm()
    await bukaIsian(user)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(await screen.findByRole('button', { name: 'Menyimpan…' })).toBeDisabled()
    expect(screen.getByLabelText('Keterangan')).toBeDisabled()
  })

  it('menyorot isian yang ditolak server di tempatnya', async () => {
    replies[`POST ${ROUTE}/`] = apiError(422, 'validasi_gagal', 'Isian salah.', {
      detail: [{ kolom: 'keterangan', pesan: 'Keterangan terlalu umum.' }],
    })
    const user = userEvent.setup()
    showForm()
    await bukaIsian(user)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(await screen.findByText('Keterangan terlalu umum.')).toBeInTheDocument()
    expect(screen.queryByText('Isian belum benar')).not.toBeInTheDocument()
  })

  it.each([
    ['validasi_gagal', 'Isian belum benar'],
    ['portal_tidak_disebut', 'Portal entitas belum dipilih'],
    ['portal_tidak_dikenal', 'Portal entitas belum dipilih'],
    ['portal_belum_siap', 'Basis data entitas ini belum tersedia'],
    ['galat_lain', 'Gagal menyimpan'],
  ])('menerjemahkan galat simpan %s', async (kode, judul) => {
    replies[`POST ${ROUTE}/`] = apiError(409, kode)
    const user = userEvent.setup()
    showForm()
    await bukaIsian(user)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(await screen.findByText(judul)).toBeInTheDocument()
  })

  it('membedakan galat simpan karena jaringan dan karena kesalahan tak terduga', async () => {
    replies[`POST ${ROUTE}/`] = NETWORK as unknown as Reply
    const user = userEvent.setup()
    showForm()
    await bukaIsian(user)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))
    expect(await screen.findByText(/Batch belum tersimpan\. Isian Anda masih ada/)).toBeInTheDocument()

    replies[`POST ${ROUTE}/`] = brokenResponse
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))
    expect(await screen.findByText(/kesalahan yang tidak terduga/)).toBeInTheDocument()
  })

  it('Kosongkan isian membuang ketikan dan galat terakhir', async () => {
    replies[`POST ${ROUTE}/`] = apiError(500, 'galat_lain')
    const user = userEvent.setup()
    showForm()
    await bukaIsian(user)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))
    expect(await screen.findByText('Gagal menyimpan')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Kosongkan isian' }))

    // Daftar klaim ikut dikosongkan, sehingga sisa isian MENUTUP kembali — persis
    // seperti keadaan sesaat setelah Tambah ditekan.
    expect(screen.queryByLabelText('Keterangan')).not.toBeInTheDocument()
    expect(screen.queryByText('Gagal menyimpan')).not.toBeInTheDocument()

    // Dibuka lagi: ketikan sebelumnya memang sudah hilang.
    await bukaIsian(user)
    expect((screen.getByLabelText('Keterangan') as HTMLInputElement).value).toBe('')
  })

  it('mewarnai sisa negatif sebagai peringatan', async () => {
    const user = userEvent.setup()
    showForm()
    await bukaIsian(user)

    await user.type(screen.getByLabelText('Nilai Klaim'), '10')
    await user.type(screen.getByLabelText('Pembayaran'), '30')

    expect((screen.getByLabelText('Sisa') as HTMLInputElement).value).toBe('-20')
  })

  it('mengisi Tahun dengan pilihan pertama begitu daftarnya tiba', async () => {
    const user = userEvent.setup()
    const view = render(
      <QueryClientProvider client={new QueryClient()}>
        <RecoveryForm nextBatch={1} year={[]} principal={[]} onSaved={vi.fn()} />
      </QueryClientProvider>,
    )
    // Isian Tahun baru ada setelah data klaim diunggah.
    await bukaIsian(user)
    expect((screen.getByLabelText('Tahun') as HTMLSelectElement).value).toBe('')

    view.rerender(
      <QueryClientProvider client={new QueryClient()}>
        <RecoveryForm nextBatch={1} year={['2030']} principal={[]} onSaved={vi.fn()} />
      </QueryClientProvider>,
    )

    await waitFor(() =>
      expect((screen.getByLabelText('Tahun') as HTMLSelectElement).value).toBe('2030'),
    )
  })
})

// ── Layar ─────────────────────────────────────────────────────────────────────────

describe('RecoveryPage', () => {
  function defaultPage() {
    replies[`GET ${ROUTE}/form`] = jsonResponse(200, FORM)
    replies[`GET ${ROUTE}/principal`] = jsonResponse(200, {
      principal: PRINCIPAL,
      total: 1,
      portal: 'ASM',
    })
    replies[`GET ${ROUTE}/?*`] = jsonResponse(200, { principal: [], total: 0, portal: 'ASM' })
  }

  it.each([
    ['portal_tidak_disebut', 'Portal entitas belum dipilih'],
    ['portal_tidak_dikenal', 'Portal entitas belum dipilih'],
    ['portal_belum_siap', 'Basis data entitas ini belum tersedia'],
    ['galat_lain', 'Layar gagal dimuat'],
  ])('menjelaskan bekal awal yang gagal dimuat: %s', async (kode, judul) => {
    defaultPage()
    replies[`GET ${ROUTE}/form`] = apiError(500, kode)
    wrap(<RecoveryPage />)

    expect(await screen.findByText(judul)).toBeInTheDocument()
  })

  it('menjelaskan gangguan jaringan dan kesalahan tak terduga saat memuat', async () => {
    defaultPage()
    replies[`GET ${ROUTE}/principal`] = NETWORK as unknown as Reply
    const view = wrap(<RecoveryPage />)
    expect(await screen.findByText(/Layar belum dapat dimuat/)).toBeInTheDocument()
    view.unmount()

    replies[`GET ${ROUTE}/principal`] = brokenResponse
    wrap(<RecoveryPage />)
    expect(await screen.findByText('Terjadi kesalahan pada sistem. Coba muat ulang.')).toBeInTheDocument()
  })

  
  
    it('menutup form lewat tombol yang sama dengan pembukanya', async () => {
    defaultPage()
    const user = userEvent.setup()
    wrap(<RecoveryPage />)

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Tutup form' }))

    expect(screen.queryByRole('form', { name: 'Catat batch recovery' })).not.toBeInTheDocument()
  })

  it('Refresh memuat ulang bekal awal, principal, dan daftar sekaligus', async () => {
    defaultPage()
    const user = userEvent.setup()
    wrap(<RecoveryPage />)

    const refresh = await screen.findByRole('button', { name: 'Refresh' })
    await waitFor(() => expect(refresh).toBeEnabled())
    const count = () => ({
      form: callsTo('GET', `${ROUTE}/form`).length,
      principal: callsTo('GET', `${ROUTE}/principal`).length,
      list: calls.filter((c) => c.url.startsWith(`${ROUTE}/?`)).length,
    })
    const before = count()

    await user.click(refresh)

    await waitFor(() => {
      const after = count()
      expect(after.form).toBeGreaterThan(before.form)
      expect(after.principal).toBeGreaterThan(before.principal)
      expect(after.list).toBeGreaterThan(before.list)
    })
    expect(await screen.findByRole('button', { name: 'Refresh' })).toBeEnabled()
  })

  it('menampilkan Memuat selama bekal awal dimuat ulang', async () => {
    defaultPage()
    replies[`GET ${ROUTE}/form`] = () => new Promise<Response>(() => undefined) as unknown as Response
    wrap(<RecoveryPage />)

    expect(await screen.findByRole('button', { name: 'Memuat…' })).toBeDisabled()
  })
})

// ── Tab Outstanding ───────────────────────────────────────────────────────────────

describe('OutstandingTable', () => {
  function batchRow(over: Record<string, unknown> = {}) {
    return {
      batch: 1,
      nama_principal: 'PT CONTOH SATU',
      tahun: '',
      tanggal_input: '',
      no_hpll: '',
      nilai_klaim: 100,
      pembayaran_sebelumnya: 0,
      pembayaran: 300,
      sisa: -200,
      keterangan: '',
      posisi_kasus: '',
      nomor_virtual_account: '',
      nomor_polis: '',
      id_dokumen: 'DOK-1',
      dokumen: { id: 'DOK-1', nama_berkas: '', input_nama: '', tanggal: 'bukan-tanggal' },
      ...over,
    }
  }

  function listOf(batch: unknown[], total = 1) {
    return jsonResponse(200, {
      principal: [
        {
          nama_principal: 'PT CONTOH SATU',
          nilai_klaim: 100,
          pembayaran_sebelumnya: 0,
          pembayaran: 300,
          sisa: -200,
          batch,
        },
      ],
      total,
      portal: 'ASM',
    })
  }

  it('mewarnai sisa negatif dan menulis tanda pisah untuk kolom riwayat yang kosong', async () => {
    replies[`GET ${ROUTE}/?*`] = listOf([batchRow()])
    const user = userEvent.setup()
    wrap(<OutstandingTable />)

    const nama = await screen.findByText('PT CONTOH SATU')
    // Satu batch saja tidak memunculkan keterangan jumlah batch.
    expect(screen.queryByText(/batch$/)).not.toBeInTheDocument()
    expect(screen.getAllByText('-200')[0]).toHaveClass('text-red-700')

    await user.click(nama)

    const riwayat = await screen.findByRole('table', { name: 'Riwayat batch principal ini' })
    // Tanggal kosong, tahun, NO HPLL, keterangan, dan posisi semuanya kosong.
    expect(within(riwayat).getAllByText('—')).toHaveLength(5)
    expect(within(riwayat).getByText('-200')).toHaveClass('text-red-700')
  })

  it('menulis tanda pisah untuk tanggal yang tidak terbaca dan nama pengunggah kosong', async () => {
    replies[`GET ${ROUTE}/?*`] = listOf([batchRow()])
    const user = userEvent.setup()
    wrap(<OutstandingTable />)

    await user.click(await screen.findByText('PT CONTOH SATU'))
    await user.click(await screen.findByRole('button', { name: 'View Document' }))

    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getAllByText('—')).toHaveLength(2)
    // Nama berkas kosong tidak digambar sama sekali.
    expect(dialog.querySelector('p.mt-3')).toBeNull()

    // Klik di dalam modal tidak menutupnya; klik di latar menutupnya.
    await user.click(within(dialog).getByText('View Dokument Pendukung'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await user.click(dialog)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('membuka berkas inline di tab baru dan mencabut URL-nya kemudian', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    replies[`GET ${ROUTE}/?*`] = listOf([batchRow()])
    replies[`GET ${ROUTE}/bukti-bayar/DOK-1`] = new Response('pdf', {
      status: 200,
      headers: { 'Content-Disposition': 'inline; filename="bukti.pdf"' },
    })
    const open = vi.fn()
    vi.stubGlobal('open', open)
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    wrap(<OutstandingTable />)

    await user.click(await screen.findByText('PT CONTOH SATU'))
    await user.click(await screen.findByRole('button', { name: 'View Document' }))
    await user.click(await screen.findByRole('button', { name: 'Buka berkas' }))

    await waitFor(() => expect(open).toHaveBeenCalledWith('blob:contoh', '_blank', 'noopener,noreferrer'))
    expect(URL.revokeObjectURL).not.toHaveBeenCalled()
    vi.advanceTimersByTime(60_000)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:contoh')
  })

  it('mengunduh berkas yang bukan inline dengan nama dari server, atau nama cadangan', async () => {
    let disposition = 'attachment; filename="bukti.xlsx"'
    replies[`GET ${ROUTE}/?*`] = listOf([
      batchRow({ dokumen: { id: 'DOK-1', nama_berkas: 'bukti.xlsx', input_nama: 'P', tanggal: '' } }),
    ])
    replies[`GET ${ROUTE}/bukti-bayar/DOK-1`] = () =>
      new Response('x', { status: 200, headers: { 'Content-Disposition': disposition } })
    const user = userEvent.setup()
    wrap(<OutstandingTable />)

    await user.click(await screen.findByText('PT CONTOH SATU'))
    await user.click(await screen.findByRole('button', { name: 'View Document' }))
    expect(await screen.findByText('bukti.xlsx')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Buka berkas' }))
    await waitFor(() => expect(clicked).toEqual([{ href: 'blob:contoh', download: 'bukti.xlsx' }]))

    disposition = 'attachment'
    await user.click(await screen.findByRole('button', { name: 'Buka berkas' }))
    await waitFor(() => expect(clicked.at(-1)?.download).toBe('bukti-bayar'))
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:contoh')
  })

  it.each([
    ['bukti_bayar_tidak_ditemukan', 'Bukti bayar tidak ditemukan'],
    ['bukti_bayar_di_penyimpanan_lain', 'Berkasnya tidak tersimpan di basis data'],
    ['galat_lain', 'Bukti bayar gagal dibuka'],
  ])('menjelaskan berkas yang gagal dibuka: %s', async (kode, judul) => {
    replies[`GET ${ROUTE}/?*`] = listOf([batchRow()])
    replies[`GET ${ROUTE}/bukti-bayar/DOK-1`] = apiError(404, kode)
    const user = userEvent.setup()
    wrap(<OutstandingTable />)

    await user.click(await screen.findByText('PT CONTOH SATU'))
    await user.click(await screen.findByRole('button', { name: 'View Document' }))
    await user.click(await screen.findByRole('button', { name: 'Buka berkas' }))

    expect(await screen.findByText(judul)).toBeInTheDocument()
  })

  it('membedakan jaringan dan kesalahan tak terduga saat membuka berkas', async () => {
    replies[`GET ${ROUTE}/?*`] = listOf([batchRow()])
    replies[`GET ${ROUTE}/bukti-bayar/DOK-1`] = NETWORK as unknown as Reply
    const user = userEvent.setup()
    wrap(<OutstandingTable />)

    await user.click(await screen.findByText('PT CONTOH SATU'))
    await user.click(await screen.findByRole('button', { name: 'View Document' }))
    await user.click(await screen.findByRole('button', { name: 'Buka berkas' }))
    expect(await screen.findByText(/Bukti bayar belum dapat dibuka/)).toBeInTheDocument()

    replies[`GET ${ROUTE}/bukti-bayar/DOK-1`] = brokenResponse
    await user.click(screen.getByRole('button', { name: 'Buka berkas' }))
    expect(await screen.findByText(/Coba beberapa saat lagi\./)).toBeInTheDocument()
  })

  it('menampilkan Membuka selama berkas diambil', async () => {
    replies[`GET ${ROUTE}/?*`] = listOf([batchRow()])
    replies[`GET ${ROUTE}/bukti-bayar/DOK-1`] = () =>
      new Promise<Response>(() => undefined) as unknown as Response
    const user = userEvent.setup()
    wrap(<OutstandingTable />)

    await user.click(await screen.findByText('PT CONTOH SATU'))
    await user.click(await screen.findByRole('button', { name: 'View Document' }))
    await user.click(await screen.findByRole('button', { name: 'Buka berkas' }))

    expect(await screen.findByRole('button', { name: 'Membuka…' })).toBeDisabled()
  })

  it.each([
    ['portal_tidak_disebut', 'Portal entitas belum dipilih'],
    ['portal_tidak_dikenal', 'Portal entitas belum dipilih'],
    ['portal_belum_siap', 'Basis data entitas ini belum tersedia'],
    ['galat_lain', 'Daftar gagal dimuat'],
  ])('menjelaskan daftar yang gagal dimuat: %s', async (kode, judul) => {
    replies[`GET ${ROUTE}/?*`] = apiError(500, kode)
    wrap(<OutstandingTable />)

    expect(await screen.findByText(judul)).toBeInTheDocument()
  })

  it('membedakan jaringan dan kesalahan tak terduga saat memuat daftar', async () => {
    replies[`GET ${ROUTE}/?*`] = NETWORK as unknown as Reply
    const view = wrap(<OutstandingTable />)
    expect(await screen.findByText(/Daftar belum dapat dimuat/)).toBeInTheDocument()
    view.unmount()

    replies[`GET ${ROUTE}/?*`] = brokenResponse
    wrap(<OutstandingTable />)
    expect(await screen.findByText('Terjadi kesalahan pada sistem. Coba tekan Refresh.')).toBeInTheDocument()
  })

  it('mengirim kata kunci ke server, kembali ke halaman pertama, lalu berpindah halaman', async () => {
    replies[`GET ${ROUTE}/?*`] = listOf([batchRow()], 25)
    const user = userEvent.setup()
    wrap(<OutstandingTable />)

    await screen.findByText('PT CONTOH SATU')
    await user.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
    await waitFor(() =>
      expect(calls.some((c) => c.url.includes('lewati=10'))).toBe(true),
    )

    await user.type(screen.getByPlaceholderText('Cari nama principal'), 'CONTOH')
    await waitFor(() =>
      expect(calls.some((c) => c.url.includes('cari=CONTOH') && c.url.includes('lewati=0'))).toBe(
        true,
      ),
    )
  })
})
