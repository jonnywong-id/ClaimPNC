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
    selisih_terencana: ['Kolom No Klaim menampilkan nomor klaim induk, bukan kunci Pega.'],
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
        tanggal_register: '2026-08-05',
        tanggal_kejadian: '2026-07-04',
        status_klaim_kode: '1147',
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

  // Keenam kolom pertama mengikuti `Section/DashboardClaim_Section2-Section.xml`.
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
  ]) {
    expect(screen.getByRole('columnheader', { name: new RegExp(judul, 'i') })).toBeVisible()
  }
})

it('menggambar kolom survei pada tile bertipe survei', async () => {
  stubFetch(ringkasanResponse(), telusurSurvei())
  renderPage()

  await screen.findByText('18')
  await userEvent.click(screen.getByRole('button', { name: /Loss Adjuster/ }))

  await screen.findByText('SRV-CONTOH-0001')

  // Kolomnya mengikuti `Section/DashboardClaim_Section1-Section.xml`.
  for (const judul of ['No Survey', 'Surveyor', 'Status Survey', 'Tanggal Survey']) {
    expect(screen.getByRole('columnheader', { name: new RegExp(judul, 'i') })).toBeVisible()
  }
})

it('menandai tanggal survei yang memang tidak diambil, bukan mengosongkannya diam-diam', async () => {
  stubFetch(ringkasanResponse(), telusurSurvei())
  renderPage()

  await screen.findByText('18')
  await userEvent.click(screen.getByRole('button', { name: /Loss Adjuster/ }))

  const sel = await screen.findByText('SRV-CONTOH-0001')

  // Kueri loss adjuster memang tidak mengambil tanggal survei. Sel kosong tidak dapat
  // dibedakan dari data yang hilang; tanda pisah menyatakan bahwa memang tidak ada.
  //
  // Diperiksa DI DALAM barisnya, bukan di seluruh layar: tanda pisah yang sama dipakai
  // DataTable untuk setiap sel kosong, sehingga mencarinya di seluruh layar tidak
  // membuktikan bahwa yang kosong adalah kolom yang dimaksud.
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
