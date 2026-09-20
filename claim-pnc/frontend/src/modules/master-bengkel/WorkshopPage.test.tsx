import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { WorkshopPage } from './WorkshopPage'

const BRANCHES = {
  portal: 'ASM',
  cabang: [
    { id: '001', nama: 'Cabang Contoh Pusat' },
    { id: '002', nama: 'Cabang Contoh Bandung' },
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

/** Bengkel rekanan yang sudah disetujui. */
const APPROVED = {
  id_bengkel: '010000000001',
  nama_bengkel: 'Bengkel Contoh Utama',
  alamat_bengkel: 'Jalan Contoh Nomor 1',
  telp_bengkel: '021-0000001',
  nohp_bengkel: '0800-0000-001',
  email: 'kontak@contoh-bengkel.example',
  email_wo: 'wo@contoh-bengkel.example',
  id_cabang: '001',
  nama_cabang: 'Cabang Contoh Pusat',
  id_kota: '3171',
  nama_kota: 'Jakarta Pusat',
  status_rekanan: '1',
  status_bengkel: '1',
  alasan_status_bengkel: 'Kerja sama aktif.',
  tanggal_status: '01/01/2026',
  login_aplikasi: 'bengkelcontoh1',
  id_bank: '002',
  nama_bank: 'Bank Contoh Satu',
  no_rekening: '1000000001',
  nama_rekening: 'Bengkel Contoh Utama',
  id_rekening: 'ACC-0001',
  nama_npwp: 'Bengkel Contoh Utama',
  no_npwp: '00.000.000.0-000.001',
  alamat_npwp: 'Jalan Contoh Nomor 1',
  jenis_pph: 'PPh 23',
  ppn: '11',
  diskon_jasa: '10',
  diskon_sparepart: '5',
  persen_material: '100',
  pct_selisih_pl: '0',
  sla: '3',
  status_disupply_asm: '1',
  supplier: '',
  status_eklaim: '1',
  status_auto_aksep: '1',
  status_payment: '1',
  status_autopayment: '0',
  status_tekno: '1',
  status_order: '1',
  id_dokumen: '',
  status: '1',
  status_label: 'Approve',
  rekanan: true,
}

/**
 * Bengkel REKANAN yang login aplikasinya kosong.
 *
 * Keadaan yang di Pega tidak terlihat di layar mana pun, dan berakibat bengkelnya tidak
 * akan pernah dapat masuk. Uji di bawah memastikan layar menjelaskannya.
 */
const APPROVED_WITHOUT_LOGIN = {
  ...APPROVED,
  id_bengkel: '010000000004',
  nama_bengkel: 'Bengkel Contoh Tanpa Login',
  login_aplikasi: '',
}

/** Bengkel yang menunggu keputusan. */
const PENDING = {
  ...APPROVED,
  id_bengkel: '010000000002',
  nama_bengkel: 'Bengkel Contoh Menunggu',
  login_aplikasi: 'bengkelcontoh2',
  status: '0',
  status_label: 'Waiting Approval',
}

/** Bengkel non-rekanan yang ditolak, dan karena itu tanpa login. */
const REJECTED = {
  ...APPROVED,
  id_bengkel: '010000000003',
  nama_bengkel: 'Bengkel Contoh Ditolak',
  login_aplikasi: '',
  status_rekanan: '0',
  rekanan: false,
  status: '2',
  status_label: 'Reject',
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
 * defaultReply melayani ketiga lookup dan daftar; mutasi dijawab pemanggil.
 *
 * Ketiga jalur lookup diperiksa LEBIH DULU karena ketiganya berawalan sama dengan jalur
 * daftar — persis alasan yang sama yang membuat rutenya didaftarkan lebih dulu di
 * backend.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/bengkel/cabang')) return { body: BRANCHES }
    if (call.url.startsWith('/api/master/bengkel/bank')) return { body: BANKS }
    if (call.url.startsWith('/api/master/bengkel/kota')) return { body: CITIES }

    if (call.method === 'GET') {
      const status = new URL(call.url, 'http://x').searchParams.get('status') ?? '1'

      let rows: unknown[] = []
      if (status === '1') rows = [APPROVED, APPROVED_WITHOUT_LOGIN]
      if (status === '0') rows = [PENDING]
      if (status === '2') rows = [REJECTED]

      return { body: { bengkel: rows, status, portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { bengkel: APPROVED, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <WorkshopPage />
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

/** listCalls mengambil permintaan daftar saja, membuang ketiga lookup. */
function listCalls() {
  return calls.filter((c) => c.method === 'GET' && c.url.startsWith('/api/master/bengkel?'))
}

/**
 * tab mengambil tombol tab menurut captionnya.
 *
 * Dicari DI DALAM bilah tab, bukan di seluruh halaman: caption tabnya — "Approve" dan
 * "Reject" — adalah caption Pega yang memang ditiru (D-13), dan kata yang sama muncul
 * pada tombol keputusan. Pencarian yang tidak dibatasi akan menemukan keduanya, dan itu
 * bukan cacat uji melainkan gambaran keadaan yang sebenarnya di layar.
 */
function tab(name: string) {
  return within(screen.getByRole('navigation', { name: 'Tab Master Bengkel' })).getByRole(
    'button',
    { name },
  )
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

describe('daftar master bengkel', () => {
  it('menampilkan judul dan kolom sesuai grid layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Bengkel HE' })).toBeInTheDocument()

    // Keenam kolom data, pada urutan yang sama dengan grid Pega — `pyColumnCount = 7`,
    // enam kolom data ditambah kolom aksi.
    const table = await screen.findByRole('table')
    const header = within(table)
      .getAllByRole('columnheader')
      .map((h) => h.textContent?.trim() ?? '')

    expect(header).toEqual([
      'ID Bengkel',
      'Nama Bengkel',
      'Alamat Bengkel',
      'Telp Bengkel',
      'No HP Bengkel',
      'Login Aplikasi',
      'Aksi',
    ])
  })

  // Urutan tab mengikuti layar Pega: Approve, Reject, Waiting Approval.
  //
  // Tab pertama adalah yang terbuka saat layar dibuka, dan petugas yang terbiasa menekan
  // tab kedua akan menekan tab yang berbeda bila urutannya diubah.
  it('menggambar tab pada urutan layar Pega', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    const label = within(screen.getByRole('navigation', { name: 'Tab Master Bengkel' }))
      .getAllByRole('button')
      .map((b) => b.textContent?.trim() ?? '')

    expect(label).toEqual(['Approve', 'Reject', 'Waiting Approval'])
  })

  // Ketiga tab adalah tiga nilai status atas SATU endpoint. Caption-nya dibaca langsung
  // dari Section/BrowseMasterHE-Section.xml.
  it('ketiga tab mengirim penyaring status yang benar', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(listCalls()[0]?.url).toContain('status=1')

    const user = userEvent.setup()

    await user.click(tab('Waiting Approval'))
    await waitFor(() => expect(listCalls().at(-1)?.url).toContain('status=0'))

    await user.click(tab('Reject'))
    await waitFor(() => expect(listCalls().at(-1)?.url).toContain('status=2'))
  })

  // Portal entitas disebut di setiap permintaan. Tanpa header itu, backend MENOLAK —
  // ia tidak pernah jatuh ke portal utama sebagai cadangan (R-20).
  it('menyebut portal entitas pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
    expect(screen.getByText('ASM')).toBeInTheDocument()
  })

  /*
    Bengkel REKANAN tanpa login aplikasi ditandai di layar.

    Keadaan itu tidak terlihat di layar Pega mana pun, dan akibatnya nyata: bengkelnya
    tidak akan pernah dapat masuk aplikasi bengkel.
  */
  it('menandai bengkel rekanan yang belum punya login aplikasi', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('rekanan tanpa login')).toBeInTheDocument()
  })

  /*
    Bengkel NON-rekanan yang login-nya kosong TIDAK ditandai.

    Sel kosongnya berarti "memang tidak diberi login", bukan "seharusnya punya tetapi
    tidak" — dan menandai keduanya akan membuat tandanya tidak berarti apa-apa.
  */
  it('tidak menandai bengkel non-rekanan yang tanpa login', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(tab('Reject'))
    expect(await screen.findByText('Bengkel Contoh Ditolak')).toBeInTheDocument()
    expect(screen.queryByText('rekanan tanpa login')).not.toBeInTheDocument()
  })
})

describe('paginasi', () => {
  /** manyRows menghasilkan sejumlah baris disetujui dengan kunci yang berbeda-beda. */
  function manyRows(count: number) {
    return Array.from({ length: count }, (_, index) => ({
      ...APPROVED,
      id_bengkel: `0100000${String(index + 100).padStart(5, '0')}`,
      nama_bengkel: `Bengkel Contoh ${index + 1}`,
    }))
  }

  function installMany(count: number) {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/bengkel/cabang')) return { body: BRANCHES }
      if (call.url.startsWith('/api/master/bengkel/bank')) return { body: BANKS }
      if (call.url.startsWith('/api/master/bengkel/kota')) return { body: CITIES }
      return { body: { bengkel: manyRows(count), status: '1', portal: 'ASM' } }
    })
  }

  /*
    Dua puluh baris per halaman.

    Angkanya bukan pilihan: `pyPageSizeOther` pada
    `Section/BrowseMasterHEApprove-Section.xml` bernilai 20, karena `pyPageSize` di
    section yang sama bernilai "Other". Layar master lain memakai 15, sehingga angka ini
    milik layar — bukan bawaan komponen tabel.
  */
  it('memotong daftar pada 20 baris per halaman', async () => {
    installMany(47)
    show()

    const table = await screen.findByRole('table')
    // Satu baris header ditambah 20 baris data.
    expect(within(table).getAllByRole('row')).toHaveLength(21)

    // Ringkasan barisnya role="status", sehingga berpindah halaman diumumkan pembaca
    // layar tanpa memindahkan fokus.
    expect(screen.getByRole('status')).toHaveTextContent('Menampilkan 1–20 dari 47 baris.')
  })

  it('berpindah halaman lewat nomor halaman', async () => {
    installMany(47)
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Halaman 3' }))

    // Halaman ketiga memuat sisanya: 47 − 40 = 7 baris.
    await waitFor(() => {
      const table = screen.getByRole('table')
      expect(within(table).getAllByRole('row')).toHaveLength(8)
    })
    expect(screen.getByRole('status')).toHaveTextContent('Menampilkan 41–47 dari 47 baris.')
  })

  // Daftar yang lebih pendek dari satu halaman tidak menggambar paginator sama sekali.
  it('tidak menggambar paginator bila hanya ada satu halaman', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.queryByRole('navigation', { name: 'Halaman tabel' })).not.toBeInTheDocument()
  })
})

describe('keputusan borongan', () => {
  // Centang dan tombol keputusan HANYA ada di tab Waiting Approval — sama seperti di
  // Pega, yang menaruh keduanya di grid persetujuan saja.
  it('tidak menampilkan centang di tab Approve', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByRole('checkbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Approve terpilih' })).not.toBeInTheDocument()
  })

  it('tombol keputusan mati sampai ada yang dicentang', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(tab('Waiting Approval'))

    const approve = await screen.findByRole('button', { name: 'Approve terpilih' })
    expect(approve).toBeDisabled()

    await user.click(await screen.findByRole('checkbox'))
    await waitFor(() => expect(approve).toBeEnabled())
  })

  /*
    SATU permintaan untuk seluruh baris yang dicentang, bukan satu per baris.

    Bentuknya mengikuti Activity/SetApprovalAllMaster — memecahnya menjadi sederet
    permintaan akan mengubah operasi yang di Pega utuh menjadi sesuatu yang dapat gagal
    separuh jalan.
  */
  it('mengirim satu permintaan berisi seluruh baris yang dicentang', async () => {
    installFetch(
      defaultReply((call) => {
        if (call.url === '/api/master/bengkel/keputusan') {
          return {
            body: {
              jumlah_berubah: 1,
              status: '1',
              status_label: 'Approve',
              portal: 'ASM',
            },
          }
        }
        return { body: { bengkel: APPROVED, portal: 'ASM' } }
      }),
    )
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(tab('Waiting Approval'))

    await user.click(await screen.findByRole('checkbox'))
    await user.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    await waitFor(() => {
      const decision = calls.filter((c) => c.url === '/api/master/bengkel/keputusan')
      expect(decision).toHaveLength(1)
      expect(decision[0]?.body).toEqual({
        id_bengkel: [PENDING.id_bengkel],
        status: '1',
      })
    })

    expect(
      await screen.findByText(/1 bengkel dipindahkan ke/),
    ).toBeInTheDocument()
  })

  // Berpindah tab membuang centang: baris yang dipilih milik tab sebelumnya, dan
  // menyimpannya berarti keputusan dapat mengenai baris yang tidak sedang dilihat.
  it('membuang centang saat berpindah tab', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(tab('Waiting Approval'))
    await user.click(await screen.findByRole('checkbox'))

    expect(await screen.findByText(/1 bengkel/)).toBeInTheDocument()

    await user.click(tab('Reject'))
    await user.click(tab('Waiting Approval'))

    expect(
      await screen.findByText('Centang bengkel yang akan diputuskan.'),
    ).toBeInTheDocument()
  })
})

describe('form bengkel', () => {
  it('menolak menyimpan tanpa nama bengkel', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.click(await screen.findByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama bengkel wajib diisi.')).toBeInTheDocument()
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
  })

  /*
    Login aplikasi wajib HANYA untuk bengkel rekanan.

    Syaratnya bukan karangan: Activity/ValidationLoginBengkel_act melompati seluruh
    urusan login pada prasyarat Local.STS_REKANAN=='0'.
  */
  it('mewajibkan login aplikasi hanya untuk bengkel rekanan', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.type(await screen.findByLabelText('Nama bengkel'), 'Bengkel Contoh Baru')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText('Login aplikasi wajib diisi untuk bengkel rekanan.'),
    ).toBeInTheDocument()

    // Diubah menjadi non-rekanan: isian yang sama tidak lagi dituntut.
    await user.selectOptions(screen.getByLabelText('Status rekanan'), '0')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.filter((c) => c.method === 'POST')).toHaveLength(1))
  })

  // Persentase ditolak bila bukan angka 0–100. SELISIH YANG DIRENCANAKAN terhadap Pega,
  // yang menerima teks apa pun.
  it('menolak persentase yang bukan angka 0-100', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.type(await screen.findByLabelText('Nama bengkel'), 'Bengkel Contoh Baru')
    await user.selectOptions(screen.getByLabelText('Status rekanan'), '0')
    await user.type(screen.getByLabelText('PPN (%)'), '101')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText('Harus berupa angka antara 0 dan 100, misalnya 11 atau 12,5.'),
    ).toBeInTheDocument()
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
  })

  /*
    Peringatan bahwa akun bengkel TIDAK diterbitkan sistem baru.

    Di Pega, menyetujui bengkel rekanan membuat operator lewat GCNMCreateOperator —
    dengan kata sandi yang sama untuk setiap bengkel. Akun itu tidak dibawa, dan tanpa
    kalimat ini petugas akan mengira bengkelnya sudah bisa masuk.
  */
  it('menyatakan bahwa akun bengkel belum diterbitkan', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(await screen.findByText(/Akun untuk masuk aplikasi bengkel belum diterbitkan/))
      .toBeInTheDocument()
  })

  // Menyunting memuat isian dari baris yang dipilih, dan menyebut kuncinya sebagai
  // keterangan yang tidak dapat diubah.
  it('memuat baris yang disunting beserta kuncinya', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    const table = screen.getByRole('table')
    await user.click(within(table).getAllByRole('button', { name: 'Ubah' })[0]!)

    expect(await screen.findByRole('heading', { name: /Ubah Bengkel Contoh Utama/ }))
      .toBeInTheDocument()
    expect(screen.getByLabelText('Nama bengkel')).toHaveValue('Bengkel Contoh Utama')

    // Dicari DI DALAM form: kunci yang sama juga tampil pada barisnya di tabel, dan itu
    // memang benar — yang diperiksa di sini adalah form menyebutkannya sebagai
    // keterangan, bukan bahwa kuncinya muncul sekali di halaman.
    const form = screen.getByRole('form', { name: /Ubah Bengkel Contoh Utama/ })
    expect(within(form).getByText(APPROVED.id_bengkel)).toBeInTheDocument()
  })

  // Menyimpan mengirim PUT ke kunci baris, bukan POST.
  it('menyimpan perubahan lewat PUT ke kunci barisnya', async () => {
    installFetch(
      defaultReply(() => ({ body: { bengkel: { ...APPROVED, status: '0' }, portal: 'ASM' } })),
    )
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    const table = screen.getByRole('table')
    await user.click(within(table).getAllByRole('button', { name: 'Ubah' })[0]!)
    await screen.findByRole('heading', { name: /Ubah Bengkel Contoh Utama/ })

    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.filter((c) => c.method === 'PUT')
      expect(saved).toHaveLength(1)
      expect(saved[0]?.url).toBe(`/api/master/bengkel/${APPROVED.id_bengkel}`)
    })
  })

  /*
    Penanda sistem menawarkan saran dari NILAI YANG SUDAH DIPAKAI baris lain.

    Daftar pilihannya tidak ada di export Pega (R-16), dan mengarang dua pilihan
    "Ya/Tidak" berarti menebak domain kolom yang menentukan kanal mana yang boleh dipakai
    bengkel. Sarannya karena itu berasal dari data.
  */
  it('menawarkan saran penanda sistem dari nilai yang sudah dipakai', async () => {
    installFetch(defaultReply())
    const view = show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await screen.findByLabelText('Status e-klaim')

    const datalist = view.container.querySelector('#bengkel-status_eklaim')
    expect(datalist).not.toBeNull()
    expect(datalist?.querySelectorAll('option')).toHaveLength(1)
    expect(datalist?.querySelector('option')?.getAttribute('value')).toBe('1')
  })
})
