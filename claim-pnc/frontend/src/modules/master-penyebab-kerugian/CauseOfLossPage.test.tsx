import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { HEADER_PORTAL } from '@/api/client'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/penyebab-kerugian'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Lima baris contoh yang DESKRIPSINYA dikarang.
 *
 * Isi `POOLDATA.M_CAUSE_OF_LOSS` yang sebenarnya belum pernah diterima dari DBA, dan tidak
 * ada satu pun CSV master untuk tabel ini. Yang diuji di berkas ini adalah PERILAKU layar,
 * bukan isi masternya.
 *
 * Yang TIDAK dikarang adalah bentuk ID-nya — kode situs `1` ditambah tiga digit, diturunkan
 * dari `Database/PEGA_M_CAUSE_OF_LOSS.prc:20`.
 *
 * Dua baris dipilih dengan sengaja, dan keduanya menguji perilaku nyata:
 *
 *   - Satu baris ber-deskripsi KOSONG, karena modul ini memang menerimanya (keputusan
 *     Work Owner 2026-09-20) sehingga baris seperti itu pasti akan muncul di daftar.
 *   - Sebagian baris punya ID LAMA dan sebagian tidak, karena kolom itu memang hanya
 *     melekat pada baris warisan.
 */
const SAMPLE = [
  { id: '1001', deskripsi: 'Contoh Golongan A', id_lama: '01' },
  { id: '1002', deskripsi: 'Contoh Golongan B', id_lama: '02' },
  { id: '1003', deskripsi: '', id_lama: '' },
  { id: '1009', deskripsi: 'Contoh Golongan I', id_lama: '' },
  { id: '1010', deskripsi: 'Contoh Golongan J', id_lama: '' },
]

/**
 * Bilah atas memuat pemilih portal, sehingga SETIAP layar di balik sesi ikut memanggil
 * /api/portal. Peladen tiruan menjawabnya otomatis supaya tiap uji di berkas ini tidak
 * perlu mengulang daftar yang sama.
 */
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
    // Kerangka layar memuat menunya sendiri sejak menu dibaca dari basis data. Ia dijawab
    // di sini supaya uji layar ini menguji layarnya, bukan jalur galat menu.
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(reply(url, init))
  })
}

/** Peladen tiruan yang menjawab daftar dan menerima simpan. */
function installDefaultFetch() {
  installFetch((url, init) => {
    if (url === ROUTES && init?.method === 'POST') {
      return jsonResponse(201, {
        penyebab_kerugian: { id: '1011', deskripsi: readDeskripsi(init), id_lama: '' },
        portal: 'ASM',
      })
    }
    if (url.startsWith(`${ROUTES}/`) && init?.method === 'PUT') {
      const id = url.slice(ROUTES.length + 1)
      return jsonResponse(200, {
        penyebab_kerugian: { id, deskripsi: readDeskripsi(init), id_lama: '01' },
        portal: 'ASM',
      })
    }
    return jsonResponse(200, {
      penyebab_kerugian: SAMPLE,
      total: SAMPLE.length,
      portal: 'ASM',
    })
  })
}

function readDeskripsi(init: RequestInit | undefined): string {
  return (JSON.parse(String(init?.body ?? '{}')) as { deskripsi?: string }).deskripsi ?? ''
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/penyebab-kerugian']}>
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
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('daftar', () => {
  it('menampilkan ID dan Deskripsi Kerugian dari server', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Contoh Golongan A')).toBeInTheDocument()
    expect(screen.getByText('Contoh Golongan J')).toBeInTheDocument()

    // Total datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/5 penyebab kerugian terdaftar/)).toBeInTheDocument()
  })

  /*
    Deskripsi kosong DITERIMA modul ini, sehingga baris seperti itu pasti muncul. Ia
    ditandai, bukan dibiarkan sebagai sel kosong yang terlihat seperti tabel rusak.
  */
  it('menandai baris yang deskripsinya kosong', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Golongan A')

    expect(screen.getByText('(tanpa deskripsi)')).toBeInTheDocument()
  })

  /*
    Grid Pega hanya memuat DUA kolom — `ID` dan `Deskripsi Kerugian`. Layar ini
    mengikutinya apa adanya, sehingga `OLD_M_COL_ID` TIDAK tampil meski ikut dikirim API.

    Uji ini menjaga supaya kolom itu tidak diam-diam muncul kembali: menambahkannya adalah
    keputusan, bukan perbaikan.
  */
  it('hanya menampilkan dua kolom data, tanpa ID lama', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Golongan A')

    expect(screen.getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Deskripsi Kerugian' })).toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: /ID lama/i })).not.toBeInTheDocument()
    // Nilainya pun tidak bocor ke layar lewat sel mana pun.
    expect(screen.queryByText('01')).not.toBeInTheDocument()
  })

  /*
    Tombolnya berbunyi **Refresh**: itulah `pyButtonLabel` yang terpasang di harness Pega,
    dan `D-13` menetapkan teks yang dilihat pengguna mengikuti layar lama.

    Ke-10 harness master di Pega seluruhnya berbunyi "Refresh", dan Work Owner memutuskan
    2026-09-20 agar seluruh modul master mengikutinya. Uji ini penjaga label itu di modul
    ini; modul lain belum punya uji setara.
  */
  it('memakai label tombol yang sama dengan Pega', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Golongan A')

    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeInTheDocument()
  })

  it('membawa token sesi di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Golongan A')

    const request = calls.find((p) => p.url === ROUTES)
    expect(request).toBeDefined()
    expect(request?.url).not.toContain('token')

    const header = request?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
  })

  /*
    Portal entitas WAJIB ikut di setiap permintaan. Tanpa header ini backend menolak
    permintaannya — dan itu memang yang diinginkan: jatuh diam-diam ke portal utama berarti
    menampilkan penyebab kerugian badan hukum lain tanpa satu pun tanda di layar (`R-20`).
  */
  it('membawa portal entitas di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Golongan A')

    const request = calls.find((p) => p.url === ROUTES)
    expect(request?.url).not.toContain('ASM')

    const header = request?.init?.headers as Record<string, string>
    expect(header[HEADER_PORTAL]).toBe('ASM')
  })

  it('menyebut entitas yang menjawab, bukan hanya yang diminta', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Golongan A')

    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('menyaring baris saat pengguna mengetik di kotak cari', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.type(screen.getByRole('searchbox'), 'golongan j')

    expect(screen.getByText('Contoh Golongan J')).toBeInTheDocument()
    expect(screen.queryByText('Contoh Golongan A')).not.toBeInTheDocument()
  })

  it('mencari juga pada kolom ID, bukan hanya deskripsi', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.type(screen.getByRole('searchbox'), '1010')

    expect(screen.getByText('Contoh Golongan J')).toBeInTheDocument()
    expect(screen.queryByText('Contoh Golongan B')).not.toBeInTheDocument()
  })

  it('menampilkan pesan yang dapat ditindaklanjuti saat pemuatan gagal', async () => {
    installFetch(() => jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' }))
    show()

    expect(await screen.findByRole('alert')).toHaveTextContent(/gagal dimuat/i)
  })
})

describe('tambah', () => {
  it('mengirim POST tanpa ID, lalu memuat ulang daftar', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = screen.getByRole('form', { name: /menambah data/i })
    // ID tidak dapat disunting: ia dibuat sistem, sama seperti di Pega.
    expect(within(form).getByText('Dibuat sistem')).toBeInTheDocument()

    await pengguna.type(within(form).getByLabelText('Deskripsi Kerugian'), 'Kebakaran')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'POST')
      expect(send).toBeDefined()
      expect(send?.url).toBe(ROUTES)
      expect(JSON.parse(String(send?.init?.body))).toEqual({ deskripsi: 'Kebakaran' })
    })

    // Daftar dimuat ulang dari server: nomor baru hanya diketahui server.
    await waitFor(() => {
      expect(calls.filter((p) => p.url === ROUTES && p.init?.method === 'GET').length).toBe(2)
    })
  })

  /*
    Ini kebalikan dari uji yang sama di Master Status Klaim, dan perbedaannya DISENGAJA.

    Layar Pega menerima deskripsi kosong — tidak ada `pyRequired` di
    `Section/BrowseCauseOfLoss-Section.xml` — dan Work Owner memutuskan perilaku itu
    dipertahankan pada 2026-09-20 sesuai `P-5`. Bila uji ini kelak gagal karena seseorang
    menambahkan `.min(1)` pada skemanya, yang perlu dibaca lebih dulu adalah keputusannya —
    bukan uji ini.
  */
  it('menerima deskripsi kosong dan tetap mengirimnya ke server', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'POST')
      expect(send).toBeDefined()
      expect(JSON.parse(String(send?.init?.body))).toEqual({ deskripsi: '' })
    })
  })

  /*
    Karena deskripsi ganda diterima, pengguna tidak akan pernah ditolak sistem — jadi
    satu-satunya cara ia tahu adalah diberi tahu. Peringatan itu menggantikan pemeriksaan
    yang sengaja tidak ada.
  */
  it('memperingatkan bahwa deskripsi ganda tidak ditolak sistem', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })

    expect(within(form).getByText(/boleh terdaftar lebih dari satu kali/i)).toBeInTheDocument()
  })

  it('menampilkan detail validasi per field dari server', async () => {
    installFetch((_url, init) => {
      if (init?.method === 'POST') {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [{ field: 'deskripsi', pesan: 'Deskripsi Kerugian paling panjang 100 karakter.' }],
        })
      }
      return jsonResponse(200, {
        penyebab_kerugian: SAMPLE,
        total: SAMPLE.length,
        portal: 'ASM',
      })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })
    await pengguna.type(within(form).getByLabelText('Deskripsi Kerugian'), 'Apa pun')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(/paling panjang 100/i)
  })

  /*
    Tabel situs yang kosong tidak dapat ditolong pengguna sama sekali — mengulang tidak akan
    menolong, dan pesannya harus mengatakan itu alih-alih menyuruh mencoba lagi.
  */
  it('menerangkan bahwa mengulang tidak menolong saat nomor tidak dapat dibentuk', async () => {
    installFetch((_url, init) => {
      if (init?.method === 'POST') {
        return jsonResponse(500, {
          kode: 'id_tidak_dapat_dibentuk',
          pesan: 'Nomor penyebab kerugian tidak dapat dibentuk.',
        })
      }
      return jsonResponse(200, {
        penyebab_kerugian: SAMPLE,
        total: SAMPLE.length,
        portal: 'ASM',
      })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })
    await pengguna.type(within(form).getByLabelText('Deskripsi Kerugian'), 'Golongan Baru')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(/tidak akan menolong/i)
    // Form tetap terbuka supaya isian pengguna tidak hilang.
    expect(within(form).getByLabelText('Deskripsi Kerugian')).toHaveValue('Golongan Baru')
  })
})

describe('ubah', () => {
  it('mengirim PUT dengan ID di jalur URL, bukan di badan permintaan', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.click(
      screen.getByRole('button', { name: /Ubah penyebab kerugian Contoh Golongan A/i }),
    )

    const form = screen.getByRole('form', { name: /memperbaharui data/i })
    // ID baris yang sedang diubah ditampilkan, dan tetap tidak dapat disunting.
    expect(within(form).getByText('1001')).toBeInTheDocument()

    const isian = within(form).getByLabelText('Deskripsi Kerugian')
    expect(isian).toHaveValue('Contoh Golongan A')

    await pengguna.clear(isian)
    await pengguna.type(isian, 'Kebakaran')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'PUT')
      expect(send).toBeDefined()
      expect(send?.url).toBe(`${ROUTES}/1001`)
      // ID TIDAK ikut di badan permintaan: menerimanya dari klien akan membuat satu
      // golongan dapat dipindahkan ke nomor lain, dan memutus rincian di bawahnya.
      expect(JSON.parse(String(send?.init?.body))).toEqual({ deskripsi: 'Kebakaran' })
    })
  })

  /*
    Form Pega memuat DUA isian berlabel "Deskripsi Kerugian" — satu terikat `COL_DESC`,
    satu lagi terikat `.Description`, keduanya berbagi `pyAutomationID` yang sama. Yang
    kedua mati: `SetCauseOfLossValue_act` tidak pernah mengisinya, dan view tidak punya
    kolom untuk membacanya kembali. Ia tidak dibawa, dan ketiadaannya dijaga di sini.
  */
  it('hanya menyediakan SATU isian yang dapat diketik', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    await pengguna.click(
      screen.getByRole('button', { name: /Ubah penyebab kerugian Contoh Golongan A/i }),
    )

    const form = screen.getByRole('form', { name: /memperbaharui data/i })
    expect(within(form).getAllByRole('textbox')).toHaveLength(1)
  })

  it('dapat mengubah baris yang deskripsinya kosong', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Golongan A')

    // Baris tanpa deskripsi tetap punya tombol Ubah; label aksesibilitasnya jatuh ke ID
    // supaya tombolnya tetap dapat disebut pengguna papan ketik dan pembaca layar.
    await pengguna.click(screen.getByRole('button', { name: /Ubah penyebab kerugian 1003/i }))

    const form = screen.getByRole('form', { name: /memperbaharui data/i })
    expect(within(form).getByLabelText('Deskripsi Kerugian')).toHaveValue('')
  })
})

describe('yang sengaja tidak ada', () => {
  /*
    Tidak ada tombol Hapus, dan ketiadaannya diuji — bukan sekadar dicatat di komentar.

    Layar Pega tidak punya tombolnya, `Database/PEGA_M_CAUSE_OF_LOSS.prc` hanya mengenal
    INSERT dan UPDATE, dan menghapus satu golongan akan membuat setiap baris
    `D_CAUSE_OF_LOSS` yang bernaung di bawahnya kehilangan induknya (`ADR-0012`).
  */
  it('tidak menyediakan tombol hapus', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Golongan A')

    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})
