import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SparepartPage } from './SparepartPage'

/**
 * Daftar Kategori dan Tipe, persis seperti yang dijawab endpoint `/pilihan`.
 *
 * Berbeda dari Master Panel, isinya DIBACA DARI BASIS DATA entitas — bukan konstanta yang
 * ditanam di activity Pega. Distribusinya sengaja tidak merata supaya penyempitan Tipe
 * menurut Kategori benar-benar terlihat bekerja.
 */
const OPTIONS = {
  kategori: [
    { kode: 'KAT01', nama: 'ENGINE' },
    { kode: 'KAT02', nama: 'HYDRAULIC' },
  ],
  tipe: [
    { kode: 'TIP01', nama: 'FILTER', kode_kategori: 'KAT01' },
    { kode: 'TIP02', nama: 'PISTON', kode_kategori: 'KAT01' },
    { kode: 'TIP03', nama: 'SEAL KIT', kode_kategori: 'KAT02' },
  ],
  portal: 'ASM',

  // Jalur unggah dinyatakan siap supaya isian berkas pada form tergambar hidup. Keadaan
  // sebaliknya diuji terpisah — lihat "isian unggah mati saat layanannya belum terpasang".
  unggah_tersedia: true,
}

/** Sparepart yang sudah disetujui, seluruh kolomnya terisi. */
const APPROVED = {
  id_sparepart: 'SP0000000001',
  nama_sparepart: 'FILTER OLI MESIN',
  nomor_sparepart: '1R-0716',
  kode_sparepart: 'FLT-ENG-001',
  harga_jual: '1250000',
  kategori_sparepart: 'KAT01',
  nama_kategori_sparepart: 'ENGINE',
  tipe_sparepart: 'TIP01',
  nama_tipe_sparepart: 'FILTER',
  berat: '850',
  panjang: '12',
  lebar: '12',
  tinggi: '18',
  stock_minimal: '5',
  stock_maximal: '40',
  kuantitas_pesanan: '10',
  tanggal_produksi: '2025-11-04',
  part_substitusi: 'FILTER OLI MESIN ALT',
  jenis_sparepart: 'ORIGINAL',
  satuan: 'PCS',
  status_aktif: '1',
  status_sparepart: 'READY',
  user_update: 'INTANHENNY',
  tanggal_update_harga: '2026-09-12T03:15:00Z',
  id_dokumen: 'DOC-0001',
  status: '1',
  status_label: 'Approve',
}

/** Sparepart yang menunggu keputusan, keempat penandanya KOSONG. */
const PENDING = {
  ...APPROVED,
  id_sparepart: 'SP0000000002',
  nama_sparepart: 'SEAL KIT BOOM CYLINDER',
  nomor_sparepart: '707-99-45600',
  kode_sparepart: 'SKT-HYD-014',
  kategori_sparepart: 'KAT02',
  nama_kategori_sparepart: 'HYDRAULIC',
  tipe_sparepart: 'TIP03',
  nama_tipe_sparepart: 'SEAL KIT',
  jenis_sparepart: '',
  satuan: '',
  status_aktif: '',
  status_sparepart: '',
  status: '0',
  status_label: 'Waiting Approval',
}

/**
 * Sparepart yang ditolak, TANPA stempel tanggal harga sama sekali.
 *
 * Keadaan yang SAH — kolomnya belum pernah terisi — dan yang paling mudah terlupa diuji
 * karena baris pertama selalu mengisinya.
 */
const REJECTED = {
  ...APPROVED,
  id_sparepart: 'SP0000000003',
  nama_sparepart: 'TRACK LINK ASSY',
  nomor_sparepart: '20Y-32-00203',
  kode_sparepart: 'TRK-UND-007',
  user_update: '',
  tanggal_update_harga: undefined,
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
      // FormData dibiarkan apa adanya: ia badan permintaan unggah, dan JSON.parse atasnya
      // MELEMPAR — kegagalan yang muncul sebagai "unggahan tidak pernah terjadi", bukan
      // sebagai galat yang menyebut sebabnya.
      body:
        init?.body instanceof FormData
          ? init.body
          : init?.body
            ? JSON.parse(init.body as string)
            : undefined,
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
 * defaultReply melayani daftar pilihan dan daftar sparepart; mutasi dijawab pemanggil.
 *
 * Jalur `/pilihan` diperiksa LEBIH DULU karena ia berawalan sama dengan jalur daftar —
 * persis alasan yang sama yang membuat rutenya didaftarkan lebih dulu di backend.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/sparepart/pilihan')) return { body: OPTIONS }

    /*
      Permintaan dokumen dijawab 404 `dokumen_belum_ada`, dan itu jawaban yang WAJAR —
      bukan kegagalan. Baris contoh memang belum punya lampiran, dan backend pun menjawab
      begitu. Dipisahkan lebih dulu karena jalur daftar di bawah akan menangkapnya dan
      menjawabnya dengan bentuk daftar yang sama sekali bukan dokumen.
    */
    if (call.url.includes('/dokumen')) {
      if (call.method === 'GET') {
        return { body: { error: { code: 'dokumen_belum_ada' } }, status: 404 }
      }
      if (mutation) return mutation(call)
    }

    if (call.method === 'GET') {
      const status = new URL(call.url, 'https://x').searchParams.get('status') ?? '1'

      let rows: unknown[] = []
      if (status === '1') rows = [APPROVED]
      if (status === '0') rows = [PENDING]
      if (status === '2') rows = [REJECTED]

      return { body: { sparepart: rows, status, portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { sparepart: APPROVED, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <SparepartPage />
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

/** listCalls mengambil permintaan daftar saja, membuang permintaan daftar pilihan. */
function listCalls() {
  return calls.filter((c) => c.method === 'GET' && c.url.startsWith('/api/master/sparepart?'))
}

/**
 * tab mengambil tombol tab menurut captionnya.
 *
 * Dicari DI DALAM bilah tab, bukan di seluruh halaman: caption tabnya — "Approve" dan
 * "Reject" — adalah caption Pega yang memang ditiru (D-13), dan kata yang sama muncul pada
 * tombol keputusan.
 */
function tab(name: string) {
  return within(screen.getByRole('navigation', { name: 'Tab Master Sparepart' })).getByRole(
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

describe('daftar master sparepart', () => {
  // Grid Pega hanya menampilkan LIMA kolom dari dua puluh tiga; sisanya hanya terlihat saat
  // baris dibuka. Susunan itu ditiru apa adanya atas keputusan Work Owner (2026-09-20).
  it('menampilkan judul dan kelima kolom grid Pega', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Sparepart HE' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const header of [
      'ID Sparepart',
      'Nama Sparepart',
      'Harga (Rp)',
      'User Update',
      'Tanggal Update',
    ]) {
      expect(within(table).getByRole('columnheader', { name: header })).toBeInTheDocument()
    }
  })

  // Kolom yang TIDAK ada di grid Pega tidak digambar — termasuk sel sisa salin-tempel yang
  // di layar lama terikat pada `.TELP_BENGKEL` dan karenanya selalu kosong.
  it('tidak menggambar kolom di luar kelima kolom Pega', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    for (const absent of ['Nomor Sparepart', 'Kode Sparepart', 'Kategori Sparepart', 'Satuan']) {
      expect(within(table).queryByRole('columnheader', { name: absent })).not.toBeInTheDocument()
    }
  })

  it('membuka tab Approve lebih dulu', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(listCalls()[0]?.url).toContain('status=1')
  })

  // Harga jual ditampilkan berpemisah ribuan, tetapi yang DIKIRIM ke server tetap teksnya.
  it('menampilkan harga jual dengan pemisah ribuan', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('1.250.000')).toBeInTheDocument()
  })

  // Stempel UTC dari server ditampilkan dalam waktu Jakarta, lewat Intl — tidak ada satu pun
  // penambahan 7 jam manual (F-5).
  it('menampilkan tanggal update dalam waktu Jakarta', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    // 2026-09-12T03:15:00Z = 10.15 WIB.
    expect(within(table).getByText(/10[.:]15/)).toBeInTheDocument()
  })

  it('menampilkan baris tanpa stempel tanggal sebagai tanda pisah', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(tab('Reject'))

    await screen.findByText('TRACK LINK ASSY')
    const table = screen.getByRole('table')
    expect(within(table).getAllByText('—').length).toBeGreaterThan(0)
  })

  it('menyebut portal entitas yang menjawab', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(screen.getByText('ASM')).toBeInTheDocument()
  })

  // Setiap tab menembak endpoint yang sama dengan penyaring yang berbeda.
  it('mengganti penyaring saat berpindah tab', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')

    await userEvent.click(tab('Waiting Approval'))
    await waitFor(() => {
      expect(listCalls().some((c) => c.url.includes('status=0'))).toBe(true)
    })

    await userEvent.click(tab('Reject'))
    await waitFor(() => {
      expect(listCalls().some((c) => c.url.includes('status=2'))).toBe(true)
    })
  })

  // Portal ikut pada SETIAP permintaan; tanpa itu server tidak tahu basis data mana yang
  // harus menjawab (ADR-0030, R-20).
  it('mengirim portal pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    for (const call of calls) {
      expect(call.header['X-Portal'] ?? call.header['x-portal']).toBe('ASM')
    }
  })
})

describe('grid saat tidak ada baris', () => {
  /*
    Grid Pega menggambar kepala kolomnya beserta pyGridNoResultsMessage di bawahnya saat
    hasilnya nol. Layar ini mengikutinya: kolom tetap terlihat, keterangannya di dalam grid.
  */
  it('tetap menggambar kelima kolom saat daftarnya kosong', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/sparepart/pilihan')) return { body: OPTIONS }
      return { body: { sparepart: [], status: '1', portal: 'ASM' } }
    })
    show()

    const table = await screen.findByRole('table')
    for (const header of [
      'ID Sparepart',
      'Nama Sparepart',
      'Harga (Rp)',
      'User Update',
      'Tanggal Update',
    ]) {
      expect(within(table).getByRole('columnheader', { name: header })).toBeInTheDocument()
    }
    expect(screen.getByText('Data tidak ada')).toBeInTheDocument()
  })

  /*
    Gagal dan kosong memakai kalimat yang SAMA — keputusan Work Owner (2026-09-24).

    Layar karena itu tidak dapat dipakai membedakan "tabelnya memang kosong" dari "tabelnya
    gagal dibaca"; yang membedakannya adalah log backend dan `claimpnc -periksa`. Uji ini
    mengunci keputusan itu supaya perubahannya kelak disengaja, bukan tergelincir.
  */
  it('memakai kalimat yang sama saat pemuatan gagal', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/sparepart/pilihan')) return { body: OPTIONS }
      return { body: { kode: 'kesalahan_sistem', pesan: 'gagal' }, status: 500 }
    })
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'ID Sparepart' })).toBeInTheDocument()
    expect(screen.getByText('Data tidak ada')).toBeInTheDocument()
  })

  // Kedua tombol unggah layar Pega digambar, dan keduanya dipagari di sini: layar ini sudah
  // sekali kehilangan salah satunya, dan kehilangan itu tidak menggagalkan satu uji pun.
  it('menggambar kedua tombol unggah', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(screen.getByRole('button', { name: 'Upload Document' })).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Upload Data Master Sparepart' }),
    ).toBeInTheDocument()
  })

  /*
    SATU panel terbuka pada satu waktu.

    Keduanya menempati tempat yang sama di bawah kepala halaman. Membuka keduanya sekaligus
    pernah benar-benar terjadi di Master Sparepart saat masing-masing memegang keadaan
    sendiri — layarnya memanjang dan dua judul muncul berdampingan tanpa satu pun menjelaskan
    yang mana yang sedang dipakai.
  */
  it('menutup panel dokumen saat panel CSV dibuka', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')

    await userEvent.click(screen.getByRole('button', { name: 'Upload Document' }))
    expect(await screen.findByRole('heading', { name: 'Upload Document' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Upload Data Master Sparepart' }))
    expect(
      await screen.findByRole('heading', { name: 'Upload Data Master Sparepart' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Upload Document' })).not.toBeInTheDocument()
  })

  // Tombolnya menutup panelnya sendiri saat ditekan ulang — sama seperti Master Panel.
  it('menutup panel CSV saat tombolnya ditekan ulang', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    const tombol = screen.getByRole('button', { name: 'Upload Data Master Sparepart' })

    await userEvent.click(tombol)
    expect(
      await screen.findByRole('heading', { name: 'Upload Data Master Sparepart' }),
    ).toBeInTheDocument()

    await userEvent.click(tombol)
    expect(
      screen.queryByRole('heading', { name: 'Upload Data Master Sparepart' }),
    ).not.toBeInTheDocument()
  })
})

describe('form master sparepart', () => {
  it('membuka form tambah dengan isian kosong', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(
      await screen.findByRole('heading', { name: 'Tambah Master Sparepart' }),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Nomor Sparepart')).toHaveValue('')
  })

  // ID, user, dan tanggal update ditampilkan sebagai KETERANGAN, bukan isian: ketiganya
  // diterbitkan server.
  it('menampilkan ID dan pelaku sebagai keterangan pada mode ubah', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah' }))

    await screen.findByRole('heading', { name: 'Ubah FILTER OLI MESIN' })

    // Dicari DI DALAM form: kolom ID pada grid memuat teks yang sama.
    const form = screen.getByRole('form', { name: 'Ubah FILTER OLI MESIN' })
    expect(within(form).getByText('SP0000000001')).toBeInTheDocument()
    expect(within(form).getByText('INTANHENNY')).toBeInTheDocument()
    expect(within(form).queryByLabelText('ID Sparepart')).not.toBeInTheDocument()
  })

  it('menolak simpan saat keempat isian wajib kosong', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nomor sparepart wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Nama sparepart wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Kode sparepart wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Harga jual wajib diisi.')).toBeInTheDocument()

    // Tidak satu pun permintaan tulis terkirim.
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
  })

  it('menolak harga jual yang bukan angka', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), '20Y-70-21120')
    await userEvent.type(screen.getByLabelText('Nama Sparepart'), 'BUCKET PIN')
    await userEvent.type(screen.getByLabelText('Kode Sparepart'), 'BKT-020')
    await userEvent.type(screen.getByLabelText('Harga Jual (Rp)'), 'seribu')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText('Harga jual harus berupa angka, misalnya 1250000 atau 1250000,50.'),
    ).toBeInTheDocument()
  })

  // Stok minimal tidak boleh melampaui stok maksimal — SELISIH YANG DIRENCANAKAN terhadap
  // sistem lama, yang tidak memeriksanya sama sekali.
  it('menolak stock minimal yang melebihi stock maximal', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), '20Y-70-21120')
    await userEvent.type(screen.getByLabelText('Nama Sparepart'), 'BUCKET PIN')
    await userEvent.type(screen.getByLabelText('Kode Sparepart'), 'BKT-020')
    await userEvent.type(screen.getByLabelText('Harga Jual (Rp)'), '2500000')
    await userEvent.type(screen.getByLabelText('Stock Minimal'), '41')
    await userEvent.type(screen.getByLabelText('Stock Maximal'), '40')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText('Stock minimal tidak boleh melebihi stock maximal.'),
    ).toBeInTheDocument()
  })

  // Tipe BERCABANG dari Kategori; daftarnya dipersempit begitu kategori dipilih.
  it('mempersempit daftar Tipe menurut Kategori yang dipilih', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const kategori = await screen.findByLabelText('Kategori Sparepart')
    const tipe = screen.getByLabelText('Tipe Sparepart')

    await userEvent.selectOptions(kategori, 'KAT01')
    await waitFor(() => {
      expect(within(tipe).getByRole('option', { name: 'FILTER' })).toBeInTheDocument()
    })
    expect(within(tipe).queryByRole('option', { name: 'SEAL KIT' })).not.toBeInTheDocument()

    await userEvent.selectOptions(kategori, 'KAT02')
    await waitFor(() => {
      expect(within(tipe).getByRole('option', { name: 'SEAL KIT' })).toBeInTheDocument()
    })
    expect(within(tipe).queryByRole('option', { name: 'FILTER' })).not.toBeInTheDocument()
  })

  /*
    Keempat penanda menawarkan NILAI YANG SUDAH DIPAKAI baris lain, bukan daftar yang
    dikarang — daftar pilihan aslinya tidak ada di export (R-16).
  */
  it('menawarkan nilai penanda yang sudah dipakai baris lain', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah' }))

    const satuan = await screen.findByLabelText('Satuan')
    expect(within(satuan).getByRole('option', { name: 'PCS' })).toBeInTheDocument()
    // Jalan keluarnya selalu tersedia, supaya nilai baru tetap dapat diketik.
    expect(within(satuan).getByRole('option', { name: 'Lainnya…' })).toBeInTheDocument()
  })

  it('mengirim isian yang benar saat menambah', async () => {
    installFetch(defaultReply(() => ({ body: { sparepart: APPROVED, portal: 'ASM' }, status: 201 })))
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), '20Y-70-21120')
    await userEvent.type(screen.getByLabelText('Nama Sparepart'), 'BUCKET PIN')
    await userEvent.type(screen.getByLabelText('Kode Sparepart'), 'BKT-020')
    await userEvent.type(screen.getByLabelText('Harga Jual (Rp)'), '2500000')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.method === 'POST')).toBe(true)
    })

    const sent = calls.find((c) => c.method === 'POST')?.body as Record<string, unknown>
    expect(sent.nomor_sparepart).toBe('20Y-70-21120')
    expect(sent.nama_sparepart).toBe('BUCKET PIN')
    // Harga dikirim sebagai TEKS, bukan angka JSON (I-12, D-51).
    expect(sent.harga_jual).toBe('2500000')
    // Kelima kolom yang diturunkan server tidak ikut dikirim.
    for (const absent of [
      'id_sparepart',
      'status',
      'user_update',
      'tanggal_update_harga',
      'id_dokumen',
    ]) {
      expect(sent).not.toHaveProperty(absent)
    }
  })

  it('menutup form hanya setelah server menjawab berhasil', async () => {
    installFetch(
      defaultReply(() => ({
        body: { kode: 'kunci_sparepart_sudah_ada', pesan: 'Sudah dipakai.' },
        status: 409,
      })),
    )
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), '1R-0716')
    await userEvent.type(screen.getByLabelText('Nama Sparepart'), 'BUCKET PIN')
    await userEvent.type(screen.getByLabelText('Kode Sparepart'), 'BKT-020')
    await userEvent.type(screen.getByLabelText('Harga Jual (Rp)'), '2500000')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Sparepart dengan kunci itu sudah ada')).toBeInTheDocument()
    // Formnya masih terbuka, dan isian pengguna tidak hilang.
    expect(screen.getByLabelText('Nama Sparepart')).toHaveValue('BUCKET PIN')
  })

  // Pelanggaran dari server disorot pada isiannya masing-masing, bukan hanya diringkas.
  it('menyorot isian yang dilaporkan server', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar.',
          detail: [{ kolom: 'kode_sparepart', pesan: 'Kode sudah dipakai sparepart lain.' }],
        },
        status: 422,
      })),
    )
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), '20Y-70-21120')
    await userEvent.type(screen.getByLabelText('Nama Sparepart'), 'BUCKET PIN')
    await userEvent.type(screen.getByLabelText('Kode Sparepart'), 'BKT-020')
    await userEvent.type(screen.getByLabelText('Harga Jual (Rp)'), '2500000')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Kode sudah dipakai sparepart lain.')).toBeInTheDocument()
  })
})

describe('ketiga tab berperilaku sama, seperti Pega', () => {
  /*
    Ketiga section tab Pega hanya memuat dua tombol — SIMPAN dan Ubah — dan NOL pxCheckbox
    serta NOL pySelected. Centang, Select All, Approve, dan Reject hanya ada di
    `ApprovalMasterSparepartHE`, yakni layar Inbox Manager.

    Uji ini mengunci pencabutan bilah keputusan (Work Owner, 2026-10-03) supaya ia tidak
    kembali tanpa disengaja.
  */
  it('tidak menggambar centang maupun tombol keputusan di tab Waiting Approval', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(tab('Waiting Approval'))

    await screen.findByText('SEAL KIT BOOM CYLINDER')
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Approve terpilih' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Reject terpilih' })).not.toBeInTheDocument()
  })

  it('menggambar kolom yang sama persis di ketiga tab', async () => {
    installFetch(defaultReply())
    show()

    for (const name of ['Approve', 'Reject', 'Waiting Approval']) {
      await userEvent.click(tab(name))

      const table = await screen.findByRole('table')
      const header = within(table)
        .getAllByRole('columnheader')
        .map((cell) => cell.textContent?.replace(/\s+/g, ' ').trim())

      expect(header).toEqual([
        'ID Sparepart',
        'Nama Sparepart',
        'Harga (Rp)',
        'User Update',
        'Tanggal Update',
        'Aksi',
      ])
    }
  })
})

describe('unggah dokumen master sparepart', () => {
  /*
    Jalur penyimpanannya SUDAH ADA di aplikasi ini — modul `dokumenpenunjang`, yang meniru
    rantai `UploadDocumentToGoogleStorage` → `InsertDokumenPNC` sampai ujungnya, termasuk
    konvensi nama `yyMdhm-sS-<jenis>-<nama>`. Modul ini menyambung ke sana, tidak
    membangunnya ulang.
  */
  it('panel tertutup sampai tombolnya ditekan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(screen.queryByLabelText('Berkas dokumen')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Upload Document' }))
    expect(await screen.findByLabelText('Berkas dokumen')).toBeEnabled()
  })

  // Dinyatakan SEBELUM pengguna memilih berkas, bukan sesudah ia memilihnya.
  it('tombol mati saat layanan penyimpanan belum terpasang', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/sparepart/pilihan')) {
        return { body: { ...OPTIONS, unggah_tersedia: false } }
      }
      return defaultReply()(call)
    })
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(screen.getByRole('button', { name: 'Upload Document' })).toBeDisabled()
  })

  /*
    Bentuknya mengikuti modal Pega: pemilih berkas sendiri (widget bawaan disembunyikan
    karena teks "No file chosen" pun membukanya), daftar "Nama File" beserta baris
    kosongnya, lalu Batal dan Submit.
  */
  it('berbentuk seperti modal Pega', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Upload Document' }))

    expect(await screen.findByLabelText('Berkas dokumen')).toHaveClass('sr-only')
    expect(screen.getByRole('button', { name: 'Pilih Berkas' })).toBeEnabled()
    expect(screen.getByText('Nama File')).toBeInTheDocument()
    expect(screen.getByText('Data tidak ada')).toBeInTheDocument()
    expect(screen.getByText('Maksimal 20 MB.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Batal' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Submit' })).toBeInTheDocument()
  })

  /*
    Urutannya yang diuji, bukan sekadar permintaannya: sparepart disimpan LEBIH DULU,
    dokumennya menyusul ke ID yang baru lahir. Pada jalur Tambah urutan itu satu-satunya
    yang mungkin — ID diterbitkan server.

    Submit pada panel hanya MENUTUPNYA; berkasnya tetap tertahan sampai Simpan ditekan,
    persis seperti `SaveFilePenunjang` yang menitipkan berkas ke halaman sementara.
  */
  it('mengunggah SESUDAH sparepart tersimpan, ke ID yang diterbitkan server', async () => {
    installFetch(
      defaultReply((call) => {
        if (call.url.includes('/dokumen')) {
          return { body: { data: { data_id: '20260000000001' } }, status: 201 }
        }
        return {
          body: { sparepart: { ...APPROVED, id_sparepart: 'SP0000000009' } },
          status: 201,
        }
      }),
    )
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Upload Document' }))
    await userEvent.upload(
      await screen.findByLabelText('Berkas dokumen'),
      new File(['isi'], 'faktur.pdf', { type: 'application/pdf' }),
    )
    await userEvent.click(screen.getByRole('button', { name: 'Submit' }))

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('heading', { name: 'Tambah Master Sparepart' })
    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'NS-9')
    await userEvent.type(screen.getByLabelText('Nama Sparepart'), 'SEAL')
    await userEvent.type(screen.getByLabelText('Kode Sparepart'), 'KD-9')
    await userEvent.type(screen.getByLabelText('Harga Jual (Rp)'), '2500000')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const unggah = calls.filter((c) => c.url.includes('/dokumen') && c.method === 'POST')
      expect(unggah).toHaveLength(1)
      expect(unggah.at(0)?.url).toContain('SP0000000009')
    })

    const simpan = calls.findIndex((c) => c.method === 'POST' && !c.url.includes('/dokumen'))
    const unggah = calls.findIndex((c) => c.method === 'POST' && c.url.includes('/dokumen'))
    expect(simpan).toBeLessThan(unggah)
  })

  // Tanpa berkas, tidak boleh ada permintaan unggah sama sekali.
  it('tidak menembak endpoint dokumen bila tidak ada berkas dipilih', async () => {
    installFetch(defaultReply(() => ({ body: { sparepart: APPROVED }, status: 201 })))
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('heading', { name: 'Tambah Master Sparepart' })

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'NS-9')
    await userEvent.type(screen.getByLabelText('Nama Sparepart'), 'SEAL')
    await userEvent.type(screen.getByLabelText('Kode Sparepart'), 'KD-9')
    await userEvent.type(screen.getByLabelText('Harga Jual (Rp)'), '2500000')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.method === 'POST' && !c.url.includes('/dokumen'))).toBe(true)
    })
    expect(calls.filter((c) => c.url.includes('/dokumen') && c.method === 'POST')).toHaveLength(0)
  })
})

/*
  Unggah CSV — "Upload Data Master Sparepart".

  Jalurnya dipagari terpisah dari unggah dokumen karena perilakunya BERBEDA pada satu hal
  yang menentukan: berkas CSV dikirim SEKETIKA, sedangkan berkas dokumen ditahan sampai
  Simpan. Menguji keduanya dalam satu blok membuat perbedaan itu mudah tertukar.
*/
describe('unggah CSV master sparepart', () => {
  const CSV_REPLY = {
    data: {
      total: 2,
      baru: 1,
      diperbarui: 0,
      gagal: 1,
      baris: [
        {
          baris: 2,
          nomor_sparepart: 'NS-9',
          id_sparepart: 'SP0000000009',
          hasil: 'baru',
          pesan: '',
        },
        {
          baris: 3,
          nomor_sparepart: '',
          id_sparepart: '',
          hasil: 'gagal',
          pesan: 'Nomor sparepart wajib diisi.',
        },
      ],
    },
  }

  function csvReply(status = 200, body?: unknown) {
    return defaultReply((call) => {
      if (call.url.includes('/unggah-csv')) return { status, body: body ?? CSV_REPLY }
      return { body: { sparepart: APPROVED, portal: 'ASM' }, status: 200 }
    })
  }

  function csv() {
    return new File(['NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n'], 'sparepart.csv', {
      type: 'text/csv',
    })
  }

  // Berbeda dari dokumen: berkas CSV DIKIRIM SEKETIKA, tanpa menunggu Simpan.
  it('mengirim berkas seketika ke jalurnya sendiri', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByText('FILTER OLI MESIN')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Sparepart' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    await waitFor(() =>
      expect(calls.filter((c) => c.url.endsWith('/unggah-csv'))).toHaveLength(1),
    )

    const sent = calls.find((c) => c.url.endsWith('/unggah-csv'))
    expect(sent).toBeDefined()
    expect(sent!.body).toBeInstanceOf(FormData)
    expect(((sent!.body as FormData).get('berkas') as File).name).toBe('sparepart.csv')

    // Jalur CSV TIDAK menyentuh endpoint dokumen — keduanya terpisah.
    expect(calls.filter((c) => c.url.includes('/dokumen') && c.method === 'POST')).toHaveLength(0)
  })

  // Laporan per baris: tanpa nomor barisnya, "1 baris gagal" memaksa pengguna menebak yang
  // mana di antara tiga ratus.
  it('menampilkan laporan per baris beserta nomor dan sebabnya', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByText('FILTER OLI MESIN')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Sparepart' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByText(/baris dibaca/)).toBeInTheDocument()
    expect(screen.getByText('Nomor sparepart wajib diisi.')).toBeInTheDocument()
    // Baris yang berhasil TETAP tersimpan — tanpa kalimat ini pengguna akan mengunggah ulang
    // seluruh berkas dan menimpa baris yang sudah benar.
    expect(screen.getByText(/tetap tersimpan/)).toBeInTheDocument()
  })

  it('menutup panel lewat Batal tanpa mengirim apa pun', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByText('FILTER OLI MESIN')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Sparepart' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByLabelText('Berkas CSV')).not.toBeInTheDocument()
    expect(calls.filter((c) => c.url.includes('/unggah-csv'))).toHaveLength(0)
  })

  // Berkas yang tidak terbaca sama sekali dibedakan dari baris yang gagal: di sini TIDAK ada
  // satu baris pun yang masuk, sehingga yang perlu diperbaiki adalah berkasnya.
  it('membedakan berkas yang tidak terbaca dari baris yang gagal', async () => {
    installFetch(
      csvReply(422, {
        kode: 'csv_tidak_sah',
        pesan: 'Header berkas tidak lengkap. Kolom yang kurang: KODE_SPART.',
      }),
    )
    show()

    const user = userEvent.setup()
    await screen.findByText('FILTER OLI MESIN')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Sparepart' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByText('Berkas tidak dapat dibaca')).toBeInTheDocument()
    expect(screen.queryByText(/baris dibaca/)).not.toBeInTheDocument()
  })

  /*
    Tombolnya TIDAK mati saat layanan penyimpanan dokumen belum terpasang.

    `unggah_tersedia` berbicara tentang layanan dokumen; unggah CSV menulis langsung ke tabel
    sparepart dan tidak menyentuh layanan itu sama sekali. Mematikannya bersama-sama adalah
    kesalahan yang mudah dibuat — satu penanda dipakai untuk dua hal yang tidak berhubungan.
  */
  it('tetap hidup meski layanan dokumen belum terpasang', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/sparepart/pilihan')) {
        return { body: { ...OPTIONS, unggah_tersedia: false } }
      }
      return defaultReply()(call)
    })
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(screen.getByRole('button', { name: 'Upload Document' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Upload Data Master Sparepart' })).toBeEnabled()
  })
})
