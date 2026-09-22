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
 * defaultReply melayani daftar pilihan dan daftar sparepart; mutasi dijawab pemanggil.
 *
 * Jalur `/pilihan` diperiksa LEBIH DULU karena ia berawalan sama dengan jalur daftar —
 * persis alasan yang sama yang membuat rutenya didaftarkan lebih dulu di backend.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/sparepart/pilihan')) return { body: OPTIONS }

    if (call.method === 'GET') {
      const status = new URL(call.url, 'http://x').searchParams.get('status') ?? '1'

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
      'Harga Jual',
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

describe('tombol unggah', () => {
  /*
    Kedua tombolnya digambar DALAM KEADAAN MATI, bukan dihapus.

    Keduanya ada di layar Pega, dan Work Owner meminta layarnya "seperti aplikasi PEGA"
    (2026-09-20). Rule yang menjalankannya tidak ada di export (R-16), sehingga membuatnya
    berfungsi berarti mengarang bentuk berkas dan aturannya.
  */
  it('menggambar kedua tombol unggah Pega dalam keadaan mati', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(screen.getByRole('button', { name: 'Upload Document' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Upload Data Master Sparepart' })).toBeDisabled()
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

describe('keputusan borongan', () => {
  it('menggambar bilah keputusan hanya pada tab Waiting Approval', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    expect(
      screen.queryByRole('button', { name: 'Approve terpilih' }),
    ).not.toBeInTheDocument()

    await userEvent.click(tab('Waiting Approval'))
    expect(
      await screen.findByRole('button', { name: 'Approve terpilih' }),
    ).toBeInTheDocument()
  })

  // TANPA isian Catatan: tabelnya tidak punya kolom penampungnya. Menggambar isian yang
  // diam-diam membuang isinya lebih buruk daripada tidak menggambarnya.
  it('tidak menggambar isian catatan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(tab('Waiting Approval'))

    await screen.findByRole('button', { name: 'Approve terpilih' })
    expect(screen.queryByLabelText('Catatan')).not.toBeInTheDocument()
  })

  it('mematikan tombol keputusan selama belum ada yang dicentang', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(tab('Waiting Approval'))

    expect(await screen.findByRole('button', { name: 'Approve terpilih' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Reject terpilih' })).toBeDisabled()
  })

  it('mengirim satu permintaan untuk seluruh baris yang dicentang', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          jumlah_berubah: 1,
          status: '1',
          status_label: 'Approve',
          portal: 'ASM',
        },
      })),
    )
    show()

    await screen.findByText('FILTER OLI MESIN')
    await userEvent.click(tab('Waiting Approval'))

    await screen.findByText('SEAL KIT BOOM CYLINDER')
    await userEvent.click(screen.getByRole('checkbox'))
    await userEvent.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    await waitFor(() => {
      expect(calls.some((c) => c.url.endsWith('/keputusan'))).toBe(true)
    })

    const sent = calls.find((c) => c.url.endsWith('/keputusan'))?.body as Record<string, unknown>
    expect(sent.id_sparepart).toEqual(['SP0000000002'])
    expect(sent.status).toBe('1')
    // Badannya TIDAK memuat catatan.
    expect(sent).not.toHaveProperty('catatan')
  })
})
