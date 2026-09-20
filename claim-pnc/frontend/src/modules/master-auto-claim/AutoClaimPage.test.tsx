import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AutoClaimPage } from './AutoClaimPage'

const BANKS = {
  portal: 'ASM',
  bank: [
    { kode: '001', nama: 'BANK CONTOH NIAGA' },
    { kode: '002', nama: 'BANK CONTOH MANDIRI' },
  ],
}

const SOURCES = {
  portal: 'ASM',
  sumber_bisnis: [
    { id: 'AGN007', nama: 'PT. LEASING CONTOH UTAMA' },
    { id: 'AGN008', nama: 'PT CONTOH ARTHA FINANSIAL' },
  ],
}

const CLIENTS = {
  portal: 'ASM',
  client: [{ id: 'CLI007', nama: 'PT. TERTANGGUNG CONTOH KETUJUH' }],
}

/** Satu baris disetujui dan benar-benar dapat dipakai klaim otomatis. */
const APPROVED = {
  inisial: 'AGN001',
  nama_penerima: 'MITRA CONTOH SEJAHTERA',
  nama_bank: 'BANK CONTOH NIAGA',
  no_rekening: '1000000001',
  pct_max: '100',
  pic_lapor: 'PIC Contoh Satu',
  email_lapor: 'pic.satu@contoh.example',
  alamat_penerima: 'Jalan Contoh Nomor 1',
  id_client: 'CLI001',
  nama_client: 'TERTANGGUNG CONTOH PERTAMA',
  claim_allowed: '1',
  komite: 'ADMINPNC',
  status: '1',
  status_label: 'Approve',
  dapat_dipakai: true,
}

/**
 * Baris yang DISETUJUI tetapi CLAIM_ALLOWED-nya bukan "1".
 *
 * Ia keadaan yang di Pega tidak terlihat di layar mana pun: statusnya Approve, tetapi
 * `GetReceiverClaimAsuransiKredit` menyaring `claim_allowed = 1` sehingga baris ini
 * tidak pernah dipakai. Uji di bawah memastikan layar menjelaskannya.
 */
const APPROVED_BUT_BLOCKED = {
  ...APPROVED,
  inisial: 'AGN002',
  nama_penerima: 'KOPERASI CONTOH BERSAMA',
  claim_allowed: '0',
  dapat_dipakai: false,
}

/** Baris menunggu, penyetujunya pengguna yang sedang masuk. */
const PENDING = {
  ...APPROVED,
  inisial: 'AGN003',
  nama_penerima: 'MULTIFINANCE CONTOH ABADI',
  id_client: 'CLI003',
  nama_client: 'TERTANGGUNG CONTOH KETIGA',
  status: '0',
  status_label: 'Waiting Approval',
  dapat_dipakai: false,
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
 * Ketiga jalur lookup diperiksa LEBIH DULU karena ketiganya berawalan sama dengan
 * jalur daftar — persis alasan yang sama yang membuat rutenya didaftarkan lebih dulu
 * di backend.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/auto-claim/bank')) return { body: BANKS }
    if (call.url.startsWith('/api/master/auto-claim/sumber-bisnis')) return { body: SOURCES }
    if (call.url.startsWith('/api/master/auto-claim/client')) return { body: CLIENTS }

    if (call.method === 'GET') {
      const status = new URL(call.url, 'http://x').searchParams.get('status') ?? '1'
      const committeeOnly =
        new URL(call.url, 'http://x').searchParams.get('komite_saya') === 'true'

      let rows: unknown[] = []
      if (status === '1') rows = [APPROVED, APPROVED_BUT_BLOCKED]
      if (status === '0') rows = committeeOnly ? [PENDING] : [PENDING]
      if (status === '2') rows = []

      return { body: { auto_claim: rows, status, portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { auto_claim: APPROVED, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AutoClaimPage />
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

/**
 * tabStrip membatasi pencarian pada bilah tab saja.
 *
 * WAJIB dipakai untuk setiap penekanan tab, karena DUA caption bertabrakan dengan nama
 * tombol di tempat lain: tab "Approve" dengan tombol Approve pada form komite, dan tab
 * "Reject" dengan tombol Reject di sebelahnya. Keduanya memang bernama sama di Pega.
 *
 * Kekeliruan yang sama sudah pernah menjatuhkan uji Master Rekening — tab bernama
 * "Approve" ikut tertangkap `queryByRole('button', { name: 'Approve' })`.
 */
function tabStrip() {
  return within(screen.getByRole('navigation', { name: 'Tab Master Auto Klaim' }))
}

/** clickTab menekan satu tab menurut captionnya. */
async function clickTab(user: ReturnType<typeof userEvent.setup>, label: string) {
  await user.click(tabStrip().getByRole('button', { name: label }))
}

/** listCalls mengambil permintaan daftar saja, membuang ketiga lookup. */
function listCalls() {
  return calls.filter(
    (c) =>
      c.method === 'GET' &&
      c.url.startsWith('/api/master/auto-claim?'),
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

describe('daftar master auto claim', () => {
  /*
    Caption DAN urutan tab dibaca dari `pyTitle` tiap container pada
    `Section/MasterAutoKlaim-Section.xml`, dalam urutan dokumen:

      offset   pyTitle            section
       84.789  Approve            BrowseAutoKlaim
      122.544  Reject             BrowseAutoKlaimReject
      164.383  Waiting Approval   BrowseAutoKlaimApproval
      202.564  Komite Approval    BrowseAutoKlaimKomite

    Uji ini ada karena versi pertama layar ini menamai tab pertama "Master Auto Klaim"
    — yang sebenarnya JUDUL LAYAR, bukan tab — sehingga tab "Approve" hilang sama sekali
    dan ketiga tab lainnya bergeser urutannya.
  */
  it('menampilkan keempat tab Pega pada urutan yang sama', async () => {
    installFetch(defaultReply())
    show()

    const captions = tabStrip()
      .getAllByRole('button')
      .map((b) => b.textContent?.trim())

    expect(captions).toEqual(['Approve', 'Reject', 'Waiting Approval', 'Komite Approval'])

    // "Master Auto Klaim" adalah judul layar, dan HANYA judul layar.
    expect(screen.getByRole('heading', { name: 'Master Auto Klaim' })).toBeInTheDocument()
    expect(tabStrip().queryByRole('button', { name: 'Master Auto Klaim' })).not.toBeInTheDocument()
  })

  /*
    Kolomnya mengikuti header grid Pega SATU PER SATU, tanpa penggabungan. Uji ini ada
    karena versi pertama layar ini menggabungkan Bank dengan No Rekening, PIC dengan
    Email, dan Nama Penerima dengan Client — tiga penggabungan yang tidak ada di Pega
    dan melanggar `D-13`.

    Urutannya ikut diuji, bukan hanya keberadaannya: kolom yang benar tetapi berpindah
    tempat tetap membuat petugas mencari.
  */
  it('menampilkan sembilan kolom Pega pada urutan yang sama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Auto Klaim' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    const headers = within(table)
      .getAllByRole('columnheader')
      .map((h) => h.textContent?.trim())

    expect(headers).toEqual([
      'Inisial',
      'Nama penerima',
      'Bank penerima',
      'No rekening',
      'Alamat penerima',
      'Email lapor',
      'PCT max',
      'PIC',
      'Komite',
      // Kolom tombol Update tanpa judul, sama seperti Pega.
      '',
    ])
  })

  /*
    Header yang benar BELUM cukup. Sebuah render yang tetap menggabungkan dua nilai ke
    dalam satu sel akan lolos uji header di atas — dan itu persis bentuk kesalahan yang
    dikoreksi Work Owner.

    Uji ini karena itu membaca SEL barisnya, satu per satu, dan menuntut tiap nilai
    berdiri di selnya sendiri.
  */
  it('menaruh setiap nilai di selnya sendiri, tidak digabung', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const firstRow = within(table).getAllByRole('row')[1]!
    const cells = within(firstRow)
      .getAllByRole('cell')
      // Tiap sel memuat DUA anak: label nama kolom untuk tampilan kartu di layar
      // sempit — ber-`aria-hidden` dan disembunyikan CSS di layar lebar — lalu
      // nilainya. jsdom tidak menerapkan CSS, sehingga `textContent` sel memuat
      // keduanya. Yang dibaca di sini hanya yang kedua.
      .map((c) => c.querySelector(':scope > span:not([aria-hidden="true"])')?.textContent?.trim())

    expect(cells).toEqual([
      'AGN001',
      'MITRA CONTOH SEJAHTERA',
      'BANK CONTOH NIAGA',
      '1000000001',
      'Jalan Contoh Nomor 1',
      'pic.satu@contoh.example',
      '100',
      'PIC Contoh Satu',
      'ADMINPNC',
      'Update',
    ])
  })

  // Tab Komite tidak punya kolom KOMITE di Pega — seluruh barisnya memang milik
  // pemanggil, sehingga kolomnya tidak memberi tahu apa pun.
  it('menghilangkan kolom Komite pada tab Komite Approval', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await clickTab(userEvent.setup(), 'Komite Approval')

    await waitFor(() => {
      const table = screen.getByRole('table')
      expect(within(table).queryByRole('columnheader', { name: 'Komite' })).not.toBeInTheDocument()
    })
  })

  // Keempat tab adalah empat kombinasi penyaring atas SATU endpoint. Nilai penyaringnya
  // dibaca langsung dari pyDeferLoadRetrievalActivityParams tiap section Pega.
  it('keempat tab mengirim penyaring yang benar', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(listCalls()[0]?.url).toContain('status=1')

    const user = userEvent.setup()

    await clickTab(user, 'Komite Approval')
    await waitFor(() => {
      const last = listCalls().at(-1)?.url ?? ''
      expect(last).toContain('status=0')
      expect(last).toContain('komite_saya=true')
    })

    await clickTab(user, 'Waiting Approval')
    await waitFor(() => {
      const last = listCalls().at(-1)?.url ?? ''
      expect(last).toContain('status=0')
      expect(last).not.toContain('komite_saya')
    })

    await clickTab(user, 'Reject')
    await waitFor(() => expect(listCalls().at(-1)?.url).toContain('status=2'))
  })

  // Portal dikirim sebagai header, bukan di URL: nilai di URL ikut tercatat di log
  // peramban, log proxy, dan header Referer.
  it('menyebut portal entitas pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  /*
    Baris yang disetujui TETAPI tidak dapat dipakai adalah keadaan yang di Pega tidak
    terlihat di layar mana pun.

    Ia dilaporkan sebagai CATATAN DI ATAS TABEL, bukan sebagai kolom: grid mengikuti Pega
    kolom per kolom, dan keterangan yang tidak ada di sana tidak boleh menyelinap masuk
    sebagai kolom tambahan.
  */
  it('melaporkan baris ber-CLAIM_ALLOWED bukan satu di luar grid', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')

    // Inisialnya dicari DI DALAM catatan, bukan di seluruh layar: AGN002 juga muncul
    // sebagai barisnya sendiri di tabel, dan pencarian global akan menemukan keduanya.
    const note = screen.getByText(/1 baris disetujui tetapi tidak dipakai/)
    expect(note).toBeInTheDocument()
    expect(note.closest('div')).toHaveTextContent(/CLAIM_ALLOWED pada baris AGN002 bukan/)

    // Dan ia BUKAN kolom.
    expect(within(table).queryByRole('columnheader', { name: 'Dipakai' })).not.toBeInTheDocument()
  })

  /*
    Tombol keputusan berada DI FORM, bukan di baris grid — itu letaknya di Pega. Baris
    grid hanya punya tombol Update, dan itu berlaku di keempat tab.

    Dibuktikan dari offset di dalam section: tombol ber-`stsapprove` ada di 152k–166k,
    jauh SEBELUM header grid di 232k; tombol ber-`INISIAL` ada di 280k, di dalam baris.
  */
  it('baris grid hanya punya tombol Update, termasuk di tab Komite', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    const table = await screen.findByRole('table')
    expect(within(table).getAllByRole('button', { name: 'Update' }).length).toBeGreaterThan(0)
    expect(within(table).queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument()

    await clickTab(user, 'Komite Approval')
    await waitFor(() => {
      const grid = screen.getByRole('table')
      expect(within(grid).getAllByRole('button', { name: 'Update' }).length).toBeGreaterThan(0)
      expect(within(grid).queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument()
    })
  })
})

/**
 * openCommitteeForm menempuh alur Pega apa adanya: buka tab Komite, tekan Update pada
 * barisnya, tunggu form termuat. Baru sesudah itu Approve dan Reject tersedia.
 */
async function openCommitteeForm(user: ReturnType<typeof userEvent.setup>) {
  await screen.findByRole('table')
  await clickTab(user, 'Komite Approval')

  await waitFor(() =>
    expect(screen.getAllByRole('button', { name: 'Update' }).length).toBeGreaterThan(0),
  )
  await user.click(screen.getAllByRole('button', { name: 'Update' })[0]!)
  return screen.findByRole('form', { name: /Ubah Master Auto Claim/ })
}

describe('keputusan komite', () => {
  /*
    Approve dan Reject berada DI FORM, bukan di baris grid. Alurnya karena itu: Update
    pada baris -> isian termuat -> komite memutuskan atas isi yang benar-benar dilihatnya.

    Itu letaknya di Pega, dan letak itu pula yang membuat cacatnya terlihat: komite yang
    TIDAK memuat barisnya lebih dulu mengirim page form yang kosong, dan kueri UPDATE
    menulis CLIENTID/CLIENTNAME kosong apa adanya — menyetujui dapat MENGHAPUS data
    client. Di sini nilainya berasal dari baris yang dimuat, sehingga tidak dapat hilang.
  */
  it('Approve mengirim seluruh isian baris, termasuk client', async () => {
    installFetch(defaultReply(() => ({ body: { auto_claim: APPROVED, portal: 'ASM' } })))
    show()

    const user = userEvent.setup()
    const form = await openCommitteeForm(user)

    await user.click(within(form).getByRole('button', { name: 'Approve' }))

    await waitFor(() => {
      const put = calls.find((c) => c.method === 'PUT')
      expect(put).toBeDefined()
      expect(put?.url).toBe('/api/master/auto-claim/AGN003')
      expect(put?.body).toMatchObject({
        status: '1',
        id_client: 'CLI003',
        nama_client: 'TERTANGGUNG CONTOH KETIGA',
        nama_bank: 'BANK CONTOH NIAGA',
        no_rekening: '1000000001',
      })
    })
  })

  it('Reject mengirim status dua', async () => {
    installFetch(defaultReply(() => ({ body: { auto_claim: PENDING, portal: 'ASM' } })))
    show()

    const user = userEvent.setup()
    const form = await openCommitteeForm(user)

    await user.click(within(form).getByRole('button', { name: 'Reject' }))

    await waitFor(() => {
      const put = calls.find((c) => c.method === 'PUT')
      expect(put?.body).toMatchObject({ status: '2' })
    })
  })

  // Tab Komite tidak punya tombol Simpan di Pega — hanya Approve dan Reject.
  it('tidak menyediakan tombol Simpan di tab Komite', async () => {
    installFetch(defaultReply())
    show()

    const form = await openCommitteeForm(userEvent.setup())
    expect(within(form).queryByRole('button', { name: 'Simpan' })).not.toBeInTheDocument()
  })

  /*
    Tab Waiting Approval BACA SAJA: di Pega ia tidak punya satu pun tombol simpan —
    hanya `UpdateMstAutoClaim_act1` yang memuat baris ke form. Yang memutuskan adalah
    komite yang ditunjuk, lewat tabnya sendiri.
  */
  it('tab Waiting Approval membuka form baca saja', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await clickTab(user, 'Waiting Approval')

    await waitFor(() =>
      expect(screen.getAllByRole('button', { name: 'Update' }).length).toBeGreaterThan(0),
    )
    await user.click(screen.getAllByRole('button', { name: 'Update' })[0]!)

    const form = await screen.findByRole('form', { name: /Ubah Master Auto Claim/ })
    expect(within(form).queryByRole('button', { name: 'Simpan' })).not.toBeInTheDocument()
    expect(within(form).queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument()
    expect(within(form).getByRole('button', { name: 'Tutup' })).toBeInTheDocument()
    expect(within(form).getByLabelText('PCT max')).toBeDisabled()
  })
})

describe('form', () => {
  it('menyimpan perubahan MENGEMBALIKAN baris ke antrean persetujuan', async () => {
    installFetch(defaultReply(() => ({ body: { auto_claim: PENDING, portal: 'ASM' } })))
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getAllByRole('button', { name: 'Update' })[0]!)
    await screen.findByRole('form', { name: /Ubah Master Auto Claim/ })

    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const put = calls.find((c) => c.method === 'PUT')
      expect(put?.body).toMatchObject({ status: '0' })
    })
  })

  /*
    Sumber Bisnis adalah kunci baris dan nama penerimanya tidak dapat diperbarui —
    `UpdateAutoClaim-SQL.xml` tidak menyebut kolomnya. Pada mode ubah keduanya karena itu
    ditampilkan sebagai keterangan, bukan isian.
  */
  it('mengunci Sumber Bisnis saat mengubah', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getAllByRole('button', { name: 'Update' })[0]!)

    const form = await screen.findByRole('form', { name: /Ubah Master Auto Claim/ })
    expect(within(form).getByText('Tidak dapat diubah.', { exact: false })).toBeInTheDocument()
    expect(within(form).queryByRole('searchbox', { name: 'Sumber Bisnis' })).not.toBeInTheDocument()
  })

  /*
    Nilai yang tersimpan SELALU berasal dari baris yang dipilih, tidak pernah dari yang
    diketik — padanan langsung penolakan "Nama penerima klaim tidak ditemukan. Jangan
    diketik manual." pada Activity/ValidasiAutoClaim.
  */
  it('menuntut Sumber Bisnis dipilih sebelum dapat disimpan', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: /Tambah Master Auto Claim/ })
    expect(within(form).getByRole('button', { name: 'Simpan' })).toBeDisabled()

    await user.type(within(form).getByRole('searchbox', { name: 'Sumber Bisnis' }), 'leasing')
    await user.click(await screen.findByRole('button', { name: /PT. LEASING CONTOH UTAMA/ }))

    await waitFor(() =>
      expect(within(form).getByRole('button', { name: 'Simpan' })).not.toBeDisabled(),
    )
  })

  /*
    CLAIM ALLOWED, KOMITE, dan APPROVAL tidak punya isian di layar. Ketiganya diturunkan
    server — kotak yang isinya selalu diabaikan lebih buruk daripada tidak ada kotak
    sama sekali.
  */
  it('tidak menyediakan isian untuk nilai yang diturunkan server', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: /Tambah Master Auto Claim/ })
    expect(within(form).queryByLabelText(/claim allowed/i)).not.toBeInTheDocument()
    expect(within(form).queryByLabelText(/komite/i)).not.toBeInTheDocument()
    expect(within(form).queryByLabelText(/approval/i)).not.toBeInTheDocument()
  })

  // PCT max diperiksa sebagai angka 0–100 — selisih yang direncanakan terhadap Pega,
  // yang menerima teks apa pun.
  it('menolak PCT max yang bukan angka 0-100', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getAllByRole('button', { name: 'Update' })[0]!)

    const form = await screen.findByRole('form', { name: /Ubah Master Auto Claim/ })
    const pct = within(form).getByLabelText('PCT max')
    await user.clear(pct)
    await user.type(pct, '150')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText(/PCT max harus berupa angka/)).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'PUT')).toBe(false)
  })

  // Koma diterima sebagai pemisah desimal: petugas Indonesia mengetiknya begitu.
  it('menerima koma sebagai pemisah desimal pada PCT max', async () => {
    installFetch(defaultReply(() => ({ body: { auto_claim: PENDING, portal: 'ASM' } })))
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getAllByRole('button', { name: 'Update' })[0]!)

    const form = await screen.findByRole('form', { name: /Ubah Master Auto Claim/ })
    const pct = within(form).getByLabelText('PCT max')
    await user.clear(pct)
    await user.type(pct, '82,5')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const put = calls.find((c) => c.method === 'PUT')
      expect(put?.body).toMatchObject({ pct_max: '82,5' })
    })
  })
})

describe('portal entitas', () => {
  // Layar tidak menembak API sebelum portal dipilih. Backend memang menolaknya
  // (TKT-F6-002), tetapi menembaknya hanya untuk menerima penolakan akan menampilkan
  // pesan galat pada layar yang sebenarnya belum siap dibuka.
  it('tidak memuat apa pun sebelum portal dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})
