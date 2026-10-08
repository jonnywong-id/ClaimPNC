import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReasMemberPage } from './ReasMemberPage'

/**
 * Seluruh kode, nama perusahaan, login, dan surel di berkas ini KARANGAN, dan domainnya
 * `contoh.invalid` yang RFC 2606 cadangkan supaya tidak pernah dapat diselesaikan DNS
 * (`D-69`).
 */

/** Baris CADANGAN — TYPE '1'. */
const CADANGAN = {
  kode_reas: 'RE-001',
  nama_reas: 'Reasuransi Nusantara Jaya',
  login: 'ReasuransiNusantaraJaya',
  email: 'pla.nusantara@contoh.invalid',
  negara: 'Indonesia',
  tipe: '1',
  cadangan: true,
}

/**
 * Baris KEDUA milik perusahaan yang SAMA, hanya berbeda TYPE.
 *
 * Inilah bentuk yang paling mudah disalahpahami di layar ini: nama yang sama muncul dua
 * kali dengan surel berbeda. Tanpa baris ini di dalam uji, kolom Tipe dan kunci baris tiga
 * kolom tidak terbukti berguna sama sekali.
 */
const TIPE_DUA = {
  kode_reas: 'RE-001',
  nama_reas: 'Reasuransi Nusantara Jaya',
  login: 'ReasuransiNusantaraJaya',
  email: 'dla.nusantara@contoh.invalid',
  negara: 'Indonesia',
  tipe: '2',
  cadangan: false,
}

/** Baris yang LOGIN, EMAIL, dan NEGARA-nya kosong — ketiganya keadaan yang mungkin. */
const TIDAK_LENGKAP = {
  kode_reas: 'RE-003',
  nama_reas: 'Bahtera Reinsurance Ltd',
  login: '',
  email: '',
  negara: '',
  tipe: '',
  cadangan: false,
}

type Call = {
  url: string
  method: string
  header: Record<string, string>
  body: unknown
}

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      header: (init?.headers as Record<string, string>) ?? {},
      body: init?.body !== undefined ? JSON.parse(init.body as string) : undefined,
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

/** defaultReply melayani daftar dengan ketiga baris contoh. */
function defaultReply(): (call: Call) => Reply {
  return () => ({
    body: { member_reas: [CADANGAN, TIPE_DUA, TIDAK_LENGKAP], portal: 'ASM' },
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ReasMemberPage />
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

describe('daftar member reas', () => {
  it('memakai kepala kolom Pega apa adanya, pada urutan yang sama', async () => {
    /*
      Ditetapkan Work Owner 2026-10-05: No · Nama Reinsurer · Login · Email · Tipe · Aksi.

      Diuji sebagai URUTAN, bukan sebagai keberadaan satu per satu: susunan kolom adalah
      bagian dari "ikuti Pega", dan kolom yang benar pada posisi yang salah tetap salah.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const kepala = within(table)
      .getAllByRole('columnheader')
      .map((h) => h.textContent?.trim())

    expect(kepala).toEqual(['No', 'Nama Reinsurer', 'Login', 'Email', 'Tipe', 'Aksi'])
  })

  it('TIDAK menggambar kolom Negara — Pega pun tidak punya', async () => {
    /*
      `COUNTRY` tetap dikirim server dan tetap terbaca di panel Detail; yang tidak ada
      hanyalah kolomnya di grid. Uji ini memagari rekonstruksi yang sempat keliru:
      susunan kolom sebelumnya menggambarnya karena section grid Pega hilang dari export
      (`R-16`), dan Work Owner mengoreksinya.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const kepala = within(table)
      .getAllByRole('columnheader')
      .map((h) => h.textContent?.trim())

    expect(kepala).not.toContain('Negara')
    expect(within(table).queryByText('Indonesia')).toBeNull()
  })

  it('menampilkan isi baris yang dijawab server', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')

    // Satu baris kepala + tiga baris isi.
    expect(within(table).getAllByRole('row')).toHaveLength(4)

    /*
      Kolom "No" berisi NOMOR URUT tampilan — 1, 2, 3 — bukan kode reas.

      Diperiksa lewat SEL PERTAMA tiap baris, bukan lewat teks di dalam barisnya: baris
      pertama kebetulan ber-`tipe` "1" juga, sehingga mencari teks "1" di seluruh baris
      menemukan dua hal yang berbeda.

      Dan dicari lewat `getByText` di dalam selnya, bukan lewat `textContent`: setiap sel
      ikut memuat judul kolomnya sebagai label tampilan kartu (`aria-hidden`), sehingga
      `textContent` sel pertama berbunyi "No1", bukan "1".
    */
    const baris = within(table).getAllByRole('row').slice(1)
    baris.forEach((row, i) => {
      const selPertama = within(row).getAllByRole('cell')[0] as HTMLElement
      expect(within(selPertama).getByText(String(i + 1))).toBeInTheDocument()
    })

    expect(within(table).getByText('pla.nusantara@contoh.invalid')).toBeInTheDocument()
    expect(within(table).getByText('dla.nusantara@contoh.invalid')).toBeInTheDocument()
  })

  it('menampilkan dua baris terpisah untuk satu perusahaan dengan TYPE berbeda', async () => {
    /*
      Kunci baris di layar adalah TIGA kolom — kode + nama + tipe. Bila ia memakai kode reas
      sendirian, React akan menganggap kedua baris ini satu baris dan hanya salah satunya
      yang tampil. Uji ini yang menangkapnya.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const nusantara = within(table)
      .getAllByRole('row')
      .filter((row) => row.textContent?.includes('Reasuransi Nusantara Jaya'))

    expect(nusantara).toHaveLength(2)
  })

  it('TIDAK menggambar penanda "cadangan" — Pega pun tidak', async () => {
    /*
      Ketetapan Work Owner 2026-10-05. Arti `TYPE = '1'` tetap dihitung server dan tetap
      dikirim sebagai field `cadangan`; yang dihapus adalah penggambarannya, bukan datanya —
      dijaga uji backend `TestListMenandaiBarisCadangan`.

      Diuji, bukan sekadar dihapus: penanda seperti ini mudah kembali saat seseorang merasa
      arti `'1'` perlu "dijelaskan di layar".
    */
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.queryByText('cadangan')).toBeNull()
  })

  it('kolom Nama Reinsurer dipatok lebar minimum, bukan hanya lewat width kolom', async () => {
    /*
      `DataTable` memakai `table-auto`, dan di mode itu `width` kolom hanya SARAN —
      peramban membagi ruang menurut panjang isi. Alamat surel (teks panjang tanpa spasi)
      menang terhadap nama perusahaan yang pendek, dan kolom ini menyusut sampai kepala
      kolomnya patah dua baris.

      Lebar minimum pada ISI selnya menaikkan lebar min-content kolom, dan itu wajib
      dipenuhi peramban. Diuji karena ia tampak seperti gaya tempelan yang boleh dirapikan
      — padahal ia yang menahan kolomnya.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    // Dua baris memang bernama sama — satu perusahaan, dua jenis dokumen.
    const nama = within(table).getAllByText('Reasuransi Nusantara Jaya')

    expect(nama).toHaveLength(2)
    nama.forEach((sel) => {
      // Dibaca dari gaya SEBARIS, bukan lewat `toHaveStyle`: yang terakhir memakai
      // getComputedStyle, dan jsdom tidak menghitung satuan `rem` sehingga nilainya
      // kembali kosong — kegagalan yang berasal dari perkakas uji, bukan dari layarnya.
      expect(sel.style.minWidth).toBe('16rem')
    })
  })

  it('TIDAK menggambar kode reas di bawah nama reinsurer', async () => {
    /*
      Sempat ditempelkan di bawah namanya supaya tetap terbaca; Work Owner mengoreksinya
      2026-10-05 — Pega tidak begitu.

      Ia tetap DAPAT DICARI lewat `value` kolom itu; lihat uji pencarian di
      ReasMemberPage.more.test.tsx.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByText('RE-001')).toBeNull()
    expect(within(table).queryByText('RE-003')).toBeNull()
  })

  it('menyatakan LOGIN dan EMAIL yang kosong, bukan membiarkannya sebagai sel kosong', async () => {
    /*
      Keduanya gagal dalam DIAM di sistem lama: mitra tanpa login tidak akan pernah melihat
      klaimnya sendiri, dan baris tanpa surel membuat dokumen PLA/DLA terbit lalu tidak
      sampai ke siapa pun. Sel kosong tidak menunjukkan keduanya.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const bahtera = within(table)
      .getAllByRole('row')
      .find((row) => row.textContent?.includes('Bahtera Reinsurance Ltd'))

    expect(bahtera).toBeDefined()
    expect(within(bahtera as HTMLElement).getAllByText('belum ada')).toHaveLength(2)
  })

  it('menyebut portal entitas yang sedang dibuka', async () => {
    installFetch(defaultReply())
    show()

    // Data master dimiliki masing-masing entitas; "data siapa ini" tidak boleh hanya
    // diandaikan pengguna (ADR-0030, R-20).
    expect(await screen.findByText('ASM')).toBeInTheDocument()
  })

  it('mengirim header portal pada permintaan daftar', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    const daftar = calls.find((c) => c.url.includes('/api/master/reas'))
    expect(daftar).toBeDefined()
    expect(daftar?.method).toBe('GET')
    expect(Object.values(daftar?.header ?? {})).toContain('ASM')
  })
})

describe('layar baca-saja', () => {
  /*
    Ketiga uji di bawah menjaga keputusan yang berdasar bukti: harness `DataMemberReas`
    memuat satu grid dan satu tombol Refresh, dan satu-satunya penulis T_REINSURER di
    sistem lama adalah alur PLA/DLA lewat `Database/UPDATEREAS.prc` — bukan layar ini.

    Bila kelak terbukti sebaliknya, ketiganya yang akan gagal lebih dulu. Itu memang yang
    diinginkan: penambahan jalur tulis harus menjadi keputusan yang disadari.
  */

  it('tidak punya tombol Tambah, dan Simpan baru muncul setelah Ubah ditekan', async () => {
    /*
      Tombol **Tambah** tidak ada, dan itu terkalibrasi: indeks rule
      `Harness/MasterLoginSurvey-Harness.xml` — layar yang terbukti punya dua tombol —
      menyebut `PYBUTTONLABEL!REFRESH` DAN `!TAMBAH`; `DataMemberReas` hanya `REFRESH`.

      Baris baru lahir dari alur PLA/DLA, bukan dari layar ini.
    */
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    expect(screen.queryByRole('button', { name: 'Tambah' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Simpan' })).toBeNull()
  })

  /** openEdit menekan tombol Ubah pada baris yang namanya disebut. */
  async function openEdit(
    user: ReturnType<typeof userEvent.setup>,
    nama: string,
  ): Promise<HTMLElement> {
    const table = await screen.findByRole('table')
    const baris = within(table)
      .getAllByRole('row')
      .find((row) => row.textContent?.includes(nama))
    await user.click(within(baris as HTMLElement).getByRole('button', { name: 'Ubah' }))

    const judul = await screen.findByRole('heading', { name: /Ubah Email/ })
    return judul.closest('section') as HTMLElement
  }

  it('form ubah berada DI ATAS tabel, bukan di bawahnya', async () => {
    /*
      Ketetapan Work Owner 2026-10-05. Alasannya praktis: form di bawah tabel berada di luar
      layar pada daftar yang panjang, sehingga menekan Ubah tampak seperti tidak melakukan
      apa-apa.

      Diuji lewat urutan dokumen, bukan lewat kelas CSS: yang menentukan adalah posisinya
      terhadap tabel, dan itulah yang dilihat pengguna.
    */
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    const panel = await openEdit(user, 'Reasuransi Nusantara Jaya')
    const table = screen.getByRole('table')

    // Node.DOCUMENT_POSITION_FOLLOWING = 4 → tabel berada SESUDAH panel.
    expect(panel.compareDocumentPosition(table) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('hanya Email yang dapat diketik; kunci dan Login hanya ditampilkan', async () => {
    /*
      `Database/UPDATEREAS.prc` pada baris yang sudah ada hanya menyentuh kolom `EMAIl`.
      `LOGIN` yang paling berakibat bila ikut dapat diubah: ia menentukan klaim mana yang
      dilihat seorang mitra reasuransi, dan mengubahnya dari sini akan memindahkan
      visibilitas klaim tanpa satu pun pesan galat (`R-20`).
    */
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    const panel = await openEdit(user, 'Reasuransi Nusantara Jaya')

    // SATU isian saja di dalam formnya.
    expect(within(panel).getAllByRole('textbox')).toHaveLength(1)
    expect(within(panel).getByLabelText('Email')).toHaveValue('pla.nusantara@contoh.invalid')

    // Keempat keterangan ditampilkan, tetapi bukan sebagai isian.
    expect(within(panel).getByText('RE-001')).toBeInTheDocument()
    expect(within(panel).getByText('ReasuransiNusantaraJaya')).toBeInTheDocument()
    expect(within(panel).getByText('Indonesia')).toBeInTheDocument()
  })

  it('mengirim ketiga kolom kunci beserta surel barunya', async () => {
    installFetch((call) =>
      call.method === 'PUT'
        ? { body: { member_reas: CADANGAN, portal: 'ASM' } }
        : { body: { member_reas: [CADANGAN, TIPE_DUA, TIDAK_LENGKAP], portal: 'ASM' } },
    )
    const user = userEvent.setup()
    show()

    const panel = await openEdit(user, 'Reasuransi Nusantara Jaya')
    const isian = within(panel).getByLabelText('Email')
    await user.clear(isian)
    await user.type(isian, 'baru@contoh.invalid')
    await user.click(within(panel).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.method === 'PUT')).toBe(true)
    })

    const put = calls.find((c) => c.method === 'PUT')
    expect(put?.body).toEqual({
      kode_reas: 'RE-001',
      nama_reas: 'Reasuransi Nusantara Jaya',
      tipe: '1',
      email: 'baru@contoh.invalid',
    })
  })

  it('menutup form lewat Batal tanpa mengirim apa pun', async () => {
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    const panel = await openEdit(user, 'Reasuransi Nusantara Jaya')
    await user.click(within(panel).getByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('heading', { name: /Ubah Email/ })).toBeNull()
    expect(calls.every((c) => c.method === 'GET')).toBe(true)
  })

  it('tidak pernah mengirim permintaan yang mengubah data', async () => {
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => {
      expect(calls.length).toBeGreaterThan(1)
    })
    expect(calls.every((c) => c.method === 'GET')).toBe(true)
  })

  it('TIDAK menggambar blok catatan di kaki halaman', async () => {
    /*
      Dihapus atas ketetapan Work Owner 2026-10-05. Isinya menyebut nama rule Pega, nama
      kolom Oracle, dan nomor keputusan — tidak berarti apa-apa bagi petugas klaim, dan
      membuat layar tampak belum selesai.

      Diuji, bukan sekadar dihapus: blok seperti ini mudah kembali saat seseorang merasa
      keterbatasannya perlu "dinyatakan di layar". Catatan resminya hidup di doc comment
      paket `masterreas` dan di `docs/`.
    */
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    expect(screen.queryByText(/Layar ini hanya menampilkan/)).toBeNull()
    expect(screen.queryByText(/dapat muncul beberapa kali/)).toBeNull()
    expect(screen.queryByText(/menentukan klaim yang dilihat mitra/)).toBeNull()
  })
})

describe('penolakan dan kegagalan', () => {
  it('meminta portal dipilih lebih dulu, tanpa menembak server', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()

    // Menembak server hanya untuk menerima penolakan akan menampilkan pesan galat pada
    // layar yang sebenarnya belum siap dibuka.
    expect(calls.filter((c) => c.url.includes('/api/master/reas'))).toHaveLength(0)
  })

  it('menjelaskan basis data entitas yang belum tersedia, bukan menyuruh mengulang', async () => {
    installFetch(() => ({
      body: { kode: 'portal_belum_siap', pesan: 'Portal belum siap.' },
      status: 503,
    }))
    show()

    expect(
      await screen.findByText('Basis data entitas ini belum tersedia'),
    ).toBeInTheDocument()
  })

  it('tidak menampilkan daftar kosong ketika pemuatannya gagal', async () => {
    /*
      Daftar kosong dan kegagalan membaca terlihat sama bila galatnya ditelan, dan pengguna
      tidak punya cara membedakan "belum ada mitra" dari "tabelnya tidak terbaca".
    */
    installFetch(() => ({
      body: { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' },
      status: 500,
    }))
    show()

    expect(await screen.findByText(/tidak dapat dimuat|kesalahan pada sistem/)).toBeInTheDocument()
    expect(screen.queryByRole('table')).toBeNull()
  })
})
