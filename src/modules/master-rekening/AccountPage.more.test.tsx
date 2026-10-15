import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, renderHook, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AccountPage } from './AccountPage'
import { useAccountList, useUpdateAccount, type AccountFields } from './api'

/**
 * Uji tambahan Master Rekening: formulir pengajuan, cabang galat, tabel, dan hook yang
 * tidak dipakai layar. Seluruh data KARANGAN (`D-69`).
 */

type Reply = { status: number; body: unknown } | 'putus' | 'rusak'

let reply: Map<string, Reply>
let request: { path: string; metode: string; body: unknown; header: Record<string, string> }[]

/** Jawaban yang membuat callAPI melempar galat selain APIError/NetworkError. */
function brokenResponse(): Response {
  return {
    get status(): number {
      throw new TypeError('jawaban rusak')
    },
  } as unknown as Response
}

function installFakeFetch() {
  request = []
  vi.stubGlobal('fetch', async (path: string, options?: RequestInit) => {
    request.push({
      path,
      metode: options?.method ?? 'GET',
      body: options?.body ? JSON.parse(String(options.body)) : undefined,
      header: (options?.headers ?? {}) as Record<string, string>,
    })
    const method = options?.method ?? 'GET'
    const key = [...reply.keys()]
      .filter((k) => {
        const [m, p] = k.split(' ')
        return m === method && path.startsWith(p!)
      })
      .sort((a, b) => b.length - a.length)[0]
    const result = key ? reply.get(key)! : { status: 404, body: null }
    if (result === 'putus') throw new TypeError('putus')
    if (result === 'rusak') return brokenResponse()
    return new Response(JSON.stringify(result.body), {
      status: result.status,
      headers: { 'Content-Type': 'application/json' },
    })
  })
}

function client() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function wrap(children: ReactNode) {
  return <QueryClientProvider client={client()}>{children}</QueryClientProvider>
}

function sampleAccount(overrides: Record<string, unknown> = {}) {
  return {
    nomor_rekening: '1234567890',
    nama_pemilik: 'BENGKEL CONTOH SEJAHTERA',
    nama_bank: 'BANK CONTOH',
    cabang_bank: 'JAKARTA PUSAT',
    alamat_bank: 'JL. CONTOH NO. 1',
    kode_bank: '014',
    tipe_rekening: 'BIASA',
    aktif: true,
    email: 'keuangan@contoh.example',
    telepon: '0211234567',
    nik: '3171000000000000',
    email_inputor: 'pengaju@contoh.example',
    catatan: 'catatan awal',
    id_dokumen: '',
    diinput_oleh: '3171999',
    status: '0',
    status_label: 'Menunggu',
    komite_approval: 'KOMITE-01',
    diinput_pada: '2026-09-17T03:00:00Z',
    status_layanan: '',
    id_rekening_kasir: '',
    respons_kasir: '',
    dapat_dipakai: false,
    ...overrides,
  }
}

function listOf(rows: unknown[], jumlah = rows.length) {
  return { status: 200, body: { rekening: rows, jumlah, batas: 50, lewati: 0 } }
}

beforeEach(() => {
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
  reply = new Map<string, Reply>([
    ['GET /api/master-rekening/bank', { status: 200, body: { bank: [{ kode: '014', nama: 'BANK BCA' }] } }],
    ['GET /api/master-rekening', listOf([sampleAccount()])],
  ])
  installFakeFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

/*
 * Nama label di bawah mengikuti layar Pega `Memperbaharui Data` huruf per huruf,
 * termasuk yang berhuruf kapital seluruhnya. Itu disengaja: uji ini sekaligus yang
 * mengunci agar label layar tidak diubah diam-diam menjadi sesuatu yang "lebih rapi"
 * tetapi tidak lagi dikenali pengguna yang terbiasa dengan layar lama.
 *
 * Tanda wajib ikut menjadi bagian label — "Email *" — karena `Field` menggambar label
 * apa adanya. Pencocokannya memakai regex berjangkar depan supaya tidak bergantung pada
 * spasi di sekitar bintangnya.
 *
 * Satu jebakan yang sudah menggigit sekali dan karena itu dicatat: jangkar `/^Email/`
 * saja mengenai DUA kolom — "Email" dan "Email Inputor" — dan `getByLabelText` memilih
 * yang pertama ditemukannya. Akibatnya salah satu kolom tetap kosong, pengajuan ditolak
 * validasi, dan uji gagal di tempat yang jauh dari sebabnya. Jangkarnya karena itu
 * menyertakan bintangnya.
 */
async function openForm() {
  render(wrap(<AccountPage />))
  await screen.findByText('1234567890')
  await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
  return screen.findByRole('button', { name: 'Simpan' })
}

async function fillForm() {
  await userEvent.type(screen.getByLabelText(/^NOMOR REKENING/), '9988776655')
  await userEvent.type(screen.getByLabelText(/^Input Nama/), 'PEMILIK CONTOH')
  await screen.findByRole('option', { name: 'BANK BCA' })
  await userEvent.selectOptions(screen.getByLabelText(/^NAMA BANK/), '014')
  await userEvent.type(screen.getByLabelText(/^NAMA CABANG BANK/), 'SLEMAN')
  await userEvent.type(screen.getByLabelText(/^NOMOR TELEPON/), '08123456789')
  await userEvent.type(screen.getByLabelText(/^ALAMAT/), 'JL. CONTOH 2')
  await userEvent.selectOptions(screen.getByLabelText(/^TIPE REKENING/), 'VA')
  await userEvent.selectOptions(screen.getByLabelText(/^STATUS AKTIF/), 'Ya')
  await userEvent.type(screen.getByLabelText(/^KTP\/NIK\/NPWP/), '3404000000000001')
  await userEvent.type(screen.getByLabelText(/^Email \*$/), 'pemilik@contoh.example')
  await userEvent.type(screen.getByLabelText(/^Email Inputor/), 'pengaju@contoh.example')
}

describe('formulir rekening baru', () => {
  it('membuka dan menutup formulir lewat tombol yang sama', async () => {
    await openForm()
    expect(screen.getByText('Memperbaharui Data')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Batal' }))
    expect(screen.queryByText('Memperbaharui Data')).not.toBeInTheDocument()
  })

  it('menolak isian kosong dan format salah di layar tanpa memanggil server', async () => {
    const submit = await openForm()
    await userEvent.click(submit)

    // Kesebelas kolom bertanda `*` di Pega harus mengeluh sekaligus. Pengguna yang
    // mengisi sebelas kolom berhak tahu seluruh yang kurang dalam satu kali tekan,
    // bukan menemukan satu kesalahan baru pada setiap percobaan.
    for (const text of [
      'Nomor rekening wajib diisi.',
      'Nama wajib diisi.',
      'Nama bank wajib dipilih.',
      'Nama cabang bank wajib diisi.',
      'Nomor telepon wajib diisi.',
      'Alamat wajib diisi.',
      'Tipe rekening wajib dipilih.',
      'Status aktif wajib dipilih.',
      'Email wajib diisi.',
      'Email inputor wajib diisi.',
      'KTP/NIK/NPWP wajib diisi.',
    ]) {
      expect(await screen.findByText(text)).toBeInTheDocument()
    }
    expect(screen.getByLabelText(/^NAMA BANK/)).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText(/^TIPE REKENING/)).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText(/^STATUS AKTIF/)).toHaveAttribute('aria-invalid', 'true')

    await userEvent.type(screen.getByLabelText(/^NOMOR REKENING/), '12A')
    await userEvent.type(screen.getByLabelText(/^Email \*$/), 'bukan-email')
    await userEvent.click(submit)
    expect(await screen.findByText('Nomor rekening hanya boleh berisi angka.')).toBeInTheDocument()
    expect(screen.getByText('Format email tidak benar.')).toBeInTheDocument()
    expect(request.some((r) => r.metode === 'POST')).toBe(false)
  })

  it('mengirim pengajuan lengkap lalu menutup formulir', async () => {
    reply.set('POST /api/master-rekening', { status: 201, body: sampleAccount() })
    const submit = await openForm()
    await fillForm()
    await userEvent.click(submit)

    await waitFor(() => expect(screen.queryByText('Memperbaharui Data')).not.toBeInTheDocument())
    const sent = request.find((r) => r.metode === 'POST')
    expect(sent?.path).toBe('/api/master-rekening')
    expect(sent?.header['X-Portal']).toBe('ASM')
    expect(sent?.body).toEqual({
      nomor_rekening: '9988776655',
      nama_pemilik: 'PEMILIK CONTOH',
      nama_bank: 'BANK BCA',
      cabang_bank: 'SLEMAN',
      alamat_bank: 'JL. CONTOH 2',
      kode_bank: '014',
      tipe_rekening: 'VA',
      email: 'pemilik@contoh.example',
      telepon: '08123456789',
      nik: '3404000000000001',
      email_inputor: 'pengaju@contoh.example',
      id_dokumen: '',
      // Pengaju tidak mengirim catatan: kolom NOTE diisi komite saat memutuskan, dan
      // layar Pega memang tidak punya kolom catatan di sisi pengaju.
      catatan: '',
      aktif: true,
      kode_bank_lama: '',
      nomor_rekening_lama: '',
      nama_pemilik_lama: '',
    })
  })

  it.each([
    {
      name: 'nomor sudah terdaftar',
      answer: { status: 409, body: { kode: 'nomor_rekening_sudah_ada', pesan: 'x' } } as Reply,
      title: 'Nomor rekening sudah terdaftar',
    },
    {
      name: 'galat API lain',
      answer: { status: 500, body: { kode: 'galat_internal', pesan: 'x' } } as Reply,
      title: 'Gagal menyimpan',
    },
    {
      name: 'jaringan putus',
      answer: 'putus' as Reply,
      title: 'Tidak dapat menghubungi server',
      description: 'Perubahan belum tersimpan. Periksa koneksi lalu coba lagi.',
    },
    {
      name: 'galat bukan API',
      answer: 'rusak' as Reply,
      title: 'Gagal menyimpan',
      description: 'Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi.',
    },
  ])('menampilkan galat pengajuan: $name', async ({ answer, title, description }) => {
    reply.set('POST /api/master-rekening', answer)
    const submit = await openForm()
    await fillForm()
    await userEvent.click(submit)

    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
    // Formulir tetap terbuka supaya pengguna dapat membetulkan isian.
    expect(screen.getByText('Memperbaharui Data')).toBeInTheDocument()
  })

  it('memindahkan galat validasi server ke kolomnya masing-masing', async () => {
    reply.set('POST /api/master-rekening', {
      status: 422,
      body: {
        kode: 'isian_tidak_sah',
        pesan: 'x',
        field: { nomor_rekening: 'Nomor ditolak server.', kolom_asing: 'Diabaikan.' },
      },
    })
    const submit = await openForm()
    await fillForm()
    await userEvent.click(submit)

    expect(await screen.findByText('Nomor ditolak server.')).toBeInTheDocument()
    expect(screen.queryByText('Diabaikan.')).not.toBeInTheDocument()

    // Kotak pesan di atas form TIDAK muncul saat pelanggarannya sudah tersorot di
    // kolomnya masing-masing: ia hanya akan mengulang hal yang sama dua kali. Perilaku
    // ini seragam dengan Master Tipe Surveyors dan master lainnya.
    expect(screen.queryByText('Ada isian yang belum benar')).not.toBeInTheDocument()
  })

  it('memberi tahu bila daftar bank gagal dimuat', async () => {
    reply.set('GET /api/master-rekening/bank', { status: 500, body: { kode: 'x', pesan: 'x' } })
    await openForm()

    expect(await screen.findByText(/Daftar bank tidak dapat dimuat/)).toBeInTheDocument()
  })
})

describe('mengubah rekening yang masih menunggu', () => {
  it('mengisi formulir dari baris yang dipilih lalu mengirim PUT ke kuncinya', async () => {
    reply.set('PUT /api/master-rekening', { status: 200, body: sampleAccount() })
    render(wrap(<AccountPage />))
    await screen.findByText('1234567890')

    // Nama tombol menyebut nomor rekeningnya — satu tabel memuat banyak tombol "Ubah",
    // dan tanpa nomornya pengguna pembaca layar mendengar deretan tombol yang
    // seluruhnya bernama sama.
    await userEvent.click(screen.getByRole('button', { name: 'Ubah rekening 1234567890' }))

    // Formulir terisi dari baris yang dipilih, bukan kosong. Tanpa ini pengguna harus
    // mengetik ulang sebelas kolom hanya untuk membetulkan satu di antaranya.
    const number = screen.getByLabelText(/^NOMOR REKENING/)
    expect(number).toHaveValue('1234567890')
    expect(screen.getByLabelText(/^Input Nama/)).toHaveValue('BENGKEL CONTOH SEJAHTERA')
    expect(screen.getByLabelText(/^Email Inputor/)).toHaveValue('pengaju@contoh.example')
    expect(screen.getByLabelText(/^STATUS AKTIF/)).toHaveValue('Ya')

    // Nomor rekening adalah bagian kuncinya; mengubahnya berarti pengajuan baru.
    expect(number).toHaveAttribute('readonly')

    await userEvent.clear(screen.getByLabelText(/^NAMA CABANG BANK/))
    await userEvent.type(screen.getByLabelText(/^NAMA CABANG BANK/), 'SURABAYA')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(request.some((r) => r.metode === 'PUT')).toBe(true))
    const sent = request.find((r) => r.metode === 'PUT')!
    expect(sent.path).toBe('/api/master-rekening/014/1234567890')
    expect(sent.body).toMatchObject({
      cabang_bank: 'SURABAYA',
      nomor_rekening: '1234567890',
      email_inputor: 'pengaju@contoh.example',
    })
  })

  it('menawarkan Ubah juga pada rekening yang sudah disetujui, dengan peringatannya', async () => {
    reply.set(
      'GET /api/master-rekening',
      listOf([sampleAccount({ status: '1', status_label: 'Komite Approve' })]),
    )
    render(wrap(<AccountPage />))
    await screen.findByText('1234567890')

    // Layar lama menyediakan tombol ini di tab Approve, dan server kini menerimanya.
    await userEvent.click(screen.getByRole('button', { name: 'Ubah rekening 1234567890' }))

    // Pengamannya bukan menyembunyikan tombol, melainkan memberi tahu akibatnya:
    // menyimpan akan mencabut persetujuan komite.
    expect(
      await screen.findByText('Menyimpan akan mencabut persetujuan komite'),
    ).toBeInTheDocument()
  })
})

describe('daftar dan tabel', () => {
  it('menampilkan keadaan memuat lalu pesan kosong', async () => {
    reply.set('GET /api/master-rekening', listOf([]))
    render(wrap(<AccountPage />))

    expect(screen.getByText('Memuat data…')).toBeInTheDocument()
    expect(
      await screen.findByText('Tidak ada rekening yang cocok dengan pencarian Anda.'),
    ).toBeInTheDocument()
  })

  it('menampilkan peringatan bila daftar gagal dimuat', async () => {
    reply.set('GET /api/master-rekening', { status: 500, body: { kode: 'x', pesan: 'x' } })
    render(wrap(<AccountPage />))

    expect(await screen.findByText('Daftar rekening gagal dimuat')).toBeInTheDocument()
  })

  it('menampilkan ID Kasir dan Response Kasir sebagai dua kolom terpisah', async () => {
    reply.set(
      'GET /api/master-rekening',
      listOf(
        [
          sampleAccount({
            nomor_rekening: '111',
            status: '1',
            status_label: 'Komite Approve',
            status_layanan: 'BERHASIL',
            id_rekening_kasir: 'KSR-9',
          }),
          sampleAccount({
            nomor_rekening: '222',
            status: '1',
            status_label: 'Komite Approve',
            status_layanan: 'BERHASIL',
          }),
          sampleAccount({
            nomor_rekening: '444',
            status: '1',
            status_label: 'Komite Approve',
            status_layanan: 'GAGAL',
            respons_kasir: 'Nomor rekening ditolak Kasir.',
          }),
        ],
        10,
      ),
    )
    render(wrap(<AccountPage />))

    // Sebelumnya keduanya digabung menjadi satu sel "Terdaftar · KSR-9". Pega
    // memisahkannya, dan penggabungan membuat ID-nya tidak dapat dicari maupun disalin
    // sebagai nilai tersendiri.
    expect(await screen.findByText('KSR-9')).toBeInTheDocument()

    // Baris kedua tidak punya ID Kasir; selnya bertanda hubung, bukan kosong — sel
    // kosong tidak dapat dibedakan dari sel yang gagal dimuat.
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)

    const gagal = screen.getByText('Nomor rekening ditolak Kasir.')
    expect(gagal).toHaveClass('text-red-700')
  })

  it('mengirim kata kunci ke SERVER, bukan menyaring di peramban', async () => {
    render(wrap(<AccountPage />))
    await screen.findByText('1234567890')

    // Satu kotak cari, dan isinya menembak server. Daftar rekening dipotong paginasi:
    // menyaring di peramban hanya menyentuh halaman yang sedang terbuka, sehingga
    // rekening di halaman berikutnya dilaporkan tidak ada.
    await userEvent.type(screen.getByLabelText(/Cari nomor rekening/), 'BENG')
    await userEvent.click(screen.getByRole('tab', { name: 'Reject' }))

    await waitFor(() =>
      expect(
        request.some((r) => r.path.includes('status=2') && r.path.includes('cari=BENG')),
      ).toBe(true),
    )
  })
})

describe('keputusan komite', () => {
  // Keputusan diambil HANYA di tab Komite Approval — satu-satunya section Pega yang
  // memuat tombol `Approve`/`Reject` beserta "KETERANGAN APPROVAL ATASAN".
  async function openPending() {
    render(wrap(<AccountPage />))
    await userEvent.click(screen.getByRole('tab', { name: 'Komite Approval' }))
    return screen.findByDisplayValue('catatan awal')
  }

  it('mengirim penolakan dengan keterangan bawaan rekening', async () => {
    reply.set('POST /api/master-rekening/014/1234567890/keputusan', {
      status: 200,
      body: sampleAccount({ status: '2' }),
    })
    await openPending()
    await userEvent.click(screen.getByRole('button', { name: 'Reject' }))

    await waitFor(() => {
      const decision = request.find((r) => r.path.endsWith('/keputusan'))
      expect(decision?.body).toEqual({ status: '2', catatan: 'catatan awal', id_dokumen: '' })
    })
  })

  it('menampilkan pelanggaran isian dari server', async () => {
    reply.set('POST /api/master-rekening/014/1234567890/keputusan', {
      status: 422,
      body: {
        kode: 'isian_tidak_sah',
        pesan: 'x',
        detail: [{ field: 'catatan', pesan: 'Keterangan wajib diisi.' }],
        field: { id_dokumen: 'Buku rekening belum diunggah.' },
      },
    })
    await openPending()
    await userEvent.click(screen.getByRole('button', { name: 'Approve' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Buku rekening belum diunggah. Keterangan wajib diisi.',
    )
  })

  it('menampilkan pesan umum untuk galat keputusan yang lain', async () => {
    reply.set('POST /api/master-rekening/014/1234567890/keputusan', 'putus')
    await openPending()
    await userEvent.click(screen.getByRole('button', { name: 'Approve' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Keputusan tidak tersimpan. Coba lagi.')
  })
})

describe('hook', () => {
  function hookWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client()}>{children}</QueryClientProvider>
  }

  it('useAccountList menyertakan batas dan lewati pada kueri', async () => {
    const { result } = renderHook(() => useAccountList({ batas: 20, lewati: 40 }), {
      wrapper: hookWrapper,
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(request.some((r) => r.path === '/api/master-rekening?batas=20&lewati=40')).toBe(true)
  })

  it('useAccountList tidak memanggil server tanpa sesi', () => {
    useSession.setState({ token: null })
    const { result } = renderHook(() => useAccountList({}), { wrapper: hookWrapper })

    expect(result.current.fetchStatus).toBe('idle')
    expect(request).toHaveLength(0)
  })

  it('useUpdateAccount mengirim PUT ke kunci lama beserta nilai lamanya', async () => {
    reply.set('PUT /api/master-rekening', { status: 200, body: sampleAccount() })
    const values: AccountFields = {
      nomorRekening: '777',
      namaPemilik: 'BARU',
      namaBank: 'BANK',
      cabangBank: 'CAB',
      alamatBank: 'ALM',
      kodeBank: '014',
      tipeRekening: 'BIASA',
      email: 'baru@contoh.example',
      telepon: '0800',
      nik: '1',
      emailInputor: 'pengaju@contoh.example',
      idDokumen: 'D1',
      catatan: 'c',
      aktif: false,
      kodeBankLama: '002',
      nomorRekeningLama: '1/2',
      namaPemilikLama: 'LAMA',
    }
    const { result } = renderHook(() => useUpdateAccount(), { wrapper: hookWrapper })

    result.current.mutate({ kodeBank: '002', nomorRekening: '1/2', values })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const sent = request.find((r) => r.metode === 'PUT')
    expect(sent?.path).toBe('/api/master-rekening/002/1%2F2')
    expect(sent?.body).toMatchObject({
      kode_bank_lama: '002',
      nomor_rekening_lama: '1/2',
      nama_pemilik_lama: 'LAMA',
      aktif: false,
    })
  })
})
