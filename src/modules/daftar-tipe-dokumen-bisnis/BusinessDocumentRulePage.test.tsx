import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { BusinessDocumentRulePage } from './BusinessDocumentRulePage'

/** Daftar utama: bisnis yang SUDAH punya aturan. */
const BUSINESS_LIST = {
  portal: 'ASM',
  total: 2,
  boleh_pilih_semua: true,
  bisnis: [
    { id: '001', nama_bisnis: 'Fire', dikecualikan_pilih_semua: false },
    { id: '004', nama_bisnis: 'Marine Cargo', dikecualikan_pilih_semua: false },
  ],
}

/**
 * Daftar pilihan: SELURUH bisnis, termasuk yang belum punya aturan.
 *
 * `10028` sengaja ikut — ia salah satu dari lima kode yang dilewati pemilihan massal di
 * Pega, dan tanpa satu pun di contoh, pengecualiannya tidak dapat diuji.
 */
const BUSINESS_CHOICES = {
  portal: 'ASM',
  total: 4,
  boleh_pilih_semua: true,
  bisnis: [
    { id: '001', nama_bisnis: 'Fire', dikecualikan_pilih_semua: false },
    { id: '004', nama_bisnis: 'Marine Cargo', dikecualikan_pilih_semua: false },
    { id: '005', nama_bisnis: 'Travel', dikecualikan_pilih_semua: false },
    { id: '10028', nama_bisnis: 'Personal Accident', dikecualikan_pilih_semua: true },
  ],
}

const DOCUMENT_TYPES = {
  portal: 'ASM',
  total: 2,
  pilihan: [
    { id: '20001', nama: 'REGISTER', id_induk: '' },
    { id: '20003', nama: 'SURVEY', id_induk: '' },
  ],
}

const DETAIL_DOCUMENTS = {
  portal: 'ASM',
  total: 3,
  pilihan: [
    { id: '40001', nama: 'Laporan Kerugian', id_induk: '20001' },
    { id: '40002', nama: 'Foto Lokasi Kejadian', id_induk: '20001' },
    { id: '40003', nama: 'Berita Acara Survei', id_induk: '20003' },
  ],
}

const OBJECT_DOCUMENTS = {
  portal: 'ASM',
  total: 1,
  pilihan: [{ id: '30001', nama: 'Bangunan', id_induk: '' }],
}

/** Aturan milik bisnis 001 — dua baris, supaya penyimpanan jamak dapat diuji. */
const RULE_LIST = {
  portal: 'ASM',
  total: 2,
  tipe_dokumen_bisnis: [
    {
      id: '10001',
      id_bisnis: '001',
      nama_bisnis: 'Fire',
      id_tipe_dokumen: '20001',
      tipe_dokumen: 'REGISTER',
      id_object_dokumen: '30001',
      object_dokumen: 'Bangunan',
      id_detail_dokumen: '40001',
      detail_dokumen: 'Laporan Kerugian',
      status_wajib: true,
      minimum_dokumen: 1,
      jenis_klaim: [],
    },
    {
      id: '10002',
      id_bisnis: '001',
      nama_bisnis: 'Fire',
      id_tipe_dokumen: '20003',
      tipe_dokumen: 'SURVEY',
      id_object_dokumen: '',
      object_dokumen: '',
      id_detail_dokumen: '40003',
      detail_dokumen: '-',
      status_wajib: false,
      minimum_dokumen: 0,
      jenis_klaim: [],
    },
  ],
}

/** Satu baris LENGKAP dengan jaminannya — hanya pengambilan satu baris yang membawanya. */
const RULE_DETAIL = {
  portal: 'ASM',
  tipe_dokumen_bisnis: { ...RULE_LIST.tipe_dokumen_bisnis[0], jenis_klaim: ['10009'] },
}

type Call = { url: string; method: string; body: unknown; header: Record<string, string> }

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

/** defaultReply melayani seluruh jalur baca; mutasi dijawab pemanggil. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.method !== 'GET') {
      if (mutation) return mutation(call)
      return { body: RULE_LIST, status: 201 }
    }
    if (call.url.startsWith('/api/master/bisnis-pilihan')) return { body: BUSINESS_CHOICES }
    if (call.url.startsWith('/api/master/tipe-dokumen-pilihan')) return { body: DOCUMENT_TYPES }
    if (call.url.startsWith('/api/master/detail-dokumen-pilihan')) return { body: DETAIL_DOCUMENTS }
    if (call.url.startsWith('/api/master/objek-dokumen-pilihan')) return { body: OBJECT_DOCUMENTS }
    if (call.url.includes('/tipe-dokumen-bisnis/bisnis/')) return { body: RULE_LIST }
    if (call.url.match(/\/tipe-dokumen-bisnis\/\d+$/)) return { body: RULE_DETAIL }
    return { body: BUSINESS_LIST }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <BusinessDocumentRulePage />
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

/** openUbah membuka layar Ubah bisnis 001 dan mengembalikan form-nya. */
async function openUbah(mutation?: (call: Call) => Reply) {
  installFetch(defaultReply(mutation))
  show()
  await userEvent.click(await screen.findByRole('button', { name: 'Ubah dokumen bisnis Fire' }))
  return screen.findByRole('form', { name: 'Update Data' })
}

/** openTambah membuka layar Tambah dan mengembalikan form-nya. */
async function openTambah(mutation?: (call: Call) => Reply) {
  installFetch(defaultReply(mutation))
  show()
  await userEvent.click(await screen.findByRole('button', { name: 'Tambah' }))
  return screen.findByRole('form', { name: 'Tambah Data' })
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

describe('daftar Tipe Dokumen Bisnis', () => {
  it('memakai judul layar lama, bukan judul butir menu', async () => {
    installFetch(defaultReply())
    show()

    expect(
      await screen.findByRole('heading', { name: 'Detail Tipe Dokumen Bisnis' }),
    ).toBeInTheDocument()
  })

  it('menyebut entitas yang menjawabnya dan mengirim portal di header', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Fire')
    expect(screen.getByText('ASM')).toBeInTheDocument()
    expect(calls[0]?.header[HEADER_PORTAL]).toBe('ASM')
  })

  it('tidak menembak server sebelum entitas dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  /**
   * Kedua tombol inilah yang ADA di layar lama. Versi pertama modul ini memakai satu
   * tombol "Detail" yang membuka grid tingkat kedua — bentuk yang saya karang sendiri.
   */
  it('menawarkan Ubah dan Copy pada setiap bisnis, bukan Detail', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('button', { name: 'Ubah dokumen bisnis Fire' })).toBeVisible()
    expect(within(table).getByRole('button', { name: 'Copy dokumen bisnis Fire' })).toBeVisible()
    expect(within(table).queryByRole('button', { name: /Detail/ })).toBeNull()
  })
})

describe('layar Ubah', () => {
  /**
   * Inti koreksi atas bentuk sebelumnya: Ubah membuka SELURUH baris bisnis itu dalam satu
   * form, bukan satu baris per layar.
   */
  it('memuat seluruh baris bisnis itu sekaligus', async () => {
    const form = await openUbah()

    expect(within(form).getByRole('textbox', { name: 'Nama File baris 1' })).toHaveValue(
      'Laporan Kerugian',
    )
    expect(within(form).getByRole('textbox', { name: 'Nama File baris 2' })).toHaveValue('-')
  })

  /** Urutan kolom ditiru apa adanya dari layar yang berjalan. */
  it('memakai urutan kolom dan label layar lama', async () => {
    const form = await openUbah()

    const headers = within(form)
      .getAllByRole('columnheader')
      .map((cell) => cell.textContent?.trim())
    expect(headers.slice(0, 6)).toEqual([
      'Tipe Dokumen',
      'Detail Dokumen',
      'Object Dokumen',
      'Nama File',
      'Status Wajib',
      'Minimum Dokumen',
    ])
  })

  /** Dropdown YA/TIDAK, bukan kotak centang — lihat komentarnya di DocumentRowsTable. */
  it('menyajikan Status Wajib sebagai pilihan YA atau TIDAK', async () => {
    const form = await openUbah()

    expect(within(form).getByRole('combobox', { name: 'Status Wajib baris 1' })).toHaveValue('YA')
    expect(within(form).getByRole('combobox', { name: 'Status Wajib baris 2' })).toHaveValue(
      'TIDAK',
    )
  })

  it('menyimpan seluruh baris sekaligus ke alamat bisnisnya', async () => {
    const form = await openUbah(() => ({ body: RULE_LIST }))

    const namaFile = within(form).getByRole('textbox', { name: 'Nama File baris 1' })
    await userEvent.clear(namaFile)
    await userEvent.type(namaFile, 'Laporan Kerugian Final')
    await userEvent.selectOptions(
      within(form).getByRole('combobox', { name: 'Status Wajib baris 2' }),
      'YA',
    )
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((call) => call.method === 'PUT')).toBe(true))
    const saved = calls.find((call) => call.method === 'PUT')
    expect(saved?.url).toBe('/api/master/tipe-dokumen-bisnis/bisnis/001')

    const body = saved?.body as { dokumen: Array<Record<string, unknown>> }
    // Kedua baris membawa ID-nya, sehingga keduanya DIPERBARUI — bukan disisipkan ulang.
    expect(body.dokumen.map((row) => row['id'])).toEqual(['10001', '10002'])
    expect(body.dokumen[0]?.['detail_dokumen']).toBe('Laporan Kerugian Final')
    expect(body.dokumen[1]?.['status_wajib']).toBe(true)
    // Bisnisnya TIDAK ikut dikirim: ia ada di alamat, dan tidak dapat diubah.
    expect(body).not.toHaveProperty('bisnis')
  })

  /**
   * Satu penekanan Simpan dapat memperbarui baris lama DAN menambah baris baru — bentuk
   * yang sama dengan sentinel `UnknownID` di sistem lama.
   */
  it('mengirim baris baru tanpa id berdampingan dengan baris lama', async () => {
    const form = await openUbah(() => ({ body: RULE_LIST }))

    await userEvent.click(within(form).getByRole('button', { name: 'Tambah baris' }))
    await userEvent.type(
      within(form).getByRole('textbox', { name: 'Nama File baris 3' }),
      'Dokumen Baru',
    )
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((call) => call.method === 'PUT')).toBe(true))
    const body = (calls.find((call) => call.method === 'PUT')?.body ?? {}) as {
      dokumen: Array<Record<string, unknown>>
    }
    expect(body.dokumen).toHaveLength(3)
    expect(body.dokumen[2]?.['id']).toBe('')
    expect(body.dokumen[2]?.['detail_dokumen']).toBe('Dokumen Baru')
  })

  it('menyatakan bahwa lini bisnis tidak dapat dipindah', async () => {
    const form = await openUbah()

    expect(within(form).getByText(/tidak dapat diubah/)).toBeInTheDocument()
    expect(within(form).queryByRole('combobox', { name: /^Nama Bisnis/ })).toBeNull()
  })

  it('menyempitkan Detail Dokumen mengikuti Tipe Dokumen pada baris yang sama', async () => {
    const form = await openUbah()

    const tipe = within(form).getByRole('combobox', { name: 'Tipe Dokumen baris 1' })
    await userEvent.clear(tipe)
    await userEvent.type(tipe, 'SURVEY (20003)')

    // Daftar pilihan baris 1 kini hanya memuat rincian milik tahap SURVEY.
    const options = Array.from(
      document.querySelectorAll<HTMLOptionElement>('#detail-dokumen-0 option'),
    ).map((option) => option.value)
    expect(options).toEqual(['Berita Acara Survei (40003)'])

    // Rincian tahap sebelumnya ikut dikosongkan, supaya pasangan yang tidak pernah muncul
    // di layar unggah mana pun tidak ikut tersimpan.
    expect(within(form).getByRole('combobox', { name: 'Detail Dokumen baris 1' })).toHaveValue('')
  })

  it('memuat jenis klaim lewat jalurnya sendiri, per baris', async () => {
    const form = await openUbah()

    await userEvent.click(within(form).getByRole('button', { name: 'Jenis Klaim baris 10001' }))

    expect(await screen.findByText('10009')).toBeInTheDocument()
    expect(calls.some((call) => call.url.includes('/tipe-dokumen-bisnis/10001'))).toBe(true)
  })

  it('menambahkan jenis klaim tanpa menyentuh penyimpanan baris', async () => {
    const form = await openUbah(() => ({ body: RULE_DETAIL }))
    await userEvent.click(within(form).getByRole('button', { name: 'Jenis Klaim baris 10001' }))

    await userEvent.type(await screen.findByLabelText('Kode Jenis Klaim'), '10012')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah jenis klaim' }))

    await waitFor(() => expect(calls.some((call) => call.method === 'POST')).toBe(true))
    const added = calls.find((call) => call.method === 'POST')
    expect(added?.url).toBe('/api/master/tipe-dokumen-bisnis/10001/jenis-klaim')
    expect(added?.body).toEqual({ id_jenis_klaim: '10012' })
  })

  it('menyatakan bahwa jenis klaim tidak dapat dibuang', async () => {
    const form = await openUbah()
    await userEvent.click(within(form).getByRole('button', { name: 'Jenis Klaim baris 10001' }))

    expect(
      await screen.findByText(/hanya dapat ditambahkan, tidak dapat dibuang/),
    ).toBeInTheDocument()
  })
})

describe('layar Tambah', () => {
  /**
   * Nama Bisnis adalah baris autocomplete yang ditambah satu per satu, bukan grid 206
   * kotak centang — lihat komentarnya di BusinessDocumentRuleCreateForm.
   */
  it('mengisi Nama Bisnis lewat baris autocomplete yang dapat ditambah', async () => {
    const form = await openTambah()

    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Nama Bisnis baris 1' }),
      'Fire (001)',
    )
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah' }))
    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Nama Bisnis baris 2' }),
      'Travel (005)',
    )

    expect(within(form).getByText(/2 bisnis × 1 dokumen/)).toBeInTheDocument()
  })

  it('mengirim perkalian bisnis kali dokumen', async () => {
    const form = await openTambah(() => ({ body: RULE_LIST, status: 201 }))

    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Nama Bisnis baris 1' }),
      'Fire (001)',
    )
    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Tipe Dokumen baris 1' }),
      'REGISTER (20001)',
    )
    await userEvent.type(
      within(form).getByRole('textbox', { name: 'Nama File baris 1' }),
      'Laporan Kerugian',
    )
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((call) => call.method === 'POST')).toBe(true))
    const body = calls.find((call) => call.method === 'POST')?.body as {
      bisnis: string[]
      dokumen: Array<Record<string, unknown>>
    }
    expect(body.bisnis).toEqual(['001'])
    expect(body.dokumen[0]?.['id_tipe_dokumen']).toBe('20001')
    // Baris baru TIDAK membawa id — jalur tambah memang selalu menyisipkan.
    expect(body.dokumen[0]).not.toHaveProperty('id')
  })

  /**
   * Teks di luar master boleh diketik dan tidak terhapus sendiri. Versi pertama form ini
   * menurunkan nilai isian dari kodenya, sehingga setiap huruf yang belum cocok
   * mengosongkan isiannya — dan isian itu menjadi mustahil diisi.
   */
  it('membiarkan teks di luar daftar diketik tanpa menghapusnya sendiri', async () => {
    const form = await openTambah()

    const tipe = within(form).getByRole('combobox', { name: 'Tipe Dokumen baris 1' })
    await userEvent.type(tipe, 'REG')
    expect(tipe).toHaveValue('REG')
  })

  it('melewati bisnis yang dikecualikan pada pemilihan massal, tetapi tetap membiarkannya diketik', async () => {
    const form = await openTambah()

    await userEvent.click(within(form).getByRole('button', { name: 'Tamban semua bisnis NONMBU' }))

    const typed = within(form)
      .getAllByRole('combobox', { name: /^Nama Bisnis baris/ })
      .map((input) => (input as HTMLInputElement).value)
    expect(typed).toEqual(['Fire (001)', 'Marine Cargo (004)', 'Travel (005)'])

    // Bisnis yang sama tetap dapat diketik sendiri — pengecualiannya hanya pada tombolnya.
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah' }))
    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Nama Bisnis baris 4' }),
      'Personal Accident (10028)',
    )
    expect(within(form).getByText(/4 bisnis × 1 dokumen/)).toBeInTheDocument()
  })

  it('menyembunyikan tombol massal bagi operator di luar NONMBU', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/bisnis-pilihan')) {
        return { body: { ...BUSINESS_CHOICES, boleh_pilih_semua: false } }
      }
      return defaultReply()(call)
    })
    show()
    await userEvent.click(await screen.findByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    expect(within(form).queryByRole('button', { name: 'Tamban semua bisnis NONMBU' })).toBeNull()
  })

  it('menampilkan pesan layar lama saat bisnis belum diisi', async () => {
    const form = await openTambah(() => ({
      body: { kode: 'nama_bisnis_belum_diisi', pesan: 'x' },
      status: 422,
    }))
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama Bisnis belum di isi')).toBeInTheDocument()
  })
})

describe('layar Copy', () => {
  /**
   * Copy membawa susunan bisnis asal ke form Tambah TANPA ID-nya — penyimpanan berikutnya
   * karena itu menerbitkan baris baru alih-alih menimpa baris asalnya.
   */
  it('membawa seluruh baris bisnis asal, tanpa ID dan tanpa bisnisnya', async () => {
    installFetch(defaultReply())
    show()
    await userEvent.click(await screen.findByRole('button', { name: 'Copy dokumen bisnis Fire' }))

    const form = await screen.findByRole('form', { name: 'Tambah Data' })
    expect(within(form).getByRole('textbox', { name: 'Nama File baris 1' })).toHaveValue(
      'Laporan Kerugian',
    )
    // Bisnis tujuan dikosongkan: menyalin ke bisnis yang sama tidak punya arti apa pun.
    expect(within(form).getByRole('combobox', { name: 'Nama Bisnis baris 1' })).toHaveValue('')
  })

  it('menyimpan salinan sebagai baris baru pada bisnis tujuan', async () => {
    installFetch(defaultReply(() => ({ body: RULE_LIST, status: 201 })))
    show()
    await userEvent.click(await screen.findByRole('button', { name: 'Copy dokumen bisnis Fire' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Nama Bisnis baris 1' }),
      'Travel (005)',
    )
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((call) => call.method === 'POST')).toBe(true))
    const posted = calls.find((call) => call.method === 'POST')
    expect(posted?.url).toBe('/api/master/tipe-dokumen-bisnis')
    const body = posted?.body as { bisnis: string[]; dokumen: Array<Record<string, unknown>> }
    expect(body.bisnis).toEqual(['005'])
    expect(body.dokumen).toHaveLength(2)
    expect(body.dokumen.every((row) => !('id' in row))).toBe(true)
  })
})

describe('galat', () => {
  it('menampilkan pesan khusus saat portal belum dipilih di server', async () => {
    installFetch(() => ({ body: { kode: 'portal_tidak_disebut', pesan: 'x' }, status: 400 }))
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
  })

  it('menampilkan pesan khusus saat basis data entitas belum siap', async () => {
    installFetch(() => ({ body: { kode: 'portal_belum_siap', pesan: 'x' }, status: 503 }))
    show()

    expect(await screen.findByText('Basis data entitas ini belum tersedia')).toBeInTheDocument()
  })

  it('memakai pesan server pada galat lain', async () => {
    installFetch(() => ({
      body: { kode: 'galat_internal', pesan: 'Basis data sibuk.' },
      status: 500,
    }))
    show()

    expect(await screen.findByText('Daftar tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Basis data sibuk.')).toBeInTheDocument()
  })

  /**
   * Baris yang hilang sejak layar dibuka membatalkan SELURUH penyimpanan — tidak ada
   * sebagian yang tersimpan, dan kalimatnya mengatakan itu.
   */
  it('menyebut bahwa tidak ada yang tersimpan saat sebuah baris hilang', async () => {
    const form = await openUbah(() => ({
      body: { kode: 'tipe_dokumen_bisnis_tidak_ditemukan', pesan: 'x' },
      status: 404,
    }))
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Baris aturan sudah tidak ada')).toBeInTheDocument()
    expect(screen.getByText(/tidak ada yang tersimpan/)).toBeInTheDocument()
  })
})

/**
 * Inilah yang menyebabkan laporan "tersimpan tetapi tidak muncul, tanpa pesan galat" pada
 * 2026-10-04.
 *
 * Isian autocomplete boleh diisi teks bebas (meniru `pyAllowFreeFormInput=true`), tetapi
 * teks yang tidak cocok menghasilkan KODE KOSONG — dan baris berkode kosong tersimpan lalu
 * hilang dari seluruh layar. Penandaannya wajib terbaca sebelum Simpan ditekan.
 */
describe('isian yang belum mengenali kodenya', () => {
  it('memperingatkan saat Tipe Dokumen diketik tetapi tidak cocok', async () => {
    installFetch(defaultReply())
    show()
    await userEvent.click(await screen.findByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Tipe Dokumen baris 1' }),
      'REGIS',
    )

    expect(within(form).getByText(/tidak akan tersimpan/)).toBeInTheDocument()
  })

  it('tidak memperingatkan setelah teksnya cocok dengan daftarnya', async () => {
    installFetch(defaultReply())
    show()
    await userEvent.click(await screen.findByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Tipe Dokumen baris 1' }),
      'REGISTER (20001)',
    )

    expect(within(form).queryByText(/tidak akan tersimpan/)).toBeNull()
  })

  it('memperingatkan saat Nama Bisnis diketik tetapi tidak cocok', async () => {
    installFetch(defaultReply())
    show()
    await userEvent.click(await screen.findByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    await userEvent.type(
      within(form).getByRole('combobox', { name: 'Nama Bisnis baris 1' }),
      'Fir',
    )

    expect(within(form).getByText(/akan dilewati/)).toBeInTheDocument()
  })

  it('juga memperingatkan di layar Ubah', async () => {
    const form = await openUbah()

    const tipe = within(form).getByRole('combobox', { name: 'Tipe Dokumen baris 1' })
    await userEvent.clear(tipe)
    await userEvent.type(tipe, 'bukan tahap mana pun')

    expect(within(form).getByText(/tidak akan tersimpan/)).toBeInTheDocument()
  })
})

/**
 * Pencarian adalah SELISIH TERENCANA — layar lama tidak punya kotak cari. Diminta Work
 * Owner 2026-10-05 karena daftarnya memuat ratusan lini bisnis.
 */
describe('pencarian daftar bisnis', () => {
  it('menyaring menurut nama maupun kodenya', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByText('Fire')

    const box = screen.getByRole('searchbox', { name: /Cari nama bisnis/ })

    await userEvent.type(box, 'Marine')
    expect(screen.queryByText('Fire')).toBeNull()
    expect(screen.getByText('Marine Cargo')).toBeInTheDocument()

    await userEvent.clear(box)
    await userEvent.type(box, '001')
    expect(screen.getByText('Fire')).toBeInTheDocument()
    expect(screen.queryByText('Marine Cargo')).toBeNull()
  })
})
