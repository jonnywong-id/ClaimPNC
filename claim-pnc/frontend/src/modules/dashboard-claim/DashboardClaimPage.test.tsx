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
  status_transfer: [
    { nilai: '', label: '---PILIH STATUS TRANSFER---' },
    { nilai: 'SUDAH', label: 'Sudah Transfer' },
    { nilai: 'BELUM', label: 'Belum Transfer' },
  ],
  status_bayar: [
    { nilai: '', label: '---PILIH STATUS PEMBAYARAN---' },
    { nilai: 'LUNAS', label: 'Lunas' },
    { nilai: 'BELUM', label: 'Belum Lunas' },
  ],
  tipe_pengguna: [
    { nilai: '', label: 'Pilih Type User' },
    { nilai: 'Admin', label: 'Admin' },
    { nilai: 'PIC Teknik', label: 'PIC Teknik' },
    { nilai: 'Other', label: 'Other' },
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
        tanggal_pendaftaran: '2026-08-05 10:00:00',
        tanggal_lapor: '2026-08-06 00:00:00',
        tanggal_kejadian: '2026-07-04',
        lama_hari: 2191,
        status_klaim_kode: '1147',
        status_klaim_label: 'Register',
        status_proses: 'New',
        // Dua posisi berjalan sekaligus, digabung koma — bentuk yang dihasilkan
        // GET_POSISI_PROGRESS_PNC. Progres kedua kosong, dan itu keadaan yang sah.
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

// Badan permintaan ikut direkam: sebagian uji memeriksa APA yang dikirim, bukan hanya ke
// mana. Tanpa itu, isian yang digambar tetapi tidak pernah terkirim akan lolos.
type Call = { url: string; init?: RequestInit | undefined }

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

it('menaruh Pilih SESUDAH No Klaim di Close Claim, dan SEBELUMNYA di Outstanding', async () => {
  // Kedua grid layar lama menyusunnya terbalik satu sama lain, dan itu ditiru apa adanya:
  //   Section/InboxManagerReopen1_Sec-Section.xml  -> No Klaim, Pilih, ...
  //   Section/InboxOutstandingClaim_Section-Section.xml -> Pilih, No Klaim, ...
  // Uji ini ada supaya "merapikan" keduanya menjadi seragam gagal, bukan lolos diam-diam.
  const judulTampak = () =>
    screen
      .getAllByRole('columnheader')
      .map((el) => (el.textContent ?? '').trim())
      .filter(Boolean)

  const closeClaim = { ...telusurKlaim(), tile: 'close-claim' as const, judul: 'Close Claim' }
  stubFetch(ringkasanResponse(), closeClaim)
  const layar = renderPage()

  await screen.findByText('1.302')
  await userEvent.click(screen.getByRole('button', { name: /Close Claim/ }))
  await screen.findByText('PNCN.26.0001')

  const close = judulTampak()
  expect(close.indexOf('No Klaim')).toBeGreaterThanOrEqual(0)
  expect(close.indexOf('No Klaim')).toBeLessThan(close.indexOf('Pilih'))

  layar.unmount()

  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()
  await screen.findByText('1.302')
  await userEvent.click(screen.getByRole('button', { name: /Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  const outs = judulTampak()
  expect(outs.indexOf('Pilih')).toBeGreaterThanOrEqual(0)
  expect(outs.indexOf('Pilih')).toBeLessThan(outs.indexOf('No Klaim'))
})

it('memakai teks pyNoSelectionText layar lama pada kedua dropdown penyaring', async () => {
  // Disalin apa adanya dari elemen `pyNoSelectionText` pada
  // `Section/FilterDashboardClaim_sec-Section.xml` (`D-13`), termasuk huruf besar dan tanda
  // hubungnya. Sebelum Section itu diterima (2026-10-08) keduanya berbunyi
  // "Semua Status Transfer" / "Semua Status Pembayaran" — karangan kita sendiri.
  //
  // Teksnya terbaca aneh, dan itu justru alasan uji ini ada: ia yang paling mungkin
  // "dirapikan" kembali oleh orang yang mengira itu kekeliruan.
  stubFetch(ringkasanResponse())
  renderPage()
  await screen.findByText('247')

  await userEvent.click(screen.getByRole('button', { name: /Penyaring|Filter/i }))

  expect(
    screen.getByRole('option', { name: '---PILIH STATUS TRANSFER---' }),
  ).toBeInTheDocument()
  expect(
    screen.getByRole('option', { name: '---PILIH STATUS PEMBAYARAN---' }),
  ).toBeInTheDocument()
  expect(screen.queryByRole('option', { name: /^Semua Status/ })).not.toBeInTheDocument()
})

it('membagi popup rincian menjadi tab, dalam urutan harness layar lama', async () => {
  // Urutan ini disalin dari elemen pyTitle pada
  // Harness/ViewTempDetailClaim-Harness.xml, bukan disusun ulang. Ia terlihat janggal —
  // Penerima Pembayaran Klaim mendahului Registrasi — dan itu justru alasan uji ini ada:
  // merapikannya agar "masuk akal" adalah membuat model sendiri lagi.
  vi.stubGlobal('fetch', (url: string) => {
    calls.push({ url })
    if (url.includes('/klaim/')) {
      return Promise.resolve(
        jsonResponse(200, {
          klaim_id: 'CONTOH-1',
          nomor_klaim: 'PNCN.26.0001',
          status_proses: 'New',
          status_klaim: '1147',
          pic_teknik: 'PIC Contoh 1',
          admin_pnc: 'Admin Contoh 1',
          didaftarkan_pada: '2026-08-05T03:00:00Z',
          dokumen: { ClaimData: { ObjectList: [{ ObjectName: 'Objek Contoh' }] } },
          portal: 'ASM',
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
  await userEvent.click(await screen.findByRole('button', { name: 'PNCN.26.0001' }))

  const popup = await screen.findByRole('dialog', { name: /Detail Klaim PNCN.26.0001/i })
  const judul = within(popup)
    .getAllByRole('tab')
    .map((el) => (el.textContent ?? '').trim())

  expect(judul).toEqual([
    'Penerima Pembayaran Klaim',
    'Registrasi',
    'Hasil Survey',
    'Estimasi',
    'Adjustment & Akseptasi',
    'Dokumen',
    'Progress Klaim',
  ])

  // Riwayat lampiran BUKAN tab — di layar lama ia grid di bawah isi tab.
  expect(within(popup).queryByRole('tab', { name: /Attachment|Lampiran/i })).not.toBeInTheDocument()
  expect(within(popup).getByText('Attachment')).toBeVisible()
})

it('klaim PA memunculkan tab Catatan Untuk Analis dan kolom objek versi PA', async () => {
  // Penjaganya disalin dari pyContainerVisibleWhen di harness dan section:
  //   tab "Catatan Untuk Analis"  IsPA      -> Policy.Quotation.GroupPanel = "002"
  //   grid objek versi PA         IsPA      -> Nama / Perkerjaan / Tanggal Lahir
  //   grid objek versi Travel     IsTravel  -> Nama Peserta / Status / KTP/Paspor / ...
  //
  // Uji ini ada karena tebakan sebelumnya TERBALIK: saya mengira kolom Peserta milik PA.
  // "Perkerjaan" memang salah ketik di layar lama, dan dibawa apa adanya.
  vi.stubGlobal('fetch', (url: string) => {
    calls.push({ url })
    if (url.includes('/klaim/')) {
      return Promise.resolve(
        jsonResponse(200, {
          klaim_id: 'CONTOH-PA',
          nomor_klaim: 'PNCN.26.0001',
          status_proses: 'New',
          status_klaim: '1147',
          pic_teknik: 'PIC Contoh 1',
          admin_pnc: 'Admin Contoh 1',
          didaftarkan_pada: '2026-08-05T03:00:00Z',
          dokumen: {
            Policy: { Quotation: { GroupPanel: '002' } },
            ClaimData: {
              ObjectList: [{ ObjectName: 'Peserta Contoh', ObjectJob: 'Karyawan' }],
            },
          },
          portal: 'ASM',
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
  await userEvent.click(await screen.findByRole('button', { name: 'PNCN.26.0001' }))

  const popup = await screen.findByRole('dialog', { name: /Detail Klaim PNCN.26.0001/i })

  // Tab bersyarat IsPA kini muncul — pada klaim non-PA ia tidak.
  expect(within(popup).getByRole('tab', { name: 'Catatan Untuk Analis' })).toBeVisible()

  await userEvent.click(within(popup).getByRole('tab', { name: 'Registrasi' }))
  expect(within(popup).getByRole('columnheader', { name: 'Perkerjaan' })).toBeVisible()
  expect(
    within(popup).queryByRole('columnheader', { name: 'KTP/Paspor' }),
  ).not.toBeInTheDocument()
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

  // Tombol unduh TIDAK ada. Layar lama tidak punya padanannya: Section2 memuat NOL aksi
  // `openUrlInWindow` — cara keempat ekspor tile dipanggil, yang muncul 4x di Section1 —
  // dan tidak ada activity `Export*` yang menyentuh klaim ber-`USERTEKNIS_1 IS NULL`.
  //
  // Satu tombol unduh pernah ada di sini dan dicabut 2026-10-07. Uji ini yang menanyakannya
  // bila ia kembali tanpa keputusan.
  expect(screen.queryByRole('button', { name: /Export|Unduh/i })).not.toBeInTheDocument()
  expect(calls.some((c) => c.url.includes('/tampungan/unduh'))).toBe(false)
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

  // Keempatnya ada di layar lama, dan SELURUHNYA milik grid — bukan panel penyaring.
  expect(screen.getByRole('button', { name: 'Export to Excel' })).toBeVisible()
  expect(screen.getByRole('button', { name: 'Transfer' })).toBeVisible()
  expect(screen.getByRole('button', { name: 'Select All' })).toBeVisible()
  expect(screen.getByRole('button', { name: 'Transfer All Case By UserID' })).toBeVisible()
})

it('tidak menggambar tombol grid sebelum sebuah kartu dipilih', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()
  await screen.findByText('247')

  // Select All dan Transfer All milik `InboxOutstandingClaim_Section` — grid Outstanding.
  // Menggambarnya di panel penyaring memisahkan tombol dari centang yang menjadi isinya,
  // dan membuatnya dapat ditekan saat tidak ada satu baris pun di layar.
  expect(screen.queryByRole('button', { name: 'Select All' })).toBeNull()
  expect(screen.queryByRole('button', { name: 'Transfer All Case By UserID' })).toBeNull()
})

/** Melayani daftar PIC Teknik beserta pengajuan transfernya. */
/**
 * @param jumlahPindah berapa klaim yang dijawab server sebagai berpindah.
 *
 * Bawaannya 1 — jawaban jalur per baris. Jalur massal tanpa centang memakai angka lain, dan
 * **nol** di sana adalah jawaban yang sah: petugasnya tidak memegang klaim berjalan.
 */
function stubTransfer(options: { gagal?: boolean; jumlahPindah?: number } = {}) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url.includes('/transfer')) {
      if (options.gagal === true) {
        return Promise.resolve(
          jsonResponse(503, {
            kode: 'transfer_belum_aktif',
            pesan: 'Pemindahan PIC Teknik belum dapat dijalankan. Hak UPDATE pada kolom PIC Teknik sedang diminta ke DBA.',
          }),
        )
      }
      return Promise.resolve(
        jsonResponse(201, {
          permintaan: { id: 'abc', jumlah_pindah: options.jumlahPindah ?? 1 },
          portal: 'ASM',
        }),
      )
    }
    if (url.includes('/pic-teknik')) {
      return Promise.resolve(
        jsonResponse(200, {
          pic: [
            { operator_id: 'PICCONTOH03', nama: 'PIC Contoh Tiga', email: '', tim: 'Tim B', beban: 1 },
            { operator_id: 'PICCONTOH01', nama: 'PIC Contoh Satu', email: '', tim: 'Tim A', beban: 3 },
          ],
          halaman: { halaman: 1, ukuran: 25, total: 2, total_halaman: 1 },
          portal: 'ASM',
        }),
      )
    }
    if (url.includes('/penyaring')) return Promise.resolve(jsonResponse(200, penyaringResponse))
    if (url.includes('/ringkasan')) return Promise.resolve(jsonResponse(200, ringkasanResponse()))
    return Promise.resolve(jsonResponse(200, telusurKlaim()))
  })
}

it('Transfer memilih PIC dari daftar, bukan mengetik User ID', async () => {
  stubTransfer()
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Transfer' }))

  // Judulnya mengikuti pyLabel layar lama, dan isinya DAFTAR — bukan isian bebas.
  await screen.findByRole('dialog', { name: 'Transfer Assignment' })
  expect(screen.queryByLabelText('User ID Baru')).toBeNull()

  // Yang tampil adalah OPERATOR ID, meski judul kolomnya "Nama". Itu bentuk layar lama:
  // satu kolom `.City`, dan nilai itu pula yang dikirim sebagai UserID ke
  // PNC_ReassignPNCTeknik.
  await screen.findByText('PICCONTOH03')

  const baris = screen.getAllByRole('button', { name: 'Assign' })
  expect(baris).toHaveLength(2)

  await userEvent.click(baris[0]!)

  // Penugasannya BENAR-BENAR berpindah, dan pesannya menyatakan itu. Bentuk pertama hanya
  // mencatat permintaan lalu berkata "belum berpindah" — antrean tanpa pelaksana.
  await screen.findByText(/PIC Teknik berpindah/i)
})

it('menggambar daftar PIC langsung, tanpa penyaring lini bisnis seperti layar lama', async () => {
  stubTransfer()
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Transfer' }))

  // Dialog TIDAK memblokir dirinya sendiri. Bentuk pertama menolak menggambar daftar tanpa
  // lini bisnis ("tidak dapat disaring"); itu aturan karangan yang mematikan fitur.
  const dialog = await screen.findByRole('dialog', { name: 'Transfer Assignment' })
  expect(screen.queryByText(/tidak dapat disaring/i)).toBeNull()

  // Daftarnya tampil LANGSUNG, tanpa satu pun pilihan yang harus diisi lebih dulu — persis
  // layar lama, yang isinya hanya daftar, paginasi, dan tombol Assign.
  await screen.findByText('PICCONTOH03')
  expect(screen.getAllByRole('button', { name: 'Assign' })).toHaveLength(2)

  // Penyaring lini bisnis TIDAK dikirim sama sekali. Di Pega ia datang dari
  // `OperatorID.pyPosition`, dan nilai itu tidak ada di sistem baru — HCC/HCQ mengembalikan
  // jabatan sebenarnya. Tiga bentuk sudah dicoba dan ketiganya keliru; yang terakhir
  // meminta pengguna memilih, langkah yang di layar lama tidak ada.
  const permintaan = calls.filter((c) => c.url.includes('/pic-teknik')).at(-1)
  expect(permintaan?.url).not.toContain('lini_bisnis')

  // Dan dropdown-nya memang tidak digambar DI DIALOG. Pencariannya dibatasi ke dialog:
  // panel penyaring layar punya "Lini Bisnis" sendiri, dan itu memang ada di layar lama.
  expect(within(dialog).queryByLabelText('Lini Bisnis')).toBeNull()
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

  // Kalimat konfirmasinya disalin apa adanya dari
  // `Section/GCNMReopenConfirmation-Section.xml` (`D-13`), termasuk huruf besar pada
  // "Membuka". Section-nya diterima 2026-10-08; sebelum itu kalimatnya karangan kita
  // sendiri. Uji ini yang menanyakannya bila ia kembali diubah.
  expect(screen.getByText(/Apakah anda yakin ingin/)).toBeVisible()
  expect(screen.getByText('Membuka')).toBeVisible()

  // Tombolnya berlabel 'Ya', mengikuti pyButtonLabel layar lama.
  await userEvent.click(screen.getByRole('button', { name: 'Ya' }))

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
    // Pertanyaan Ex Gratia ada di layar lama; jawabannya dibiarkan pada nilai bawaan.
  await screen.findByText(/Apakah klaim ini tipe Ex Gratia/i)
  await userEvent.click(
    screen.getAllByRole('button', { name: 'Copy Klaim' }).at(-1)!,
  )

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

it('mengirim ketiga isian teks panel penyaring sebagai parameter tersendiri', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  await userEvent.type(screen.getByLabelText('Nopolis'), 'POL-9')

  // Mengetik saja TIDAK menyaring. Layar lama punya tombol "Search Data", dan penyaringnya
  // baru berlaku saat tombol itu ditekan.
  expect(
    calls.filter((c) => c.url.includes('/outstanding?')).at(-1)?.url,
  ).not.toContain('nomor_polis')

  await userEvent.click(screen.getByRole('button', { name: 'Search Data' }))
  await screen.findByText('PNCN.26.0001')

  // Ketiganya adalah parameter TERSENDIRI, bukan digabung ke kotak cari: layar lama pun
  // memisahkannya, dan menggabungkannya membuat "PIC bernama POL-9" ikut cocok.
  const terakhir = calls.filter((c) => c.url.includes('/outstanding?')).at(-1)
  expect(terakhir?.url).toContain('nomor_polis=POL-9')
})

it('Clear Filter mengosongkan kotaknya DAN penyaring yang sedang berlaku', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  await userEvent.type(screen.getByLabelText('Nopolis'), 'POL-9')
  await userEvent.click(screen.getByRole('button', { name: 'Search Data' }))
  await screen.findByText('PNCN.26.0001')
  expect(
    calls.filter((c) => c.url.includes('/outstanding?')).at(-1)?.url,
  ).toContain('nomor_polis=POL-9')

  await userEvent.click(screen.getByRole('button', { name: 'Clear Filter' }))
  await screen.findByText('PNCN.26.0001')

  // Kotaknya kosong DAN penyaringnya dicabut. Mengosongkan kotaknya saja akan menyisakan
  // daftar tersaring tanpa satu pun petunjuk mengapa.
  expect(screen.getByLabelText('Nopolis')).toHaveValue('')
  expect(
    calls.filter((c) => c.url.includes('/outstanding?')).at(-1)?.url,
  ).not.toContain('nomor_polis')
})

it('mengirim kedua dropdown panel sebagai parameter tersendiri', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  await userEvent.selectOptions(screen.getByLabelText('Status Transfer'), 'SUDAH')
  await userEvent.selectOptions(screen.getByLabelText('Status Pembayaran'), 'LUNAS')
  await userEvent.click(screen.getByRole('button', { name: 'Search Data' }))
  await screen.findByText('PNCN.26.0001')

  // Keduanya berlaku BERSAMAAN. Dropdown kedua yang menggantikan yang pertama akan terlihat
  // seperti penyaring yang bekerja, padahal hasilnya salah.
  const terakhir = calls.filter((c) => c.url.includes('/outstanding?')).at(-1)?.url
  expect(terakhir).toContain('status_transfer=SUDAH')
  expect(terakhir).toContain('status_bayar=LUNAS')
})

it('menyaring kartu ringkasan dengan penyaring yang sama seperti daftarnya', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()
  await screen.findByText('247')

  await userEvent.selectOptions(screen.getByLabelText('Status Pembayaran'), 'LUNAS')
  await userEvent.click(screen.getByRole('button', { name: 'Search Data' }))
  await screen.findByText('247')

  // Kartu yang disaring berbeda dari daftarnya menampilkan angka yang tidak cocok dengan
  // isi tabel di bawahnya, dan tidak ada galat yang muncul. Sempat terjadi: kunci cache
  // ringkasan tidak memuat isian panel, sehingga kartunya tidak pernah ditembak ulang.
  const ringkasanTerakhir = calls.filter((c) => c.url.includes('/ringkasan')).at(-1)
  expect(ringkasanTerakhir?.url).toContain('status_bayar=LUNAS')
})

it('Transfer All Case By UserID membawa Type User, bukan hanya User ID', async () => {
  stubTransfer({ jumlahPindah: 4 })
  renderPage()
  await screen.findByText('247')

  // Tombolnya di atas GRID, bukan di panel penyaring: di layar lama ia milik
  // `InboxOutstandingClaim_Section`, bersebelahan dengan "Select All" yang menyuapinya.
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'Transfer All Case By UserID' }))
  await screen.findByRole('dialog', { name: 'Transfer All Case By UserID' })

  await userEvent.type(screen.getByLabelText('User ID Lama'), 'PICLAMA01')
  await userEvent.selectOptions(screen.getByLabelText('Type User'), 'PIC Teknik')

  await screen.findByText('PICCONTOH03')
  await userEvent.click(screen.getAllByRole('button', { name: 'Assign' })[0]!)

  // Jumlahnya datang dari SERVER, dan layar menyebutkannya. "Seluruh pekerjaan si A" tidak
  // dapat dihitung di muka — bisa satu klaim, bisa empat puluh — sehingga tanpa angka ini
  // pengguna tidak tahu apakah ada yang berpindah sama sekali.
  await screen.findByText(/4 klaim berpindah/i)

  // `GCNMTransferDataKlaim_act` BERCABANG pada ketiga nilai Type User. Permintaan tanpa
  // nilai yang dikenal tidak akan cocok dengan satu cabang pun saat dijalankan — dan
  // kegagalannya senyap, bukan berupa galat.
  const kirim = calls.filter((c) => c.url.includes('/transfer') && c.init?.method === 'POST').at(-1)
  const badan = JSON.parse(String(kirim?.init?.body ?? '{}'))
  expect(badan.tipe_pengguna).toBe('PIC Teknik')
  expect(badan.user_id_lama).toBe('PICLAMA01')
})

it('nol klaim berpindah dilaporkan sebagai peringatan, bukan keberhasilan', async () => {
  // User ID Lama yang salah ketik mengembalikan nol baris — bukan galat, karena perintahnya
  // memang sah. Bila jawabannya digambar hijau seperti pemindahan yang berhasil, pengguna
  // menutup dialognya dengan yakin bahwa pekerjaan seseorang sudah berpindah, padahal tidak.
  stubTransfer({ jumlahPindah: 0 })
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'Transfer All Case By UserID' }))
  await screen.findByRole('dialog', { name: 'Transfer All Case By UserID' })
  await userEvent.type(screen.getByLabelText('User ID Lama'), 'SALAHKETIK')

  await screen.findByText('PICCONTOH03')
  await userEvent.click(screen.getAllByRole('button', { name: 'Assign' })[0]!)

  await screen.findByText(/Tidak ada klaim yang berpindah/i)
  expect(screen.queryByText(/klaim berpindah\./i)).toBeNull()
})

it('Select All berarti SELURUH hasil penyaring, dan transfernya satu permintaan', async () => {
  stubTransfer({ jumlahPindah: 247 })
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  // Satu penyaring diisi lebih dulu, supaya uji ini membuktikan penyaringnya BENAR-BENAR
  // ikut terkirim — bukan sekadar bahwa alamatnya terbentuk.
  await userEvent.selectOptions(screen.getByLabelText('Lini Bisnis'), 'PA')
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'Select All' }))

  // Setiap baris tercentang, dan centangnya DIMATIKAN: melepas satu baris dari "seluruhnya"
  // menuntut daftar pengecualian yang belum ada, dan layar tidak boleh menjanjikannya.
  const centang = screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0001' })
  expect(centang).toBeChecked()
  expect(centang).toBeDisabled()

  // Jumlahnya disebut. "Select All" yang menyentuh 247 klaim dan yang menyentuh 3 terlihat
  // sama persis tanpa angka ini — dan yang pertama tidak dapat dibatalkan dengan mudah.
  await screen.findByText(/247/)
  await screen.findByText(/seluruh hasil\s+penyaring/i)

  await userEvent.click(screen.getByRole('button', { name: 'Transfer All Case By UserID' }))
  await screen.findByRole('dialog', { name: 'Transfer All Case By UserID' })

  await screen.findByText('PICCONTOH03')
  await userEvent.click(screen.getAllByRole('button', { name: 'Assign' })[0]!)
  await screen.findByText(/247 klaim berpindah/i)

  // SATU permintaan, bukan satu per klaim. Dengan 1.639 baris di produksi, perulangan per
  // klaim membebani server dan membuat kegagalan di tengah meninggalkan sebagian berpindah.
  const kirim = calls.filter((c) => c.url.includes('/transfer') && c.init?.method === 'POST')
  expect(kirim).toHaveLength(1)

  const badan = JSON.parse(String(kirim[0]?.init?.body ?? '{}'))
  expect(badan.lingkup).toBe('saring')

  // Penyaringnya ikut sebagai parameter kueri, dibaca server dengan pembaca yang SAMA dengan
  // daftarnya. Tanpa itu, tombolnya memindahkan himpunan yang berbeda dari yang terlihat.
  expect(kirim[0]?.url).toContain('lini_bisnis=')
})

it('Select All bertahan saat berpindah halaman — centang per baris tidak', async () => {
  stubTransfer()
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: 'Select All' }))
  expect(screen.getByRole('checkbox', { name: 'Pilih klaim PNCN.26.0001' })).toBeChecked()

  // Tombolnya berganti label supaya pembatalannya tidak perlu ditebak.
  expect(screen.getByRole('button', { name: 'Batalkan Pilihan' })).toBeInTheDocument()

  // Inilah yang diminta Work Owner: "kalo ke halaman next jg ttp ke select dong kan select
  // all". Centang per baris memang dibuang saat halaman berganti — itu disengaja, supaya
  // ReOpen tidak mengirim klaim yang tidak terlihat — tetapi "seluruhnya" bukan centang per
  // baris, dan membuangnya akan membuat namanya berbohong.
  const berikut = screen.queryByRole('button', { name: 'Halaman berikutnya' })
  if (berikut !== null) {
    await userEvent.click(berikut)
    await screen.findByText('PNCN.26.0001')
    expect(screen.getByRole('button', { name: 'Batalkan Pilihan' })).toBeInTheDocument()
  }
})

it('menggambar jam pada Tanggal Pendaftaran dan Report Date, serta menomori barisnya', async () => {
  stubFetch(ringkasanResponse(), telusurKlaim())
  renderPage()
  await screen.findByText('247')
  await userEvent.click(screen.getByRole('button', { name: /^Outstanding/ }))
  await screen.findByText('PNCN.26.0001')

  // Layar Pega menggambar keduanya dengan jam (24 Jan 20 14:54:24). Tanggal saja
  // menghilangkan isi: dua klaim yang didaftarkan pada hari yang sama menjadi tidak
  // terbedakan urutannya, padahal grid lama mengurutkannya tepat dengan nilai ini.
  await screen.findByText('5 Agustus 2026 10:00:00')

  // Jam 00:00:00 DIGAMBAR apa adanya, sama seperti Pega — menyembunyikannya membuat tengah
  // malam dan "baris yang tidak membawa jam" tampak sama.
  await screen.findByText('6 Agustus 2026 00:00:00')

  // Kolom nomor baris: Pega menggambarnya paling kiri, dan halaman pertama mulai dari 1.
  const tabel = screen.getAllByRole('table')[0]!
  expect(within(tabel).getByText('1')).toBeInTheDocument()
})

it('nomor klaim membuka popup rincian, bukan berpindah halaman', async () => {
  const calls2: { url: string }[] = []
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls2.push({ url })
    calls.push({ url, init })
    if (url.includes('/klaim/')) {
      return Promise.resolve(
        jsonResponse(200, {
          klaim_id: 'CONTOH-1',
          nomor_klaim: 'PNCN.26.0001',
          status_proses: 'New',
          status_klaim: '1147',
          pic_teknik: 'PIC Contoh 1',
          admin_pnc: 'Admin Contoh 1',
          didaftarkan_pada: '2026-08-05T03:00:00Z',
          dokumen: {},
          portal: 'ASM',
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

  // Di Pega ia pranala yang menjalankan showHarness bertarget POPUP — bukan pindah halaman.
  await userEvent.click(await screen.findByRole('button', { name: 'PNCN.26.0001' }))

  const dialog = await screen.findByRole('dialog', { name: /Detail Klaim PNCN.26.0001/i })
  expect(within(dialog).getByText('PIC Contoh 1')).toBeInTheDocument()

  // Rinciannya diambil SAAT diklik, bukan ikut pada setiap baris daftar. Menyertakannya di
  // daftar akan membawa puluhan dokumen klaim pada setiap pemuatan halaman.
  expect(calls2.filter((c) => c.url.includes('/klaim/'))).toHaveLength(1)

  // Dokumen kosong DINYATAKAN, bukan digambar sebagai layar hampa — klaim yang belum punya
  // baris di JSON_KLAIM memang begitu, dan itu keadaan yang sah.
  expect(within(dialog).getByText(/belum punya dokumen rincian/i)).toBeInTheDocument()
})

it('baris grid di dalam popup membuka sub-popup, dan Escape menutup yang itu saja', async () => {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url.includes('/klaim/')) {
      return Promise.resolve(
        jsonResponse(200, {
          klaim_id: 'CONTOH-1',
          nomor_klaim: 'PNCN.26.0001',
          status_proses: 'New',
          status_klaim: '1147',
          pic_teknik: 'PIC Contoh 1',
          admin_pnc: 'Admin Contoh 1',
          didaftarkan_pada: '2026-08-05T03:00:00Z',
          // Satu baris coverage — di Pega barisnya membuka Flow Action ViewCoverageAdj.
          dokumen: {
            ClaimData: {
              ObjectCoverageList: [
                { CoverageNote: 'Kebakaran', CoverageID: 'PLAN-1', SumTSI: '1000000' },
              ],
            },
          },
          portal: 'ASM',
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
  await userEvent.click(await screen.findByRole('button', { name: 'PNCN.26.0001' }))

  const popup = await screen.findByRole('dialog', { name: /Detail Klaim PNCN.26.0001/i })

  // Coverage ada di tab 'Adjustment & Akseptasi', bukan di gulungan panjang: popup layar lama
  // dibagi menjadi tab lewat 'pyTitle' pada ViewTempDetailClaim.
  await userEvent.click(within(popup).getByRole('tab', { name: 'Adjustment & Akseptasi' }))
  expect(within(popup).getByText('Kebakaran')).toBeInTheDocument()

  // Padanan pyEditAction: barisnya punya tombol Detail.
  await userEvent.click(within(popup).getAllByRole('button', { name: 'Detail' })[0]!)
  const sub = await screen.findByRole('dialog', { name: 'Detail Coverage' })
  expect(within(sub).getByText('Nama Coverage')).toBeInTheDocument()

  // Escape menutup sub-popup SAJA. Tanpa stopPropagation keduanya tertutup sekaligus, dan
  // pengguna yang hanya ingin kembali ke daftar kehilangan seluruh rinciannya.
  await userEvent.keyboard('{Escape}')
  expect(screen.queryByRole('dialog', { name: 'Detail Coverage' })).toBeNull()
  expect(screen.getByRole('dialog', { name: /Detail Klaim PNCN.26.0001/i })).toBeInTheDocument()
})
