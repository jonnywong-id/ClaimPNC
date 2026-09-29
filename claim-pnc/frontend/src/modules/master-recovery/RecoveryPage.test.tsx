import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
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

/**
 * Isi tab Outstanding yang dijawab uji ini.
 *
 * Dibiarkan KOSONG secara baku supaya uji yang tidak berurusan dengan daftar tidak perlu
 * memikirkannya. Uji yang menguji daftarnya mengisi variabel ini lebih dulu.
 */
let OUTSTANDING: unknown[] = []

function installDefaultFetch() {
  installFetch((url, init) => {
    if (url === `${ROUTE}/form`) return jsonResponse(200, FORM)
    // Daftar Outstanding. Jalurnya selalu membawa query string (limit dan lewati),
    // sehingga dicocokkan dengan awalan — bukan kesamaan penuh.
    if (url.startsWith(`${ROUTE}/?`)) {
      return jsonResponse(200, {
        principal: OUTSTANDING,
        total: OUTSTANDING.length,
        portal: 'ASM',
      })
    }
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
/**
 * Membuka panel entri lewat tombol **Tambah**.
 *
 * Diperlukan sejak layar disusun ulang mengikuti Pega (2026-09-29): isi utama layar adalah
 * DAFTAR, dan form entri dibuka lewat Tambah — bukan tergelar sejak layar dibuka.
 */
async function openEntry(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('button', { name: 'Tambah' }))
}

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
  OUTSTANDING = []
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
    const user = userEvent.setup()
    show()
    await openEntry(user)

    expect(await screen.findByText('4')).toBeInTheDocument()
    // Kata "perkiraan" WAJIB terbaca. Nomor batch di sistem lama tampak pasti padahal
    // tidak, dan dua petugas yang membuka layar bersamaan melihat angka yang sama.
    expect(screen.getByText(/Perkiraan/i)).toBeInTheDocument()
  })

  it('mengisi pilihan tahun dari server, terbaru lebih dulu', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await openEntry(user)

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
    await openEntry(user)

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
    await openEntry(user)

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
    await openEntry(user)

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
    await openEntry(user)

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
    await openEntry(user)

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
    await openEntry(user)

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
    await openEntry(user)

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
    await openEntry(user)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    expect(await screen.findByText('Batch 4 tersimpan')).toBeInTheDocument()
    expect(screen.getByText(/Sisa yang tercatat: Rp 155\.000/)).toBeInTheDocument()
  })

  it('mengosongkan isian hanya SETELAH server menjawab berhasil', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await openEntry(user)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Transfer Recovery' }))

    // Panel entri MENUTUP setelah berhasil, sehingga yang terlihat berikutnya adalah
    // daftar. Isiannya diperiksa setelah panelnya dibuka kembali — dan di situlah
    // pertanyaannya sebenarnya berarti: yang dibuka petugas berikutnya harus bersih,
    // bukan membawa sisa ketikan batch sebelumnya.
    await waitFor(() => expect(screen.getByText(/Batch 4 tersimpan/i)).toBeInTheDocument())

    await openEntry(user)
    expect((screen.getByLabelText('Keterangan') as HTMLInputElement).value).toBe('')
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
    await openEntry(user)

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
    await openEntry(user)

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

describe('tab Outstanding', () => {
  /**
   * Menggantikan uji lama yang justru MENUNTUT daftar itu tidak ada.
   *
   * Tuntutan itu keliru: ia berdiri di atas kesimpulan bahwa layar lama tidak punya
   * daftar, yang ditarik dari tidak adanya kueri pembaca di export — padahal export-nya
   * sendiri tidak lengkap (`R-16`). Grid Outstanding ada di
   * `Section/OutstandingMasterRecovery-Section.xml` dan berisi data di Pega yang berjalan.
   */
  /** Satu principal dengan tiga batch — bentuk yang sama dengan layar Pega. */
  function contohPrincipal() {
    const batch = (
      nomor: number,
      jam: string,
      sebelum: number,
      bayar: number,
      sisa: number,
      dokumen: string,
    ) => ({
      batch: nomor,
      nama_principal: 'PT CONTOH PENJAMINAN NUSANTARA',
      tahun: '2018',
      tanggal_input: `2025-05-14T${jam}:00+07:00`,
      no_hpll: '',
      nilai_klaim: 160000,
      pembayaran_sebelumnya: sebelum,
      pembayaran: bayar,
      sisa,
      keterangan: 'OK',
      posisi_kasus: 'test',
      nomor_virtual_account: '0000000000000001',
      nomor_polis: '',
      id_dokumen: dokumen,
      dokumen:
        dokumen === ''
          ? null
          : {
              id: dokumen,
              nama_berkas: 'bukti-transfer.pdf',
              input_nama: 'PETUGASCONTOH',
              tanggal: `2025-05-14T${jam}:00+07:00`,
            },
    })

    return [
      {
        nama_principal: 'PT CONTOH PENJAMINAN NUSANTARA',
        // Angka baris luar = batch TERAKHIR, bukan jumlah.
        nilai_klaim: 160000,
        pembayaran_sebelumnya: 7000,
        pembayaran: 100000,
        sisa: 153000,
        batch: [
          batch(1, '10:49', 0, 5000, 155000, ''),
          batch(2, '10:53', 5000, 2000, 155000, ''),
          batch(3, '10:54', 7000, 100000, 153000, 'DOK-3'),
        ],
      },
    ]
  }

  it('menampilkan SATU baris per principal beserta kelima kolom layar lama', async () => {
    OUTSTANDING = contohPrincipal()
    installDefaultFetch()
    show()

    // Kelima kolom grid LUAR, dengan judul yang sama persis.
    for (const judul of [
      'Nama Principal',
      'Nilai Klaim',
      'Nilai Pembayaran Sebelumnya',
      'Pembayaran',
      'Sisa',
    ]) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }

    // Tiga batch, tetapi hanya SATU baris luar.
    expect(await screen.findByText('PT CONTOH PENJAMINAN NUSANTARA')).toBeInTheDocument()
    // Angka uang diformat di layar, bukan dikirim server sudah terformat.
    expect(await screen.findByText('153.000')).toBeInTheDocument()
    // Riwayatnya belum terbuka sebelum barisnya diklik.
    expect(screen.queryByText('Tanggal Input')).not.toBeInTheDocument()
  })

  it('membuka riwayat batch saat baris diklik, dari yang paling lama', async () => {
    OUTSTANDING = contohPrincipal()
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await user.click(await screen.findByText('PT CONTOH PENJAMINAN NUSANTARA'))

    // Kesepuluh kolom grid DALAM.
    for (const judul of [
      'Tanggal Input',
      'NO HPLL',
      'Tahun',
      'Nilai Pembayaran',
      'Sisa Klaim',
      'Keterangan',
      'Posisi',
    ]) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }

    // Urutannya dari yang paling lama — 10:49 lebih dulu, bukan 10:54.
    const waktu = screen.getAllByText(/^14\/05\/2025 10:\d\d$/).map((el) => el.textContent)
    expect(waktu).toEqual(['14/05/2025 10:49', '14/05/2025 10:53', '14/05/2025 10:54'])
  })

  it('membuka modal View Dokument Pendukung, dan menulis Data Tidak Ada saat kosong', async () => {
    OUTSTANDING = contohPrincipal()
    installDefaultFetch()
    const user = userEvent.setup()
    show()

    await user.click(await screen.findByText('PT CONTOH PENJAMINAN NUSANTARA'))

    // Tombolnya ada di SETIAP batch, sama seperti layar lama — termasuk yang tanpa
    // lampiran. Yang membedakan adalah isi modalnya.
    const tombol = await screen.findAllByRole('button', { name: 'View Document' })
    expect(tombol).toHaveLength(3)

    // Batch pertama (10:49) tidak punya lampiran.
    await user.click(tombol[0]!)
    expect(await screen.findByText('View Dokument Pendukung')).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Input Nama' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Tanggal' })).toBeInTheDocument()
    // Kalimatnya sama persis dengan layar lama.
    expect(screen.getByText('Data Tidak Ada')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Tutup' }))

    // Batch ketiga (10:54) punya lampiran; modalnya menyebut pengunggah dan tanggalnya.
    await user.click(tombol[2]!)
    const dialog = await screen.findByRole('dialog')
    // Dicari DI DALAM modal: tanggal yang sama juga tampil di grid dalam di belakangnya.
    expect(within(dialog).getByText('PETUGASCONTOH')).toBeInTheDocument()
    expect(within(dialog).getByText('14/05/2025 10:54')).toBeInTheDocument()
    expect(within(dialog).queryByText('Data Tidak Ada')).not.toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Buka berkas' })).toBeInTheDocument()
  })

  it('meminta halaman ke server, bukan memotongnya di peramban', async () => {
    installDefaultFetch()
    show()

    await waitFor(() => {
      const list = calls.find((c) => c.url.startsWith(`${ROUTE}/?`))
      expect(list).toBeDefined()
      // Sepuluh baris per halaman, sama dengan pyRDLPageSize grid lama.
      expect(list?.url).toContain('limit=10')
      expect(list?.url).toContain('lewati=0')
    })
  })
})

describe('yang sengaja tidak ada', () => {
  it('tidak menyediakan Ubah maupun Hapus', async () => {
    installDefaultFetch()
    show()

    // Ketiadaan ini berdiri di atas ISI, bukan di atas ketiadaan bukti:
    // INSERTMASTERRECOVERYKLAIM.prc hanya mengenal INSERT, dan tidak ada satu pun UPDATE
    // maupun DELETE atas tabel ini di seluruh export.
    await screen.findByRole('button', { name: 'Tambah' })
    expect(screen.queryByRole('button', { name: /^Ubah/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Hapus/i })).not.toBeInTheDocument()
  })
})
