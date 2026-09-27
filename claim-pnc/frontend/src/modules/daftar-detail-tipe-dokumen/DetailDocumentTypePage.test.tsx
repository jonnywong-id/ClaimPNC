import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DetailDocumentTypePage } from './DetailDocumentTypePage'

/** Jawaban keempat daftar pilihan — SATU respons, sama seperti servernya. */
const REFERENCES = {
  portal: 'ASM',
  tipe_dokumen: [
    { id: '10001', nama: 'Dokumen Registrasi' },
    { id: '10002', nama: 'Dokumen Survey' },
  ],
  penyebab_kerugian: [
    { id: '1001', nama: 'Contoh Golongan A' },
    { id: '1004', nama: 'Contoh Golongan D' },
  ],
  objek_dokumen: [
    { id: '10001', nama: 'KTP Tertanggung' },
    { id: '10002', nama: 'Polis Asli' },
  ],
  bisnis: [
    { id: '003', nama: 'ANEKA' },
    { id: '006', nama: 'FIRE / PROPERTY' },
  ],
  tidak_tersedia: [],
}

/**
 * Jawaban daftar. Perhatikan `bisnis` SELALU kosong di sini — itu memang bentuk
 * jawabannya, karena kueri daftar tidak membaca aturan per lini bisnis.
 */
const LIST = {
  portal: 'ASM',
  total: 2,
  detail_tipe_dokumen: [
    {
      id: '100001',
      id_tipe_dokumen: '10001',
      nama_tipe_dokumen: 'Dokumen Registrasi',
      detail_dokumen: 'Formulir Laporan Kerugian',
      status_tertanggung: 'Tertanggung',
      id_penyebab_kerugian: '1001',
      keterangan_penyebab_kerugian: 'Contoh Golongan A',
      id_objek_dokumen: '10002',
      keterangan_objek_dokumen: 'Polis Asli',
      resiko: '0',
      bisnis: [],
    },
    {
      // Baris yang rujukannya sudah tidak ada di master mana pun. Ia SAH dan harus tetap
      // tampil — keterangan kosong, bukan baris yang hilang.
      id: '100005',
      id_tipe_dokumen: '19999',
      nama_tipe_dokumen: '',
      detail_dokumen: 'Dokumen Warisan Tanpa Master',
      status_tertanggung: '',
      id_penyebab_kerugian: '1999',
      keterangan_penyebab_kerugian: '',
      id_objek_dokumen: '19999',
      keterangan_objek_dokumen: '',
      resiko: '0',
      bisnis: [],
    },
  ],
}

/** Jawaban GET satu baris — di sinilah daftar bisnisnya ikut, bukan di daftar. */
const DETAIL = {
  portal: 'ASM',
  detail_tipe_dokumen: {
    id: '100001',
    id_tipe_dokumen: '10001',
    nama_tipe_dokumen: 'Dokumen Registrasi',
    detail_dokumen: 'Formulir Laporan Kerugian',
    status_tertanggung: 'Tertanggung',
    id_penyebab_kerugian: '1001',
    keterangan_penyebab_kerugian: 'Contoh Golongan A',
    id_objek_dokumen: '10002',
    keterangan_objek_dokumen: 'Polis Asli',
    resiko: '0',
    bisnis: [
      { id_bisnis: '003', nama_bisnis: 'ANEKA', status_wajib: true, minimum_dokumen: 1 },
      {
        // Bisnis yang sudah tidak ada di master — namanya kosong, kodenya tetap ada.
        id_bisnis: '099',
        nama_bisnis: '',
        status_wajib: false,
        minimum_dokumen: 2,
      },
    ],
  },
}

type Call = {
  url: string
  method: string
  body: unknown
  header: Record<string, string>
}

let calls: Call[] = []

/** reply memetakan jalur permintaan ke respons yang dikembalikan fetch tiruan. */
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

/** defaultReply melayani daftar, detail, dan daftar pilihan. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    // Diperiksa PALING DULU: jalurnya berada di bawah sub-rute yang sama dengan
    // pengambilan satu baris, dan urutan yang terbalik akan membuatnya terbaca sebagai ID.
    if (call.url.startsWith('/api/master/detail-tipe-dokumen/pilihan')) {
      return { body: REFERENCES }
    }
    if (call.method === 'GET' && call.url.includes('/detail-tipe-dokumen/')) {
      return { body: DETAIL }
    }
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return { body: DETAIL, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DetailDocumentTypePage />
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

/** Membuka form ubah pada baris 100001 dan menunggu daftar bisnisnya selesai dimuat. */
async function openEditForm() {
  await screen.findByRole('table')
  await userEvent.click(screen.getByRole('button', { name: 'Ubah Formulir Laporan Kerugian' }))

  const form = await screen.findByRole('form', { name: 'Detail Tipe Dokumen' })
  await waitFor(() => {
    expect(within(form).getByLabelText('ID Bisnis baris 1')).toHaveValue('003')
  })
  return form
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
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('DetailDocumentTypePage', () => {
  it('menampilkan daftar beserta entitas yang menjawabnya', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('Formulir Laporan Kerugian')).toBeInTheDocument()
    // Entitasnya disebut terang-terangan — "data siapa ini" tidak boleh diandaikan
    // (`R-20`).
    expect(screen.getByText('ASM')).toBeInTheDocument()
  })

  it('mengirim portal yang dipilih pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await waitFor(() => {
      expect(calls.length).toBeGreaterThan(0)
    })
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  it('menampilkan baris yang rujukannya sudah tidak ada di master', async () => {
    // Penyimpangan yang disengaja dari kueri lama, yang memakai INNER JOIN dan
    // menyembunyikannya. Baris yang hilang dari layar master tidak dapat diperbaiki
    // petugas, sementara aturannya TETAP berlaku pada klaim.
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('Dokumen Warisan Tanpa Master')).toBeInTheDocument()
  })

  it('memuat ulang satu baris saat form ubah dibuka, supaya lini bisnisnya ikut', async () => {
    // Daftar TIDAK membawa aturan bisnis. Memakai baris dari daftar akan membuat form
    // tampak seolah seluruh lini bisnisnya sudah dihapus — dan menyimpannya benar-benar
    // menghapusnya.
    installFetch(defaultReply())
    show()

    const form = await openEditForm()

    expect(within(form).getByLabelText('ID Bisnis baris 1')).toHaveValue('003')
    expect(within(form).getByLabelText('ID Bisnis baris 2')).toHaveValue('099')
    expect(
      calls.some((call) => call.method === 'GET' && call.url.includes('/detail-tipe-dokumen/100001')),
    ).toBe(true)
  })

  it('mengisi form dengan KETERANGAN pada dua isian rujukan, bukan kodenya', async () => {
    // Inilah yang dilihat petugas di Pega: `Section/BrowseListDetailTypeDocument-Section.xml`
    // mengikat isian "Dokumen kolom ID" ke TempDTDoc.DOC_COL_INFO dan "Objek Dokumen" ke
    // TempDTDoc.OBJ_DOC_DESC. Kodenya hanya target tersembunyi.
    //
    // "ID Tipe Dokumen" sebaliknya memang berisi kode — labelnya pun menyebutnya demikian.
    installFetch(defaultReply())
    show()

    const form = await openEditForm()

    expect(within(form).getByLabelText('ID Tipe Dokumen')).toHaveValue('10001')
    expect(within(form).getByLabelText('Dokumen kolom ID')).toHaveValue('Contoh Golongan A')
    expect(within(form).getByLabelText('Objek Dokumen')).toHaveValue('Polis Asli')
  })

  it('menampilkan pasangan kode dan keterangan sebagai petunjuk', async () => {
    // Petunjuknya menggantikan peran daftar autocomplete Pega setelah pilihannya ditutup:
    // pada isian berisi kode ditampilkan namanya, pada isian berisi keterangan ditampilkan
    // kodenya.
    installFetch(defaultReply())
    show()

    const form = await openEditForm()

    expect(within(form).getByText('Tipe dokumen: Dokumen Registrasi')).toBeInTheDocument()
    expect(within(form).getByText('Kode penyebab kerugian: 1001')).toBeInTheDocument()
    expect(within(form).getByText('Kode objek dokumen: 10002')).toBeInTheDocument()
  })

  it('mengirim kode DAN keterangan berpasangan saat disimpan', async () => {
    installFetch(defaultReply(() => ({ body: DETAIL })))
    show()

    const form = await openEditForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'PUT')).toBe(true)
    })
    const saved = calls.find((call) => call.method === 'PUT')?.body as Record<string, unknown>
    expect(saved.keterangan_penyebab_kerugian).toBe('Contoh Golongan A')
    expect(saved.id_penyebab_kerugian).toBe('1001')
    expect(saved.keterangan_objek_dokumen).toBe('Polis Asli')
    expect(saved.id_objek_dokumen).toBe('10002')
  })

  it('mengirim keterangan yang diketik bebas TANPA kode', async () => {
    // `pyAllowFreeFormInput=true` di layar lama mengizinkannya, dan kedua keterangan punya
    // kolomnya sendiri sehingga isian itu tersimpan utuh. Membuangnya karena kodenya tidak
    // ada berarti menghilangkan apa yang baru saja diketik petugas.
    installFetch(defaultReply(() => ({ body: DETAIL })))
    show()

    const form = await openEditForm()
    const field = within(form).getByLabelText('Objek Dokumen')
    await userEvent.clear(field)
    await userEvent.type(field, 'Objek yang tidak ada di master')
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'PUT')).toBe(true)
    })
    const saved = calls.find((call) => call.method === 'PUT')?.body as Record<string, unknown>
    expect(saved.keterangan_objek_dokumen).toBe('Objek yang tidak ada di master')
    expect(saved.id_objek_dokumen).toBe('')
  })

  it('mengganti SELURUH daftar bisnis saat disimpan, bukan menambahinya', async () => {
    // Grid mengirim susunan akhir yang dikehendaki petugas, dan tidak ada satu pun penanda
    // di sana yang menyatakan baris mana yang baru, mana yang berubah, dan mana yang
    // dibuang.
    installFetch(defaultReply(() => ({ body: DETAIL })))
    show()

    const form = await openEditForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Hapus lini bisnis baris 2' }))
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'PUT')).toBe(true)
    })
    const saved = calls.find((call) => call.method === 'PUT')?.body as {
      bisnis: { id_bisnis: string }[]
    }
    expect(saved.bisnis).toHaveLength(1)
    expect(saved.bisnis[0]?.id_bisnis).toBe('003')
  })

  it('mengirim status wajib sebagai boolean, bukan teks', async () => {
    // Kontraknya boolean; yang TERSIMPAN teks "Ya"/"Tidak". Server yang menjembataninya,
    // dan layar tidak boleh ikut mengirim teksnya.
    installFetch(defaultReply(() => ({ body: DETAIL })))
    show()

    const form = await openEditForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'PUT')).toBe(true)
    })
    const saved = calls.find((call) => call.method === 'PUT')?.body as {
      bisnis: { status_wajib: unknown }[]
    }
    expect(saved.bisnis[0]?.status_wajib).toBe(true)
  })

  it('membuang baris bisnis yang kodenya belum diisi', async () => {
    // Grid selalu menyisakan baris yang baru ditambahkan tetapi belum diisi. Menyimpannya
    // berarti menulis aturan yang tidak menunjuk lini bisnis mana pun.
    installFetch(defaultReply(() => ({ body: DETAIL })))
    show()

    const form = await openEditForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah Bisnis' }))
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'PUT')).toBe(true)
    })
    const saved = calls.find((call) => call.method === 'PUT')?.body as {
      bisnis: unknown[]
    }
    expect(saved.bisnis).toHaveLength(2)
  })

  it('menerima seluruh isian kosong, mengikuti layar lama yang tanpa validasi', async () => {
    installFetch(defaultReply(() => ({ body: DETAIL, status: 201 })))
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Detail Tipe Dokumen' })
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'POST')).toBe(true)
    })
  })

  it('menolak Resiko yang bukan angka sebelum permintaan dikirim', async () => {
    // Ini bentuk kolom, bukan aturan bisnis: `SetTypePDFAdjustment` membacanya dengan
    // `@toDecimal(.RISK)`, dan teks yang bukan angka akan terbaca NOL di sana tanpa satu
    // pun tanda.
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Detail Tipe Dokumen' })
    await userEvent.type(within(form).getByLabelText('Resiko'), 'abc')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Resiko hanya boleh berisi angka.')).toBeInTheDocument()
    expect(calls.some((call) => call.method === 'POST')).toBe(false)
  })

  it('tetap dapat menyimpan meski sebagian daftar pilihan gagal dimuat', async () => {
    // Keempat kode boleh diketik sendiri, sehingga daftar yang gagal dimuat hanya
    // menghilangkan kenyamanan memilih — bukan kemampuan menyimpan.
    installFetch((call) => {
      if (call.url.startsWith('/api/master/detail-tipe-dokumen/pilihan')) {
        return { body: { ...REFERENCES, tipe_dokumen: [], tidak_tersedia: ['tipe_dokumen'] } }
      }
      if (call.method === 'GET') return { body: LIST }
      return { body: DETAIL, status: 201 }
    })
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(
      await screen.findByText(/Saran untuk isian Tipe Dokumen tidak tersedia/),
    ).toBeInTheDocument()

    const form = screen.getByRole('form', { name: 'Detail Tipe Dokumen' })
    await userEvent.type(within(form).getByLabelText('ID Tipe Dokumen'), '10001')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'POST')).toBe(true)
    })
  })

  it('tidak menembak server sebelum portal dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('menjelaskan penolakan portal yang belum siap', async () => {
    installFetch(() => ({
      body: { kode: 'portal_belum_siap', pesan: 'portal belum siap' },
      status: 503,
    }))
    show()

    expect(await screen.findByText('Basis data entitas ini belum tersedia')).toBeInTheDocument()
  })
})
