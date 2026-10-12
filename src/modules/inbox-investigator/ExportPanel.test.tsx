import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ExportPanel } from './ExportPanel'

/**
 * Uji panel Export Data Investigation.
 *
 * Yang diuji BUKAN isi berkasnya — itu milik uji backend, yang membandingkan judul kolomnya
 * dengan teks tetap pada `Activity/ExportDataInvestigator-Act.xml:2332`. Yang diuji di sini
 * adalah apa yang dikirim layar dan apa yang dikatakannya kepada pengguna.
 */

let calls: { url: string; init?: RequestInit }[] = []

/** Jawaban berkas CSV yang berhasil. */
function csvReply(): Response {
  return {
    ok: true,
    status: 200,
    headers: new Headers({
      'Content-Type': 'text/csv; charset=utf-8',
      'Content-Disposition': 'attachment; filename="data-investigasi.csv"',
    }),
    blob: async () => new Blob(['Tanggal Investigasi\n'], { type: 'text/csv' }),
    json: async () => ({}),
  } as unknown as Response
}

/** Jawaban penolakan penyaring dari server. */
function rejectReply(message: string): Response {
  return {
    ok: false,
    status: 400,
    headers: new Headers(),
    blob: async () => new Blob(),
    json: async () => ({ kode: 'permintaan_cacat', pesan: message }),
  } as unknown as Response
}

function installFetch(...replies: Response[]) {
  let index = 0
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, init?: RequestInit) => {
      calls.push({ url: String(url), ...(init ? { init } : {}) })
      const reply = replies[Math.min(index, replies.length - 1)]
      index += 1
      return Promise.resolve(reply)
    }),
  )
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ExportPanel />
    </QueryClientProvider>,
  )
}

/**
 * fill mengisi ketiga kendali; nilai kosong berarti kendalinya dilewati.
 *
 * Tanggalnya diketik **DD/MM/YYYY**, bukan ISO: `DateField` adalah isian teks bertopeng,
 * bukan `<input type="date">`. Itu bentuk yang dipakai seluruh layar aplikasi ini, dan
 * mengetiknya sebagai ISO menghasilkan isian yang tetap kosong tanpa satu pun galat.
 */
async function fill(from: string, to: string, investigated: string) {
  if (from) {
    await userEvent.type(screen.getByLabelText(/^dari$/i), from)
  }
  if (to) {
    await userEvent.type(screen.getByLabelText(/^sampai$/i), to)
  }
  if (investigated) {
    await userEvent.selectOptions(screen.getByLabelText(/pilih investigation/i), investigated)
  }
}

function pressExport() {
  return userEvent.click(screen.getByRole('button', { name: /export data investigation/i }))
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')

  // jsdom tidak punya keduanya; tanpa ini jalur unduhan yang BERHASIL justru melempar.
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:contoh'),
    revokeObjectURL: vi.fn(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('panel Export Data Investigation', () => {
  /** Ketiga kendali layar lama digambar, dengan label yang sama. */
  it('menggambar ketiga kendali layar lama', () => {
    installFetch(csvReply())
    show()

    expect(screen.getByLabelText(/^dari$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^sampai$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/pilih investigation/i)).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /export data investigation/i }),
    ).toBeInTheDocument()
  })

  /**
   * Isian yang belum lengkap menghentikan ekspor SEBELUM server ditembak.
   *
   * Masing-masing menyebut isian mana yang kurang. Pesan yang hanya berbunyi "lengkapi
   * isian" memaksa pengguna menebak, dan pada tiga kendali itu tebakannya sering salah.
   */
  it('menyebut isian yang kurang dan tidak menembak server', async () => {
    installFetch(csvReply())
    show()

    await pressExport()
    expect(await screen.findByRole('alert')).toHaveTextContent(/tanggal "dari"/i)
    expect(calls).toHaveLength(0)

    await fill('01/09/2026', '', '')
    await pressExport()
    expect(await screen.findByRole('alert')).toHaveTextContent(/tanggal "sampai"/i)
    expect(calls).toHaveLength(0)

    await fill('', '30/09/2026', '')
    await pressExport()
    expect(await screen.findByRole('alert')).toHaveTextContent(/pilih investigation/i)
    expect(calls).toHaveLength(0)
  })

  /** Rentang terbalik ditolak dengan kalimatnya sendiri. */
  it('menolak rentang tanggal yang terbalik', async () => {
    installFetch(csvReply())
    show()

    await fill('30/09/2026', '01/09/2026', '1')
    await pressExport()

    expect(await screen.findByRole('alert')).toHaveTextContent(/lebih awal/i)
    expect(calls).toHaveLength(0)
  })

  /**
   * Ketiga kendali dikirim sebagai parameter kueri, beserta header sesi dan portal.
   *
   * Header portal yang hilang berarti berkas data medis satu badan hukum dapat sampai ke
   * petugas badan hukum lain (`R-20`), dan itu tidak terlihat sebagai galat — berkasnya
   * terunduh dengan isi yang tampak masuk akal.
   */
  it('mengirim ketiga kendali beserta header sesi dan portal', async () => {
    installFetch(csvReply())
    show()

    await fill('01/09/2026', '30/09/2026', '1')
    await pressExport()

    await vi.waitFor(() => expect(calls).toHaveLength(1))

    const call = calls[0]!
    expect(call.url).toContain('/api/inbox/investigator/ekspor?')
    expect(call.url).toContain('dari=2026-09-01')
    expect(call.url).toContain('sampai=2026-09-30')
    expect(call.url).toContain('investigasi=1')

    const header = call.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
    expect(Object.values(header)).toContain('ASM')
  })

  /** Pilihan "belum diinvestigasi" dikirim sebagai `0`, bukan sebagai labelnya. */
  it('mengirim kode pilihan, bukan labelnya', async () => {
    installFetch(csvReply())
    show()

    await fill('01/09/2026', '30/09/2026', '0')
    await pressExport()

    await vi.waitFor(() => expect(calls).toHaveLength(1))
    expect(calls[0]!.url).toContain('investigasi=0')
  })

  /** Penolakan server ditampilkan dengan pesannya sendiri, bukan dengan kalimat umum. */
  it('menampilkan pesan penolakan dari server', async () => {
    installFetch(rejectReply('Ekspor tidak dapat dijalankan: rentangnya terlalu lebar.'))
    show()

    await fill('01/09/2026', '30/09/2026', '1')
    await pressExport()

    expect(await screen.findByText(/rentangnya terlalu lebar/i)).toBeInTheDocument()
  })

  /**
   * TIDAK ada satu pun teks keterangan yang Pega tak punya.
   *
   * Dua paragraf sempat ditulis — "isinya berbeda dari daftar" dan peringatan data medis —
   * dan keduanya dicabut. Uji ini yang gagal lebih dulu bila salah satunya kembali: alasan
   * menambahkannya selalu terasa kuat saat menulisnya, dan selalu diminta hapus.
   */
  it('tidak menambahkan keterangan yang tidak ada di layar lama', () => {
    installFetch(csvReply())
    show()

    expect(screen.queryByText(/data medis/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/tidak sama dengan daftar/i)).not.toBeInTheDocument()
  })
})
