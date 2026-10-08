import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/masking'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Baris contoh yang BENTUKNYA sama dengan produksi, isinya tidak.
 *
 * Yang ditiru: sub modul dipisah koma dengan koma di ujung, kuota yang sangat besar, dan
 * satu baris nonaktif. Ketiganya adalah keadaan yang benar-benar ada di portal ASM dan
 * paling mudah salah ditangani layar.
 *
 * Yang TIDAK ditiru: nama penggunanya. Modul ini memuat kewenangan melihat data pribadi —
 * menyalin daftar nama sungguhan ke berkas yang di-commit berarti menerbitkan daftar siapa
 * yang boleh membuka nomor KTP nasabah (`D-69`).
 */
const SAMPLE = [
  {
    id: '1',
    cabang: '100081',
    nama_cabang: 'KANTOR PUSAT',
    login: 'CONTOH.ADMIN',
    modul: 'PNCSearchKlaim',
    sub_modul: 'Registrasi,Dokumen,',
    maks_cari: 100000,
    maks_lihat: 100000,
    lihat_ktp: true,
    lihat_email: true,
    lihat_notelp: true,
    aktif: true,
    dicatat_oleh: 'CONTOH.SUPERVISOR',
    dicatat_pada: '2026-09-01T08:00:00Z',
  },
  {
    id: '2',
    cabang: '100069',
    nama_cabang: 'BOGOR',
    login: 'CONTOH.TEKNIK',
    modul: 'PNCSearchKlaim',
    sub_modul: '',
    maks_cari: 5,
    maks_lihat: 5,
    lihat_ktp: true,
    lihat_email: false,
    lihat_notelp: false,
    aktif: false,
    dicatat_oleh: 'CONTOH.SUPERVISOR',
    dicatat_pada: '2026-09-01T08:00:00Z',
  },
]

const BRANCHES = {
  cabang: [
    { kode: '100001', nama: 'AGENCY MANADO' },
    { kode: '100081', nama: 'KANTOR PUSAT' },
  ],
  total: 2,
  portal: 'ASM',
}

/**
 * Petugas contoh cabang 100001, mengisi tabel pada form Tambah.
 *
 * Loginnya dikarang — modul ini memuat kewenangan melihat data pribadi, dan menyalin
 * daftar nama sungguhan ke berkas yang di-commit berarti menerbitkan daftar siapa yang
 * boleh membuka nomor KTP nasabah (`D-69`).
 */
const OPERATORS = {
  pengguna: [
    { login: 'CONTOH.BARU.SATU', nama: 'CONTOH BARU SATU' },
    { login: 'CONTOH.BARU.DUA', nama: 'CONTOH BARU DUA' },
    { login: 'contoh.surel@example.invalid', nama: 'CONTOH SUREL' },
  ],
  total: 3,
  portal: 'ASM',
}

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
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    if (url.startsWith(`${ROUTE}/cabang`)) return Promise.resolve(jsonResponse(200, BRANCHES))
    if (url.startsWith(`${ROUTE}/pengguna`)) return Promise.resolve(jsonResponse(200, OPERATORS))
    return Promise.resolve(reply(url, init))
  })
}

/** Peladen tiruan yang menjawab daftar dan menerima simpan serta ubah status. */
function installDefaultFetch() {
  installFetch((url, init) => {
    // POST berbentuk MASSAL: satu cabang, banyak baris, dan jawabannya melaporkan hasil
    // per baris. Tiruan ini meniru bentuk itu — bukan bentuk satu baris yang lama.
    if (url === ROUTE && init?.method === 'POST') {
      const body = JSON.parse(String(init.body ?? '{}')) as {
        baris: { login: string }[]
      }
      return jsonResponse(201, {
        tersimpan: body.baris.length,
        ditolak: 0,
        hasil: body.baris.map((b) => ({ login: b.login, tersimpan: true })),
        portal: 'ASM',
      })
    }
    if (url.endsWith('/status') && init?.method === 'PUT') {
      const body = JSON.parse(String(init.body ?? '{}')) as { aktif: boolean }
      return jsonResponse(200, { masking: { ...SAMPLE[0], aktif: body.aktif }, portal: 'ASM' })
    }
    if (url.startsWith(`${ROUTE}/`) && init?.method === 'PUT') {
      const body = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
      return jsonResponse(200, { masking: { ...SAMPLE[0], ...body }, portal: 'ASM' })
    }
    return jsonResponse(200, { masking: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/masking']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
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

describe('daftar', () => {
  // Kolom dan urutannya diambil apa adanya dari `MasterProteksi_Sec`. Uji ini menjaga
  // susunannya, karena itulah yang paling mudah bergeser saat layar disunting kelak — dan
  // pergeserannya tidak menghasilkan galat apa pun, hanya layar yang tidak lagi sama.
  it('menyusun kolom persis seperti grid layar lama', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')

    const header = screen.getAllByRole('columnheader').map((cell) => cell.textContent?.trim())
    expect(header).toEqual([
      'Cabang',
      'Login',
      'Status Aktif',
      'KTP',
      'Email',
      'Notelp',
      'Max Lihat',
      'Max Cari',
      'Lihat Modul',
      'Aksi',
    ])
  })

  it('menampilkan baris nonaktif, tidak menyembunyikannya', async () => {
    installDefaultFetch()
    show()

    // `SearchData.Type == "1"` tidak menyaring apa pun. Sepuluh dari 25 baris produksi
    // berstatus tidak aktif; menyembunyikannya secara baku akan membuat 40% isinya lenyap.
    expect(await screen.findByText('CONTOH.TEKNIK')).toBeInTheDocument()
    expect(screen.getByText('TIDAK AKTIF')).toBeInTheDocument()
    expect(screen.getByText('AKTIF')).toBeInTheDocument()
  })

  // Ketiga kewenangan ditulis "Ya"/"Tidak" apa adanya, sama dengan layar lama. Lencana
  // berwarna sempat dipakai di sini dan dicabut — `D-13` menetapkan tampilan mengikuti
  // Pega, dan lencana adalah bacaan yang berbeda dari yang dikenal petugas.
  it('menulis kewenangan sebagai Ya/Tidak, bukan lencana', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    expect(screen.getAllByText('Ya').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Tidak').length).toBeGreaterThan(0)
  })

  // Angka kuota ditulis apa adanya. Pemisah ribuan sempat ditambahkan dan dicabut: layar
  // lama menulis `100000`, dan `100.000` adalah bentuk yang berbeda dari yang dihafal
  // petugas.
  it('menulis kuota apa adanya tanpa pemisah ribuan', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    expect(screen.getAllByText('100000').length).toBeGreaterThan(0)
    expect(screen.queryByText('100.000')).not.toBeInTheDocument()
  })

  // Kolom LIHAT MODUL berisi TOMBOL VIEW, bukan teks modulnya.
  // Tombol VIEW membuka DIALOG ViewDataModul berisi kotak centang — bukan panel berisi
  // teks modul. Panel teks sempat dipakai dan dicabut: ia menuliskan nilai mentah kolom
  // alih-alih label yang dibaca petugas, dan menghilangkan sub modul yang tidak dicentang.
  it('membuka dialog ViewDataModul berisi kotak centang', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    const view = screen.getByRole('button', { name: 'Lihat modul CONTOH.ADMIN' })
    expect(view).toHaveTextContent('VIEW')

    // Nilai mentah kolom TIDAK pernah tampil di layar.
    expect(screen.queryByText('PNCSearchKlaim')).not.toBeInTheDocument()
    await user.click(view)

    const dialog = await screen.findByRole('dialog', { name: 'ViewDataModul' })
    expect(within(dialog).getByText('MODUL')).toBeInTheDocument()
    expect(within(dialog).getByText('SUB MODUL')).toBeInTheDocument()

    // MODUL ditulis dengan LABEL-nya, bukan nilai kolomnya.
    expect(within(dialog).getByLabelText('VIEW HISTORY KLAIM: berlaku')).toBeChecked()

    // SUB MODUL: SELURUH pilihan tampil, tercentang maupun tidak. Baris contoh pertama
    // menyimpan "Registrasi,Dokumen," — dua tercentang, satu tidak.
    expect(within(dialog).getByLabelText('REGISTRASI: berlaku')).toBeChecked()
    expect(within(dialog).getByLabelText('DOKUMEN: berlaku')).toBeChecked()
    expect(
      within(dialog).getByLabelText('PENERIMA PEMBAYARAN KLAIM: tidak berlaku'),
    ).not.toBeChecked()
  })

  it('menutup dialog ViewDataModul lewat tombol tutup', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Lihat modul CONTOH.ADMIN' }))

    const dialog = await screen.findByRole('dialog', { name: 'ViewDataModul' })
    await user.click(within(dialog).getByRole('button', { name: 'Tutup' }))

    await waitFor(() => {
      expect(screen.queryByRole('dialog', { name: 'ViewDataModul' })).not.toBeInTheDocument()
    })
  })

  it('menyebut entitas yang menjawab, bukan hanya yang diminta', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('mengirim header portal pada setiap permintaan', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    const listCall = calls.find((c) => c.url.startsWith(ROUTE) && !c.url.includes('/cabang'))
    expect(listCall).toBeDefined()
    expect((listCall?.init?.headers as Record<string, string>)['X-Portal']).toBe('ASM')
  })
})

describe('pencarian', () => {
  // Empat tipe, sama dengan `SearchData.Type` layar lama.
  it('menawarkan keempat tipe pencarian layar lama', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    const pilihan = within(screen.getByLabelText('Tipe Pencarian'))
      .getAllByRole('option')
      .map((option) => option.textContent?.trim())

    expect(pilihan).toEqual(['Semua data', 'Nama cabang', 'Nama pengguna', 'Status aktif'])
  })

  it('mengirim tipe dan kata kunci sebagai parameter, bukan dirangkai', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.type(screen.getByLabelText('Nama Pencarian'), 'teknik')
    await user.click(screen.getByRole('button', { name: 'Cari' }))

    await waitFor(() => {
      const searched = calls.find((c) => c.url.includes('kata_kunci='))
      expect(searched).toBeDefined()
      expect(searched?.url).toContain('cari_di=login')
      expect(searched?.url).toContain('kata_kunci=teknik')
    })
  })

  // Tipe status memakai DROPDOWN, bukan kotak teks — sama seperti layar lama, yang
  // menampilkan kontrol berbeda untuk `SearchData.Type=='4'`.
  it('menukar kotak teks dengan dropdown saat mencari menurut status', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.selectOptions(screen.getByLabelText('Tipe Pencarian'), 'status')

    expect(await screen.findByLabelText('Status Aktif')).toBeInTheDocument()
    expect(screen.queryByLabelText('Nama Pencarian')).not.toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Status Aktif'), 'TIDAK AKTIF')
    await user.click(screen.getByRole('button', { name: 'Cari' }))

    await waitFor(() => {
      const searched = calls.find((c) => c.url.includes('cari_di=status'))
      expect(searched).toBeDefined()
      // URLSearchParams menuliskan spasi sebagai '+', bukan '%20'.
      expect(searched?.url).toContain('kata_kunci=TIDAK+AKTIF')
    })
  })
})

describe('menambah — form massal', () => {
  // Form Tambah layar lama berbentuk DAFTAR: satu cabang di atas, lalu tabel petugas
  // cabang itu dengan Template Akses per baris. Uji ini menjaga bentuk itu.
  it('menampilkan tabel Nama User / Login / Template Akses, bukan satu baris isian', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    const header = within(form)
      .getAllByRole('columnheader')
      .map((cell) => cell.textContent?.trim())
    expect(header).toEqual(['', 'Nama User', 'Login', 'Template Akses'])

    // Sebelum cabang dipilih, tabelnya kosong — sama seperti "Data Tidak Ada" di Pega.
    expect(within(form).getByText('Pilih cabang lebih dulu.')).toBeInTheDocument()
  })

  it('mengisi tabel dengan petugas cabang yang dipilih', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))

    expect(await within(form).findByText('CONTOH BARU SATU')).toBeInTheDocument()
    expect(within(form).getByText('CONTOH.BARU.DUA')).toBeInTheDocument()

    // Permintaannya menyebut cabangnya.
    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('/pengguna?cabang=100001'))).toBe(true)
    })
  })

  it('mengirim satu cabang dan beberapa baris sekaligus', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await within(form).findByText('CONTOH BARU SATU')

    await user.click(
      within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.SATU'),
    )
    await user.click(within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.DUA'))
    await user.click(within(form).getByLabelText('KTP untuk CONTOH.BARU.SATU'))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.init?.method === 'POST')
      expect(posted).toBeDefined()

      const body = JSON.parse(String(posted?.init?.body ?? '{}')) as {
        cabang: string
        baris: Record<string, unknown>[]
      }
      // Cabang berada di LUAR daftar baris — satu kali Simpan tidak pernah menyentuh
      // lebih dari satu cabang.
      expect(body.cabang).toBe('100001')
      expect(body.baris).toHaveLength(2)
      expect(body.baris[0]).not.toHaveProperty('cabang')
      expect(body.baris[0]).not.toHaveProperty('aktif')
      expect(body.baris[0]?.['lihat_ktp']).toBe(true)
      expect(body.baris[1]?.['lihat_ktp']).toBe(false)
    })
  })

  it('hanya mengirim baris yang dicentang', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await within(form).findByText('CONTOH BARU SATU')

    // Tiga petugas tampil; hanya satu dicentang.
    await user.click(within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.DUA'))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.init?.method === 'POST')
      const body = JSON.parse(String(posted?.init?.body ?? '{}')) as { baris: unknown[] }
      expect(body.baris).toHaveLength(1)
    })
  })

  // Form MENUTUP setelah seluruh baris tersimpan. Membiarkannya terbuka membuat petugas
  // mengira penyimpanannya belum selesai, lalu menekan Simpan sekali lagi.
  it('menutup form setelah seluruh baris tersimpan', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await within(form).findByText('CONTOH BARU SATU')
    await user.click(within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.SATU'))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(screen.queryByRole('form', { name: 'Input data masking' })).not.toBeInTheDocument()
    })
  })

  // MODUL dan SUB MODUL berupa KOTAK CENTANG, dan bentuk tersimpannya harus sama persis
  // dengan sistem lama: MODUL polos, SUB MODUL dipisah koma DENGAN koma di ujung.
  it('menyimpan modul dan sub modul dari centang, dalam bentuk kolom yang benar', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await within(form).findByText('CONTOH BARU SATU')
    await user.click(within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.SATU'))

    // Setiap baris petugas punya centangnya sendiri, jadi lingkupnya baris — bukan form.
    const row = within(form).getByRole('row', { name: /CONTOH BARU SATU/ })

    // Isian berupa centang, bukan kotak teks.
    expect(within(row).queryByLabelText('Modul')).not.toBeInTheDocument()
    // MODUL sudah tercentang sebagai nilai awal — satu-satunya modul yang ada.
    expect(within(row).getByLabelText('VIEW HISTORY KLAIM')).toBeChecked()

    await user.click(within(row).getByLabelText('REGISTRASI'))
    await user.click(within(row).getByLabelText('DOKUMEN'))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.init?.method === 'POST')
      const body = JSON.parse(String(posted?.init?.body ?? '{}')) as {
        baris: Record<string, string>[]
      }
      // MODUL: nilai kolom, bukan labelnya, dan tanpa koma.
      expect(body.baris[0]?.['modul']).toBe('PNCSearchKlaim')
      // SUB MODUL: urut sesuai daftar pilihan, DENGAN koma di ujung.
      expect(body.baris[0]?.['sub_modul']).toBe('Registrasi,Dokumen,')
    })
  })

  // Kuota diketik langsung tanpa menghapus apa pun lebih dulu, dan angka yang terkirim
  // sama dengan yang terbaca di layar.
  //
  // Dilaporkan petugas: kotak berisi `0`, diketik `10`, yang tampil `010`. Nilai yang
  // terkirim memang benar — yang dibaca petugas tidak.
  it('mengetik kuota pada kotak kosong mengirim angka yang sama dengan yang tampil', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await within(form).findByText('CONTOH BARU SATU')
    await user.click(within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.SATU'))

    const row = within(form).getByRole('row', { name: /CONTOH BARU SATU/ })
    const cari = within(row).getByLabelText('Max Cari Data') as HTMLInputElement
    const lihat = within(row).getByLabelText('Max Lihat Data') as HTMLInputElement

    // Tidak ada `0` yang harus dihapus lebih dulu — dari keharusan itulah `010` lahir.
    expect(cari.value).toBe('')

    await user.click(cari)
    await user.keyboard('10')
    await user.click(lihat)
    await user.keyboard('6')

    expect(cari.value).toBe('10')
    expect(lihat.value).toBe('6')

    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.init?.method === 'POST')
      const body = JSON.parse(String(posted?.init?.body ?? '{}')) as {
        baris: Record<string, number>[]
      }
      expect(body.baris[0]?.['maks_cari']).toBe(10)
      expect(body.baris[0]?.['maks_lihat']).toBe(6)
    })
  })

  it('tidak dapat disimpan sebelum ada petugas yang dicentang', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    expect(within(form).getByRole('button', { name: 'Simpan' })).toBeDisabled()
    expect(within(form).getByText('Belum ada petugas yang dipilih.')).toBeInTheDocument()
  })

  // Sebagian berhasil dan sebagian ditolak dilaporkan PER BARIS — bukan satu kata.
  it('melaporkan hasil per baris saat sebagian ditolak', async () => {
    installFetch((url, init) => {
      if (url === ROUTE && init?.method === 'POST') {
        return jsonResponse(201, {
          tersimpan: 1,
          ditolak: 1,
          hasil: [
            { login: 'CONTOH.BARU.SATU', tersimpan: true },
            {
              login: 'CONTOH.BARU.DUA',
              tersimpan: false,
              pesan: 'Petugas ini sudah punya data masking di cabang tersebut.',
            },
          ],
          portal: 'ASM',
        })
      }
      return jsonResponse(200, { masking: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await within(form).findByText('CONTOH BARU SATU')
    await user.click(within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.SATU'))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('1 tersimpan, 1 ditolak')).toBeInTheDocument()
    expect(within(form).getByText(/sudah punya data masking/)).toBeInTheDocument()
  })

  // SELURUH baris ditolak — alasannya tetap harus sampai ke layar.
  //
  // Ini pernah gagal, dan kegagalannya tidak kelihatan dari sisi mana pun secara terpisah.
  // Server menjawab 409 beserta laporan per barisnya; klien memperlakukan setiap non-2xx
  // sebagai galat, mencari `pesan` yang memang tidak ada di badan laporan, lalu jatuh ke
  // kalimat umum "Terjadi kesalahan pada sistem". Laporannya terbentuk dengan benar dan
  // dibuang tepat sebelum digambar.
  //
  // Karena itu yang diuji di sini bukan nomor statusnya, melainkan yang DIBACA petugas:
  // alasan penolakan muncul, dan form tidak menutup.
  it('menampilkan alasan saat SELURUH baris ditolak, dan form tidak menutup', async () => {
    installFetch((url, init) => {
      if (url === ROUTE && init?.method === 'POST') {
        return jsonResponse(200, {
          tersimpan: 0,
          ditolak: 1,
          hasil: [
            {
              login: 'CONTOH.BARU.SATU',
              tersimpan: false,
              pesan: 'Petugas ini sudah punya data masking di cabang tersebut.',
            },
          ],
          portal: 'ASM',
        })
      }
      return jsonResponse(200, { masking: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Input data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await within(form).findByText('CONTOH BARU SATU')
    await user.click(within(form).getByLabelText('Beri kewenangan kepada CONTOH.BARU.SATU'))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('0 tersimpan, 1 ditolak')).toBeInTheDocument()
    expect(within(form).getByText(/sudah punya data masking/)).toBeInTheDocument()
    expect(screen.queryByText('Terjadi kesalahan pada sistem.')).not.toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Input data masking' })).toBeInTheDocument()
  })
})

describe('aksi pada baris', () => {
  // `ActionMaskingData_Sec` memasang syarat `.STS_AKTF=='AKTIF'` pada KEDUA tombolnya.
  // Baris nonaktif karena itu tidak punya aksi apa pun di layar lama.
  it('hanya menampilkan aksi pada baris aktif', async () => {
    installDefaultFetch()
    show()

    expect(
      await screen.findByRole('button', { name: 'Edit masking CONTOH.ADMIN' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Delete masking CONTOH.ADMIN' }),
    ).toBeInTheDocument()

    // Baris nonaktif: tidak ada Ubah, tidak ada Nonaktifkan.
    expect(
      screen.queryByRole('button', { name: 'Edit masking CONTOH.TEKNIK' }),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /masking CONTOH\.TEKNIK/ }),
    ).not.toBeInTheDocument()
  })

  it('tombolnya disebut Nonaktifkan, bukan Hapus', async () => {
    installDefaultFetch()
    show()

    // Layar lama menamainya DELETE padahal ia tidak pernah membuang baris. Nama yang
    // keliru membuat petugas mengira datanya hilang lalu menambahkannya lagi dari awal.
    await screen.findByText('CONTOH.ADMIN')
    expect(screen.queryByRole('button', { name: /Hapus/ })).not.toBeInTheDocument()
  })

  it('mengirim status lewat jalurnya sendiri', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Delete masking CONTOH.ADMIN' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.url.endsWith('/status'))
      expect(sent).toBeDefined()
      expect(sent?.init?.method).toBe('PUT')

      // Badannya menyebut keadaan yang dituju, bukan menyuruh "balikkan". Dua klik yang
      // tiba bersamaan tidak boleh saling meniadakan sampai pengguna tidak tahu kewenangan
      // itu akhirnya hidup atau mati.
      const body = JSON.parse(String(sent?.init?.body ?? '{}')) as { aktif: boolean }
      expect(body.aktif).toBe(false)
    })
  })
})

describe('mengubah', () => {
  // Form layar lama MEMUAT isian "STATUS" (`MasterProteksi_Sec:22449`), sehingga status
  // ikut tersimpan saat form disimpan.
  it('menampilkan isian status dan mengirimkannya', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Edit masking CONTOH.ADMIN' }))

    const form = await screen.findByRole('form', { name: 'Ubah data masking' })
    expect(within(form).getByLabelText('Status')).toBeInTheDocument()

    await user.selectOptions(within(form).getByLabelText('Status'), 'false')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.init?.method === 'PUT' && !c.url.endsWith('/status'))
      expect(sent).toBeDefined()
      const body = JSON.parse(String(sent?.init?.body ?? '{}')) as Record<string, unknown>
      expect(body['aktif']).toBe(false)

      // Kuota ikut terkirim sebagai ANGKA meski tidak disentuh.
      //
      // Form Ubah memakai NumberField lewat `setValue`, bukan `register`. Bila kaitannya
      // putus, yang terkirim akan berupa NaN atau hilang sama sekali — dan kuota yang
      // hilang pada modul ini berarti kewenangan membuka data pribadi berubah diam-diam.
      expect(body['maks_cari']).toBe(100000)
      expect(body['maks_lihat']).toBe(100000)
    })

    // Form MENUTUP setelah tersimpan — sama seperti form Tambah.
    await waitFor(() => {
      expect(screen.queryByRole('form', { name: 'Ubah data masking' })).not.toBeInTheDocument()
    })
  })

  // Form Ubah memakai kotak centang yang sama, dan harus MEMBACA nilai tersimpan dengan
  // benar — baris contoh pertama menyimpan "Registrasi,Dokumen,".
  it('membaca modul dan sub modul tersimpan sebagai centang', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Edit masking CONTOH.ADMIN' }))

    const form = await screen.findByRole('form', { name: 'Ubah data masking' })
    expect(within(form).getByLabelText('VIEW HISTORY KLAIM')).toBeChecked()
    expect(within(form).getByLabelText('REGISTRASI')).toBeChecked()
    expect(within(form).getByLabelText('DOKUMEN')).toBeChecked()
    expect(within(form).getByLabelText('PENERIMA PEMBAYARAN KLAIM')).not.toBeChecked()
  })

  // Mencabut satu centang harus menghasilkan nilai kolom yang benar — tanpa koma ganda,
  // tanpa kehilangan koma penutup.
  it('menyimpan kembali bentuk kolom yang benar setelah centang diubah', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Edit masking CONTOH.ADMIN' }))

    const form = await screen.findByRole('form', { name: 'Ubah data masking' })
    await user.click(within(form).getByLabelText('DOKUMEN')) // dicabut
    await user.click(within(form).getByLabelText('PENERIMA PEMBAYARAN KLAIM')) // ditambah
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.init?.method === 'PUT' && !c.url.endsWith('/status'))
      const body = JSON.parse(String(sent?.init?.body ?? '{}')) as Record<string, string>
      expect(body['modul']).toBe('PNCSearchKlaim')
      // Urutannya mengikuti daftar pilihan, bukan urutan mencentang.
      expect(body['sub_modul']).toBe('Penerimaan Pembayaran Klaim,Registrasi,')
    })
  })

  // Mengubah kuota di form Ubah mengirim ANGKA barunya, bukan teks dan bukan nilai lama.
  it('mengubah kuota mengirim angka barunya', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Edit masking CONTOH.ADMIN' }))

    const form = await screen.findByRole('form', { name: 'Ubah data masking' })
    const cari = within(form).getByLabelText('Max Cari Data') as HTMLInputElement

    // Nilai tersimpan terbaca apa adanya — bukan kosong, bukan NaN.
    expect(cari.value).toBe('100000')

    await user.clear(cari)
    await user.type(cari, '25')
    expect(cari.value).toBe('25')

    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.init?.method === 'PUT' && !c.url.endsWith('/status'))
      const body = JSON.parse(String(sent?.init?.body ?? '{}')) as Record<string, unknown>
      expect(body['maks_cari']).toBe(25)
      expect(body['maks_lihat']).toBe(100000)
    })
  })
})

describe('portal', () => {
  it('menuntun memilih portal alih-alih menampilkan galat', async () => {
    installDefaultFetch()
    useSelectedPortal.getState().clear()
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    // Permintaan daftar tidak pernah ditembakkan tanpa portal.
    expect(calls.some((c) => c.url.startsWith(ROUTE))).toBe(false)
  })
})
