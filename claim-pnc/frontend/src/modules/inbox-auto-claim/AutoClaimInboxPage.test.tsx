import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AutoClaimInboxPage } from './AutoClaimInboxPage'

// Ketiga tab beserta tabelnya datang dari server — layar tidak memuat daftarnya sendiri.
//
// Urutannya ditetapkan Work Owner (2026-09-20): Asuransi Kredit lebih dulu, dan tab
// pertama itulah yang terbuka. Ia SENGAJA berbeda dari urutan harness Pega, yang menyebut
// ANEKA lebih dulu.
//
// Label ANEKA juga tidak sama dengan nama rule-nya (`*_AutoClaim`) — itu sebabnya label
// datang dari server, bukan diturunkan dari nama rule.
const TABS = {
  bawaan: 'kredit',
  tab: [
    { kode: 'kredit', label: 'Asuransi Kredit', tabel: 'POOLDATA.TMP_BATCH_CLAIM_KREDIT' },
    { kode: 'aneka', label: 'ANEKA', tabel: 'POOLDATA.TMP_BATCH_AUTO_CLAIM' },
    { kode: 'travel', label: 'Travel', tabel: 'POOLDATA.TMP_BATCH_AUTO_TRAVEL' },
  ],
}

const TEMPLATE = {
  // Empat kolom wajib, mengikuti `Activity/InsertKlaimToTable_Other-Act.xml`. Yang TIDAK
  // ada di sini sama pentingnya: inisialid, prodke, dan currency ketiganya hasil
  // pencarian polis, bukan isian pengunggah.
  kolom_wajib: ['policyno', 'claimamount', 'dateofloss', 'reportdate'],
  kolom_opsional: ['causeofloss', 'keyword', 'alasanklaim'],
  batas_baris: 5000,
}

const BATCH_LIST = {
  portal: 'ASM',
  paginasi: { halaman: 1, ukuran: 15, total: 3, total_halaman: 1 },
  batch: [
    {
      kode_perusahaan: 'MFIN',
      nama_perusahaan: 'Mitra Finansial Nusantara',
      batch: '1',
      tanggal_proses: '12/01/2026',
      jumlah_upload: 4,
      jumlah_proses: 4,
      jumlah_berhasil: 2,
      jumlah_gagal: 2,
      jumlah_belum_proses: 0,
      user_upload: 'ADMINPNC',
    },
    {
      kode_perusahaan: 'MFIN',
      nama_perusahaan: 'Mitra Finansial Nusantara',
      batch: '2',
      tanggal_proses: '05/01/2026',
      jumlah_upload: 2,
      jumlah_proses: 0,
      jumlah_berhasil: 0,
      jumlah_gagal: 0,
      jumlah_belum_proses: 2,
      user_upload: 'ADMINPNC',
    },
    {
      // Batch tanpa baris master. Ia yang membuktikan LEFT JOIN terbawa sampai ke layar.
      kode_perusahaan: 'ZZZZ',
      nama_perusahaan: '',
      batch: '1',
      tanggal_proses: '02/03/2026',
      jumlah_upload: 1,
      jumlah_proses: 1,
      jumlah_berhasil: 0,
      jumlah_gagal: 1,
      jumlah_belum_proses: 0,
      user_upload: 'ADMINPNC',
    },
  ],
}

const LINE_LIST = {
  portal: 'ASM',
  kode_perusahaan: 'MFIN',
  batch: '1',
  paginasi: { halaman: 1, ukuran: 15, total: 2, total_halaman: 1 },
  baris: [
    {
      nomor_polis: '0100120260001',
      prod_ke: '1',
      nomor_klaim: 'PNCN.26.101',
      nomor_aksep: 'AKS-26-0001',
      mata_uang: 'IDR',
      nilai_klaim: '12500000.00',
      penyebab_kerugian: '12002',
      tanggal_kejadian: '03/01/2026',
      tanggal_lapor: '05/01/2026',
      tanggal_proses: '12/01/2026',
      catatan: '',
      nama_objek: '',
      flag_tidak_bayar: '',
      keyword: 'REF-1',
      keterangan: 'Sukses Klaim',
      hasil: 'berhasil',
    },
    {
      nomor_polis: '0100120260003',
      prod_ke: '1',
      nomor_klaim: '',
      nomor_aksep: '',
      mata_uang: 'IDR',
      nilai_klaim: '4300000.00',
      penyebab_kerugian: '',
      tanggal_kejadian: '05/01/2026',
      tanggal_lapor: '07/01/2026',
      tanggal_proses: '12/01/2026',
      catatan: '',
      nama_objek: '',
      flag_tidak_bayar: '',
      keyword: 'REF-3',
      keterangan: 'Penyebab kerugian tidak ditemukan',
      hasil: 'gagal',
    },
  ],
}

type Call = {
  url: string
  method: string
  header: Record<string, string>
}

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      header: (init?.headers as Record<string, string>) ?? {},
    }
    calls.push(call)

    const { body, status = 200 } = map(call)
    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

// SUMMARY WAJIB cocok dengan BATCH_LIST, dan kecocokannya bukan kerapian: aturan modul
// ini adalah angka ringkasan SAMA dengan total grid, sehingga data uji yang melanggarnya
// akan meloloskan layar yang menampilkan dua angka bertentangan.
//
// BATCH_LIST memuat tiga baris — MFIN dua, ZZZZ satu — dan paginasinya menyebut total 3.
// Angka di bawah mengikutinya persis.
//
// TIDAK ADA baris berjumlah nol, dan itu bukan kelalaian: server hanya mengirim
// perusahaan yang punya batch di tab ini (keputusan Work Owner 2026-09-20). Baris nol
// pernah ada di sini, dan justru itu yang menutupi cacatnya — setiap klik padanya
// menghasilkan grid kosong, sehingga penyaringnya terlihat "tidak berfungsi".
const SUMMARY = {
  portal: 'ASM',
  total: 3,
  perusahaan: [
    { kode: 'MFIN', nama: 'Mitra Finansial Nusantara', jumlah_batch: 2 },
    { kode: 'ZZZZ', nama: '', jumlah_batch: 1 },
  ],
}

/** defaultReply melayani kelima GET; unggahan dijawab pemanggil lewat `upload`. */
function defaultReply(upload?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/inbox-auto-claim/tab')) return { body: TABS }
    if (call.url.startsWith('/api/inbox-auto-claim/ringkasan')) return { body: SUMMARY }
    if (call.url.startsWith('/api/inbox-auto-claim/format-unggahan')) return { body: TEMPLATE }
    if (call.url.startsWith('/api/inbox-auto-claim/unggah')) {
      if (upload) return upload(call)
      return { body: { batch: [], jumlah_baris: 0, ditolak: [], portal: 'ASM' }, status: 201 }
    }
    // Rincian batch: /api/inbox-auto-claim/MFIN/1?halaman=1
    if (/\/api\/inbox-auto-claim\/[^/?]+\/[^/?]+/.test(call.url)) return { body: LINE_LIST }

    // Daftar batch — dan stub ini benar-benar MENYARING.
    //
    // Versi pertama mengembalikan BATCH_LIST apa adanya untuk permintaan apa pun,
    // sehingga uji penyaring hanya dapat memeriksa URL-nya. Cacat berupa "permintaannya
    // terkirim tetapi tabelnya tidak berubah" — persis yang dilaporkan Work Owner —
    // TIDAK DAPAT ditangkap stub seperti itu.
    const filter = new URL(call.url, 'http://uji').searchParams.get('perusahaan') ?? ''
    if (filter === '') return { body: BATCH_LIST }

    const batch = BATCH_LIST.batch.filter((b) => b.kode_perusahaan === filter)
    return {
      body: {
        ...BATCH_LIST,
        batch,
        paginasi: { ...BATCH_LIST.paginasi, total: batch.length },
      },
    }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AutoClaimInboxPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function startSession() {
  useSession.getState().login({
    token: 'token-contoh',
    user: {
      identitas: '90000001',
      nama: 'Contoh Administrator',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: '',
      perusahaan: 'ASM',
    },
    validUntil: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  })
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
  startSession()
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('daftar batch Inbox Auto Claim', () => {
  it('menampilkan judul dan kesembilan kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Inbox Auto Claim' })).toBeInTheDocument()

    const table = await screen.findByRole('table', { name: 'Daftar batch' })
    for (const kolom of [
      'KODE',
      'Nama Perusahaan',
      'Batch',
      'Tgl Proses',
      'Di Upload',
      'Diproses',
      'Berhasil',
      'Gagal',
      'User Upload',
    ]) {
      expect(
        within(table).getByRole('columnheader', { name: new RegExp(kolom) }),
      ).toBeInTheDocument()
    }
  })

  it('mengirim header portal pada setiap permintaan data entitas', async () => {
    // Backend MENOLAK permintaan tanpa portal, dan tidak pernah jatuh ke portal utama
    // sebagai cadangan (R-20). Layar harus benar-benar mengirimkannya.
    installFetch(defaultReply())
    show()

    await screen.findByRole('table', { name: 'Daftar batch' })

    // Dua rute sengaja TIDAK menyebut portal: bentuk berkas unggahan dan daftar tab sama
    // untuk setiap entitas, dan tidak satu baris data entitas pun dibacanya. Menuntut
    // portal di sana justru membuat layar gagal menggambar tabnya sebelum entitas dipilih.
    const dataCall = calls.filter(
      (c) => !c.url.includes('format-unggahan') && !c.url.startsWith('/api/inbox-auto-claim/tab'),
    )
    expect(dataCall.length).toBeGreaterThan(0)
    for (const call of dataCall) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  it('menandai batch yang perusahaannya tidak terdaftar di master', async () => {
    // Dengan INNER JOIN barisnya akan hilang tanpa satu pun tanda, padahal baris seperti
    // itu justru yang tidak akan pernah berhasil diproses.
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Tidak terdaftar di Master Auto Claim')).toBeInTheDocument()
    expect(screen.getByText(/Kode ZZZZ perlu didaftarkan/)).toBeInTheDocument()
  })

  it('menyebutkan baris yang belum diproses pada batch yang baru diunggah', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('+2 menunggu')).toBeInTheDocument()
  })

  it('menampilkan Total Data dan tombol halaman', async () => {
    // Bentuknya mengikuti Section/ButtonPagingInbox-Section.xml.
    installFetch(defaultReply())
    show()

    await screen.findByRole('table', { name: 'Daftar batch' })
    const bar = screen.getByRole('navigation', { name: 'Navigasi halaman' })
    expect(within(bar).getByText(/Total Data/)).toBeInTheDocument()
    expect(within(bar).getByRole('button', { name: 'Halaman pertama' })).toBeInTheDocument()
    expect(within(bar).getByRole('button', { name: 'Halaman terakhir' })).toBeInTheDocument()
  })

  it('menyaring pada KODE perusahaan, bukan namanya', async () => {
    // Penyaring lama merangkai NAMA ke dalam teks SQL. Yang dikirim sekarang kodenya, dan
    // uji ini yang menjaganya tidak kembali ke nama.
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })
    calls = []

    await userEvent.click(within(ringkasan).getByRole('button', { name: /Mitra Finansial/ }))

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('perusahaan=MFIN'))).toBe(true)
    })
    expect(calls.some((c) => c.url.includes('Mitra'))).toBe(false)
  })

  it('tidak lagi memasang dropdown perusahaan di bilah judul grid', async () => {
    // Penyaringnya panel ringkasan, sama seperti layar Pega (keputusan Work Owner
    // 2026-09-20). Uji ini yang menjaga dropdown-nya tidak kembali diam-diam dan membuat
    // layar punya DUA penyaring yang dapat berselisih.
    installFetch(defaultReply())
    show()

    await screen.findByRole('table', { name: 'Daftar batch' })
    expect(screen.queryByRole('combobox', { name: 'Nama Perusahaan' })).not.toBeInTheDocument()
  })

  it('menampilkan ketiga tombol yang mesinnya belum dibangun dalam keadaan nonaktif', async () => {
    // Ditampilkan, bukan disembunyikan: menyembunyikannya membuat layar tampak selesai
    // padahal separuh alurnya belum ada.
    installFetch(defaultReply())
    show()

    for (const label of ['Proses Klaim', 'Generate DLA', 'Cek Premi']) {
      expect(screen.getByRole('button', { name: label })).toBeDisabled()
    }
  })
})

describe('rincian batch', () => {
  it('membuka rincian dan menampilkan baris beserta hasilnya', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table', { name: 'Daftar batch' })
    await userEvent.click(screen.getByRole('button', { name: 'Detail batch 1 MFIN' }))

    expect(await screen.findByRole('heading', { name: /Rincian batch 1/ })).toBeInTheDocument()
    expect(await screen.findByText('PNCN.26.101')).toBeInTheDocument()

    // Lencana dicari DI DALAM tabel rincian, bukan di seluruh halaman: "Berhasil" dan
    // "Gagal" juga menjadi judul kolom pada grid batch di atasnya.
    const detail = screen.getAllByRole('table').at(-1)
    expect(detail).toBeDefined()
    expect(within(detail as HTMLElement).getByText('Berhasil')).toBeInTheDocument()
    expect(within(detail as HTMLElement).getByText('Gagal')).toBeInTheDocument()
    expect(
      within(detail as HTMLElement).getByText('Penyebab kerugian tidak ditemukan'),
    ).toBeInTheDocument()
  })

  it('menampilkan nilai uang dengan pemisah ribuan tanpa mengubah desimalnya', async () => {
    // I-12: nilai disimpan presisi penuh, pembulatan hanya saat ditampilkan.
    installFetch(defaultReply())
    show()

    await screen.findByRole('table', { name: 'Daftar batch' })
    await userEvent.click(screen.getByRole('button', { name: 'Detail batch 1 MFIN' }))

    expect(await screen.findByText('12.500.000,00')).toBeInTheDocument()
  })
})

describe('tombol ekspor', () => {
  it('menonaktifkan Export Berhasil pada batch yang belum punya baris berhasil', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table', { name: 'Daftar batch' })

    // Batch 2 MFIN: 0 berhasil, 0 gagal — kedua tombol ekspornya nonaktif.
    expect(screen.getByRole('button', { name: 'Export berhasil batch 2 MFIN' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Export gagal batch 2 MFIN' })).toBeDisabled()

    // Batch 1 MFIN: 2 berhasil dan 2 gagal — keduanya aktif.
    expect(screen.getByRole('button', { name: 'Export berhasil batch 1 MFIN' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Export gagal batch 1 MFIN' })).toBeEnabled()
  })
})

describe('unggah berkas klaim', () => {
  it('menampilkan judul kolom yang harus ada di berkas', async () => {
    installFetch(defaultReply())
    show()

    await userEvent.click(screen.getByRole('button', { name: 'Upload Data Klaim' }))

    expect(await screen.findByRole('heading', { name: 'Upload Data Klaim' })).toBeInTheDocument()
    expect(screen.getByText(/policyno, claimamount, dateofloss, reportdate/)).toBeInTheDocument()
  })

  it('menyatakan kode perusahaan tidak perlu diisi karena dicari dari polis', async () => {
    // Ketiadaan kolomnya justru yang paling membingungkan: petugas yang terbiasa
    // mengisi kode perusahaan akan mencari kolomnya dan mengira daftarnya kurang.
    installFetch(defaultReply())
    show()

    await userEvent.click(screen.getByRole('button', { name: 'Upload Data Klaim' }))

    expect(
      await screen.findByText(/Kode perusahaan, nomor produk, dan mata uang/),
    ).toBeInTheDocument()
  })

  it('mengirim berkas sebagai multipart dan melaporkan nomor batch yang terbit', async () => {
    installFetch(
      defaultReply(() => ({
        status: 201,
        body: {
          portal: 'ASM',
          jumlah_baris: 2,
          ditolak: [],
          batch: [
            {
              kode_perusahaan: 'MFIN',
              nama_perusahaan: 'Mitra Finansial Nusantara',
              batch: '3',
              jumlah_baris: 2,
              jumlah_lolos: 2,
              jumlah_bertanda: 0,
            },
          ],
        },
      })),
    )
    show()

    await userEvent.click(screen.getByRole('button', { name: 'Upload Data Klaim' }))

    const file = new File(['policyno,claimamount\nP1,1.00\n'], 'klaim.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Berkas CSV'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    // Nomor batch yang terbit disebut di ringkasan: pengguna tidak punya cara lain
    // mengetahuinya, karena nomornya diterbitkan server.
    const note = (await screen.findByText('2 baris tersimpan')).closest('div')
    expect(note).not.toBeNull()
    expect(
      within(note as HTMLElement).getByText(/Mitra Finansial Nusantara — batch/),
    ).toBeInTheDocument()

    // Content-Type SENGAJA tidak diisi klien: peramban yang mengisinya lengkap dengan
    // boundary. Menuliskannya akan membuat server menolak badan permintaan.
    const uploadCall = calls.find((c) => c.url.includes('/unggah'))
    expect(uploadCall?.method).toBe('POST')
    expect(uploadCall?.header['Content-Type']).toBeUndefined()
    expect(uploadCall?.header['X-Portal']).toBe('ASM')
  })

  it('membedakan baris yang TERSIMPAN BERTANDA dari baris yang DITOLAK', async () => {
    // Pembedaan ini bukan kerapian tampilan. Baris "bertanda" tersimpan dan terlihat di
    // grid; baris "ditolak" tidak ada di mana pun. Pengguna yang mengira keduanya sama
    // akan mencari baris yang tidak pernah tersimpan.
    installFetch(
      defaultReply(() => ({
        status: 201,
        body: {
          portal: 'ASM',
          jumlah_baris: 1,
          batch: [
            {
              kode_perusahaan: 'MFIN',
              nama_perusahaan: 'Mitra Finansial Nusantara',
              batch: '3',
              jumlah_baris: 1,
              jumlah_lolos: 0,
              jumlah_bertanda: 1,
            },
          ],
          ditolak: [
            { baris: 4, nomor_polis: '0900120260500', pesan: 'Sumber Bisnis Tidak ditemukan' },
          ],
        },
      })),
    )
    show()

    await userEvent.click(screen.getByRole('button', { name: 'Upload Data Klaim' }))
    const file = new File(['policyno,claimamount\nP1,1.00\n'], 'klaim.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Berkas CSV'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    const note = (await screen.findByText('1 baris tersimpan')).closest('div')
    expect(note).not.toBeNull()
    const panel = note as HTMLElement

    expect(within(panel).getByText(/1 bertanda gagal/)).toBeInTheDocument()
    expect(within(panel).getByText('1 baris TIDAK tersimpan')).toBeInTheDocument()

    // Nomor barisnya disebut: tanpa itu, pengguna dengan berkas ratusan baris hanya tahu
    // "ada yang gagal" dan harus mencarinya sendiri. Barisnya dicari sebagai satu
    // <li> utuh, bukan dengan mencocokkan potongan teks — potongan "Baris" juga muncul
    // di kalimat penjelas, dan pencocokan longgar akan lulus tanpa membuktikan apa pun.
    const rejected = within(panel)
      .getAllByRole('listitem')
      .map((item) => item.textContent?.replace(/\s+/g, ' ').trim())

    expect(rejected).toContain('Baris 4 — 0900120260500 — Sumber Bisnis Tidak ditemukan')
  })

  it('menampilkan SELURUH pelanggaran baris, bukan yang pertama saja', async () => {
    // Kesetaraan perilaku (P-5): pada berkas ratusan baris, satu galat per percobaan
    // berarti mengunggah ulang ratusan kali.
    installFetch(
      defaultReply(() => ({
        status: 422,
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada baris yang belum benar. Perbaiki berkasnya lalu unggah lagi.',
          detail: [
            { kolom: 'baris 2 · inisialid', pesan: 'Kode perusahaan wajib diisi.' },
            { kolom: 'baris 3 · nopolis', pesan: 'Nomor polis wajib diisi.' },
          ],
        },
      })),
    )
    show()

    await userEvent.click(screen.getByRole('button', { name: 'Upload Data Klaim' }))

    const file = new File(['inisialid\n\n'], 'klaim.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Berkas CSV'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    expect(await screen.findByText('2 baris perlu diperbaiki')).toBeInTheDocument()
    expect(screen.getByText('baris 2 · inisialid')).toBeInTheDocument()
    expect(screen.getByText('baris 3 · nopolis')).toBeInTheDocument()
  })
})

describe('portal belum dipilih', () => {
  it('menuntun pengguna memilih portal alih-alih menembak server', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls.some((c) => c.url.startsWith('/api/inbox-auto-claim?'))).toBe(false)
  })
})

describe('panel ringkasan per perusahaan', () => {
  it('menampilkan baris All beserta tiap perusahaan', async () => {
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })

    // Baris All memakai total dari SERVER, bukan hasil menjumlahkan irisan di layar.
    // Bedanya baru terasa bila ringkasannya kelak dipotong — jumlah di layar akan
    // diam-diam lebih kecil, dan barisnya tetap bertuliskan "All".
    const all = within(ringkasan).getByRole('button', { name: /All/ })
    expect(all).toBeInTheDocument()
    expect(
      within(ringkasan).getByRole('button', { name: /Mitra Finansial Nusantara/ }),
    ).toBeInTheDocument()
  })

  it('angka ringkasan memakai satuan yang sama dengan total grid', async () => {
    // Inilah janji panel ini kepada pengguna: yang tertulis "2" menghasilkan 2 baris.
    // Uji ini menjaga kedua angka itu tidak pernah memakai satuan berbeda.
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })

    // Dibaca PER SEL, bukan dari textContent barisnya: textContent menyatukan sel tanpa
    // pemisah ("All3"), sehingga pencocokan teksnya menguji cara DOM menggabungkan string
    // alih-alih menguji angkanya.
    const jumlah = (nama: RegExp) => {
      const baris = within(ringkasan)
        .getAllByRole('row')
        .find((r) => nama.test(r.textContent ?? ''))
      const sel = baris === undefined ? [] : within(baris).getAllByRole('cell')
      return sel[sel.length - 1]?.textContent?.trim()
    }

    expect(jumlah(/^All/)).toBe('3')
    expect(jumlah(/Mitra Finansial/)).toBe('2')
  })

  it('menandai perusahaan yang tidak terdaftar di master, bukan menyembunyikannya', async () => {
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })
    expect(within(ringkasan).getByText('tidak terdaftar di Master Auto Claim')).toBeInTheDocument()
  })

  it('mengeklik satu perusahaan menyaring daftar batch', async () => {
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })
    await userEvent.click(within(ringkasan).getByRole('button', { name: /Mitra Finansial/ }))

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('perusahaan=MFIN'))).toBe(true)
    })
  })

  it('ISI tabel batch benar-benar berubah, bukan hanya permintaannya', async () => {
    // Uji sebelumnya hanya memeriksa URL. Work Owner melaporkan cacat yang lolos darinya:
    // permintaan terkirim, tetapi tabel di bawah tidak berubah. Uji ini memeriksa BARIS
    // yang tampil, satu-satunya hal yang benar-benar dilihat pengguna.
    //
    // Tabelnya DIAMBIL ULANG setiap kali, tidak disimpan di variabel. DataTable berganti
    // ke kerangka pemuatan lalu ke keadaan kosong, dan pada keduanya elemen <table>
    // benar-benar dilepas dari DOM — simpulan dari elemen lama akan menguji DOM yang
    // sudah tidak ada di layar.
    installFetch(defaultReply())
    show()

    const baris = () =>
      within(screen.getByRole('table', { name: 'Daftar batch' })).getAllByRole('row')

    await screen.findByRole('table', { name: 'Daftar batch' })
    await waitFor(() => expect(baris()).toHaveLength(4)) // 1 judul + 3

    const ringkasan = screen.getByRole('table', { name: 'Jumlah batch per perusahaan' })
    await userEvent.click(within(ringkasan).getByRole('button', { name: /Mitra Finansial/ }))

    // MFIN punya dua batch pada BATCH_LIST.
    //
    // Pemeriksaan isinya dibatasi PADA GRID, bukan seluruh layar: nama perusahaan yang
    // sama juga tertulis di panel ringkasan, dan pencarian selebar layar akan selalu
    // menemukannya.
    await waitFor(() => expect(baris()).toHaveLength(3))
    const grid = () => within(screen.getByRole('table', { name: 'Daftar batch' }))
    expect(grid().queryByText('Kode ZZZZ perlu didaftarkan')).not.toBeInTheDocument()

    // Berpindah ke perusahaan LAIN, bukan kembali ke semua: inilah gerakan yang
    // dilaporkan tidak berpengaruh.
    await userEvent.click(within(ringkasan).getByRole('button', { name: /ZZZZ/ }))
    await waitFor(() => expect(baris()).toHaveLength(2))
    expect(grid().queryByText('Mitra Finansial Nusantara')).not.toBeInTheDocument()
  })

  it('mengeklik perusahaan yang sedang dipilih membatalkan penyaringnya', async () => {
    // Sejak dropdown dibuang, ini SATU-SATUNYA jalan kembali ke "semua" selain baris All.
    // Tanpanya pengguna yang menyaring lewat panel terkunci pada satu perusahaan.
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })
    const mfin = within(ringkasan).getByRole('button', { name: /Mitra Finansial/ })

    await userEvent.click(mfin)
    await waitFor(() => expect(mfin).toHaveAttribute('aria-pressed', 'true'))

    await userEvent.click(mfin)
    await waitFor(() => expect(mfin).toHaveAttribute('aria-pressed', 'false'))
  })

  it('setiap baris ringkasan menghasilkan batch, tidak ada yang berjumlah nol', async () => {
    // Kebalikan dari uji sebelumnya di tempat ini, yang menuntut perusahaan berjumlah nol
    // TETAP tampil supaya dapat dipilih. Itu keliru begitu ketiga tab ada: barisnya sama
    // di ketiga tab, dan setiap kliknya menghasilkan grid kosong.
    //
    // Janji panel ini sekarang sederhana dan dapat diperiksa: apa pun yang tampil di
    // sini, bila diklik, menghasilkan baris di grid.
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })

    const angka = within(ringkasan)
      .getAllByRole('row')
      .slice(2) // lewati baris judul dan baris All
      .map((r) => {
        const sel = within(r).getAllByRole('cell')
        return Number(sel[sel.length - 1]?.textContent?.trim())
      })

    expect(angka.length).toBeGreaterThan(0)
    for (const n of angka) expect(n).toBeGreaterThan(0)
  })
})

describe('tab jenis klaim', () => {
  it('menampilkan ketiga tab dengan Asuransi Kredit terbuka lebih dulu', async () => {
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    for (const label of ['ANEKA', 'Asuransi Kredit', 'Travel']) {
      expect(within(tablist).getByRole('tab', { name: label })).toBeInTheDocument()
    }

    // Urutannya juga dari server, dan Work Owner menetapkan Asuransi Kredit lebih dulu.
    const nama = within(tablist)
      .getAllByRole('tab')
      .map((t) => t.textContent)
    expect(nama).toEqual(['Asuransi Kredit', 'ANEKA', 'Travel'])

    // Tab bawaannya datang dari server (`bawaan`), bukan dipilih layar.
    expect(within(tablist).getByRole('tab', { name: 'Asuransi Kredit' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('berpindah tab mengirim ?sumber= yang baru', async () => {
    // Ketiga tab membaca TABEL yang berbeda. Bila parameternya tidak terkirim, layar akan
    // menampilkan data tab bawaan di bawah judul tab lain — salah tanpa satu pun tanda.
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    calls = []

    await userEvent.click(within(tablist).getByRole('tab', { name: 'ANEKA' }))

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('sumber=aneka'))).toBe(true)
    })
    // Ringkasannya ikut berpindah tabel, bukan hanya gridnya.
    expect(
      calls.some(
        (c) =>
          c.url.startsWith('/api/inbox-auto-claim/ringkasan') && c.url.includes('sumber=aneka'),
      ),
    ).toBe(true)
  })

  it('berpindah tab mengosongkan penyaring perusahaan', async () => {
    // Kode perusahaan pada satu tabel belum tentu ada di tabel lain. Membawanya ikut
    // berpindah menghasilkan grid kosong yang sebabnya tidak terbaca di layar.
    installFetch(defaultReply())
    show()

    const ringkasan = await screen.findByRole('table', { name: 'Jumlah batch per perusahaan' })
    const mfin = within(ringkasan).getByRole('button', { name: /Mitra Finansial/ })
    await userEvent.click(mfin)
    await waitFor(() => expect(mfin).toHaveAttribute('aria-pressed', 'true'))

    calls = []
    const tablist = screen.getByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('sumber=travel'))).toBe(true)
    })
    expect(calls.some((c) => c.url.includes('perusahaan=MFIN'))).toBe(false)
  })

  it('menyebut tabel sumber sesuai tab yang terbuka', async () => {
    // Keterangannya datang bersama tabnya. Teks tetap "POOLDATA.TMP_BATCH_AUTO_CLAIM"
    // akan SALAH pada dua dari tiga tab, dan salah dengan cara yang menyesatkan penelusuran
    // selisih angka ke DBA.
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Sumber: POOLDATA.TMP_BATCH_CLAIM_KREDIT')).toBeInTheDocument()

    const tablist = screen.getByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))

    expect(await screen.findByText('Sumber: POOLDATA.TMP_BATCH_AUTO_TRAVEL')).toBeInTheDocument()
  })
})
