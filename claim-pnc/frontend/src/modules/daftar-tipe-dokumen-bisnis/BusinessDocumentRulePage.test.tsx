import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { BusinessDocumentRulePage } from './BusinessDocumentRulePage'

/** Grid tingkat pertama: bisnis yang SUDAH punya aturan. */
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
 * `10028` sengaja ikut — ia salah satu dari lima kode yang dilewati "Pilih semua" di Pega,
 * dan tanpa satu pun di contoh, pengecualiannya tidak dapat diuji.
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

/**
 * Aturan milik bisnis 001.
 *
 * Baris kedua sengaja ber-`detail_dokumen` "-": ia yang membuktikan penanda
 * "disembunyikan" benar-benar muncul. Baris ketiga sengaja ditandai wajib — daftar tidak
 * membawa jaminannya, dan itu pun diuji.
 */
const RULE_LIST = {
  portal: 'ASM',
  total: 3,
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
      return { body: RULE_DETAIL, status: 201 }
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

    // "Detail Tipe Dokumen Bisnis" dari Section Pega — bukan "Daftar Tipe Dokumen Bisnis"
    // yang ada di tabel menu. Keduanya ditiru apa adanya di tempatnya masing-masing.
    expect(
      screen.getByRole('heading', { name: 'Detail Tipe Dokumen Bisnis' }),
    ).toBeInTheDocument()

    await screen.findByRole('table')
  })

  it('menyebut entitas yang menjawabnya dan mengirim portal di header', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('ASM')).toBeInTheDocument()
    expect(calls[0]?.header[HEADER_PORTAL]).toBe('ASM')
  })

  it('tidak menembak server sebelum entitas dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    await screen.findByText('Portal entitas belum dipilih')
    expect(calls).toHaveLength(0)
  })

  it('membuka dokumen sebuah bisnis lewat tombol Detail', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )

    expect(await screen.findByText('Laporan Kerugian')).toBeInTheDocument()
    expect(calls.some((call) => call.url.includes('/tipe-dokumen-bisnis/bisnis/001'))).toBe(true)
  })

  it('menandai baris ber-detail dokumen "-" sebagai disembunyikan', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )

    expect(await screen.findByText('— disembunyikan')).toBeInTheDocument()
  })
})

describe('menambah aturan dokumen', () => {
  it('mengirim perkalian bisnis kali dokumen, dan menunjukkan jumlahnya lebih dulu', async () => {
    installFetch(defaultReply(() => ({ body: { ...RULE_LIST, total: 2 }, status: 201 })))
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    // Dua bisnis, satu baris dokumen → dua baris akan lahir, dan layar menyebutkannya
    // sebelum petugas menekan Simpan.
    await userEvent.click(within(form).getByRole('checkbox', { name: /Fire/ }))
    await userEvent.click(within(form).getByRole('checkbox', { name: /Marine Cargo/ }))
    expect(within(form).getByText(/2 bisnis × 1 dokumen/)).toBeInTheDocument()

    // Isian rujukan kini autocomplete ketik-cari, bukan dropdown — meniru
    // `pyAutoComplete` ber-`pyAllowFreeFormInput=true` di Pega. Yang diketik adalah NAMA;
    // kodenya dicarikan di belakang.
    await userEvent.type(within(form).getByLabelText('Tipe Dokumen'), 'REGISTER (20001)')
    await userEvent.type(
      within(form).getByLabelText('Detail Dokumen'),
      'Laporan Kerugian (40001)',
    )
    await userEvent.type(within(form).getByLabelText('Nama Dokumen'), 'Laporan Kerugian')

    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'POST')
      expect(saved?.body).toEqual({
        bisnis: ['001', '004'],
        dokumen: [
          {
            id_tipe_dokumen: '20001',
            id_object_dokumen: '',
            id_detail_dokumen: '40001',
            detail_dokumen: 'Laporan Kerugian',
            status_wajib: false,
            minimum_dokumen: 0,
          },
        ],
      })
    })
  })

  it('menyempitkan pilihan Detail Dokumen mengikuti Tipe Dokumen', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    // Saran dibaca dari <datalist>, bukan dari <select>. Ia diambil lewat id-nya karena
    // opsi di dalam datalist tidak terekspos sebagai `role="option"` yang dapat
    // diandalkan di jsdom.
    const suggestions = () =>
      Array.from(document.getElementById('detail-dokumen-0')?.children ?? []).map((node) =>
        node.getAttribute('value'),
      )

    // Sebelum tahap dipilih, ketiganya ditawarkan.
    expect(suggestions()).toHaveLength(3)

    // Setelah tahap REGISTER dipilih, hanya rincian milik tahap itu yang tersisa. Tanpa
    // penyempitan ini, petugas dapat menyimpan pasangan yang tidak akan pernah muncul di
    // layar unggah mana pun.
    await userEvent.type(within(form).getByLabelText('Tipe Dokumen'), 'REGISTER (20001)')
    expect(suggestions()).toHaveLength(2)
    expect(suggestions().some((value) => value?.includes('Berita Acara Survei'))).toBe(false)
  })

  it('menampilkan kalimat layar lama saat bisnis belum dipilih', async () => {
    installFetch(
      defaultReply(() => ({
        body: { kode: 'nama_bisnis_belum_diisi', pesan: 'Nama Bisnis belum di isi.' },
        status: 422,
      })),
    )
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama Bisnis belum di isi')).toBeInTheDocument()
  })
})

describe('pilih semua bisnis', () => {
  // Lima kode lini MBU dilewati pemilihan massal
  // (`Activity/SetAllBusiness-Act.xml:984`). Daftarnya kini datang dari konfigurasi
  // (`D-15`), tetapi perilakunya sama persis.
  it('melewati bisnis yang dikecualikan, tetapi tetap membiarkannya dipilih sendiri', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    await userEvent.click(within(form).getByRole('button', { name: 'Pilih semua' }))

    // Tiga dari empat: Personal Accident (10028) dilewati.
    expect(within(form).getByText(/3 dari 4 dipilih/)).toBeInTheDocument()

    // Tetapi ia TIDAK terkunci — pengecualiannya hanya berlaku pada tombolnya.
    const excluded = within(form).getByRole('checkbox', { name: /Personal Accident/ })
    expect(excluded).not.toBeDisabled()
    await userEvent.click(excluded)
    expect(within(form).getByText(/4 dari 4 dipilih/)).toBeInTheDocument()
  })

  // Di Pega tombolnya hanya TERLIHAT oleh operator ber-`pyPosition='NONMBU'`. Itu
  // penyembunyian tampilan, bukan kewenangan — dan ditiru sebagai penyembunyian pula.
  it('menyembunyikan tombolnya bagi operator di luar NONMBU', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/bisnis-pilihan')) {
        return { body: { ...BUSINESS_CHOICES, boleh_pilih_semua: false } }
      }
      return defaultReply()(call)
    })
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    expect(within(form).queryByRole('button', { name: 'Pilih semua' })).toBeNull()
    // Kosongkan tetap ada: ia bukan bagian dari pengecualian mana pun.
    expect(within(form).getByRole('button', { name: 'Kosongkan' })).toBeInTheDocument()
  })
})

describe('menyalin aturan ke bisnis lain', () => {
  // Inilah guna tombol Copy: aturan tidak dapat DIPINDAHKAN antar lini bisnis
  // (`BUSINESSID` tidak ikut diubah saat menyunting), sehingga menyalin adalah
  // satu-satunya cara memakai ulang susunan yang sudah ada.
  it('membawa seluruh baris bisnis asal ke form tambah, tanpa ID-nya', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(await screen.findByRole('button', { name: 'Copy' }))

    const form = await screen.findByRole('form', { name: 'Tambah Data' })

    // Kedua baris bisnis 001 ikut tersalin, sudah terisi.
    expect(within(form).getByText(/2 dokumen/)).toBeInTheDocument()
    expect(within(form).getByDisplayValue('Laporan Kerugian (40001)')).toBeInTheDocument()

    // Bisnis tujuannya BELUM dipilih — itulah yang harus diisi petugas.
    expect(within(form).getByText(/0 dari 4 dipilih/)).toBeInTheDocument()
  })

  it('menyimpan salinan sebagai baris baru pada bisnis tujuan', async () => {
    installFetch(defaultReply(() => ({ body: { ...RULE_LIST, total: 2 }, status: 201 })))
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(await screen.findByRole('button', { name: 'Copy' }))

    const form = await screen.findByRole('form', { name: 'Tambah Data' })
    await userEvent.click(within(form).getByRole('checkbox', { name: /Travel/ }))
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'POST')
      const body = saved?.body as { bisnis: string[]; dokumen: unknown[] }
      expect(body.bisnis).toEqual(['005'])
      expect(body.dokumen).toHaveLength(2)
      // Tidak satu pun membawa `id` — salinan selalu menjadi baris baru.
      expect(JSON.stringify(body.dokumen)).not.toContain('"id"')
    })
  })
})

describe('mengubah aturan dokumen', () => {
  it('memuat ulang baris dari server, bukan memakai baris dari daftar', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(
      await screen.findByRole('button', { name: 'Ubah aturan Laporan Kerugian' }),
    )

    // Inilah uji yang menjaga sebuah KEPUTUSAN: daftar TIDAK membawa jaminan, pengambilan
    // satu baris membawanya. Bila kelak seseorang "merapikan" keduanya menjadi sama, layar
    // akan menampilkan setiap baris seolah tanpa jaminan — dan itu perbedaan antara
    // dokumen yang wajib dan yang tidak.
    await screen.findByRole('form', { name: 'Update Data' })
    expect(calls.some((call) => call.url.match(/\/tipe-dokumen-bisnis\/10001$/))).toBe(true)
    expect(await screen.findByText('10009')).toBeInTheDocument()
  })

  it('menampilkan bisnis sebagai tidak dapat dipindah', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(
      await screen.findByRole('button', { name: 'Ubah aturan Laporan Kerugian' }),
    )

    const form = await screen.findByRole('form', { name: 'Update Data' })
    expect(within(form).getByText('(tidak dapat dipindah)')).toBeInTheDocument()
    expect(within(form).queryByLabelText('Nama Bisnis')).toBeNull()
  })

  it('mengirim isian yang diubah tanpa menyertakan bisnis', async () => {
    installFetch(defaultReply(() => ({ body: RULE_DETAIL })))
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(
      await screen.findByRole('button', { name: 'Ubah aturan Laporan Kerugian' }),
    )

    const form = await screen.findByRole('form', { name: 'Update Data' })
    const minimum = within(form).getByLabelText('Minimum Dokumen')
    await userEvent.clear(minimum)
    await userEvent.type(minimum, '3')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'PUT')
      expect(saved?.url).toContain('/tipe-dokumen-bisnis/10001')
      expect(saved?.body).toEqual({
        id_tipe_dokumen: '20001',
        id_object_dokumen: '30001',
        id_detail_dokumen: '40001',
        detail_dokumen: 'Laporan Kerugian',
        status_wajib: true,
        minimum_dokumen: 3,
      })
    })
  })
})

describe('jenis klaim', () => {
  it('memperingatkan bahwa wajib tanpa jenis klaim tidak wajib di klaim', async () => {
    // Baris ini ditandai wajib TETAPI daftar jaminannya kosong — keadaan yang di layar
    // lama tidak terlihat sama sekali, dan yang membuat petugas yakin telah mewajibkan
    // dokumen yang sebenarnya tetap opsional.
    installFetch((call) => {
      if (call.method === 'GET' && call.url.match(/\/tipe-dokumen-bisnis\/\d+$/)) {
        return { body: { portal: 'ASM', tipe_dokumen_bisnis: RULE_LIST.tipe_dokumen_bisnis[0] } }
      }
      return defaultReply()(call)
    })
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(
      await screen.findByRole('button', { name: 'Ubah aturan Laporan Kerugian' }),
    )

    expect(
      await screen.findByText('Wajib di sini belum berarti wajib di klaim'),
    ).toBeInTheDocument()
  })

  it('menambahkan jenis klaim lewat jalurnya sendiri', async () => {
    installFetch(defaultReply(() => ({ body: RULE_DETAIL })))
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(
      await screen.findByRole('button', { name: 'Ubah aturan Laporan Kerugian' }),
    )

    await userEvent.type(await screen.findByLabelText('Kode Jenis Klaim'), '10015')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah jenis klaim' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.url.includes('/jenis-klaim'))
      expect(saved?.method).toBe('POST')
      expect(saved?.body).toEqual({ id_jenis_klaim: '10015' })
    })
  })

  it('menyatakan bahwa jenis klaim tidak dapat dibuang', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await userEvent.click(
      within(table).getByRole('button', { name: 'Lihat dokumen bisnis Fire' }),
    )
    await userEvent.click(
      await screen.findByRole('button', { name: 'Ubah aturan Laporan Kerugian' }),
    )

    expect(
      await screen.findByText(/hanya dapat ditambahkan, tidak dapat dibuang/),
    ).toBeInTheDocument()
  })
})
