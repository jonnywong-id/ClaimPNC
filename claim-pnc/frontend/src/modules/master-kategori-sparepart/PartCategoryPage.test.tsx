import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { PartCategoryPage } from './PartCategoryPage'

/** Kategori yang sudah disetujui. */
const APPROVED = {
  id_kategori_sparepart: '1',
  nama_kategori_sparepart: 'ENGINE',
  status: '1',
  status_label: 'Approve',
}

/** Kategori yang menunggu keputusan. */
const PENDING = {
  id_kategori_sparepart: '4',
  nama_kategori_sparepart: 'ELECTRICAL',
  status: '0',
  status_label: 'Waiting Approval',
}

/**
 * Kategori yang DITOLAK.
 *
 * Ia yang membuat perilaku paling mengejutkan pada modul ini dapat diuji: namanya tetap
 * memblokir pemakaian nama itu, padahal barisnya tidak terlihat dari tab mana pun selain
 * tab Reject.
 */
const REJECTED = {
  id_kategori_sparepart: '6',
  nama_kategori_sparepart: 'ATTACHMENT',
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

/** defaultReply melayani daftar; mutasi dijawab pemanggil. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.method === 'GET') {
      const status = new URL(call.url, 'http://x').searchParams.get('status') ?? '1'

      let rows: unknown[] = []
      if (status === '1') rows = [APPROVED]
      if (status === '0') rows = [PENDING]
      if (status === '2') rows = [REJECTED]

      return { body: { kategori_sparepart: rows, status, portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { kategori_sparepart: APPROVED, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PartCategoryPage />
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

/** listCalls mengambil permintaan daftar saja. */
function listCalls() {
  return calls.filter(
    (c) => c.method === 'GET' && c.url.startsWith('/api/master/kategori-sparepart?'),
  )
}

/**
 * tab mengambil tombol tab menurut captionnya.
 *
 * Dicari DI DALAM bilah tab, bukan di seluruh halaman: caption tabnya — "Approve" dan
 * "Reject" — adalah caption Pega yang memang ditiru (D-13), dan kata yang sama muncul pada
 * tombol keputusan.
 */
function tab(name: string) {
  return within(
    screen.getByRole('navigation', { name: 'Tab Master Kategori Sparepart' }),
  ).getByRole('button', { name })
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

describe('daftar master kategori sparepart', () => {
  // Grid Pega hanya punya DUA kolom, dan tabelnya memang hanya punya tiga — yang ketiga
  // adalah status, yang sudah menjadi tab.
  it('menampilkan judul dan kedua kolom grid Pega', async () => {
    installFetch(defaultReply())
    show()

    expect(
      screen.getByRole('heading', { name: 'Master Kategori Sparepart' }),
    ).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const header of ['ID Kategori Sparepart', 'Nama Kategori Sparepart']) {
      expect(within(table).getByRole('columnheader', { name: header })).toBeInTheDocument()
    }
  })

  // Tabelnya tidak punya kolom pencatat pelaku maupun stempel waktu. Menggambar kolom yang
  // selamanya kosong bukan kesetaraan melainkan peniruan yang keliru.
  it('tidak menggambar kolom User Update maupun Tanggal Update', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(
      within(table).queryByRole('columnheader', { name: 'User Update' }),
    ).not.toBeInTheDocument()
    expect(
      within(table).queryByRole('columnheader', { name: 'Tanggal Update' }),
    ).not.toBeInTheDocument()
  })

  it('membuka pada tab Approve', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(listCalls()[0]?.url).toContain('status=1')
    expect(await screen.findByText('ENGINE')).toBeInTheDocument()
  })

  // Urutan tab mengikuti layar lama: Approve, Reject, lalu Waiting Approval. Reject berada
  // di TENGAH, dan itu memang urutan yang dilihat petugas hari ini (D-13).
  it('menggambar ketiga tab pada urutan layar lama', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    const nav = screen.getByRole('navigation', { name: 'Tab Master Kategori Sparepart' })
    const label = within(nav)
      .getAllByRole('button')
      .map((b) => b.textContent)
    expect(label).toEqual(['Approve', 'Reject', 'Waiting Approval'])
  })

  it('berpindah tab menembak status yang sesuai', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(tab('Waiting Approval'))
    await waitFor(() => expect(listCalls().at(-1)?.url).toContain('status=0'))
    expect(await screen.findByText('ELECTRICAL')).toBeInTheDocument()

    await userEvent.click(tab('Reject'))
    await waitFor(() => expect(listCalls().at(-1)?.url).toContain('status=2'))
    expect(await screen.findByText('ATTACHMENT')).toBeInTheDocument()
  })

  // Portal ikut di setiap permintaan. Tanpa itu, server menolaknya — dan layar yang tidak
  // mengirimkannya akan menampilkan galat yang tidak dapat diperbaiki pengguna (R-20).
  it('mengirim portal entitas pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    for (const call of listCalls()) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  it('menolak memuat sebelum portal dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(listCalls()).toHaveLength(0)
  })
})

describe('keputusan persetujuan', () => {
  // Kolom centang HANYA ada di tab Waiting Approval: baris yang sudah diputuskan tidak
  // menunggu keputusan siapa pun.
  it('menggambar kolom pilih hanya pada tab Waiting Approval', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    expect(
      within(screen.getByRole('table')).queryByRole('columnheader', { name: 'Pilih' }),
    ).not.toBeInTheDocument()

    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('ELECTRICAL')
    expect(
      within(screen.getByRole('table')).getByRole('columnheader', { name: 'Pilih' }),
    ).toBeInTheDocument()
  })

  it('mematikan tombol keputusan selama belum ada yang dicentang', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')
    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('ELECTRICAL')

    expect(screen.getByRole('button', { name: 'Approve terpilih' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Reject terpilih' })).toBeDisabled()
  })

  it('mengirim satu permintaan untuk seluruh baris yang dicentang', async () => {
    installFetch(
      defaultReply(() => ({
        body: { jumlah_berubah: 1, status: '1', status_label: 'Approve', portal: 'ASM' },
      })),
    )
    show()
    await screen.findByRole('table')
    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('ELECTRICAL')

    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih ELECTRICAL' }))
    await userEvent.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    await waitFor(() => {
      const decision = calls.find((c) =>
        c.url.startsWith('/api/master/kategori-sparepart/keputusan'),
      )
      expect(decision).toBeDefined()
      expect(decision?.method).toBe('POST')
      expect(decision?.body).toEqual({ id_kategori_sparepart: ['4'], status: '1' })
    })

    expect(
      await screen.findByText(/1 kategori sparepart dipindahkan ke/),
    ).toBeInTheDocument()
  })

  // Centang dibuang saat berpindah tab: baris yang dipilih milik tab sebelumnya, dan
  // menyimpannya berarti keputusan dapat mengenai baris yang tidak sedang dilihat siapa pun.
  it('membuang centang saat berpindah tab', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')
    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('ELECTRICAL')

    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih ELECTRICAL' }))
    expect(screen.getByText('1 kategori')).toBeInTheDocument()

    await userEvent.click(tab('Approve'))
    await screen.findByText('ENGINE')
    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('ELECTRICAL')

    expect(screen.getByRole('button', { name: 'Approve terpilih' })).toBeDisabled()
  })
})

describe('form kategori sparepart', () => {
  it('menambah kategori baru tanpa mengirim ID maupun status', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kategori_sparepart: {
            id_kategori_sparepart: '7',
            nama_kategori_sparepart: 'FINAL DRIVE',
            status: '0',
            status_label: 'Waiting Approval',
          },
          portal: 'ASM',
        },
        status: 201,
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(
      screen.getByLabelText(/Nama Kategori Sparepart/),
      'FINAL DRIVE',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const created = calls.find((c) => c.method === 'POST' && !c.url.includes('keputusan'))
      expect(created).toBeDefined()
      // SATU field saja. ID diterbitkan server, status ditetapkan alurnya.
      expect(created?.body).toEqual({ nama_kategori_sparepart: 'FINAL DRIVE' })
    })
  })

  // Form penambahan tidak menggambar ID sama sekali: nomornya diterbitkan server dari isi
  // tabel, sehingga layar tidak punya cara menebaknya — dan tidak boleh mencoba.
  it('tidak menggambar ID pada form penambahan', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: 'Tambah Kategori Sparepart' })
    expect(within(form).queryByText('ID Kategori Sparepart')).not.toBeInTheDocument()
  })

  it('menampilkan ID yang tidak dapat diubah saat menyunting', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Ubah' }))
    const form = screen.getByRole('form', { name: 'Ubah Kategori Sparepart' })
    expect(within(form).getByText('ID Kategori Sparepart')).toBeInTheDocument()
    expect(within(form).getByText(/tidak dapat diubah/)).toBeInTheDocument()
  })

  it('menyimpan perubahan lewat PUT ke jalur ber-ID', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kategori_sparepart: { ...APPROVED, status: '0', status_label: 'Waiting Approval' },
          portal: 'ASM',
        },
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Ubah' }))
    const field = screen.getByLabelText(/Nama Kategori Sparepart/)
    await userEvent.clear(field)
    await userEvent.type(field, 'ENGINE ASSEMBLY')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.find((c) => c.method === 'PUT')
      expect(saved?.url).toBe('/api/master/kategori-sparepart/1')
      expect(saved?.body).toEqual({ nama_kategori_sparepart: 'ENGINE ASSEMBLY' })
    })
  })

  // Akibat menyimpan dinyatakan di muka, bukan ditemukan setelah tombol ditekan.
  it('memberi tahu bahwa menyimpan mengembalikan baris ke antrean persetujuan', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    // Dicari DI DALAM form, bukan di seluruh halaman: "Waiting Approval" juga merupakan
    // caption salah satu tab, dan pencarian selebar halaman akan cocok pada keduanya.
    const form = screen.getByRole('form', { name: 'Tambah Kategori Sparepart' })
    expect(within(form).getByText('Waiting Approval')).toBeInTheDocument()
    expect(
      within(form).getByText(/tidak muncul sebagai pilihan di layar Master Sparepart/),
    ).toBeInTheDocument()
  })

  it('menolak nama kosong tanpa menembak server', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')
    const before = calls.length

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText('Nama kategori sparepart wajib diisi.'),
    ).toBeInTheDocument()
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
    expect(calls.length).toBe(before)
  })

  // Nama ganda dijawab 409 dan pesannya MENYEBUT tab Reject.
  //
  // Tanpa keterangan itu, penolakannya tidak dapat dijelaskan dari layar: barisnya tidak
  // terlihat di tab Approve maupun Waiting Approval. Perilaku memblokirnya sendiri ditiru
  // dari sistem lama (P-5).
  it('menjelaskan bahwa nama bisa terpakai oleh baris yang sudah ditolak', async () => {
    installFetch(
      defaultReply(() => ({
        status: 409,
        body: {
          kode: 'kunci_kategori_sparepart_sudah_ada',
          pesan: 'Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain.',
          detail: [
            {
              kolom: 'nama_kategori_sparepart',
              pesan:
                'Nama ini sudah dipakai kategori lain — periksa juga tab Reject, karena ' +
                'kategori yang sudah ditolak pun tetap memakai namanya.',
            },
          ],
        },
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText(/Nama Kategori Sparepart/), 'ATTACHMENT')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama itu sudah dipakai')).toBeInTheDocument()
    expect(screen.getAllByText(/tab Reject/).length).toBeGreaterThan(0)
  })

  // Form tetap terbuka saat penyimpanan gagal; isian pengguna tidak dibuang.
  it('membiarkan form terbuka saat penyimpanan ditolak', async () => {
    installFetch(
      defaultReply(() => ({
        status: 409,
        body: {
          kode: 'kunci_kategori_sparepart_sudah_ada',
          pesan: 'Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain.',
          detail: [{ kolom: 'nama_kategori_sparepart', pesan: 'sudah dipakai' }],
        },
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText(/Nama Kategori Sparepart/), 'HYDRAULIC')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await screen.findByText('Nama itu sudah dipakai')
    expect(screen.getByRole('form', { name: 'Tambah Kategori Sparepart' })).toBeInTheDocument()
    expect(screen.getByLabelText(/Nama Kategori Sparepart/)).toHaveValue('HYDRAULIC')
  })

  // Tabelnya tidak punya kolom penampung alasan penolakan.
  it('tidak menggambar isian catatan', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(screen.queryByLabelText(/Catatan/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/Alasan/)).not.toBeInTheDocument()
  })
})
