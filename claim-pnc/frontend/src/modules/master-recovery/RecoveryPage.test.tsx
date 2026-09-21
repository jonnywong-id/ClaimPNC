import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/recovery'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Principal contoh. Nilainya DIKARANG, bukan disalin dari portal ASM.
 *
 * Kedua baris nyata memuat nomor rekening virtual dan surel pegawai yang benar-benar ada,
 * dan `D-69` menetapkan data seperti itu tidak pernah ditulis ke berkas yang di-commit.
 * Yang ditiru adalah bentuknya, bukan isinya.
 */
const PRINCIPAL = [
  {
    client_id: 'CONTOH-PRINCIPAL-001',
    nama_principal: 'PT CONTOH PENJAMINAN NUSANTARA',
    nomor_virtual_account: '0000000000000001',
    email_inputor_va: 'contoh.satu@example.invalid',
  },
]

const FORM = { nomor_batch_perkiraan: 4, tahun: ['2027', '2026', '2025'], portal: 'ASM' }

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(reply: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal dan menunya sendiri, sehingga SETIAP layar di balik
    // sesi ikut memanggil keduanya. Keduanya dijawab di sini supaya uji ini menguji
    // layarnya, bukan jalur galat kerangka.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(reply(url, init))
  })
}

/** Badan permintaan simpan yang terakhir dikirim layar. */
function lastSaveBody(): Record<string, unknown> {
  const call = [...calls].reverse().find((c) => c.url === `${ROUTE}/` && c.init?.method === 'POST')
  return JSON.parse(String(call?.init?.body ?? '{}')) as Record<string, unknown>
}

function installDefaultFetch() {
  installFetch((url, init) => {
    if (url === `${ROUTE}/form`) return jsonResponse(200, FORM)
    if (url === `${ROUTE}/principal`) {
      return jsonResponse(200, { principal: PRINCIPAL, total: PRINCIPAL.length, portal: 'ASM' })
    }
    if (url === `${ROUTE}/` && init?.method === 'POST') {
      const body = JSON.parse(String(init.body)) as Record<string, unknown>
      return jsonResponse(201, {
        recovery: {
          nomor_batch: 4,
          nama_principal: body['nama_principal'],
          client_id: body['client_id'],
          nomor_virtual_account: body['nomor_virtual_account'],
          tahun: body['tahun'],
          nilai_klaim: body['nilai_klaim'],
          pembayaran_sebelumnya: body['pembayaran_sebelumnya'],
          pembayaran: body['pembayaran'],
          // Sisa DARI SERVER, sengaja dibuat berbeda dari yang dihitung layar — uji di
          // bawah memastikan yang ditampilkan adalah yang ini.
          sisa: 155000,
          keterangan: body['keterangan'],
          posisi_kasus: body['posisi_kasus'],
          id_dokumen: '',
          dicatat_oleh: '90000001',
          nomor_polis: body['nomor_polis'],
          id_lini_bisnis: '',
          id_cabang: '',
          id_agen: '',
          id_marketing: '',
          baris_klaim: [],
        },
        identitas_polis_terisi: true,
        portal: 'ASM',
      })
    }
    return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak dilayani uji ini' })
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/recovery']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * Menunggu daftar tahun tiba, lalu mengisi seluruh isian wajib.
 *
 * Penantiannya bukan basa-basi: isian Tahun baru terisi setelah `/form` dijawab, dan
 * menekan Simpan sebelum itu akan ditolak "Tahun wajib dipilih" — persis seperti yang akan
 * dialami petugas yang mengetik lebih cepat daripada jaringannya.
 */
async function fillRequired(user: ReturnType<typeof userEvent.setup>) {
  const tahun = (await screen.findByLabelText('Tahun')) as HTMLSelectElement
  await waitFor(() => expect(tahun.value).not.toBe(''))

  await user.type(screen.getByLabelText('Nama Principal'), 'PT CONTOH PENJAMINAN')
  await user.type(screen.getByLabelText('Nilai Klaim (Rp)'), '160000')
  await user.type(screen.getByLabelText('Pembayaran (Rp)'), '5000')
  await user.type(screen.getByLabelText('Keterangan'), 'pengembalian sebagian')
  await user.type(screen.getByLabelText('Posisi Kasus'), 'dalam proses')
}

beforeEach(() => {
  calls = []
  // Layar berada di balik sesi. Tanpa ini SessionGuard melempar ke layar masuk.
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

describe('bekal awal layar', () => {
  it('menampilkan nomor batch perkiraan dan menyebutnya perkiraan', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('4')).toBeInTheDocument()
    // Kata "perkiraan" WAJIB terbaca. Nomor batch di sistem lama tampak pasti padahal
    // tidak, dan dua petugas yang membuka layar bersamaan melihat angka yang sama.
    expect(screen.getByText(/Perkiraan/i)).toBeInTheDocument()
  })

  it('mengisi pilihan tahun dari server, terbaru lebih dulu', async () => {
    installDefaultFetch()
    show()

    const tahun = (await screen.findByLabelText('Tahun')) as HTMLSelectElement
    // Ditunggu sampai pilihannya BENAR-BENAR terisi. Elemen select-nya sudah ada sejak
    // render pertama — jauh sebelum permintaan /form dijawab — sehingga memeriksa isinya
    // seketika akan selalu menemukan daftar kosong.
    await waitFor(() => expect(tahun.options.length).toBeGreaterThan(1))

    const pilihan = Array.from(tahun.options).map((o) => o.value)
    expect(pilihan).toContain('2027')
    expect(pilihan).toContain('2026')
    // Terbaru dipilih lebih dulu, setelah daftarnya tiba.
    await waitFor(() => expect(tahun.value).toBe('2027'))
  })

  it('menyebut portal entitas yang menjawab', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Portal entitas:')).toBeInTheDocument()
  })
})

describe('aturan Sisa', () => {
  it('memakai Pembayaran ketika tidak ada pembayaran sebelumnya', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await user.type(await screen.findByLabelText('Nilai Klaim (Rp)'), '160000')
    await user.type(screen.getByLabelText('Pembayaran (Rp)'), '5000')

    expect(await screen.findByText('Rp 155.000')).toBeInTheDocument()
    expect(screen.getByText(/Nilai Klaim − Pembayaran/)).toBeInTheDocument()
  })

  it('MENGABAIKAN Pembayaran ketika ada pembayaran sebelumnya', async () => {
    // Perilaku yang tampak keliru dibaca sekilas, dan justru karena itu diuji: ketiga
    // baris produksi membuktikan itu memang aturannya, dan `P-5` menetapkan perilaku
    // dipertahankan lebih dulu.
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await user.type(await screen.findByLabelText('Nilai Klaim (Rp)'), '160000')
    await user.type(screen.getByLabelText('Nilai Pembayaran Sebelumnya (Rp)'), '7000')
    await user.type(screen.getByLabelText('Pembayaran (Rp)'), '100000')

    expect(await screen.findByText('Rp 153.000')).toBeInTheDocument()
    expect(
      screen.getByText(/Nilai Klaim − Nilai Pembayaran Sebelumnya/),
    ).toBeInTheDocument()
  })

  it('memperingatkan ketika sisa negatif alih-alih menyembunyikannya', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await user.type(await screen.findByLabelText('Nilai Klaim (Rp)'), '10000')
    await user.type(screen.getByLabelText('Pembayaran (Rp)'), '50000')

    expect(await screen.findByText(/Sisa negatif/)).toBeInTheDocument()
  })
})

describe('validasi', () => {
  it('menandai SELURUH isian wajib sekaligus, bukan satu per satu', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await user.click(await screen.findByRole('button', { name: 'Transfer Recovery' }))

    // Keempat pesan tampil bersamaan — menggantikan satu kalimat "Wajib ISI semua field"
    // yang tidak menyebut isian mana pun.
    expect(await screen.findByText('Nama principal wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Keterangan wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Posisi kasus wajib diisi.')).toBeInTheDocument()

    // Tidak ada permintaan simpan yang terkirim.
    expect(calls.some((c) => c.url === `${ROUTE}/` && c.init?.method === 'POST')).toBe(false)
  })

  it('menolak nilai uang yang bukan angka', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await user.type(await screen.findByLabelText('Nilai Klaim (Rp)'), 'seribu')
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(
      await screen.findByText('Nilai klaim harus berupa angka rupiah utuh.'),
    ).toBeInTheDocument()
  })
})

describe('simpan', () => {
  it('mengirim nilai uang sebagai angka, bukan teks berpemisah ribuan', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    const tahun = (await screen.findByLabelText('Tahun')) as HTMLSelectElement
    await waitFor(() => expect(tahun.value).not.toBe(''))

    await user.type(screen.getByLabelText('Nama Principal'), 'PT CONTOH')
    // Diketik DENGAN pemisah ribuan, seperti yang benar-benar dilakukan petugas.
    await user.type(screen.getByLabelText('Nilai Klaim (Rp)'), '160.000')
    await user.type(screen.getByLabelText('Pembayaran (Rp)'), '5.000')
    await user.type(screen.getByLabelText('Keterangan'), 'pengembalian')
    await user.type(screen.getByLabelText('Posisi Kasus'), 'proses')
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    await waitFor(() => expect(lastSaveBody()['nilai_klaim']).toBe(160000))
    expect(lastSaveBody()['pembayaran']).toBe(5000)
  })

  it('TIDAK mengirim sisa, nomor batch, maupun identitas polis', async () => {
    // Keempatnya milik server. Backend bahkan MENOLAK badan yang memuatnya, sehingga
    // mengirimnya akan membuat setiap penyimpanan gagal dengan "permintaan cacat".
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    await waitFor(() => expect(lastSaveBody()['nama_principal']).toBeDefined())
    const body = lastSaveBody()
    expect(body).not.toHaveProperty('sisa')
    expect(body).not.toHaveProperty('nomor_batch')
    expect(body).not.toHaveProperty('id_lini_bisnis')
    expect(body).not.toHaveProperty('dicatat_oleh')
  })

  it('menampilkan sisa DARI SERVER, bukan yang dihitung layar', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(await screen.findByText('Batch 4 tersimpan')).toBeInTheDocument()
    expect(screen.getByText(/Sisa yang tercatat: Rp 155\.000/)).toBeInTheDocument()
  })

  it('mengosongkan isian hanya SETELAH server menjawab berhasil', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    await waitFor(() =>
      expect((screen.getByLabelText('Keterangan') as HTMLInputElement).value).toBe(''),
    )
  })

  it('mempertahankan isian ketika penyimpanan ditolak', async () => {
    installFetch((url, init) => {
      if (url === `${ROUTE}/form`) return jsonResponse(200, FORM)
      if (url === `${ROUTE}/principal`) {
        return jsonResponse(200, { principal: PRINCIPAL, total: 1, portal: 'ASM' })
      }
      if (url === `${ROUTE}/` && init?.method === 'POST') {
        return jsonResponse(409, {
          kode: 'nomor_batch_sudah_dipakai',
          pesan: 'Nomor batch baru saja dipakai petugas lain.',
        })
      }
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: '' })
    })
    const user = userEvent.setup()
    show()

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(await screen.findByText('Nomor batch baru saja dipakai')).toBeInTheDocument()
    // Isian TETAP ada — membuangnya akan memaksa petugas mengetik ulang seluruhnya.
    expect((screen.getByLabelText('Keterangan') as HTMLInputElement).value).toBe(
      'pengembalian sebagian',
    )
  })
})

describe('memilih principal dari master', () => {
  it('mengisi nama, Client ID, dan nomor VA sekaligus', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    const pilih = (await screen.findByLabelText('Pilih dari master')) as HTMLSelectElement
    // Sama seperti Tahun: pilihannya baru terisi setelah /principal dijawab.
    await waitFor(() => expect(pilih.options.length).toBeGreaterThan(1))
    await user.selectOptions(pilih, 'CONTOH-PRINCIPAL-001')

    expect((screen.getByLabelText('Nama Principal') as HTMLInputElement).value).toBe(
      'PT CONTOH PENJAMINAN NUSANTARA',
    )
    expect((screen.getByLabelText('Client ID') as HTMLInputElement).value).toBe(
      'CONTOH-PRINCIPAL-001',
    )
    expect(
      (screen.getByLabelText('Virtual Account Number') as HTMLInputElement).value,
    ).toBe('0000000000000001')
  })
})

describe('portal entitas', () => {
  it('menuntun memilih portal alih-alih menampilkan pesan galat', async () => {
    useSelectedPortal.getState().clear()
    installDefaultFetch()
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    // Permintaan tidak pernah ditembakkan: menembaknya hanya untuk menerima penolakan akan
    // menampilkan galat pada layar yang memang belum siap dibuka.
    expect(calls.some((c) => c.url.startsWith(ROUTE))).toBe(false)
  })
})

describe('yang sengaja tidak ada', () => {
  it('tidak menampilkan daftar batch tersimpan, dan mengatakan alasannya', async () => {
    installDefaultFetch()
    show()

    // Sistem lama tidak punya cara membaca kembali batch yang sudah tercatat, dan itu
    // ditiru apa adanya. Layar mengatakannya terus terang supaya tidak ada yang mengira
    // tabel di bawah adalah isi basis data.
    expect(
      await screen.findByText(/hanya berisi batch yang Anda simpan sejak layar dibuka/i),
    ).toBeInTheDocument()
  })
})
