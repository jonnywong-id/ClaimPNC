import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SupplierPage } from './SupplierPage'

const BRANCHES = {
  portal: 'ASM',
  cabang: [
    { id: '001', nama: 'Cabang Contoh Pusat' },
    { id: '002', nama: 'Cabang Contoh Bandung' },
  ],
}

const COUNTRIES = {
  portal: 'ASM',
  negara: [
    { id: 'ID', nama: 'Indonesia' },
    { id: 'SG', nama: 'Singapura' },
  ],
}

const BANKS = {
  portal: 'ASM',
  bank: [
    { kode: '002', nama: 'Bank Contoh Satu' },
    { kode: '008', nama: 'Bank Contoh Dua' },
  ],
}

const CITIES = {
  portal: 'ASM',
  kota: [
    { id: '3171', nama: 'Jakarta Pusat' },
    { id: '3174', nama: 'Jakarta Selatan' },
  ],
}

/**
 * Kelima daftar sandi, sebagaimana server menggabungkannya.
 *
 * Perhatikan bedanya: `status_supply` dan `status_aktif` punya LABEL yang artinya terbukti
 * dari activity, sedangkan `status_rekanan` dan `jenis_supplier` labelnya berisi sandinya
 * sendiri — keduanya hanya ditemukan di data, dan artinya memang tidak diketahui (R-16).
 */
const CODES = {
  portal: 'ASM',
  status_rekanan: [
    { nilai: '1', label: '1' },
    { nilai: '0', label: '0' },
  ],
  status_supply: [
    { nilai: '1', label: 'Heavy Equipment' },
    { nilai: '0', label: 'Selain Heavy Equipment' },
  ],
  jenis_supplier: [
    { nilai: '1', label: '1' },
    { nilai: '2', label: '2' },
  ],
  status_aktif: [
    { nilai: '1', label: 'Aktif' },
    { nilai: '0', label: 'Tidak aktif' },
  ],
  status_autopayment: [
    { nilai: '1', label: 'Ya' },
    { nilai: '0', label: 'Tidak' },
  ],
}

/** Supplier aktif yang termasuk Heavy Equipment. */
const ACTIVE = {
  id_supplier: '0100000000001',
  id_lama: '',
  nama: 'Supplier Contoh Utama',
  alamat: 'Jalan Contoh Nomor 1',
  kota: 'Jakarta Pusat',
  nama_cabang: 'Cabang Contoh Pusat',
  kode_pos: '10110',
  negara: 'Indonesia',
  telepon: '021-0000001',
  fax: '021-0000002',
  email: 'kontak@contoh-supplier.example',
  npwp: '00.000.000.0-000.000',
  contact_person: 'Contoh Narahubung',
  status_rekanan: '1',
  status_supply: '1',
  supplier_he: '1',
  term_of_payment: '30',
  term_of_delivery: '14',
  keterangan: 'Kerja sama aktif.',
  bank: 'Bank Contoh Satu',
  no_account: '1000000001',
  account_name: 'Supplier Contoh Utama',
  bank_branch: 'KCP Contoh Pusat',
  jenis_supplier: '1',
  status_aktif: '1',
  status_aktif_berlaku: '1',
  status_autopayment: '1',
  diubah_oleh: 'CONTOH.PETUGAS',
  diubah_pada: '01/01/2026',
  heavy_equipment: true,
  aktif: true,
}

/**
 * Supplier yang DIMINTA aktif tetapi BELUM berlaku aktif.
 *
 * Keadaan yang paling membingungkan di layar ini: `CreateNewMasterSupplier_post` step 6
 * menetapkan STS_AKTIF := "0" tanpa syarat, sehingga supplier yang baru disimpan sebagai
 * aktif tampil sebagai belum aktif. Uji di bawah memastikan layar menjelaskannya.
 */
const WAITING = {
  ...ACTIVE,
  id_supplier: '0100000000002',
  nama: 'Supplier Contoh Menunggu',
  status_aktif: '1',
  status_aktif_berlaku: '0',
  aktif: false,
}

/** Supplier yang memang tidak aktif — bukan menunggu apa pun. */
const INACTIVE = {
  ...ACTIVE,
  id_supplier: '0100000000003',
  nama: 'Supplier Contoh Nonaktif',
  kota: 'Surabaya',
  status_supply: '0',
  supplier_he: '0',
  heavy_equipment: false,
  status_aktif: '0',
  status_aktif_berlaku: '0',
  aktif: false,
}

type Call = {
  url: string
  method: string
  body: unknown
  header: Record<string, string>
}

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
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

/**
 * defaultReply melayani kelima lookup dan daftar; mutasi dijawab pemanggil.
 *
 * Kelima jalur lookup diperiksa LEBIH DULU karena kelimanya berawalan sama dengan jalur
 * daftar — persis alasan yang sama yang membuat rutenya didaftarkan lebih dulu di backend.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/supplier/cabang')) return { body: BRANCHES }
    if (call.url.startsWith('/api/master/supplier/negara')) return { body: COUNTRIES }
    if (call.url.startsWith('/api/master/supplier/bank')) return { body: BANKS }
    if (call.url.startsWith('/api/master/supplier/kota')) return { body: CITIES }
    if (call.url.startsWith('/api/master/supplier/sandi')) return { body: CODES }

    if (call.method === 'GET') {
      return { body: { supplier: [ACTIVE, WAITING, INACTIVE], portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { supplier: ACTIVE, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <SupplierPage />
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

/** listCalls mengambil permintaan daftar saja, membuang kelima lookup. */
function listCalls() {
  return calls.filter(
    (c) => c.method === 'GET' && new URL(c.url, 'http://x').pathname === '/api/master/supplier',
  )
}

/** saveCalls mengambil permintaan penambahan dan penyimpanan. */
function saveCalls() {
  return calls.filter((c) => c.method === 'POST' || c.method === 'PUT')
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

describe('daftar master supplier', () => {
  it('menampilkan judul dan baris dari server', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Supplier' })).toBeInTheDocument()
    expect(await screen.findByText('Supplier Contoh Utama')).toBeInTheDocument()
    expect(screen.getByText('Supplier Contoh Nonaktif')).toBeInTheDocument()
  })

  // Tombolnya memakai caption layar lama apa adanya (D-13):
  // `Harness/MasterSupplier-Harness.xml` memakai "New Supplier" dan "Refresh".
  it('memakai caption tombol layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('button', { name: 'New Supplier' })).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Refresh' })).toBeInTheDocument()
  })

  // Entitas yang sedang dilihat disebut terang-terangan: satu aplikasi melayani empat
  // badan hukum dengan basis data terpisah (ADR-0030, R-20).
  it('menyebutkan portal entitas yang menjawab', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('ASM')).toBeInTheDocument()
  })

  it('mengirim header portal pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  /*
    Keadaan yang paling membingungkan di layar ini, dan yang paling perlu dijelaskan:
    supplier yang DIMINTA aktif tetapi BELUM berlaku aktif.

    Tanpa keterangan itu, petugas melihat supplier yang "sudah disimpan sebagai aktif"
    tampil sebagai belum aktif dan tidak punya cara mengetahui sebabnya.
  */
  it('menjelaskan supplier yang menunggu persetujuan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Menunggu')
    expect(screen.getByText('menunggu persetujuan')).toBeInTheDocument()
  })

  // Supplier yang memang tidak aktif TIDAK diberi keterangan itu — ia tidak menunggu apa
  // pun, dan mengatakannya menunggu akan menyesatkan.
  it('tidak menyebut menunggu pada supplier yang memang tidak aktif', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/supplier/')) return defaultReply()(call)
      if (call.method === 'GET') return { body: { supplier: [INACTIVE], portal: 'ASM' } }
      return defaultReply()(call)
    })
    show()

    await screen.findByText('Supplier Contoh Nonaktif')
    expect(screen.queryByText('menunggu persetujuan')).not.toBeInTheDocument()
  })

  /*
    Kesembilan kolom Pega, pada urutan dan caption yang sama.

    Uji ini ada karena percobaan pertama MENGGABUNGKAN kesembilannya menjadi tujuh kolom
    majemuk — dan yang menangkapnya adalah Work Owner yang melihat layarnya, bukan satu
    pun uji. Sejak sekarang, penggabungan semacam itu gagal di sini lebih dulu.
  */
  it('menggambar kesembilan kolom Pega pada urutan yang sama', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')

    const headers = screen
      .getAllByRole('columnheader')
      .map((cell) => cell.textContent?.trim())

    expect(headers).toEqual([
      'ID',
      'Input Nama',
      'Alamat',
      'Telp',
      'Jenis Supplier',
      'Status Rekanan',
      'Status Aktif',
      'Posisi',
      'Option',
    ])
  })

  /*
    Kolom bersandi menampilkan LABEL-nya, bukan sandinya — sama seperti `_NOTE` di Pega.

    Labelnya dicari dari daftar `/sandi`. Bila artinya tidak diketahui, yang tampil adalah
    sandinya sendiri: pada contoh ini `status_rekanan` "1" berlabel "1", karena arti sandi
    itu memang tidak ada di export (R-16).
  */
  it('menampilkan label sandi, dan sandinya sendiri bila artinya tidak diketahui', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')

    // status_supply "1" punya label yang diketahui.
    expect(screen.getAllByText('Heavy Equipment').length).toBeGreaterThan(0)
    // status_aktif "1" juga.
    expect(screen.getAllByText('Aktif').length).toBeGreaterThan(0)
  })

  // Kolom POSISI tetap digambar meski isinya tidak dapat dibaca dari mana pun — pada
  // layar Pega yang sesungguhnya pun kolom itu tampak kosong.
  it('tetap menggambar kolom POSISI meski selalu kosong', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    expect(screen.getByRole('columnheader', { name: /Posisi/i })).toBeInTheDocument()
  })

  /*
    Dua puluh baris per halaman, dibaca dari `pyPageSize` pada
    `Section/InboxMasterSupplier-Section.xml`.

    Angkanya diuji di sini — bukan hanya di uji DataTable — karena yang menentukannya
    adalah LAYAR, bukan komponen: Master Bengkel juga 20, sementara lima layar master lain
    memakai 15. Uji komponen membuktikan paginasinya bekerja; uji ini membuktikan layar
    ini memakai angka yang benar.
  */
  it('memaginasi 20 baris per halaman, sesuai pyPageSize layar lama', async () => {
    const banyak = Array.from({ length: 25 }, (_, index) => ({
      ...ACTIVE,
      id_supplier: `01000000000${String(index + 10).padStart(2, '0')}`,
      nama: `Supplier Contoh ${String(index + 1).padStart(2, '0')}`,
    }))

    installFetch((call) => {
      if (call.url.startsWith('/api/master/supplier/')) return defaultReply()(call)
      if (call.method === 'GET') return { body: { supplier: banyak, portal: 'ASM' } }
      return defaultReply()(call)
    })
    show()

    await screen.findByText('Supplier Contoh 01')

    expect(screen.getByText('Supplier Contoh 20')).toBeInTheDocument()
    expect(screen.queryByText('Supplier Contoh 21')).not.toBeInTheDocument()
    expect(screen.getByText(/1–20/)).toBeInTheDocument()
  })

  it('membuka baris halaman kedua', async () => {
    const user = userEvent.setup()
    const banyak = Array.from({ length: 25 }, (_, index) => ({
      ...ACTIVE,
      id_supplier: `01000000000${String(index + 10).padStart(2, '0')}`,
      nama: `Supplier Contoh ${String(index + 1).padStart(2, '0')}`,
    }))

    installFetch((call) => {
      if (call.url.startsWith('/api/master/supplier/')) return defaultReply()(call)
      if (call.method === 'GET') return { body: { supplier: banyak, portal: 'ASM' } }
      return defaultReply()(call)
    })
    show()

    await screen.findByText('Supplier Contoh 01')
    await user.click(
      within(screen.getByRole('navigation', { name: 'Halaman tabel' })).getByRole('button', {
        name: 'Halaman 2',
      }),
    )

    expect(screen.getByText('Supplier Contoh 25')).toBeInTheDocument()
    expect(screen.queryByText('Supplier Contoh 01')).not.toBeInTheDocument()
  })

  // Tiga baris contoh tidak melampaui satu halaman, sehingga tombolnya tidak digambar.
  it('tidak menggambar tombol halaman bila barisnya sedikit', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    expect(screen.queryByRole('navigation', { name: 'Halaman tabel' })).not.toBeInTheDocument()
    expect(screen.getByText(/1–3/)).toBeInTheDocument()
  })

  it('menolak memuat daftar sebelum portal dipilih', () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(screen.getByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(listCalls()).toHaveLength(0)
  })

  /*
    Kelima dropdown bersandi wajib diisi. Bila daftarnya gagal dimuat, form-nya tidak
    dapat dipakai sama sekali — dan itu harus dikatakan, bukan dibiarkan tampak sebagai
    form yang rusak.
  */
  it('memberi tahu bila daftar pilihan status gagal dimuat', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/supplier/sandi')) {
        return { body: { kode: 'galat', pesan: 'gagal' }, status: 500 }
      }
      return defaultReply()(call)
    })
    show()

    expect(
      await screen.findByText('Daftar pilihan status tidak dapat dimuat'),
    ).toBeInTheDocument()
  })
})

describe('form master supplier', () => {
  it('membuka form tambah lewat tombol New Supplier', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getByRole('button', { name: 'New Supplier' }))

    expect(
      await screen.findByRole('heading', { name: 'Tambah Master Supplier' }),
    ).toBeInTheDocument()
  })

  /*
    Nama DIKUNCI pada mode ubah.

    Aturannya dibaca dari layar lamanya: `Section/CreateMasterSupplier_Sec-Section.xml`
    memasang `pyReadOnlyCondition: MasterSupplier.ID != ''` pada isian ini — satu-satunya
    isian di form itu yang punya syarat semacam itu.
  */
  it('mengunci isian Nama saat mengubah', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)

    const nama = await screen.findByLabelText('Nama')
    expect(nama).toHaveAttribute('readonly')
    expect(screen.getByText('Nama terkunci sejak baris ini tersimpan.')).toBeInTheDocument()
  })

  // Pada mode tambah, isian yang sama justru harus dapat diketik.
  it('membiarkan isian Nama dapat diketik saat menambah', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getByRole('button', { name: 'New Supplier' }))

    const nama = await screen.findByLabelText('Nama')
    expect(nama).not.toHaveAttribute('readonly')
  })

  // ID diterbitkan server; pada mode ubah ia ditampilkan sebagai keterangan, bukan isian.
  it('menampilkan ID sebagai keterangan saat mengubah', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)

    // Dicari DI DALAM form, bukan di seluruh halaman: ID yang sama juga tampil di baris
    // daftarnya, dan pencarian yang tidak dibatasi akan menemukan keduanya.
    const form = await screen.findByRole('form', { name: /Ubah Supplier Contoh Utama/ })
    expect(within(form).getByText('ID supplier')).toBeInTheDocument()
    expect(within(form).getByText('0100000000001')).toBeInTheDocument()
  })

  /*
    Kedua jalur berperilaku berbeda, dan bedanya menyentuh supplier mana yang boleh
    dipakai — sehingga harus disebut SEBELUM tombol Simpan ditekan.
  */
  it('memberi tahu bahwa supplier baru selalu belum aktif', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getByRole('button', { name: 'New Supplier' }))

    expect(await screen.findByText(/selalu tersimpan sebagai/)).toBeInTheDocument()
  })

  it('memberi tahu bahwa menonaktifkan berlaku seketika', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)

    expect(await screen.findByText(/Menonaktifkannya berlaku/)).toBeInTheDocument()
  })

  // Kelima dropdown bersandi diisi dari server, bukan dari daftar yang ditanam di layar.
  it('mengisi dropdown bersandi dari server', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getByRole('button', { name: 'New Supplier' }))

    const supply = await screen.findByLabelText('Status Supply')
    expect(supply).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Heavy Equipment' })).toBeInTheDocument()

    // Sandi yang artinya TIDAK diketahui tampil apa adanya, tanpa tebakan.
    expect(screen.getByLabelText('Jenis Supplier')).toBeInTheDocument()
  })

  // SELURUH pelanggaran dikirim server sekaligus (P-5), dan layar menyorotnya satu per
  // satu. Pada form lima belas isian wajib, ringkasan saja tidak memberi tahu yang mana.
  it('menyorot setiap isian yang ditolak server', async () => {
    const user = userEvent.setup()
    installFetch(
      defaultReply(() => ({
        status: 422,
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar.',
          // Nama fieldnya `kolom` — itu salah satu dari dua bentuk yang dibaca
          // `APIError.violations()`. Nama lain membuat layar menampilkan kotak galat umum
          // alih-alih menyorot isiannya, dan uji ini yang menjaganya.
          detail: [
            { kolom: 'nama', pesan: 'Nama wajib diisi.' },
            { kolom: 'telepon', pesan: 'Telepon wajib diisi.' },
          ],
        },
      })),
    )
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)
    await screen.findByLabelText('Nama')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(screen.getByText('Nama wajib diisi.')).toBeInTheDocument()
    })
    expect(screen.getByText('Telepon wajib diisi.')).toBeInTheDocument()
  })

  // Form ditutup HANYA setelah server menjawab berhasil — pada form dua puluh tiga isian,
  // menutupnya lebih dulu membuang isian pengguna saat penyimpanan gagal.
  it('mempertahankan form saat penyimpanan gagal', async () => {
    const user = userEvent.setup()
    installFetch(
      defaultReply(() => ({
        status: 409,
        body: { kode: 'nama_supplier_sudah_ada', pesan: 'Nama supplier tersebut telah digunakan.' },
      })),
    )
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)
    await screen.findByLabelText('Nama')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama supplier itu sudah dipakai')).toBeInTheDocument()
    expect(screen.getByLabelText('Alamat')).toBeInTheDocument()
  })

  // Menyimpan tanpa mengubah apa pun harus berhasil, dan badannya membawa nama yang
  // tersimpan apa adanya — nama tidak pernah ikut berubah lewat jalur ini.
  it('menyimpan perubahan lewat PUT dan menutup form', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)
    await screen.findByLabelText('Nama')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(saveCalls()).toHaveLength(1)
    })

    const sent = saveCalls()[0]!
    expect(sent.method).toBe('PUT')
    expect(sent.url).toBe('/api/master/supplier/0100000000001')
    expect((sent.body as Record<string, unknown>)['nama']).toBe('Supplier Contoh Utama')

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Simpan' })).not.toBeInTheDocument()
    })
  })

  /*
    Kota dikirim sebagai NAMA, bukan kode.

    Dokumen supplier tidak punya kunci KOTA_ID — `GetDataEditMasterSupller-SQL.xml`
    membaca KOTA saja. Mengirim kodenya berarti menyimpan sesuatu yang tidak dapat dibaca
    layar Pega mana pun.
  */
  it('mengirim kota sebagai nama, bukan kode', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)
    await screen.findByLabelText('Nama')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(saveCalls()).toHaveLength(1)
    })
    expect((saveCalls()[0]!.body as Record<string, unknown>)['kota']).toBe('Jakarta Pusat')
  })

  // Nilai turunan TIDAK ikut dikirim: server yang menurunkannya, dan menerimanya dari
  // klien akan membuka dua sumber untuk satu nilai.
  it('tidak mengirim nilai yang diturunkan server', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getAllByRole('button', { name: 'Edit' })[0]!)
    await screen.findByLabelText('Nama')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(saveCalls()).toHaveLength(1)
    })

    const body = saveCalls()[0]!.body as Record<string, unknown>
    for (const derived of [
      'id_supplier',
      'id_lama',
      'supplier_he',
      'status_aktif_berlaku',
      'diubah_oleh',
      'diubah_pada',
      'heavy_equipment',
      'aktif',
    ]) {
      expect(body).not.toHaveProperty(derived)
    }
  })

  it('membatalkan form tanpa mengirim apa pun', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Supplier Contoh Utama')
    await user.click(screen.getByRole('button', { name: 'New Supplier' }))
    await screen.findByLabelText('Nama')
    await user.click(screen.getByRole('button', { name: 'Batal' }))

    await waitFor(() => {
      expect(screen.queryByLabelText('Nama')).not.toBeInTheDocument()
    })
    expect(saveCalls()).toHaveLength(0)
  })
})
