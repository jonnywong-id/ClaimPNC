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
  kolom_tanggal: ['dateofloss', 'reportdate'],
  titik_ribuan: false,
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

/** Tabel perusahaan — tabel utama layar ini. */
async function companyTable() {
  return screen.findByRole('table', { name: 'Daftar perusahaan' })
}

/**
 * Membuka baris satu perusahaan lalu mengembalikan grid batch di dalamnya.
 *
 * Grid itu DIAMBIL ULANG oleh pemanggil bila perlu: DataTable melepas elemen <table>-nya
 * saat berganti ke keadaan memuat, sehingga simpulan lama menunjuk DOM yang sudah hilang.
 */
async function openCompany(name: string) {
  const table = await companyTable()
  await userEvent.click(within(table).getByRole('button', { name: `Buka batch ${name}` }))
  return screen.findByRole('table', { name: `Batch ${name}` })
}

/** Nama perusahaan per baris tabel perusahaan, sesuai urutan tampil. */
function companyNames(table: HTMLElement) {
  return within(table)
    .getAllByRole('button', { name: /^(Buka|Tutup) batch / })
    .map((b) => b.getAttribute('aria-label')?.replace(/^(Buka|Tutup) batch /, ''))
}

describe('tabel perusahaan', () => {
  it('menampilkan judul, kolom perusahaan paling kiri, dan tanpa grafik', async () => {
    installFetch(defaultReply())
    const { container } = show()

    expect(screen.getByRole('heading', { name: 'Inbox Auto Claim' })).toBeInTheDocument()

    const table = await companyTable()
    const kepala = within(table)
      .getAllByRole('columnheader')
      .map((h) => h.textContent)
    expect(kepala).toEqual(['Nama Perusahaan & Kode', 'Jumlah Batch', 'Aksi'])

    // Donut sudah dibuang (keputusan Work Owner 2026-09-27). Recharts menggambar <svg>
    // berkelas recharts-surface; tidak boleh ada satu pun.
    expect(container.querySelector('.recharts-surface')).toBeNull()
  })

  it('menampilkan kode di bawah nama dan jumlah batch per perusahaan', async () => {
    installFetch(defaultReply())
    show()

    const table = await companyTable()
    const mfin = within(table)
      .getAllByRole('row')
      .find((r) => r.textContent?.includes('Mitra Finansial Nusantara'))
    expect(mfin).toBeDefined()
    const sel = within(mfin as HTMLElement).getAllByRole('cell')
    expect(sel[0]?.textContent).toContain('MFIN')
    expect(sel[1]?.textContent?.trim()).toBe('2')
  })

  it('menandai perusahaan yang tidak terdaftar di master, bukan menyembunyikannya', async () => {
    installFetch(defaultReply())
    show()

    const table = await companyTable()
    expect(within(table).getByText('tidak terdaftar di Master Auto Claim')).toBeInTheDocument()
  })

  it('pencarian menyaring menurut nama maupun kode', async () => {
    installFetch(defaultReply())
    show()

    await companyTable()
    const cari = screen.getByLabelText('Cari nama perusahaan')

    await userEvent.type(cari, 'mitra')
    expect(companyNames(await companyTable())).toEqual(['Mitra Finansial Nusantara'])

    await userEvent.clear(cari)
    await userEvent.type(cari, 'zzz')
    expect(companyNames(await companyTable())).toEqual(['ZZZZ'])

    await userEvent.clear(cari)
    await userEvent.type(cari, 'tidak-ada')
    expect(await screen.findByText(/Tidak ada perusahaan yang cocok/)).toBeInTheDocument()
  })

  it('memaginasi lebih dari 15 perusahaan', async () => {
    // Alasan bentuk baru ini: tab Asuransi Kredit dapat memuat lebih dari 20 perusahaan.
    const banyak = Array.from({ length: 22 }, (_, i) => ({
      kode: `K${String(i + 1).padStart(2, '0')}`,
      nama: `Perusahaan ${String(i + 1).padStart(2, '0')}`,
      jumlah_batch: 1,
    }))
    installFetch((call) =>
      call.url.startsWith('/api/inbox-auto-claim/ringkasan')
        ? { body: { portal: 'ASM', total: 22, perusahaan: banyak } }
        : defaultReply()(call),
    )
    show()

    expect(companyNames(await companyTable())).toHaveLength(15)
    expect(screen.getByText(/dari 22 baris/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Halaman 2' }))
    const kedua = companyNames(await companyTable())
    expect(kedua).toHaveLength(7)
    expect(kedua[0]).toBe('Perusahaan 16')
  })

  it('tidak ada baris berjumlah nol', async () => {
    // Apa pun yang tampil di sini, bila dibuka, menghasilkan batch.
    installFetch(defaultReply())
    show()

    const table = await companyTable()
    const angka = within(table)
      .getAllByRole('row')
      .slice(1)
      .map((r) => Number(within(r).getAllByRole('cell')[1]?.textContent?.trim()))
    expect(angka.length).toBeGreaterThan(0)
    for (const n of angka) expect(n).toBeGreaterThan(0)
  })

  it('tidak memasang dropdown perusahaan', async () => {
    installFetch(defaultReply())
    show()

    await companyTable()
    expect(screen.queryByRole('combobox', { name: 'Nama Perusahaan' })).not.toBeInTheDocument()
  })

  it('mengirim header portal pada setiap permintaan data entitas', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')

    // Dua rute sengaja TIDAK menyebut portal: bentuk berkas unggahan dan daftar tab sama
    // untuk setiap entitas.
    const dataCall = calls.filter(
      (c) => !c.url.includes('format-unggahan') && !c.url.startsWith('/api/inbox-auto-claim/tab'),
    )
    expect(dataCall.length).toBeGreaterThan(0)
    for (const call of dataCall) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  it('menampilkan ketiga tombol yang mesinnya belum dibangun dalam keadaan nonaktif', async () => {
    installFetch(defaultReply())
    show()

    for (const label of ['Proses Klaim', 'Generate DLA', 'Cek Premi']) {
      expect(screen.getByRole('button', { name: label })).toBeDisabled()
    }
  })
})

describe('baris perusahaan yang melar', () => {
  it('tidak memuat batch sebelum ada baris yang dibuka', async () => {
    installFetch(defaultReply())
    show()

    await companyTable()
    expect(calls.some((c) => c.url.startsWith('/api/inbox-auto-claim?'))).toBe(false)
    expect(screen.queryByRole('table', { name: /^Batch / })).not.toBeInTheDocument()
  })

  it('membuka grid batch 9 kolom di dalam baris, tanpa jendela sembulan', async () => {
    installFetch(defaultReply())
    show()

    const grid = await openCompany('Mitra Finansial Nusantara')
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
        within(grid).getByRole('columnheader', { name: new RegExp(kolom) }),
      ).toBeInTheDocument()
    }
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    // Grid itu ada DI DALAM tabel perusahaan, bukan tabel terpisah di bawahnya.
    expect((await companyTable()).contains(grid)).toBe(true)
  })

  it('menyaring pada KODE perusahaan, bukan namanya', async () => {
    installFetch(defaultReply())
    show()

    await companyTable()
    calls = []
    await openCompany('Mitra Finansial Nusantara')

    expect(calls.some((c) => c.url.includes('perusahaan=MFIN'))).toBe(true)
    expect(calls.some((c) => c.url.includes('Mitra'))).toBe(false)
  })

  it('isi grid hanya milik perusahaan yang dibuka', async () => {
    // Cacat yang dilaporkan Work Owner: grid menampilkan batch perusahaan lain.
    installFetch(defaultReply())
    show()

    const grid = await openCompany('Mitra Finansial Nusantara')
    await waitFor(() =>
      expect(
        within(screen.getByRole('table', { name: 'Batch Mitra Finansial Nusantara' })).getAllByRole(
          'row',
        ),
      ).toHaveLength(3),
    ) // 1 judul + 2 batch MFIN
    expect(within(grid).queryByText('Kode ZZZZ perlu didaftarkan')).not.toBeInTheDocument()
    expect(within(grid).getByText('+2 menunggu')).toBeInTheDocument()
  })

  it('hanya satu baris terbuka sekaligus, dan mengekliknya lagi menutupnya', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    await openCompany('ZZZZ')

    expect(screen.queryByRole('table', { name: 'Batch Mitra Finansial Nusantara' })).toBeNull()
    const grid = screen.getByRole('table', { name: 'Batch ZZZZ' })
    expect(await within(grid).findByText('Kode ZZZZ perlu didaftarkan')).toBeInTheDocument()

    const table = await companyTable()
    const tutup = within(table).getByRole('button', { name: 'Tutup batch ZZZZ' })
    expect(tutup).toHaveAttribute('aria-expanded', 'true')
    await userEvent.click(tutup)
    expect(screen.queryByRole('table', { name: 'Batch ZZZZ' })).toBeNull()
  })

  it('mengeklik di mana pun pada baris ikut membukanya', async () => {
    // Sorotan hover menjanjikan seluruh baris dapat diklik — janji itu harus ditepati.
    installFetch(defaultReply())
    show()

    const table = await companyTable()
    const baris = within(table)
      .getAllByRole('row')
      .find((r) => r.textContent?.includes('Mitra Finansial Nusantara'))
    const angka = within(baris as HTMLElement).getAllByRole('cell')[1]
    await userEvent.click(angka as HTMLElement)

    expect(
      await screen.findByRole('table', { name: 'Batch Mitra Finansial Nusantara' }),
    ).toBeInTheDocument()
  })

  it('menampilkan Total Data dan tombol halaman grid batch', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    const bar = screen.getByRole('navigation', { name: 'Navigasi halaman' })
    expect(within(bar).getByText(/Total Data/)).toBeInTheDocument()
    expect(within(bar).getByRole('button', { name: 'Halaman pertama' })).toBeInTheDocument()
  })

  it('tombol ekspor nonaktif pada batch yang belum punya baris berhasil atau gagal', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    expect(
      await screen.findByRole('button', { name: 'Export berhasil batch 2 MFIN' }),
    ).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Export gagal batch 2 MFIN' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Export berhasil batch 1 MFIN' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Export gagal batch 1 MFIN' })).toBeEnabled()
  })
})

describe('rincian batch', () => {
  it('membuka rincian di dalam baris dan menampilkan hasilnya', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    await userEvent.click(await screen.findByRole('button', { name: 'Detail batch 1 MFIN' }))

    expect(await screen.findByRole('heading', { name: /Rincian batch 1/ })).toBeInTheDocument()
    expect(await screen.findByText('PNCN.26.101')).toBeInTheDocument()

    // Rinciannya juga di dalam baris yang terbuka, bukan di bawah halaman.
    const detail = screen.getAllByRole('table').at(-1) as HTMLElement
    expect((await companyTable()).contains(detail)).toBe(true)
    expect(within(detail).getByText('Penyebab kerugian tidak ditemukan')).toBeInTheDocument()
  })

  it('menampilkan nilai uang dengan pemisah ribuan tanpa mengubah desimalnya', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    await userEvent.click(await screen.findByRole('button', { name: 'Detail batch 1 MFIN' }))

    expect(await screen.findByText('12.500.000,00')).toBeInTheDocument()
  })
})

describe('unggah berkas klaim', () => {
  it('dibuka sebagai pop-up yang menyebut tab tujuan, dan Escape menutupnya', async () => {
    // Mengikuti local action Pega yang terbuka sebagai modal (Work Owner 2026-09-27).
    installFetch(defaultReply())
    show()

    await screen.findByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(screen.getByRole('button', { name: 'Upload Data Klaim' }))

    const dialog = await screen.findByRole('dialog', { name: 'Upload Data Klaim' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(within(dialog).getByText('Asuransi Kredit')).toBeInTheDocument()

    // Bentuk berkas diminta UNTUK TAB INI: berkas Kredit tidak punya tanggal kejadian
    // maupun tanggal lapor, dan petunjuk ANEKA justru yang membuatnya ditolak.
    await waitFor(() =>
      expect(
        calls.some((c) => c.url === '/api/inbox-auto-claim/format-unggahan?sumber=kredit'),
      ).toBe(true),
    )

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

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
    expect(calls.some((c) => c.url.startsWith('/api/inbox-auto-claim/ringkasan'))).toBe(false)
  })
})

describe('tab jenis klaim', () => {
  it('menampilkan ketiga tab dengan Asuransi Kredit terbuka lebih dulu', async () => {
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    const nama = within(tablist)
      .getAllByRole('tab')
      .map((t) => t.textContent)
    expect(nama).toEqual(['Asuransi Kredit', 'ANEKA', 'Travel'])
    expect(within(tablist).getByRole('tab', { name: 'Asuransi Kredit' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('berpindah tab mengirim ?sumber= yang baru ke daftar perusahaan', async () => {
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    calls = []
    await userEvent.click(within(tablist).getByRole('tab', { name: 'ANEKA' }))

    await waitFor(() => {
      expect(
        calls.some(
          (c) =>
            c.url.startsWith('/api/inbox-auto-claim/ringkasan') && c.url.includes('sumber=aneka'),
        ),
      ).toBe(true)
    })
  })

  it('berpindah tab menutup baris yang terbuka dan mengosongkan pencarian', async () => {
    // Kode perusahaan pada satu tabel belum tentu ada di tabel lain.
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    await userEvent.type(screen.getByLabelText('Cari nama perusahaan'), 'mitra')

    const tablist = screen.getByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))

    await companyTable()
    expect(screen.queryByRole('table', { name: /^Batch / })).toBeNull()
    expect(screen.getByLabelText('Cari nama perusahaan')).toHaveValue('')
  })

  it('grid batch yang dibuka membaca tab yang sedang terbuka', async () => {
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))
    calls = []
    await openCompany('Mitra Finansial Nusantara')

    const batchCall = calls.find((c) => c.url.startsWith('/api/inbox-auto-claim?'))
    expect(batchCall?.url).toContain('sumber=travel')
    expect(batchCall?.url).toContain('perusahaan=MFIN')
  })

  it('menyebut tabel sumber sesuai tab yang terbuka', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Sumber: POOLDATA.TMP_BATCH_CLAIM_KREDIT')).toBeInTheDocument()

    const tablist = screen.getByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))

    expect(await screen.findByText('Sumber: POOLDATA.TMP_BATCH_AUTO_TRAVEL')).toBeInTheDocument()
  })
})
