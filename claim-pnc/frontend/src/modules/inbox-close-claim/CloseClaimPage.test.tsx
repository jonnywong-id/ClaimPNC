import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CloseClaimPage } from './CloseClaimPage'
import type { DaftarResponse, KlaimTutup, PenyaringResponse } from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen.
 */
function klaim(partial: Partial<KlaimTutup> = {}): KlaimTutup {
  return {
    klaim_id: 'ASM-FW-GCNMFW-WORK PNCN.26.0001',
    nomor_klaim: 'PNCN.26.0001',
    nomor_polis: 'POL-FIRE-0001',
    nama_tertanggung: 'PT Bumi Contoh Sentosa',
    nama_bisnis: 'Fire',
    sumber_bisnis: 'Broker',
    nama_cabang: 'JKT',
    pic_teknik: 'BUDISANTOSO',
    admin_pnc: 'ADMINPNC',
    tanggal_pendaftaran: '2026-01-12',
    tanggal_kejadian: '2026-01-07',
    tanggal_tutup: '2026-02-22',
    lama_hari: 41,
    status_proses: 'Resolved-Completed',
    status_tampil: 'Close',
    status_klaim_kode: '1163',
    status_klaim_label: 'Paid',
    sudah_transfer: true,
    permintaan_tertunda: [],
    ...partial,
  }
}

function daftarResponse(partial: Partial<DaftarResponse> = {}): DaftarResponse {
  return {
    klaim: [klaim()],
    total: 1,
    permintaan_terbaca: true,
    boleh_mengajukan: true,
    pelaksana_belum_ada: true,
    selisih_terencana: ['Kolom "Lama Waktu Klaim" berisi umur klaim dalam hari.'],
    ...partial,
  }
}

const penyaringResponse: PenyaringResponse = {
  lini_bisnis: [
    { nilai: 'ALL', label: 'Semua Lini Bisnis' },
    { nilai: 'NONMBU', label: 'Non-MBU' },
    { nilai: 'BONDING', label: 'Bonding' },
    { nilai: 'PA', label: 'Personal Accident' },
    { nilai: 'TRAVEL', label: 'Travel' },
  ],
  status_transfer: [
    { nilai: '', label: 'Semua Status Transfer' },
    { nilai: 'SUDAH TRANSFER', label: 'Sudah Transfer' },
    { nilai: 'BELUM TRANSFER', label: 'Belum Transfer' },
  ],
  status_bayar: [
    { nilai: '', label: 'Semua Status Bayar' },
    { nilai: 'LUNAS', label: 'Lunas' },
    { nilai: 'BELUM LUNAS', label: 'Belum Lunas' },
  ],
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/**
 * Menjawab kedua GET layar ini sekaligus, dan menyerahkan POST ke pemanggil.
 *
 * Layar memanggil `/penyaring` dan daftar bersamaan; menjawab keduanya di satu tempat
 * membuat tiap uji hanya perlu menyatakan yang berbeda.
 */
function stubFetch(
  daftar: DaftarResponse | (() => Response),
  onPost?: (body: unknown) => Response,
) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })

    if (init?.method === 'POST') {
      const body: unknown = init.body ? JSON.parse(String(init.body)) : {}
      return Promise.resolve(
        onPost ? onPost(body) : jsonResponse(500, { kode: 'galat_internal', pesan: 'tak diduga' }),
      )
    }
    if (url.includes('/penyaring')) {
      return Promise.resolve(jsonResponse(200, penyaringResponse))
    }
    return Promise.resolve(typeof daftar === 'function' ? daftar() : jsonResponse(200, daftar))
  })
}

function renderPage() {
  // Cache dimatikan supaya tiap uji berdiri sendiri; retry dimatikan supaya jalur galat
  // tidak menunggu percobaan ulang.
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <CloseClaimPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: '2026-09-24T12:00:00Z',
  })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/**
 * Judul kolom mengikuti `Section/InboxManagerReopen1_Sec-Section.xml` apa adanya (`D-13`).
 *
 * Mengganti judulnya ke bahasa Inggris akan lulus uji lain tetapi membuat layar berbeda dari
 * yang dikenal pengguna — dan layar ini satu-satunya yang judul kolomnya memang berbahasa
 * Indonesia di Pega.
 */
it('memakai judul kolom yang sama dengan section rujukan', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0001')

  for (const title of [
    'No Klaim',
    'No Polis',
    'Nama Tertanggung',
    'Nama Bisnis',
    'Sumber Bisnis',
    'Nama Cabang',
    'Tanggal Pendaftaran',
    'Lama Waktu Klaim',
    'PIC Teknik',
    'Admin PNC',
  ]) {
    expect(screen.getByRole('columnheader', { name: new RegExp(title, 'i') })).toBeVisible()
  }
})

/**
 * Kolom "Lama Waktu Klaim" berisi ANGKA HARI, bukan tanggal.
 *
 * Inilah selisih terencana yang paling terlihat: di Pega kolom itu menampilkan tanggal
 * pendaftaran untuk kedua kalinya.
 */
it('menampilkan lama klaim sebagai jumlah hari, bukan tanggal', async () => {
  stubFetch(daftarResponse())
  renderPage()

  expect(await screen.findByText('41 hari')).toBeVisible()
})

/**
 * Klaim yang tanggal tutupnya tidak tercatat ditandai BERBEDA.
 *
 * Angkanya dihitung sampai hari ini, dan itu arti yang berbeda dari lamanya klaim berjalan.
 * Menampilkannya sama persis akan membuat keduanya tidak dapat dibedakan.
 */
it('menandai lama klaim yang dihitung sampai hari ini', async () => {
  stubFetch(daftarResponse({ klaim: [klaim({ tanggal_tutup: '', lama_hari: 900 })] }))
  renderPage()

  expect(await screen.findByText('900 hari*')).toBeVisible()
})

it('menyatakan selisih terencana kepada pengguna', async () => {
  stubFetch(daftarResponse())
  renderPage()

  expect(await screen.findByText(/Tiga hal yang berbeda dari layar lama/)).toBeVisible()
})

/**
 * Penyaring dikirim ke SERVER, bukan disaring di peramban.
 *
 * Tabel klaim berisi puluhan juta baris, dan layar ini membaca bagian yang paling besar
 * darinya. Menyaring di peramban membuat hasilnya BOHONG — ia hanya menyentuh halaman yang
 * sedang terbuka.
 */
it('mengirim penyaring ke server sebagai parameter', async () => {
  stubFetch(daftarResponse())
  renderPage()
  await screen.findByText('PNCN.26.0001')

  await userEvent.selectOptions(screen.getByLabelText('Status Bayar'), 'LUNAS')

  await waitFor(() => {
    expect(calls.some((call) => call.url.includes('status_bayar=LUNAS'))).toBe(true)
  })
})

/**
 * Mengubah penyaring mengembalikan halaman ke awal.
 *
 * Tanpa ini, menyaring dari halaman empat menampilkan tabel kosong yang tampak rusak — dan
 * pengguna menyimpulkan penyaringnya tidak menemukan apa pun.
 */
it('kembali ke halaman pertama setiap kali penyaring berubah', async () => {
  stubFetch(daftarResponse({ total: 120 }))
  renderPage()
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
  await waitFor(() => expect(calls.some((call) => call.url.includes('lewati=25'))).toBe(true))

  calls = []
  await userEvent.selectOptions(screen.getByLabelText('Lini Bisnis'), 'PA')

  await waitFor(() => {
    const terakhir = calls.filter((call) => !call.url.includes('/penyaring')).at(-1)
    expect(terakhir?.url).toContain('lini=PA')
    expect(terakhir?.url).not.toContain('lewati=')
  })
})

/**
 * Dialog konfirmasi menyatakan bahwa klaim BELUM berubah.
 *
 * Ini kalimat yang paling penting di layar ini: tombolnya berhasil ditekan, tetapi klaimnya
 * tidak berubah. Tanpa kalimat itu, pengguna menutup layar dengan mengira klaimnya sudah
 * terbuka kembali.
 */
it('menjelaskan bahwa klaim belum berubah sebelum permintaan dikirim', async () => {
  stubFetch(daftarResponse())
  renderPage()
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'ReOpen' }))

  const dialog = await screen.findByRole('dialog')
  expect(within(dialog).getByText(/Klaim belum berubah sekarang/)).toBeVisible()
  expect(within(dialog).getByText(/Reopen Claim/)).toBeVisible()
})

it('mengirim permintaan reopen beserta kunci klaim dan alasannya', async () => {
  let dikirim: unknown = null
  stubFetch(daftarResponse(), (body) => {
    dikirim = body
    return jsonResponse(201, {
      permintaan: {
        id: 'abc',
        jenis: 'reopen',
        status: 'menunggu',
        alasan: 'dokumen susulan',
        pemohon: 'BUDI',
        pemohon_nama: 'Budi',
        pada: '2026-09-23 11:00',
        efek_status_kerja: 'New',
        efek_status_klaim: '1164',
        lingkup_salin: '',
      },
      pesan: 'Permintaan buka kembali tercatat. Klaim belum berubah.',
    })
  })
  renderPage()
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'ReOpen' }))
  await userEvent.type(screen.getByLabelText(/Alasan/), 'dokumen susulan')
  await userEvent.click(screen.getByRole('button', { name: 'Ajukan buka kembali' }))

  await waitFor(() => {
    expect(dikirim).toEqual({
      jenis: 'reopen',
      klaim_id: 'ASM-FW-GCNMFW-WORK PNCN.26.0001',
      alasan: 'dokumen susulan',
    })
  })

  expect(await screen.findByText(/Klaim belum berubah/)).toBeVisible()
})

/**
 * Baris yang permintaannya sudah diajukan TIDAK lagi menampilkan tombolnya.
 *
 * Menyisakan tombol yang pasti ditolak server (409) hanya mengundang penekanan yang berakhir
 * sebagai pesan galat.
 */
it('mengganti tombol dengan penanda saat permintaan masih menunggu', async () => {
  stubFetch(
    daftarResponse({
      klaim: [
        klaim({
          permintaan_tertunda: [
            {
              id: 'abc',
              jenis: 'reopen',
              status: 'menunggu',
              alasan: '',
              pemohon: 'BUDI',
              pemohon_nama: 'Budi',
              pada: '2026-09-23 11:00',
              efek_status_kerja: 'New',
              efek_status_klaim: '1164',
              lingkup_salin: '',
            },
          ],
        }),
      ],
    }),
  )
  renderPage()

  expect(await screen.findByText('ReOpen diminta')).toBeVisible()
  expect(screen.queryByRole('button', { name: 'ReOpen' })).toBeNull()

  // Jenis yang lain tetap boleh diajukan: yang dibatasi adalah satu permintaan menunggu per
  // (jenis, klaim).
  expect(screen.getByRole('button', { name: 'Copy Klaim' })).toBeVisible()
})

/**
 * Tabel permintaan yang belum ada dinyatakan, bukan didiamkan.
 *
 * Bila tidak, penanda yang tidak muncul akan terbaca sebagai "belum pernah diminta" — dan
 * pengguna mengajukannya lagi, yang justru keadaan yang hendak dicegah.
 */
it('menyatakan saat penanda permintaan tidak dapat dibaca', async () => {
  stubFetch(daftarResponse({ permintaan_terbaca: false }))
  renderPage()

  expect(await screen.findByText(/Penanda permintaan tidak dapat ditampilkan/)).toBeVisible()
  // Daftarnya TETAP tampil — itulah yang membedakannya dari kegagalan.
  expect(screen.getByText('PNCN.26.0001')).toBeVisible()
})

it('menampilkan galat permintaan di dalam dialog, tanpa menutupnya', async () => {
  stubFetch(daftarResponse(), () =>
    jsonResponse(409, {
      kode: 'permintaan_masih_menunggu',
      pesan: 'Permintaan sejenis atas klaim ini masih menunggu dijalankan.',
    }),
  )
  renderPage()
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'Copy Klaim' }))
  await userEvent.click(screen.getByRole('button', { name: 'Ajukan salin klaim' }))

  const dialog = await screen.findByRole('dialog')
  expect(within(dialog).getByText(/masih menunggu dijalankan/)).toBeVisible()
})

it('menuntun pengguna memilih entitas sebelum menembak server', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(daftarResponse())
  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeVisible()
  expect(calls).toHaveLength(0)
})

/**
 * Pengajuan yang tertutup dinyatakan DI LAYAR, dan tombolnya dimatikan.
 *
 * Penjaganya When rule `IsGCNMUser` yang isinya `1 = 2` — selalu salah, sehingga hari ini
 * kedua aksi tidak dapat dipakai siapa pun (keputusan Work Owner 2026-09-23).
 *
 * Yang dijaga uji ini bukan penolakannya — itu urusan server — melainkan bahwa pengguna
 * mengetahuinya SEBELUM menekan. Tombol yang tampak aktif lalu selalu gagal terbaca sebagai
 * gangguan sistem, bukan sebagai kewenangan yang memang tidak ada.
 */
it('menonaktifkan kedua tombol dan menyebutkan alasannya saat pengajuan tertutup', async () => {
  stubFetch(
    daftarResponse({
      boleh_mengajukan: false,
      alasan_tidak_boleh: 'Pengajuan sedang tidak tersedia untuk semua pengguna.',
      pelaksana_belum_ada: false,
    }),
  )
  renderPage()
  await screen.findByText('PNCN.26.0001')

  expect(screen.getByText('Buka kembali dan salin klaim belum tersedia')).toBeVisible()
  expect(
    screen.getByText('Pengajuan sedang tidak tersedia untuk semua pengguna.'),
  ).toBeVisible()

  expect(screen.getByRole('button', { name: 'ReOpen' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Copy Klaim' })).toBeDisabled()
})

/**
 * Tombolnya TETAP digambar, bukan disembunyikan.
 *
 * Kolom "Tindakan" yang kosong tanpa sebab yang terbaca akan dilaporkan sebagai fitur yang
 * hilang. Tombol mati yang menjelaskan dirinya sendiri lebih jujur daripada ruang kosong.
 */
it('tetap menggambar tombolnya meski tidak dapat ditekan', async () => {
  stubFetch(daftarResponse({ boleh_mengajukan: false, pelaksana_belum_ada: false }))
  renderPage()
  await screen.findByText('PNCN.26.0001')

  expect(screen.getByRole('button', { name: 'ReOpen' })).toBeVisible()
  expect(screen.getByRole('button', { name: 'Copy Klaim' })).toBeVisible()
})

/**
 * Keterangan pelaksana muncul saat pengajuannya TERBUKA, dan hanya saat itu.
 *
 * Menampilkan keduanya sekaligus akan membuat pengguna mengira tombolnya mati karena
 * pelaksananya belum ada, padahal sebabnya kewenangan.
 */
it('menyatakan bahwa permintaan belum ada yang menjalankan', async () => {
  stubFetch(daftarResponse({ boleh_mengajukan: true, pelaksana_belum_ada: true }))
  renderPage()
  await screen.findByText('PNCN.26.0001')

  expect(
    screen.getByText('Permintaan tercatat, tetapi belum ada yang menjalankannya'),
  ).toBeVisible()
  expect(screen.queryByText('Buka kembali dan salin klaim belum tersedia')).toBeNull()
})

/**
 * Sebelum daftarnya dimuat, kewenangan BELUM DIKETAHUI — dan itu diperlakukan sebagai tidak
 * boleh.
 *
 * Menganggapnya boleh akan membuat tombolnya tampak aktif sekejap lalu berubah, dan pengguna
 * yang menekannya tepat pada saat itu menerima galat yang tidak dapat ia jelaskan.
 */
it('tidak mengaktifkan tombol sebelum kewenangannya diketahui', async () => {
  // Jawaban ditahan supaya layar tetap dalam keadaan memuat.
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url.includes('/penyaring')) {
      return Promise.resolve(jsonResponse(200, penyaringResponse))
    }
    return new Promise<Response>(() => {})
  })
  renderPage()

  await waitFor(() => expect(calls.length).toBeGreaterThan(0))
  expect(screen.queryByRole('button', { name: 'ReOpen' })).toBeNull()
})
