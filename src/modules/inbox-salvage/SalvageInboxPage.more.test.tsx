import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { NetworkError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DetailSalvagePanel } from './DetailSalvagePanel'
import { SalvageInboxPage } from './SalvageInboxPage'
import { TambahSalvageForm } from './TambahSalvageForm'
import type { ListResponse, MetadataResponse, SalvageRow, Tab } from './types'

/**
 * Uji tambahan Inbox Salvage: galat, ekspor, form Tambah, dan unggahan detail.
 * Seluruh nilai KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-salvage'

const TAB: Tab = {
  kode: 'histori',
  nama: 'Histori',
  keterangan: 'Seluruh pengajuan.',
  kolom: [
    { kunci: 'tanggal_input', judul: 'Tanggal Input', angka: false },
    { kunci: 'no_klaim', judul: 'No Klaim', angka: false },
    { kunci: 'nilai_pengajuan_pic', judul: 'Nilai Pengajuan PIC', angka: true },
  ],
  label_pencarian: 'CARI SALVAGE',
  pencarian_cocok_persis: false,
  kunci_rincian: 'pengajuan',
  membuka_form_sunting: false,
}

const META: MetadataResponse = {
  daftar: [TAB],
  daftar_bawaan: 'histori',
  pilihan_status_salvage: [{ kode: '2', label: 'Salvage DiTerima' }],
  kolom_berkas_unggahan: ['item', 'quantity'],
  pilihan_mata_uang: [{ kode: 'IDR', label: 'IDR' }],
  portal: 'ASM',
}

function row(over: Partial<SalvageRow> = {}): SalvageRow {
  return {
    referensi: '201',
    no_klaim: 'PNC-1',
    id_salvage: '201',
    tanggal_input: '2026-09-09',
    tanggal_kejadian: '',
    pic: '',
    nama_bisnis: '',
    nama_objek: '',
    jenis_salvage: '',
    lokasi_salvage: '',
    status_lelang: '',
    nilai_pengajuan_pic: '1500000',
    nilai_request_balai_lelang: '',
    email: '',
    keterangan_pic: '',
    tipe_pengajuan: '',
    aging: '',
    catatan: '',
    ...over,
  }
}

function list(rows: SalvageRow[], over: Partial<ListResponse> = {}): ListResponse {
  return {
    daftar: TAB,
    baris: rows,
    paginasi: { halaman: 1, ukuran: 20, total: rows.length, total_halaman: 1 },
    cari: '',
    portal: 'ASM',
    ...over,
  }
}

type Call = { url: string; init: RequestInit | undefined }
let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

type Answer = Response | 'fail' | 'hold' | null

function stubFetch(answer: (url: string, init?: RequestInit) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    const reply = answer(url, init)
    if (reply === 'fail') return Promise.reject(new TypeError('Failed to fetch'))
    if (reply === 'hold') return new Promise(() => {})
    if (reply) return Promise.resolve(reply)
    if (url === `${PATH}/daftar`) return Promise.resolve(jsonResponse(200, META))
    if (url === `${PATH}/ringkas`) return Promise.resolve(jsonResponse(200, { baris: [], portal: 'ASM' }))
    return Promise.resolve(jsonResponse(200, list([row()])))
  })
}

function wrap(children: ReactNode, entry = '/inbox-salvage') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[entry]}>{children}</MemoryRouter>
    </QueryClientProvider>,
  )
}

let clicked: string[] = []

/**
 * Mengisi SELURUH isian wajib form Tambah; tanpa itu peramban menahan Submit.
 *
 * Termasuk kedua daftar pilihan: keduanya wajib begitu masternya terbaca, dan
 * melewatkannya membuat Submit tertahan peramban tanpa satu pun pesan yang terlihat uji.
 */
async function fillRequired(user: ReturnType<typeof userEvent.setup>, scope: HTMLElement) {
  for (const label of [/Nomor Klaim/, /Nama Object/, /Nama Coverage/, /Jenis Salvage/]) {
    const field = within(scope).getByLabelText(label) as HTMLInputElement
    if (field.value === '' && !field.readOnly) await user.type(field, 'x')
  }

  for (const [label, value] of [
    ['Mata Uang', 'IDR'],
    ['Status Salvage', '2'],
  ] as const) {
    const pilihan = within(scope).queryByLabelText(label) as HTMLSelectElement | null
    if (pilihan !== null && pilihan.value === '') await user.selectOptions(pilihan, value)
  }
}

beforeEach(() => {
  calls = []
  clicked = []
  Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() })
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push(this.download)
  })
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('SalvageInboxPage — keadaan layar', () => {
  it('menuntun memilih entitas tanpa menembak server', () => {
    stubFetch(() => null)
    useSelectedPortal.getState().clear()
    wrap(<SalvageInboxPage />)

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('menyatakan layar tidak dapat dibuka saat bentuknya gagal dimuat', async () => {
    stubFetch((url) =>
      url === `${PATH}/daftar` ? jsonResponse(500, { kode: 'x', pesan: 'Bentuk rusak.' }) : null,
    )
    wrap(<SalvageInboxPage />)

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Bentuk rusak.')).toBeInTheDocument()
  })

  it('memakai pesan jaringan saat daftar tidak dapat dimuat', async () => {
    stubFetch((url) => (url.startsWith(`${PATH}?`) ? 'fail' : null))
    wrap(<SalvageInboxPage />)

    expect(await screen.findByText('Daftar tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
  })

  it('menyebut ringkasan sedang dihitung', async () => {
    stubFetch((url) => (url === `${PATH}/ringkas` ? 'hold' : null))
    wrap(<SalvageInboxPage />)

    expect(await screen.findByText('Menghitung ringkasan status salvage…')).toBeInTheDocument()
  })

  it('memformat tanggal dan uang, dan menulis tanda pisah bila rincian tak berkunci', async () => {
    stubFetch((url) =>
      url.startsWith(`${PATH}?`)
        ? jsonResponse(200, list([row(), row({ referensi: '202', id_salvage: '', nilai_pengajuan_pic: 'abc', no_klaim: 'PNC-2' })]))
        : null,
    )
    wrap(<SalvageInboxPage />)

    const table = await screen.findByRole('table', { name: 'Daftar Histori' })
    expect(within(table).getByText('1.500.000')).toBeInTheDocument()
    // Nilai yang tidak terbaca angka digambar apa adanya, bukan nol.
    expect(within(table).getByText('abc')).toBeInTheDocument()
    expect(within(table).getAllByRole('button', { name: 'Detail' })).toHaveLength(1)
    expect(within(table).queryByText('2026-09-09')).not.toBeInTheDocument()
  })

  /**
   * Catatan daftar TIDAK LAGI DIGAMBAR (2026-10-08).
   *
   * Uji ini dulu membuktikan sebaliknya. Ia diubah menjadi kebalikannya — bukan dihapus —
   * karena yang perlu dijaga sekarang justru KETIADAANNYA: isian itu dicabut dari kontrak
   * server, dan layar yang diam-diam menggambarnya lagi berarti kontraknya hidup kembali
   * tanpa ada yang memutuskannya.
   */
  it('TIDAK menggambar catatan daftar, dan tetap menyebut daftar yang kosong', async () => {
    const withNote = { ...TAB, catatan_daftar: 'Catatan khusus daftar.' }
    stubFetch((url) => {
      if (url === `${PATH}/daftar`) return jsonResponse(200, { ...META, daftar: [withNote] })
      if (url.startsWith(`${PATH}?`)) return jsonResponse(200, list([], { daftar: withNote }))
      return null
    })
    wrap(<SalvageInboxPage />)

    expect(await screen.findByText('Belum ada pengajuan pada daftar ini.')).toBeInTheDocument()

    // Dikirim server pun, ia tidak digambar.
    expect(screen.queryByText('Catatan khusus daftar.')).not.toBeInTheDocument()
  })

  it('berpindah halaman lewat paginasi lalu kembali ke halaman pertama', async () => {
    stubFetch((url) =>
      url.startsWith(`${PATH}?`)
        ? jsonResponse(
            200,
            list([row()], {
              paginasi: {
                halaman: url.includes('halaman=2') ? 2 : 1,
                ukuran: 20,
                total: 40,
                total_halaman: 2,
              },
            }),
          )
        : null,
    )
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
    await waitFor(() => expect(calls.some((c) => c.url.includes('halaman=2'))).toBe(true))

    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Halaman pertama' })).not.toBeDisabled(),
    )
    await user.click(screen.getByRole('button', { name: 'Halaman pertama' }))
    // Halaman pertama dilayani dari cache, sehingga paginatornya kembali ke posisi awal.
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Halaman pertama' })).toBeDisabled(),
    )
  })

  it('memuat ulang daftar lewat Refresh dan mengosongkan pencarian', async () => {
    stubFetch(() => null)
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />, '/inbox-salvage?cari=A')

    await screen.findByRole('table', { name: 'Daftar Histori' })
    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBeGreaterThan(before))

    await user.clear(screen.getByLabelText('CARI SALVAGE'))
    await waitFor(() => {
      const last = [...calls].reverse().find((c) => c.url.startsWith(PATH + '?'))
      expect(last?.url).not.toContain('cari=')
    })
  })
})

describe('SalvageInboxPage — ekspor', () => {
  it('mengunduh berkas bernama dari server dengan daftar dan kata kunci', async () => {
    stubFetch((url) =>
      url.startsWith(`${PATH}/ekspor`)
        ? new Response('x', {
            status: 200,
            headers: { 'Content-Disposition': 'attachment; filename="histori.csv"' },
          })
        : null,
    )
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />, '/inbox-salvage?cari=PNC')

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Export Data' }))

    await waitFor(() => expect(clicked).toEqual(['histori.csv']))
    const request = calls.find((c) => c.url.startsWith(`${PATH}/ekspor`))
    expect(request?.url).toBe(`${PATH}/ekspor?daftar=histori&cari=PNC`)
    expect((request?.init?.headers as Record<string, string>)['X-Portal']).toBe('ASM')
  })

  it('memakai nama cadangan saat server tidak menyebut nama', async () => {
    stubFetch((url) => (url.startsWith(`${PATH}/ekspor`) ? new Response('x', { status: 200 }) : null))
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    await waitFor(() => expect(clicked).toEqual(['inbox-salvage.csv']))
  })

  it('memakai nama cadangan saat Content-Disposition tidak bernama', async () => {
    stubFetch((url) =>
      url.startsWith(`${PATH}/ekspor`)
        ? new Response('x', { status: 200, headers: { 'Content-Disposition': 'attachment' } })
        : null,
    )
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    await waitFor(() => expect(clicked).toEqual(['inbox-salvage.csv']))
  })

  it.each([
    [jsonResponse(403, { pesan: 'Tidak berwenang mengekspor.' }), 'Tidak berwenang mengekspor.'],
    [new Response('bukan json', { status: 500 }), 'Berkas ekspor tidak dapat diambil.'],
  ])('menyebut pesan ekspor yang gagal %#', async (reply, text) => {
    stubFetch((url) => (url.startsWith(`${PATH}/ekspor`) ? reply : null))
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    expect(await screen.findByText('Berkas ekspor tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('memakai pesan umum saat ekspor gagal tanpa pesan', async () => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      if (url.startsWith(`${PATH}/ekspor`)) return Promise.reject(new Error(''))
      if (url === `${PATH}/daftar`) return Promise.resolve(jsonResponse(200, META))
      if (url === `${PATH}/ringkas`) return Promise.resolve(jsonResponse(200, { baris: [], portal: 'ASM' }))
      return Promise.resolve(jsonResponse(200, list([row()])))
    })
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    expect(
      await screen.findByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })
})

describe('SalvageInboxPage — form Tambah', () => {
  it('menampilkan kabar tersimpan setelah Submit berhasil', async () => {
    stubFetch((url, init) =>
      url === PATH && init?.method === 'POST'
        ? jsonResponse(201, {
            id_salvage: '300',
            jumlah_detail_item: 0,
            pesan: 'Pengajuan 300 tersimpan.',
            balai_lelang: { dicoba: true, diterima: true },
            pemberitahuan: { dicoba: true, terkirim: true },
            portal: 'ASM',
          })
        : null,
    )
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
    await fillRequired(user, form)
    await user.click(within(form).getByRole('button', { name: 'Submit Pengajuan Salvage' }))

    expect(await screen.findByText('Pengajuan 300 tersimpan.')).toHaveAttribute('role', 'status')

    // Keduanya berhasil, sehingga bilahnya HIJAU.
    expect(screen.getByText('Pengajuan 300 tersimpan.').className).toContain('emerald')

    // onSaved TIDAK menutup form jalur Tambah — penutupnya onClose.
    await waitFor(() =>
      expect(screen.queryByRole('form', { name: 'Menambahkan Data Salvage' })).not.toBeInTheDocument(),
    )
  })

  // Pengajuan yang tersimpan tetapi TIDAK sampai ke balai lelang bukan keberhasilan penuh.
  //
  // Bilah hijau yang berbunyi "BELUM diterima" adalah isyarat bercampur — warnanya
  // mengatakan beres, kalimatnya mengatakan sebaliknya — dan isyarat semacam itu yang
  // membuat orang berhenti membaca bilah pemberitahuan sama sekali.
  it('menggambar bilah PERINGATAN bila balai lelang menolak pengajuannya', async () => {
    stubFetch((url, init) =>
      url === PATH && init?.method === 'POST'
        ? jsonResponse(201, {
            id_salvage: '302',
            jumlah_detail_item: 0,
            pesan: 'Pengajuan salvage tersimpan. Pengajuan sudah dikirim ke balai lelang tetapi BELUM diterima.',
            balai_lelang: { dicoba: true, diterima: false, keterangan: 'ID sudah terdaftar' },
            pemberitahuan: { dicoba: true, terkirim: true },
            portal: 'ASM',
          })
        : null,
    )
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await screen.findByRole('table', { name: 'Daftar Histori' })
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
    await fillRequired(user, form)
    await user.click(within(form).getByRole('button', { name: 'Submit Pengajuan Salvage' }))

    const bilah = await screen.findByText(/BELUM diterima/)
    expect(bilah).toHaveAttribute('role', 'status')
    expect(bilah.className).toContain('amber')
    expect(bilah.className).not.toContain('emerald')
  })

  it('menutup form isian-klaim setelah tersimpan', async () => {
    stubFetch((url, init) => {
      if (url.startsWith(`${PATH}/pengajuan/`)) {
        return jsonResponse(200, {
          id_salvage: '',
          no_klaim: 'PNC-1',
          ada_pengajuan: false,
          nama_object: 'Gudang',
          nama_coverage: 'PAR',
          riwayat: [],
          barang: [],
        })
      }
      if (url === PATH && init?.method === 'POST') {
        return jsonResponse(201, {
          id_salvage: '301',
          jumlah_detail_item: 0,
          pesan: 'Tersimpan untuk klaim.',
          balai_lelang: { dicoba: true, diterima: true },
          pemberitahuan: { dicoba: true, terkirim: true },
          portal: 'ASM',
        })
      }
      return null
    })
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await user.click(await screen.findByRole('button', { name: 'Detail' }))
    const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
    expect(within(form).getByText('PNC-1')).toBeInTheDocument()
    await fillRequired(user, form)
    await user.click(within(form).getByRole('button', { name: 'Submit Pengajuan Salvage' }))

    expect(await screen.findByText('Tersimpan untuk klaim.')).toHaveAttribute('role', 'status')
    expect(screen.queryByRole('form', { name: 'Menambahkan Data Salvage' })).not.toBeInTheDocument()
  })

  it('menutup form isian-klaim lewat Batal', async () => {
    stubFetch((url) =>
      url.startsWith(`${PATH}/pengajuan/`)
        ? jsonResponse(200, {
            id_salvage: '',
            no_klaim: 'PNC-1',
            ada_pengajuan: false,
            nama_object: '',
            nama_coverage: '',
            riwayat: [],
            barang: [],
          })
        : null,
    )
    const user = userEvent.setup()
    wrap(<SalvageInboxPage />)

    await user.click(await screen.findByRole('button', { name: 'Detail' }))
    const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
    await user.click(within(form).getByRole('button', { name: 'Batal' }))
    expect(screen.getByRole('table', { name: 'Daftar Histori' })).toBeInTheDocument()
  })
})

describe('TambahSalvageForm', () => {
  function show() {
    const onSaved = vi.fn()
    const onClose = vi.fn()
    const view = wrap(
      <TambahSalvageForm
        statusOptions={[{ kode: '2', label: 'Salvage DiTerima' }]}
        currencyOptions={[{ kode: 'IDR', label: 'IDR' }]}
        uploadColumns={['item', 'quantity']}
        onClose={onClose}
        onSaved={onSaved}
      />,
    )
    const file = view.container.querySelector('input[type="file"]') as HTMLInputElement
    return { onSaved, onClose, file }
  }

  it('mengirim seluruh isian beserta item hasil unggahan yang tersisa', async () => {
    stubFetch((url) => {
      if (url === `${PATH}/unggah-detail`) {
        return jsonResponse(200, {
          detail_item_salvage: [
            { nama_item: 'Besi', jumlah_item: '3', satuan: 'kg', remark: 'karat' },
            { nama_item: 'Kayu', jumlah_item: '1', satuan: 'm3', remark: '' },
          ],
          pesan: 'Dua baris terbaca. Belum tersimpan.',
        })
      }
      return jsonResponse(201, {
        id_salvage: '1',
        jumlah_detail_item: 1,
        pesan: 'ok',
        balai_lelang: { dicoba: true, diterima: true, id_balai_lelang: 'SB-1' },
        pemberitahuan: { dicoba: true, terkirim: true },
        portal: 'ASM',
      })
    })
    // Lima belas isian diketik huruf demi huruf. Jeda bawaan user-event di antara ketukan
    // membuat test ini melewati batas waktu di runner CI yang sibuk; tanpa jeda, setiap
    // ketukan tetap dikirim sebagai event terpisah — yang diuji tidak berkurang.
    const user = userEvent.setup({ delay: null })
    const { onSaved, onClose, file } = show()

    expect(screen.getAllByText('Data Tidak Ada')[0]!).toBeInTheDocument()
    expect(screen.getByText('item, quantity')).toBeInTheDocument()

    await user.upload(file, new File(['a,b'], 'detail.csv', { type: 'text/csv' }))
    expect(await screen.findByText('Dua baris terbaca. Belum tersimpan.')).toBeInTheDocument()
    const items = screen.getByRole('table', { name: 'Detail Item Salvage' })
    expect(within(items).getByDisplayValue('Besi')).toBeInTheDocument()
    expect(file.value).toBe('')
    // "Hapus" ada SATU, di atas tabel, dan ia membuang baris TERAKHIR — bukan satu
    // tautan per baris. Baris terakhir pula yang paling sering dibatalkan orang.
    await user.click(screen.getByRole('button', { name: 'Hapus' }))
    expect(within(items).queryByDisplayValue('Kayu')).not.toBeInTheDocument()

    const fields: [RegExp, string][] = [
      [/Nomor Klaim/, 'PNC-9'],
      [/Nama Object/, 'Obj'],
      [/Nama Coverage/, 'Cov'],
      [/Jenis Salvage/, 'Besi Tua'],
      [/Lokasi Salvage$/, 'Gudang'],
      // "Nilai Penawaran" tidak lagi digambar pada form Tambah — nilainya tetap terkirim
      // apa adanya, tetapi tidak ada kolom untuk mengetiknya di sini.
      [/Total Nilai Salvage/, '1000'],
      [/Share Tertanggung/, '10'],
      [/^Email$/, 'a@example.invalid'],
      [/Nama PIC Survey/, 'Surveyor'],
      [/No Telp PIC Survey/, '0210'],
      [/Email PIC Survey/, 'b@example.invalid'],
      // Dipatok tepat: baris grid juga berlabel 'Remark baris 1'.
      [/^Remark$/, 'catatan'],
    ]
    for (const [label, value] of fields) {
      await user.type(screen.getByLabelText(label), value)
    }
    await user.selectOptions(screen.getByLabelText('Mata Uang'), 'IDR')
    await user.selectOptions(screen.getByLabelText('Status Salvage'), '2')
    await user.click(screen.getByLabelText('Lokasi Salvage Di Jabodatabek'))
    // "Tanggal Input" TIDAK dapat diubah — di layar lama pun ia teks biasa, bukan kotak
    // isian. Nilainya tetap dikirim, dan itulah yang diperiksa di bawah.
    await user.click(screen.getByRole('button', { name: 'Submit Pengajuan Salvage' }))

    await waitFor(() =>
      expect(onSaved).toHaveBeenCalledWith({ pesan: 'ok', perluPerhatian: false }),
    )
    expect(onClose).toHaveBeenCalledTimes(1)
    const sent = JSON.parse(String(calls.find((c) => c.url === PATH)?.init?.body))
    expect(sent).toMatchObject({
      mode: 'insert',
      id_salvage: '',
      nomor_klaim: 'PNC-9',
      jenis_salvage: 'Besi Tua',
      status_salvage: '2',
      lokasi_salvage: 'Gudang',
      lokasi_salvage_di_jabodetabek: true,
      tanggal_input: expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/),
      minimum_salvage: '1000',
      email_pic_survey: 'b@example.invalid',
      remark: 'catatan',
      detail_item_salvage: [{ nama_item: 'Besi', jumlah_item: '3', satuan: 'kg', remark: 'karat' }],
    })
    // Unggahan tidak mengirim JSON — berkasnya dikirim sebagai FormData.
    expect(calls.find((c) => c.url === `${PATH}/unggah-detail`)?.init?.body).toBeInstanceOf(FormData)
  })

  it('mengunci tombol unggah selama berkas dibaca', async () => {
    stubFetch((url) => (url === `${PATH}/unggah-detail` ? 'hold' : null))
    const user = userEvent.setup()
    const { file } = show()

    await user.upload(file, new File(['a'], 'a.csv', { type: 'text/csv' }))
    expect(await screen.findByRole('button', { name: 'Membaca berkas…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Submit Pengajuan Salvage' })).not.toBeDisabled()
  })

  it('menyebut galat unggah dari server', async () => {
    stubFetch((url) =>
      url === `${PATH}/unggah-detail`
        ? jsonResponse(422, { kode: 'validasi_gagal', pesan: 'Pemisah bukan koma.' })
        : null,
    )
    const user = userEvent.setup()
    const { file } = show()

    await user.upload(file, new File(['a'], 'a.csv', { type: 'text/csv' }))
    expect(await screen.findByText('Berkas tidak dapat dibaca')).toBeInTheDocument()
    expect(screen.getByText('Pemisah bukan koma.')).toBeInTheDocument()
  })

  it('menandai galat detail item dan menyebut galat simpan', async () => {
    stubFetch(() =>
      jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Permintaan belum benar.',
        detail: [{ field: 'detail_item_salvage', pesan: 'Item wajib bernama.' }],
      }),
    )
    const user = userEvent.setup()
    show()

    await fillRequired(user, document.body)
    await user.click(screen.getByRole('button', { name: 'Submit Pengajuan Salvage' }))
    expect(await screen.findByText('Pengajuan tidak tersimpan')).toBeInTheDocument()
    expect(screen.getAllByText('Permintaan belum benar.').length).toBeGreaterThan(0)
    expect(screen.getByText('Item wajib bernama.')).toHaveAttribute('role', 'alert')
  })

  it('memakai pesan jaringan dan menampilkan Menyimpan selagi mengirim', async () => {
    let mode: 'hold' | 'fail' = 'fail'
    stubFetch(() => (mode === 'fail' ? 'fail' : 'hold'))
    const user = userEvent.setup()
    show()

    await fillRequired(user, document.body)
    await user.click(screen.getByRole('button', { name: 'Submit Pengajuan Salvage' }))
    expect((await screen.findAllByText('Tidak dapat menghubungi server Claim PNC.')).length).toBeGreaterThan(0)

    mode = 'hold'
    await user.click(screen.getByRole('button', { name: 'Submit Pengajuan Salvage' }))
    expect(await screen.findByRole('button', { name: 'Menyimpan…' })).toBeDisabled()
  })

  it('tidak menembak server bila tidak ada berkas yang dipilih', async () => {
    stubFetch(() => null)
    const { file } = show()

    file.dispatchEvent(new Event('change', { bubbles: true }))
    expect(calls).toHaveLength(0)
  })
})

describe('DetailSalvagePanel', () => {
  it.each([
    [new NetworkError(), 'Sambungan ke server terputus. Periksa jaringan lalu coba lagi.'],
    [new TypeError('x'), 'Terjadi kesalahan pada sistem.'],
  ])('menjelaskan galat rincian di luar APIError %#', (error, text) => {
    render(
      <DetailSalvagePanel
        detailKey="pengajuan"
        reference="1"
        data={undefined}
        isPending={false}
        isError
        error={error}
        onClose={() => {}}
      />,
    )
    expect(screen.getByText('Detail tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })
})
