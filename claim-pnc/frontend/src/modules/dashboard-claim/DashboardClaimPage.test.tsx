import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DashboardClaimPage } from './DashboardClaimPage'
import type { PenyaringResponse, RingkasanResponse, TelusurResponse } from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen.
 */
const penyaringResponse: PenyaringResponse = {
  lini_bisnis: [
    { nilai: 'ALL', label: 'Semua Lini Bisnis' },
    { nilai: 'NONMBU', label: 'Non-MBU' },
    { nilai: 'BONDING', label: 'Bonding' },
    { nilai: 'PA', label: 'Personal Accident' },
    { nilai: 'TRAVEL', label: 'Travel' },
  ],
  tile: [
    { tile: 'outstanding', judul: 'Outstanding', bentuk: 'klaim' },
    { tile: 'close-claim', judul: 'Close Claim', bentuk: 'klaim' },
    { tile: 'loss-adjuster', judul: 'Loss Adjuster', bentuk: 'survei' },
    { tile: 'internal-surveyor', judul: 'Internal Surveyor', bentuk: 'survei' },
  ],
  portal: 'ASM',
}

function ringkasanResponse(partial: Partial<RingkasanResponse> = {}): RingkasanResponse {
  return {
    kartu: [
      { tile: 'outstanding', judul: 'Outstanding', jumlah: 247, bentuk: 'klaim' },
      { tile: 'close-claim', judul: 'Close Claim', jumlah: 1302, bentuk: 'klaim' },
      { tile: 'loss-adjuster', judul: 'Loss Adjuster', jumlah: 18, bentuk: 'survei' },
      { tile: 'internal-surveyor', judul: 'Internal Surveyor', jumlah: 64, bentuk: 'survei' },
    ],
    lini_bisnis: 'ALL',
    portal: 'ASM',
    catatan_warisan: [
      'Angka pada kartu Loss Adjuster menghitung jumlah klaim, telusurnya menampilkan baris survei.',
    ],
    ...partial,
  }
}

function telusurKlaim(): TelusurResponse {
  return {
    tile: 'outstanding',
    judul: 'Outstanding',
    bentuk: 'klaim',
    klaim: [
      {
        klaim_id: 'CONTOH-BERJALAN-1',
        nomor_klaim: 'PNCN.26.0001',
        nomor_polis: 'POLIS-CONTOH-0001',
        nama_tertanggung: 'Tertanggung Contoh A',
        nama_bisnis: 'Aneka',
        sumber_bisnis: 'Cabang',
        nama_cabang: 'Jakarta',
        pic_teknik: 'PIC Contoh 1',
        admin_pnc: 'Admin Contoh 1',
        tanggal_pendaftaran: '2026-08-05',
        tanggal_lapor: '2026-08-06',
        tanggal_kejadian: '2026-07-04',
        lama_hari: 2191,
        status_klaim_kode: '1147',
        status_klaim_label: 'Register',
        status_proses: 'New',
      },
    ],
    survei: [],
    halaman: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
    lini_bisnis: 'ALL',
    portal: 'ASM',
  }
}

function telusurSurvei(): TelusurResponse {
  return {
    tile: 'loss-adjuster',
    judul: 'Loss Adjuster',
    bentuk: 'survei',
    klaim: [],
    survei: [
      {
        survei_id: 'CONTOH-SURVEI-1',
        nomor_survei: 'SRV-CONTOH-0001',
        nomor_klaim: 'PNCN.26.0001',
        nomor_polis: 'POLIS-CONTOH-0001',
        nama_tertanggung: 'Tertanggung Contoh A',
        nomor_referensi: 'REF-CONTOH-01',
        nama_surveyor: 'Adjuster Contoh 1',
        pic_teknik: 'PIC Contoh 1',
        pic_adjuster: 'PIC Adjuster Contoh 1',
        lokasi_survei: 'Jakarta',
        tanggal_survei: '',
        tanggal_tugas: '2026-08-06',
        lama_hari: 2191,
        status_survei: 'Assigned',
        status_proses: 'Open',
      },
    ],
    halaman: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
    lini_bisnis: 'ALL',
    portal: 'ASM',
  }
}

type Call = { url: string }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/**
 * Menjawab ketiga GET layar ini, dan mencatat setiap URL yang diminta.
 *
 * URL-nya dicatat supaya uji dapat memeriksa APA yang dikirim layar — penyaring yang tidak
 * ikut terkirim adalah kegagalan yang tampak seperti "datanya memang begitu".
 */
function stubFetch(
  ringkasan: RingkasanResponse | (() => Response),
  telusur?: TelusurResponse | (() => Response),
) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push({ url })

    if (url.includes('/penyaring')) {
      return Promise.resolve(jsonResponse(200, penyaringResponse))
    }
    if (url.includes('/ringkasan')) {
      return Promise.resolve(
        typeof ringkasan === 'function' ? ringkasan() : jsonResponse(200, ringkasan),
      )
    }
    if (telusur === undefined) {
      return Promise.resolve(jsonResponse(500, { kode: 'galat_internal', pesan: 'tak diduga' }))
    }
    return Promise.resolve(typeof telusur === 'function' ? telusur() : jsonResponse(200, telusur))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DashboardClaimPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: '2026-09-27T12:00:00Z',
  })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('menggambar keempat kartu dalam urutan yang sama dengan sistem lama', async () => {
  stubFetch(ringkasanResponse())
  renderPage()

  await screen.findByText('247')

  // Urutannya `param.tipe` 0…3 pada `SetDashboardClaim` — bukan abjad (`D-13`).
  const judul = screen.getAllByRole('button', { pressed: false }).map((b) => b.textContent ?? '')
  const berurutan = judul.filter((teks) =>
    /Outstanding|Close Claim|Loss Adjuster|Internal Surveyor/.test(teks),
  )

  expect(berurutan[0]).toContain('Outstanding')
  expect(berurutan[1]).toContain('Close Claim')
  expect(berurutan[2]).toContain('Loss Adjuster')
  expect(berurutan[3]).toContain('Internal Surveyor')
})

it('menyatakan bahwa angkanya milik seluruh organisasi, bukan pengguna', async () => {
  stubFetch(ringkasanResponse())
  renderPage()

  await screen.findByText('247')

  // Layar ini mudah tertukar dengan My Inbox, dan tertukarnya tidak menghasilkan galat
  // apa pun — hanya angka yang dibaca keliru. Kalimatnya karena itu bagian dari layar.
  expect(screen.getByText(/seluruh organisasi/i)).toBeInTheDocument()
})

it('tidak menelusur apa pun sebelum sebuah kartu dipilih', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()

  await screen.findByText('247')

  expect(screen.getByText(/Pilih salah satu kartu/i)).toBeInTheDocument()
  expect(calls.some((c) => /dashboard-claim\/(outstanding|close-claim)/.test(c.url))).toBe(false)
})

it('menelusur tile yang dipilih dan menggambar kolom klaim', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()

  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /Outstanding/ }))

  await screen.findByText('PNCN.26.0001')

  // Kedua belas kolom grid Outstanding pada layar lama, dalam urutannya.
  //
  // Dicari sebagai `columnheader`, bukan sebagai teks: DataTable menggambar nama kolom DUA
  // KALI — sekali di `<th>`, sekali lagi sebagai label kartu pada tampilan sempit. Mencari
  // teksnya akan menemukan keduanya dan gagal karena ganda, bukan karena kolomnya salah.
  for (const judul of [
    'No Klaim',
    'No Polis',
    'Nama Tertanggung',
    'Nama Bisnis',
    'Sumber Bisnis',
    'Nama Cabang',
    'Tanggal Pendaftaran',
    'Report Date',
    'Lama Waktu Klaim',
    'PIC Teknik',
    'Admin PNC',
    'Claim status',
  ]) {
    expect(screen.getByRole('columnheader', { name: judul })).toBeVisible()
  }
})

it('tidak menggambar Report Date dan Claim status pada tile Close Claim', async () => {
  const closeClaim = { ...telusurKlaim(), tile: 'close-claim' as const, judul: 'Close Claim' }
  stubFetch(ringkasanResponse(), closeClaim)
  renderPage()

  await screen.findByText('1.302')
  await userEvent.click(screen.getByRole('button', { name: /Close Claim/ }))
  await screen.findByText('PNCN.26.0001')

  // Layar lama tidak menggambar keduanya di sini. Menyamakannya dengan Outstanding akan
  // menampilkan kolom yang datanya memang tidak dibaca — dan kolom kosong tidak dapat
  // dibedakan dari data yang hilang.
  expect(screen.queryByRole('columnheader', { name: /Report Date/i })).not.toBeInTheDocument()
  expect(screen.queryByRole('columnheader', { name: /Claim status/i })).not.toBeInTheDocument()

  // Yang memang ada di grid Close Claim.
  for (const judul of ['Lama Waktu Klaim', 'PIC Teknik', 'Admin PNC']) {
    expect(screen.getByRole('columnheader', { name: judul })).toBeVisible()
  }
})

it('menggambar kolom survei pada tile bertipe survei', async () => {
  stubFetch(ringkasanResponse(), telusurSurvei())
  renderPage()

  await screen.findByText('18')
  await userEvent.click(screen.getByRole('button', { name: /Loss Adjuster/ }))

  await screen.findByText('SRV-CONTOH-0001')

  // Kesepuluh kolom grid Loss Adjuster pada layar lama — judulnya berbahasa INGGRIS, dan
  // itu memang begitu di sana (`D-13`).
  for (const judul of [
    'Appointment No',
    'Reference No',
    'Claim No',
    'Policy No',
    'Adjuster',
    'PIC Adjuster',
    'Insured Name',
    'PIC ASM',
    'Status',
    'Aging',
  ]) {
    expect(screen.getByRole('columnheader', { name: judul })).toBeVisible()
  }

  // Lokasi dan Tanggal Survey TIDAK ada di grid Loss Adjuster — kuerinya memang tidak
  // mengambil tanggalnya.
  expect(screen.queryByRole('columnheader', { name: /Lokasi/i })).not.toBeInTheDocument()
  expect(screen.queryByRole('columnheader', { name: /Tanggal Survey/i })).not.toBeInTheDocument()
})

it('menggambar kolom Internal Surveyor yang BERBEDA dari Loss Adjuster', async () => {
  const internal = {
    ...telusurSurvei(),
    tile: 'internal-surveyor' as const,
    judul: 'Internal Surveyor',
  }
  stubFetch(ringkasanResponse(), internal)
  renderPage()

  await screen.findByText('64')
  await userEvent.click(screen.getByRole('button', { name: /Internal Surveyor/ }))
  await screen.findByText('SRV-CONTOH-0001')

  // Kesepuluh kolomnya, judulnya berbahasa INDONESIA — berbeda dari Loss Adjuster.
  for (const judul of [
    'Nomor Case',
    'No Klaim',
    'No Polis',
    'Nama Tertanggung',
    'Aging',
    'Tanggal Survey',
    'Lokasi Survey',
    'Status Survey',
    'PIC ASM',
    'Surveyor',
  ]) {
    expect(screen.getByRole('columnheader', { name: judul })).toBeVisible()
  }

  // Judul khas Loss Adjuster tidak boleh muncul di sini.
  expect(screen.queryByRole('columnheader', { name: /Appointment No/i })).not.toBeInTheDocument()
  expect(screen.queryByRole('columnheader', { name: /PIC Adjuster/i })).not.toBeInTheDocument()
})

it('menandai tanggal survei yang kosong pada tile Internal Surveyor', async () => {
  const internal = {
    ...telusurSurvei(),
    tile: 'internal-surveyor' as const,
    judul: 'Internal Surveyor',
  }
  stubFetch(ringkasanResponse(), internal)
  renderPage()

  await screen.findByText('64')
  await userEvent.click(screen.getByRole('button', { name: /Internal Surveyor/ }))

  const sel = await screen.findByText('SRV-CONTOH-0001')

  // Hanya tile INI yang menggambar kolom Tanggal Survey — grid Loss Adjuster tidak
  // memilikinya sama sekali, dan itu sudah dijaga uji kolomnya sendiri.
  //
  // Sel kosong tidak dapat dibedakan dari data yang hilang; tanda pisah menyatakan bahwa
  // memang tidak ada. Diperiksa DI DALAM barisnya, bukan di seluruh layar.
  const baris = sel.closest('tr')
  expect(baris).not.toBeNull()
  expect(within(baris as HTMLElement).getByText('—')).toBeVisible()
})

it('mengirim penyaring lini bisnis ke server, bukan menyaring di peramban', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()

  await screen.findByText('247')
  await userEvent.selectOptions(screen.getByLabelText('Lini Bisnis'), 'PA')

  await waitFor(() => {
    expect(calls.some((c) => c.url.includes('lini_bisnis=PA'))).toBe(true)
  })
})

it('tidak menggambar panel keterangan apa pun', async () => {
  stubFetch(ringkasanResponse())
  renderPage()

  await screen.findByText('247')

  // Kedua panel dihapus dari layar atas permintaan Work Owner 2026-09-26. Field-nya tetap
  // ada di respons `/ringkasan` dan dibaca penguji gerbang 1; yang berubah hanya
  // tampilannya.
  //
  // Uji ini menjaga penghapusan itu tetap disengaja: bila panelnya kembali tanpa keputusan,
  // uji inilah yang menanyakannya.
  expect(screen.queryByText(/Perbedaan yang disengaja/i)).not.toBeInTheDocument()
  expect(screen.queryByText(/Kenapa angka kartu bisa berbeda/i)).not.toBeInTheDocument()
})

it('menampilkan galat ringkasan tanpa mengosongkan layar', async () => {
  stubFetch(() => jsonResponse(503, { kode: 'portal_belum_siap', pesan: 'Portal belum siap.' }))
  renderPage()

  await screen.findByText(/Ringkasan tidak dapat dibaca/i)

  // Kerangka kartunya tetap tergambar. Layar yang kosong total tidak memberi tahu apa pun
  // tentang apa yang gagal.
  expect(screen.getByText('Outstanding')).toBeInTheDocument()
})

it('menolak menembak apa pun sebelum portal dipilih', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(ringkasanResponse())
  renderPage()

  expect(screen.getByText(/Portal entitas belum dipilih/i)).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})

it('menggambar kedua tab layar lama', async () => {
  stubFetch(ringkasanResponse())
  renderPage()

  await screen.findByText('247')

  // Layar lama punya DUA tab. Memindahkan berarti membawa keduanya.
  expect(screen.getByRole('tab', { name: 'Dashboard Claim' })).toBeVisible()
  expect(screen.getByRole('tab', { name: 'Inbox Tampungan PIC' })).toBeVisible()
})

it('membuka tab Tampungan PIC dengan kolom grid layar lama', async () => {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push({ url })
    if (url.includes('/penyaring')) return Promise.resolve(jsonResponse(200, penyaringResponse))
    if (url.includes('/ringkasan')) return Promise.resolve(jsonResponse(200, ringkasanResponse()))
    if (url.includes('/tampungan')) {
      return Promise.resolve(
        jsonResponse(200, {
          klaim: [
            {
              klaim_id: 'CONTOH-TAMPUNG-1',
              nomor_klaim: 'PNCN.26.0201',
              nomor_polis: 'POLIS-CONTOH-0201',
              nama_tertanggung: 'Tertanggung Contoh H',
              nama_bisnis: 'Aneka',
              sumber_bisnis: 'Cabang',
              nama_cabang: 'Jakarta',
              admin_pnc: 'Admin Contoh 1',
              tanggal_pendaftaran: '2026-08-14',
            },
          ],
          halaman: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
          portal: 'ASM',
        }),
      )
    }
    return Promise.resolve(jsonResponse(500, { kode: 'galat_internal', pesan: 'tak diduga' }))
  })

  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('tab', { name: 'Inbox Tampungan PIC' }))

  await screen.findByText('PNCN.26.0201')

  // Kedelapan kolom grid layar lama.
  for (const judul of [
    'No Klaim',
    'No Polis',
    'Nama Tertanggung',
    'Nama Bisnis',
    'Sumber Bisnis',
    'Nama Cabang',
    'Tanggal Pendaftaran',
    'Admin PNC',
  ]) {
    expect(screen.getByRole('columnheader', { name: judul })).toBeVisible()
  }

  // PIC Teknik TIDAK ada: menurut definisi tab ini, klaimnya memang belum punya.
  expect(screen.queryByRole('columnheader', { name: 'PIC Teknik' })).not.toBeInTheDocument()
})

it('tidak menembak tab Tampungan sebelum tabnya dibuka', async () => {
  stubFetch(ringkasanResponse())
  renderPage()

  await screen.findByText('247')

  // Membuka layar tidak boleh menarik daftar yang belum tentu dilihat siapa pun.
  expect(calls.some((c) => c.url.includes('/tampungan'))).toBe(false)
})

it('menggambar tombol Export dan Transfer pada tile Outstanding', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()

  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  // Ketiganya ada di layar lama: Export to Excel, Transfer per baris, dan Transfer All.
  expect(screen.getByRole('button', { name: 'Export to Excel' })).toBeVisible()
  expect(screen.getByRole('button', { name: 'Transfer' })).toBeVisible()
  expect(screen.getByRole('button', { name: 'Transfer All Case By UserID' })).toBeVisible()
})

it('menyatakan bahwa Transfer hanya MENCATAT permintaan', async () => {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url })
    if (init?.method === 'POST') {
      return Promise.resolve(
        jsonResponse(201, {
          permintaan: {
            id: 'abc',
            lingkup: 'baris',
            nomor_klaim: 'PNCN.26.0001',
            user_id_baru: 'BUDI',
            status: 'menunggu',
            pemohon: 'jonny',
            pada: '2026-10-05 14:00',
          },
          portal: 'ASM',
          pelaksana_belum_ada: true,
        }),
      )
    }
    if (url.includes('/penyaring')) return Promise.resolve(jsonResponse(200, penyaringResponse))
    if (url.includes('/ringkasan')) return Promise.resolve(jsonResponse(200, ringkasanResponse()))
    return Promise.resolve(jsonResponse(200, telusurKlaim()))
  })

  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'Transfer' }))
  await userEvent.type(screen.getByLabelText('User ID Baru'), 'BUDI')
  await userEvent.click(screen.getByRole('button', { name: 'Ajukan Transfer' }))

  // Penugasannya BELUM berpindah — barisnya masih akan tampil. Tanpa keterangan ini,
  // pengguna menekan Transfer lagi, dan setiap penekanan mencatat satu permintaan.
  await screen.findByText(/belum berpindah/i)
})

it('menolak Transfer massal tanpa User ID Lama di sisi server', async () => {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url })
    if (init?.method === 'POST') {
      return Promise.resolve(
        jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Permintaan tidak dapat diproses.',
          detail: [{ field: 'user_id_lama', pesan: 'User ID Lama wajib diisi pada transfer massal.' }],
        }),
      )
    }
    if (url.includes('/penyaring')) return Promise.resolve(jsonResponse(200, penyaringResponse))
    return Promise.resolve(jsonResponse(200, ringkasanResponse()))
  })

  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: 'Transfer All Case By UserID' }))
  await userEvent.type(screen.getByLabelText('User ID Baru'), 'BUDI')
  await userEvent.click(screen.getByRole('button', { name: 'Ajukan Transfer' }))

  // Pesannya ditempelkan ke isian yang bersangkutan, bukan ditulis sebagai galat umum.
  await screen.findByText(/User ID Lama wajib diisi/i)
})

/** Telusur tile Close Claim — dua baris, supaya pemilihan sebagian dapat diuji. */
function telusurCloseClaim(): TelusurResponse {
  const dasar = telusurKlaim()
  const satu = dasar.klaim[0]!
  return {
    ...dasar,
    tile: 'close-claim',
    judul: 'Close Claim',
    klaim: [
      satu,
      { ...satu, klaim_id: 'CONTOH-TUTUP-2', nomor_klaim: 'PNCN.26.0002' },
    ],
    halaman: { halaman: 1, ukuran: 25, total: 2, total_halaman: 1 },
  }
}

/**
 * Melayani keempat jalur yang dipakai tile Close Claim, termasuk `/penyaring` milik modul
 * Inbox Close Claim yang menjawab izin mengajukan.
 */
function stubCloseClaim(options: { boleh?: boolean; alasan?: string; gagalKe?: string } = {}) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url })

    if (url.includes('/inbox-close-claim/permintaan')) {
      const body = JSON.parse(String(init?.body ?? '{}')) as { klaim_id?: string }
      if (options.gagalKe !== undefined && body.klaim_id === options.gagalKe) {
        return Promise.resolve(
          jsonResponse(409, { kode: 'konflik', pesan: 'Permintaan serupa sudah tercatat.' }),
        )
      }
      return Promise.resolve(jsonResponse(201, { permintaan: { id: 'x' } }))
    }
    if (url.includes('/inbox-close-claim/penyaring')) {
      return Promise.resolve(
        jsonResponse(200, {
          boleh_mengajukan: options.boleh ?? true,
          ...(options.alasan !== undefined ? { alasan_tidak_boleh: options.alasan } : {}),
        }),
      )
    }
    if (url.includes('/penyaring')) return Promise.resolve(jsonResponse(200, penyaringResponse))
    if (url.includes('/ringkasan')) return Promise.resolve(jsonResponse(200, ringkasanResponse()))
    return Promise.resolve(jsonResponse(200, telusurCloseClaim()))
  })
}

async function bukaCloseClaim() {
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Close Claim/ }))
  await screen.findByText('PNCN.26.0002')
}

it('menggambar kolom Pilih pada Close Claim, dan TIDAK pada Outstanding', async () => {
  stubCloseClaim()
  await bukaCloseClaim()

  // "Pilih" adalah caption nyata pada Section/InboxManagerReopen1_Sec.
  expect(screen.getByRole('columnheader', { name: 'Pilih' })).toBeVisible()
  expect(screen.getAllByRole('checkbox')).toHaveLength(2)

  // Tidak ada kotak centang kepala kolom: Pega hanya punya SATU kontrol checkbox, yaitu
  // yang per baris. "Pilih Semua" akan memudahkan tindakan yang tidak dapat dibatalkan.
  expect(screen.queryByRole('checkbox', { name: /semua/i })).toBeNull()
})

it('mematikan ReOpen dan Copy Klaim sampai ada baris dicentang', async () => {
  stubCloseClaim()
  await bukaCloseClaim()

  expect(screen.getByRole('button', { name: 'ReOpen' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Copy Klaim' })).toBeDisabled()

  await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0001' }))

  expect(screen.getByRole('button', { name: 'ReOpen' })).toBeEnabled()
  expect(screen.getByRole('button', { name: 'Copy Klaim' })).toBeEnabled()
})

it('mengirim satu permintaan per baris yang dicentang', async () => {
  stubCloseClaim()
  await bukaCloseClaim()

  await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0001' }))
  await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0002' }))
  await userEvent.click(screen.getByRole('button', { name: 'ReOpen' }))

  // Dialog menyebut berapa klaim yang akan diajukan sebelum apa pun dikirim.
  await screen.findByText('2 klaim dipilih.')
  await userEvent.click(screen.getByRole('button', { name: /Ajukan ReOpen/ }))

  await screen.findByText(/2 permintaan ReOpen tercatat/i)

  // Endpoint-nya menerima SATU klaim_id, jadi dua baris berarti dua permintaan.
  const kirim = calls.filter((c) => c.url.includes('/inbox-close-claim/permintaan'))
  expect(kirim).toHaveLength(2)
})

it('melaporkan baris yang gagal satu per satu, bukan sebagai satu kegagalan', async () => {
  stubCloseClaim({ gagalKe: 'CONTOH-TUTUP-2' })
  await bukaCloseClaim()

  await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0001' }))
  await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0002' }))
  await userEvent.click(screen.getByRole('button', { name: 'Copy Klaim' }))
  await userEvent.click(screen.getByRole('button', { name: /Ajukan Copy Klaim/ }))

  // Yang berhasil DAN yang gagal sama-sama dinyatakan. Meringkasnya menjadi satu "gagal"
  // akan membuat pengguna mengirim ulang seluruhnya.
  await screen.findByText(/1 permintaan Copy Klaim tercatat/i)
  expect(screen.getByText(/1 permintaan tidak tercatat/i)).toBeVisible()
  expect(screen.getByText(/Permintaan serupa sudah tercatat/i)).toBeVisible()
})

it('menyatakan sebabnya saat peran tidak boleh mengajukan', async () => {
  stubCloseClaim({ boleh: false, alasan: 'Hanya Manager PNC yang dapat mengajukan.' })
  await bukaCloseClaim()

  // Alasannya datang dari domain modul Inbox Close Claim, bukan ditulis di layar ini.
  await screen.findByText('Hanya Manager PNC yang dapat mengajukan.')

  await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0001' }))
  expect(screen.getByRole('button', { name: 'ReOpen' })).toBeDisabled()
})

it('membuang centang saat berpindah halaman', async () => {
  stubCloseClaim()
  await bukaCloseClaim()

  await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0001' }))
  expect(screen.getByRole('button', { name: 'ReOpen' })).toBeEnabled()

  // Berpindah tile lalu kembali menyetarakan berpindah halaman: barisnya berganti, dan
  // centang yang tidak terlihat tidak boleh ikut terkirim.
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await userEvent.click(screen.getByRole('button', { name: /^Close Claim/ }))
  await screen.findByText('PNCN.26.0002')

  expect(screen.getByRole('button', { name: 'ReOpen' })).toBeDisabled()
})
