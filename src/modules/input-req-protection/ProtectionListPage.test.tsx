import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ProtectionListPage } from './ProtectionListPage'
import type { Protection, ProtectionListResponse } from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen.
 */
function protection(partial: Partial<Protection> = {}): Protection {
  return {
    nomor_proteksi: 'OPCN.26.0001',
    nomor_polis: '99.001.2026.00000001',
    nomor_klaim: '',
    tipe_proteksi: '1',
    nama_tipe_proteksi: 'General',
    tanggal_proteksi: '2026-09-16',
    keterangan: 'Permintaan contoh.',
    user_create: 'ADMINCONTOH',
    dapat_disunting: true,
    premi: false,
    ...partial,
  }
}

function listResponse(partial: Partial<ProtectionListResponse> = {}): ProtectionListResponse {
  return { proteksi: [protection()], total: 1, ...partial }
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/**
 * Master tipe proteksi, sebagaimana dikirim `GET /api/input-req-protection/tipe`.
 *
 * Kesembilannya disalin dari `POOLDATA.M_CLAIM_PROTECTION_TYPE`. Uji yang memakai daftar
 * karangan akan lulus sambil menyembunyikan salah pemetaan kode ke nama — dan salah itu
 * baru terlihat di layar pengguna.
 */
const KLAIM = {
  nomor_klaim: 'PNCN.26.0007',
  nomor_polis: '99.001.2026.00000001',
  nama_tertanggung: 'TERTANGGUNG CONTOH',
  dol: '2026-08-01',
  penyebab_kerugian: 'Kebakaran',
  nama_objek: 'OBJEK CONTOH',
  nama_cabang: 'CABANG CONTOH',
}

/**
 * Coverage klaim, sebagaimana dikirim `GET …/klaim/{nomor}/coverage`.
 *
 * Isinya DISALIN dari data nyata klaim `PNC-1452` — termasuk pengulangannya: `Resiko A`
 * muncul TIGA KALI dengan Penyebab Kerugian berbeda. Data karangan yang setiap barisnya
 * bernama unik akan membuat pemilihan lewat nama tampak berhasil, dan cacatnya baru muncul
 * di produksi tempat nama memang berulang.
 */
const COVERAGE = {
  coverage: [
    {
      id_objek: '1', id_coverage: '1', nama_objek: 'JackHugh', nama_coverage: 'Resiko A',
      penyebab_kerugian: 'ILLNESS', penyebab_kerugian_id: '12001',
    },
    {
      id_objek: '1', id_coverage: '2', nama_objek: 'JackHugh', nama_coverage: 'Resiko A',
      penyebab_kerugian: 'STORM', penyebab_kerugian_id: '12002',
    },
    {
      id_objek: '1', id_coverage: '3', nama_objek: 'JackHugh', nama_coverage: 'Katastropi',
      penyebab_kerugian: 'WINDSTORM', penyebab_kerugian_id: '12003',
    },
    {
      id_objek: '1', id_coverage: '4', nama_objek: 'JackHugh', nama_coverage: 'Resiko A',
      penyebab_kerugian: 'HURRICANE', penyebab_kerugian_id: '12004',
    },
  ],
}

/** Pilihan dropdown "Next Cause Of Loss", dari `POOLDATA.D_CAUSE_OF_LOSS`. */
const PENYEBAB = {
  penyebab_kerugian: [
    { kode: '11997', nama: 'FIRE - OPEN FLAME', kode_kerugian: 'A' },
    { kode: '12033', nama: 'WRECK REMOVAL', kode_kerugian: '' },
  ],
}

const TIPE = {
  tipe: [
    { kode: '1', nama: 'General' },
    { kode: '2', nama: 'Premi Belum Lunas' },
    { kode: '3', nama: 'Asuransi Kredit' },
    { kode: '4', nama: 'Pengkinian Data' },
    { kode: '5', nama: 'Currency Klaim' },
    { kode: '6', nama: 'Klaim >= 50 M' },
    { kode: '7', nama: 'Perubahan DOL' },
    { kode: '8', nama: 'Perubahan COL' },
    { kode: '9', nama: 'Nama Rekening Tidak Sesuai' },
  ],
}

/**
 * Memasang tiruan fetch.
 *
 * Permintaan master tipe DIJAWAB SENDIRI di sini dan tidak pernah sampai ke `answer`. Layar
 * membacanya pada setiap render, dan tanpa jawaban yang benar pilihan tipe menjadi kosong —
 * sehingga uji yang sama sekali tidak berurusan dengan master pun ikut gagal.
 *
 * Tiruan yang menjawab SATU bentuk untuk semua URL memang lebih ringkas, tetapi ia berhenti
 * dapat dipercaya begitu layar menembak lebih dari satu endpoint: yang diuji menjadi
 * kebetulan, bukan kontrak.
 *
 * Uji yang kelak perlu menguji KEGAGALAN master memasang `vi.stubGlobal` sendiri. Belum ada
 * yang membutuhkannya, dan menyiapkan sakelar untuk kebutuhan yang belum ada hanya menambah
 * bentuk yang harus dibaca.
 */
function stubFetch(answer: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })

    if (url.includes('/tipe')) return Promise.resolve(jsonResponse(200, TIPE))

    // Dokumen penunjang diperiksa SEBELUM `/klaim/`, dan urutannya menentukan.
    //
    // Jalurnya `/api/klaim/{nomor}/dokumen-penunjang`, sehingga `/klaim/` ikut cocok.
    // Bila ia lebih dulu, daftar dokumen akan menerima badan pencarian klaim — yang tidak
    // punya `data`, dan panelnya galat pada setiap uji yang membuka form.
    //
    // Sejak sakelar `FITUR_DOKUMEN_PENUNJANG_AKTIF` dinyalakan (2026-10-03), cabang ini
    // BENAR-BENAR terpakai: panelnya dirender pada form dan menembak jalur ini setiap kali
    // form dibuka. Sebelumnya ia dipasang untuk saat itu tiba.
    if (url.includes('/dokumen-penunjang')) {
      return Promise.resolve(jsonResponse(200, { data: [] }))
    }

    // Keduanya diperiksa SEBELUM `/klaim/`, dengan alasan yang sama seperti dokumen
    // penunjang di atas: jalurnya bersarang di bawah klaim, sehingga `/klaim/` ikut cocok.
    // Bila urutannya terbalik, panel pemilih menerima badan pencarian klaim — yang tidak
    // punya `coverage`, dan tabelnya kosong tanpa satu pun galat.
    if (url.includes('/coverage')) return Promise.resolve(jsonResponse(200, COVERAGE))
    if (url.includes('/penyebab-kerugian')) return Promise.resolve(jsonResponse(200, PENYEBAB))

    if (url.includes('/klaim/')) return Promise.resolve(jsonResponse(200, KLAIM))
    return Promise.resolve(answer(url, init))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProtectionListPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-30T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// Judul kolom mengikuti `Section/InboxReqProtection_Section-Section.xml` apa adanya
// (`D-13`). Uji ini menjaga kesetaraan itu: mengganti judul akan lulus uji lain tetapi
// membuat layar berbeda dari yang dikenal pengguna.
it('memakai ketujuh judul kolom yang sama dengan section rujukan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')

  // Ditanyakan lewat peran `columnheader`, bukan lewat teks: sebagian judul kolom juga
  // muncul sebagai label isian pada form, dan pencarian berbasis teks akan menemukan
  // keduanya lalu gagal karena ganda.
  const headers = screen.getAllByRole('columnheader').map((cell) => cell.textContent ?? '')

  for (const title of [
    'No Proteksi',
    'No Polis',
    'No Klaim',
    'Tipe Proteksi',
    'Tanggal Proteksi Dibuat',
    'Keterangan',
    'User Create',
  ]) {
    expect(headers.some((header) => header.includes(title))).toBe(true)
  }
})

// Aturannya dari `Section/InboxReqProtection_Section-Section.xml:8657`, yang menonaktifkan
// tautan baris ketika `.CaseID` terisi.
it('mematikan tautan baris yang sudah tertaut klaim', async () => {
  stubFetch(() =>
    jsonResponse(
      200,
      listResponse({
        proteksi: [
          protection({ nomor_proteksi: 'OPCN.26.0001', dapat_disunting: true }),
          protection({
            nomor_proteksi: 'OPCN.26.0002',
            nomor_klaim: 'PNCN.26.0007',
            dapat_disunting: false,
          }),
        ],
        total: 2,
      }),
    ),
  )
  renderPage()

  // Yang masih dapat disunting berupa tombol; yang terkunci bukan.
  expect(await screen.findByRole('button', { name: 'OPCN.26.0001' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'OPCN.26.0002' })).not.toBeInTheDocument()
  expect(screen.getByText('OPCN.26.0002')).toBeInTheDocument()
})

it('menyatakan proteksi yang belum tertaut klaim, bukan membiarkan selnya kosong', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('belum tertaut')).toBeInTheDocument()
})

// Nama tipe datang BERSAMA barisnya, dari master `M_CLAIM_PROTECTION_TYPE`. Kode yang tidak
// terdaftar di master tetap TAMPIL sebagai kodenya — bukan kosong dan bukan tanda hubung.
//
// Aturan itu yang menyelamatkan tipe '9': ia dipakai 25 baris produksi dan NOL kemunculan di
// export Pega, sehingga daftar tebakan mana pun akan melewatkannya.
it('menampilkan nama tipe dari master, dan kodenya apa adanya bila tak terdaftar', async () => {
  stubFetch(() =>
    jsonResponse(
      200,
      listResponse({
        proteksi: [
          protection({
            nomor_proteksi: 'OPCN.26.0001',
            tipe_proteksi: '7',
            nama_tipe_proteksi: 'Perubahan DOL',
          }),
          // Kode yang tidak ada di master: namanya kosong, sehingga kodenya yang tampil.
          protection({
            nomor_proteksi: 'OPCN.26.0002',
            tipe_proteksi: '99',
            nama_tipe_proteksi: '',
          }),
        ],
        total: 2,
      }),
    ),
  )
  renderPage()

  expect(await screen.findByText('Perubahan DOL')).toBeInTheDocument()
  expect(screen.getByText('99')).toBeInTheDocument()
})

// Pencarian dikerjakan SERVER: daftarnya hanya dibatasi "belum diakseptasi", sehingga
// menyaring satu halaman dari sepuluh akan memberi jawaban yang salah.
it('mengirim kata pencarian ke server, bukan menyaring di peramban', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')

  await userEvent.type(
    screen.getByLabelText('Cari No Proteksi / No Polis / No Klaim'),
    '99.001',
  )

  await waitFor(() => {
    expect(calls.some((call) => call.url.includes('cari=99.001'))).toBe(true)
  })
})

it('membuka form kosong saat tombol Input Open Protection ditekan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Input Open Protection' }))

  expect(screen.getByRole('heading', { name: 'Input Open Protection' })).toBeInTheDocument()
  // Pencocokan label PERSIS, bukan pola: judul kolom tabel memuat teks yang sama, dan pola
  // akan menemukan keduanya.
  expect(screen.getByLabelText('No Polis (dari klaim)')).toHaveValue('')
})

// Work Owner menegaskan 2026-09-24 bahwa ClaimNo dan ClaimID berisi nilai yang SAMA, dan
// server menurunkan ClaimID dari No Klaim.
//
// Dua isian yang wajib sama tetapi diketik terpisah akan berbeda cepat atau lambat, dan
// perbedaannya tidak menghasilkan galat apa pun — hanya proteksi yang menunjuk dua klaim
// berbeda. Uji ini menggagalkan pengembalian isian kedua itu.
it('tidak menanyakan ClaimID kedua kalinya, dan tidak mengirimkannya', async () => {
  let badan: Record<string, unknown> = {}

  stubFetch((_url, init) => {
    if (init?.method === 'POST') {
      badan = JSON.parse(String(init.body)) as Record<string, unknown>
      return jsonResponse(201, {})
    }
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Input Open Protection' }))

  expect(screen.queryByLabelText(/Referensi Klaim/)).not.toBeInTheDocument()

  await userEvent.type(screen.getByLabelText('No Klaim'), 'PNCN.26.0007')
  await userEvent.selectOptions(screen.getByLabelText('Tipe Proteksi'), '1')
  await userEvent.type(screen.getByLabelText('Keterangan'), 'Keterangan contoh.')
  await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

  await waitFor(() => expect(badan['nomor_klaim']).toBe('PNCN.26.0007'))
  expect(badan).not.toHaveProperty('referensi_klaim')
})

// Pelanggaran validasi dikirim backend SEKALIGUS dan ditempelkan ke kolomnya masing-masing
// (`11-CROSSCUTTING` §1.2 butir 1). Menampilkannya sebagai satu pesan di atas form akan
// membuat pengguna menebak kolom mana yang dimaksud.
it('menempelkan pelanggaran validasi ke kolomnya masing-masing', async () => {
  stubFetch((_url, init) => {
    if (init?.method === 'POST') {
      return jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Isian belum lengkap atau belum benar.',
        detail: [
          { field: 'no_klaim', pesan: 'No Klaim wajib diisi.' },
          { field: 'keterangan', pesan: 'Keterangan wajib diisi.' },
        ],
      })
    }
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Input Open Protection' }))

  // Isian wajib diisi lebih dulu supaya peramban benar-benar MENGIRIM permintaannya —
  // isian ber-`required` yang kosong ditahan peramban, dan penolakan backend tidak akan
  // pernah terjadi. Yang diuji di sini adalah penolakan BACKEND, bukan penolakan peramban.
  await userEvent.type(screen.getByLabelText('No Klaim'), 'PNCN.26.0007')
  await userEvent.selectOptions(screen.getByLabelText('Tipe Proteksi'), '7')
  await userEvent.type(screen.getByLabelText('Next Date Of Loss'), '2026-08-17')
  await userEvent.type(screen.getByLabelText('Keterangan'), 'Keterangan contoh.')

  await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

  expect(await screen.findByText('No Klaim wajib diisi.')).toBeInTheDocument()
  expect(screen.getByText('Keterangan wajib diisi.')).toBeInTheDocument()
})

// `TKT-F6-002`: permintaan tanpa portal ditolak server, dan layar tidak menembaknya lebih
// dulu hanya untuk menerima penolakan.
it('menuntun memilih entitas lebih dulu dan tidak menembak server', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})

// ── Klaim yang tidak ditemukan ───────────────────────────────────────────────────
//
// Work Owner 2026-09-27: *"kenapa ketika di test kosong terus"* pada Object Name.
//
// Sebabnya bukan Object Name-nya. Form hanya membaca `claim.data` dan TIDAK PERNAH
// memeriksa `claim.isError`, sehingga nomor klaim yang tidak ditemukan terlihat PERSIS SAMA
// dengan klaim yang ditemukan tetapi datanya kosong — keenam field turunan blank, tanpa
// satu pun pesan. Pada `PENYIMPANAN=memori`, hampir setiap nomor klaim nyata memang tidak
// ada.
//
// Kedua uji di bawah memasang `vi.stubGlobal` SENDIRI, tidak memakai `stubFetch`: tiruan
// bersama itu mencegat `/klaim/` dan selalu menjawabnya 200, persis seperti catatan di
// kepalanya sudah menyatakan.

/** stubKlaimGagal menjawab pencarian klaim dengan galat, sisanya normal. */
function stubKlaimGagal(status: number, badan: unknown) {
  vi.stubGlobal('fetch', (url: string) => {
    if (url.includes('/tipe')) return Promise.resolve(jsonResponse(200, TIPE))
    if (url.includes('/dokumen-penunjang')) {
      return Promise.resolve(jsonResponse(200, { data: [] }))
    }
    if (url.includes('/klaim/')) return Promise.resolve(jsonResponse(status, badan))
    return Promise.resolve(jsonResponse(200, listResponse()))
  })
}

async function bukaFormLaluKetik(nomor: string) {
  renderPage()
  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('button', { name: 'Input Open Protection' }))
  await userEvent.type(screen.getByLabelText('No Klaim'), nomor)
}

it('menyatakan ketika klaimnya TIDAK DITEMUKAN, bukan membiarkan field blank', async () => {
  stubKlaimGagal(404, { kode: 'tidak_ditemukan', pesan: 'Klaim tidak ditemukan.' })
  await bukaFormLaluKetik('PNC-9999')

  expect(await screen.findByText('Klaim tidak ditemukan')).toBeInTheDocument()

  // Pesannya menyebut Object Name secara eksplisit — itulah field yang ditanyakan, dan
  // menyebutnya menutup dugaan bahwa field itu yang rusak.
  expect(screen.getByText(/Object Name/)).toBeInTheDocument()
})

it('membedakan gangguan pembacaan dari klaim yang tidak ada', async () => {
  stubKlaimGagal(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' })
  await bukaFormLaluKetik('PNC-1865')

  // Nomor yang salah ketik dapat diperbaiki pengguna; gangguan pembacaan tidak. Menyamakan
  // pesannya membuat pengguna mengetik ulang nomor yang sebenarnya sudah benar.
  expect(await screen.findByText('Data klaim gagal dibaca')).toBeInTheDocument()
  expect(screen.queryByText('Klaim tidak ditemukan')).not.toBeInTheDocument()
})

// Klaim yang DITEMUKAN tetapi memang tanpa objek bukan galat: 547 dari 2.196 baris
// `T_CLAIM_PNC` tidak punya satu pun baris di `T_CLAIM_OBJECTLIST`. Keadaan itu harus
// terbedakan dari klaim yang tidak ada.
it('tidak mengeluh ketika klaim ADA tetapi tanpa Object Name', async () => {
  vi.stubGlobal('fetch', (url: string) => {
    if (url.includes('/tipe')) return Promise.resolve(jsonResponse(200, TIPE))
    if (url.includes('/dokumen-penunjang')) {
      return Promise.resolve(jsonResponse(200, { data: [] }))
    }
    if (url.includes('/klaim/')) {
      return Promise.resolve(jsonResponse(200, { ...KLAIM, nama_objek: '' }))
    }
    return Promise.resolve(jsonResponse(200, listResponse()))
  })
  await bukaFormLaluKetik('PNCN.26.0009')

  await waitFor(() =>
    expect(screen.getByLabelText('No Polis (dari klaim)')).toHaveValue(KLAIM.nomor_polis),
  )
  expect(screen.queryByText('Klaim tidak ditemukan')).not.toBeInTheDocument()
  expect(screen.queryByText('Data klaim gagal dibaca')).not.toBeInTheDocument()
})

// Panel pemilih adalah inti permintaan perubahan Cause Of Loss.
//
// Yang dijaga uji ini bukan tampilannya melainkan SATU hal: baris yang namanya sama persis
// tetap dapat dibedakan dan dipilih. Pada klaim nyata `PNC-1452`, `JackHugh / Resiko A`
// muncul tiga kali — kalau pemilihan bersandar pada nama, dua dari tiga pilihan akan menunjuk
// baris yang salah, dan tidak ada galat yang memberi tahu.
it('menampilkan SETIAP baris coverage, termasuk yang namanya berulang', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  await bukaFormLaluKetik('PNCN.26.0007')

  await userEvent.selectOptions(screen.getByLabelText('Tipe Proteksi'), '8')

  // Keempat baris tampil, dan ketiga `Resiko A` tidak dilebur menjadi satu.
  await waitFor(() => {
    expect(screen.getAllByText('Resiko A')).toHaveLength(3)
  })
  expect(screen.getByText('Katastropi')).toBeInTheDocument()

  // Kode ditampilkan berdampingan dengan deskripsinya: dua baris dapat bernama sama
  // sementara kodenya berbeda, dan akseptasi mengubah keduanya.
  expect(screen.getByText('WINDSTORM (12003)')).toBeInTheDocument()
})

// "Next Cause Of Loss" adalah DROPDOWN, bukan kotak teks (Work Owner, 2026-10-05).
//
// Teks bebas membuat salah ketik tersimpan apa adanya sebagai penyebab kerugian klaim —
// penyimpanan berhasil, layar normal, dan kodenya tidak cocok dengan master mana pun.
it('menawarkan Next Cause Of Loss sebagai pilihan, bukan isian bebas', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  await bukaFormLaluKetik('PNCN.26.0007')

  await userEvent.selectOptions(screen.getByLabelText('Tipe Proteksi'), '8')

  const dropdown = await screen.findByLabelText('Next Cause Of Loss')
  expect(dropdown.tagName).toBe('SELECT')

  // Pilihannya datang dari master, beserta LOSS_CODE-nya bila ada.
  await waitFor(() => {
    expect(screen.getByRole('option', { name: 'FIRE - OPEN FLAME (A)' })).toBeInTheDocument()
  })
  expect(screen.getByRole('option', { name: 'WRECK REMOVAL' })).toBeInTheDocument()
})

// Menekan Pilih menetapkan SASARAN perubahan, dan sasaran itulah yang dikirim.
//
// Tanpa keduanya, permintaan tersimpan tanpa tujuan — dan kegagalannya baru terlihat saat
// seseorang menyetujuinya lalu klaimnya tidak berubah.
it('mengirim id_objek dan id_coverage dari baris yang ditekan Pilih', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  await bukaFormLaluKetik('PNCN.26.0007')

  await userEvent.selectOptions(screen.getByLabelText('Tipe Proteksi'), '8')
  await waitFor(() => {
    expect(screen.getAllByRole('button', { name: 'Pilih' })).toHaveLength(4)
  })

  // Baris KETIGA — Katastropi / WINDSTORM, id_coverage 3.
  await userEvent.click(screen.getAllByRole('button', { name: 'Pilih' })[2]!)

  // Nilai "sebelum" mengikuti baris yang dipilih, bukan coverage pertama klaim.
  await waitFor(() => {
    expect(screen.getByLabelText('Cause Of Loss Dipilih')).toHaveValue('WINDSTORM (12003)')
  })

  await userEvent.selectOptions(screen.getByLabelText('Next Cause Of Loss'), '11997')
  await userEvent.type(screen.getByLabelText('Keterangan'), 'ubah penyebab')
  await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

  await waitFor(() => {
    const simpan = calls.find((c) => c.init?.method === 'POST')
    expect(simpan).toBeDefined()
    const body = JSON.parse(String(simpan?.init?.body)) as {
      detail_perubahan: { id_objek: string; id_coverage: string; penyebab_kerugian_baru: string }
    }
    expect(body.detail_perubahan.id_objek).toBe('1')
    expect(body.detail_perubahan.id_coverage).toBe('3')
    expect(body.detail_perubahan.penyebab_kerugian_baru).toBe('11997')
  })
})
