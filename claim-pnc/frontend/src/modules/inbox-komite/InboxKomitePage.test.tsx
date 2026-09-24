import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSession } from '@/app/session'
import type { KomiteCase } from '@/api/types'

const PATH = '/api/komite/inbox'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Ellen Supriyati',
  jenis: 'KARYAWAN',
  login: 'ELLENSUPRIYATI',
  email: 'contoh.komite@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kasus komite contoh.
 *
 * Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas yang
 * di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
function kasus(partial: Partial<KomiteCase> = {}): KomiteCase {
  return {
    nomor_case: 'K-2601',
    nomor_klaim: 'PNCN.26.0101',
    nomor_polis: 'CONTOH-PL-000117',
    nama_tertanggung: 'PT Harapan Sentosa',
    nama_bisnis: 'Property All Risk',
    sumber_bisnis: 'Direct',
    cabang: 'Jakarta Pusat',
    group_panel: '006',
    pic_klaim: 'PICTEKNIK1',
    tanggal_komite: '2026-09-11T02:00:00Z',
    tanggal_input: '2026-09-11T02:00:00Z',
    aging_komite: 9,
    status_kerja: 'Open',
    tipe_komite: 'Adjustment',
    nilai_klaim: '45000000.00',
    nilai_asm_share: '31500000.00',
    nilai_or_asm: '22500000.00',
    ada_penilaian_ai: true,
    jawaban_ai: 'DITERIMA',
    note_ai_diterima: 'Dokumen lengkap.',
    tanggal_ai: '2026-09-11T02:00:00Z',
    penjenjangan: {
      kesimpulan: 'menunggu',
      jumlah_jenjang: 0,
      jenjang_kini: 1,
      jenjang_disetujui: 0,
      jumlah_jenjang_belum_diketahui: true,
      selesai: false,
      sudah_saya_putuskan: false,
      keputusan: [],
    },
    ...partial,
  }
}

const SAMPLES: KomiteCase[] = [
  kasus(),
  kasus({
    nomor_case: 'K-2603',
    nomor_klaim: 'PNCN.26.0103',
    nama_tertanggung: 'CV Bina Karya',
    nama_bisnis: 'Aneka',
    tipe_komite: 'Survey Komite',
    aging_komite: 4,
    nilai_klaim: '7200000.00',
    nilai_asm_share: '7200000.00',
    nilai_or_asm: '0.00',
    // Tanpa penilaian AI — ia WAJIB tetap muncul.
    ada_penilaian_ai: false,
    jawaban_ai: '',
    note_ai_diterima: '',
    tanggal_ai: '',
  }),
]

const SUMMARY = { outstanding: 2, diterima: 1, ditolak: 0 }

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal dan menunya sendiri, sehingga SETIAP layar di
    // balik sesi ikut memanggil keduanya. Dijawab otomatis supaya uji ini menguji
    // layarnya, bukan jalur galat menu.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url, init))
  })
}

function listBody(rows: KomiteCase[] = SAMPLES, extra: Record<string, unknown> = {}) {
  return {
    kasus: rows,
    total: rows.length,
    lewati: 0,
    batas: 25,
    ringkasan: SUMMARY,
    kotak: 'outstanding',
    operator: 'ELLENSUPRIYATI',
    sekarang: '2026-09-20T02:00:00Z',
    ...extra,
  }
}

function stubDefaultFetch() {
  stubFetch((url, init) => {
    if (url.includes('/keputusan') && init?.method === 'POST') {
      return jsonResponse(200, {
        kasus: kasus({
          penjenjangan: {
            kesimpulan: 'menunggu',
            jumlah_jenjang: 0,
            jenjang_kini: 2,
            jenjang_disetujui: 1,
            jumlah_jenjang_belum_diketahui: true,
            selesai: false,
            sudah_saya_putuskan: true,
            keputusan: [
              {
                id: 'abc',
                jenjang: 1,
                keputusan: 'setuju',
                oleh: 'ELLENSUPRIYATI',
                pada: '2026-09-20T02:00:00Z',
              },
            ],
          },
        }),
        sekarang: '2026-09-20T02:00:00Z',
      })
    }
    return jsonResponse(200, listBody())
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/komite/inbox']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** Permintaan daftar terakhir yang dikirim layar. */
function lastListCall(): Call | undefined {
  return [...calls]
    .reverse()
    .find((c) => c.url.startsWith(PATH) && (c.init?.method ?? 'GET') === 'GET')
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('daftar', () => {
  it('menampilkan kasus komite beserta nilainya dari server', async () => {
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText('PT Harapan Sentosa')).toBeInTheDocument()
    expect(screen.getByText('K-2601')).toBeInTheDocument()
    expect(screen.getByText('PNCN.26.0101')).toBeInTheDocument()

    // Nilai uang diformat dari teks desimal kanonik, bukan dari angka JSON.
    expect(screen.getByText('Rp 45.000.000')).toBeInTheDocument()

    // Jumlah datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/2 kasus pada kotak ini/)).toBeInTheDocument()
  })

  // Kasus tanpa penilaian AI WAJIB tetap muncul. Kueri lama menyambungkannya dengan OUTER
  // JOIN; mengubahnya menjadi INNER akan membuat pekerjaan menghilang tanpa satu pun tanda.
  it('menampilkan kasus yang belum dinilai AI', async () => {
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText('CV Bina Karya')).toBeInTheDocument()
  })

  // Umur ditampilkan sebagai ANGKA, bukan hanya warna: warna tidak terbaca oleh sebagian
  // pengguna, dan yang dipertaruhkan di sini adalah pekerjaan yang mendekati batas
  // persetujuan otomatis.
  it('menampilkan umur kasus dalam hari', async () => {
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText('9 hari')).toBeInTheDocument()
    expect(screen.getByText('4 hari')).toBeInTheDocument()
  })

  it('membuka kotak outstanding lebih dulu', async () => {
    stubDefaultFetch()
    renderPage()

    await screen.findByText('PT Harapan Sentosa')
    expect(lastListCall()?.url).toContain('kotak=outstanding')
  })
})

describe('penyaring', () => {
  it('mengirim kotak yang dipilih ke server dan mengembalikan halaman ke awal', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.click(screen.getByRole('tab', { name: /Ditolak/ }))

    await waitFor(() => {
      expect(lastListCall()?.url).toContain('kotak=ditolak')
    })
    expect(lastListCall()?.url).not.toContain('lewati=')
  })

  // Pencarian dikerjakan SERVER. Menyaring di peramban hanya menyentuh halaman yang
  // sedang terbuka, sehingga hasilnya bohong pada data berpaginasi.
  it('mengirim kata kunci pencarian ke server', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.type(screen.getByRole('searchbox'), 'K-2603')

    await waitFor(() => {
      expect(lastListCall()?.url).toContain('cari=K-2603')
    })
  })

  it('mengirim rentang tanggal input ke server', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.type(screen.getByLabelText('Tgl input dari'), '2026-09-01')

    await waitFor(() => {
      expect(lastListCall()?.url).toContain('dari=2026-09-01')
    })
  })

  // Rentang terbalik ditandai DI LAYAR sebelum dikirim, dan tidak ditukar diam-diam:
  // menukarnya akan menampilkan hasil yang benar untuk pertanyaan yang tidak diajukan.
  it('menandai rentang tanggal yang terbalik', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.type(screen.getByLabelText('Tgl input dari'), '2026-09-20')
    await userEvent.type(screen.getByLabelText('Tgl input sampai'), '2026-09-01')

    expect(
      await screen.findByText(/Tanggal sampai tidak boleh lebih awal/),
    ).toBeInTheDocument()
  })
})

describe('keputusan', () => {
  it('mengirim persetujuan tanpa menyebut jenjang maupun waktu', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.click(screen.getAllByRole('button', { name: /Beri keputusan komite/ })[0]!)
    await userEvent.click(screen.getByRole('radio', { name: /Setuju/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    await waitFor(() => {
      const post = calls.find((c) => c.init?.method === 'POST')
      expect(post).toBeDefined()
      expect(post?.url).toContain('/api/komite/inbox/K-2601/keputusan')

      const body = JSON.parse(String(post?.init?.body))
      expect(body).toEqual({ keputusan: 'setuju', catatan: '' })
      // Jenjang, waktu, dan identitas pemutus milik SERVER — klien tidak pernah
      // menyebutnya.
      expect(body).not.toHaveProperty('jenjang')
      expect(body).not.toHaveProperty('pada')
      expect(body).not.toHaveProperty('oleh')
    })
  })

  // Penolakan tanpa catatan ditolak DI LAYAR lebih dulu, sehingga pengguna tidak perlu
  // menunggu perjalanan ke server untuk tahu isiannya kurang. Server tetap menolaknya
  // juga — pemeriksaan di layar adalah kenyamanan, bukan penegakan.
  it('menolak penolakan tanpa catatan sebelum dikirim', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.click(screen.getAllByRole('button', { name: /Beri keputusan komite/ })[0]!)
    await userEvent.click(screen.getByRole('radio', { name: /Tolak/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    expect(await screen.findByText(/Catatan wajib diisi/)).toBeInTheDocument()
    expect(calls.find((c) => c.init?.method === 'POST')).toBeUndefined()
  })

  // Keputusan yang MENGHENTIKAN komite menempuh satu langkah konfirmasi, dan konfirmasinya
  // menyebutkan nomor kasus — supaya yang dihentikan bukan kasus yang salah.
  it('meminta konfirmasi sebelum menolak', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.click(screen.getAllByRole('button', { name: /Beri keputusan komite/ })[0]!)
    await userEvent.click(screen.getByRole('radio', { name: /Tolak/ }))
    await userEvent.type(screen.getByLabelText(/Catatan/), 'Nilai melebihi sisa TSI.')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    const peringatan = await screen.findByRole('alert')
    expect(peringatan).toHaveTextContent('K-2601')
    expect(peringatan).toHaveTextContent(/tidak dapat ditarik kembali/)
    expect(calls.find((c) => c.init?.method === 'POST')).toBeUndefined()

    await userEvent.click(screen.getByRole('button', { name: 'Ya, lanjutkan' }))

    await waitFor(() => {
      expect(calls.find((c) => c.init?.method === 'POST')).toBeDefined()
    })
  })

  // Persetujuan TIDAK meminta konfirmasi: ia tidak menutup apa pun selama masih ada
  // jenjang berikutnya, dan ia pekerjaan yang paling sering dilakukan.
  it('tidak meminta konfirmasi untuk persetujuan', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.click(screen.getAllByRole('button', { name: /Beri keputusan komite/ })[0]!)
    await userEvent.click(screen.getByRole('radio', { name: /Setuju/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    await waitFor(() => {
      expect(calls.find((c) => c.init?.method === 'POST')).toBeDefined()
    })
    expect(screen.queryByRole('button', { name: 'Ya, lanjutkan' })).not.toBeInTheDocument()
  })

  // Konflik (409) diberi tindakan yang benar: muat ulang. Menyuruh "coba lagi" akan
  // membuat pengguna menekan tombol berulang kali pada sesuatu yang tidak akan berubah.
  it('menjelaskan keputusan yang sudah tercatat dari tab lain', async () => {
    stubFetch((url, init) => {
      if (url.includes('/keputusan') && init?.method === 'POST') {
        return jsonResponse(409, {
          kode: 'komite_sudah_selesai',
          pesan: 'Keputusan atas kasus ini sudah tercatat. Muat ulang untuk melihat keadaan terbarunya.',
        })
      }
      return jsonResponse(200, listBody())
    })
    renderPage()
    await screen.findByText('PT Harapan Sentosa')

    await userEvent.click(screen.getAllByRole('button', { name: /Beri keputusan komite/ })[0]!)
    await userEvent.click(screen.getByRole('radio', { name: /Setuju/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    expect(await screen.findByText('Keputusan sudah tercatat')).toBeInTheDocument()
  })

  // Tombol disembunyikan pada kasus yang sudah diputuskan. Itu KENYAMANAN TAMPILAN;
  // penegakannya tetap di server, yang menjawab 409.
  it('tidak menawarkan tombol putuskan pada kasus yang sudah diputuskan', async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        listBody([
          kasus({
            penjenjangan: {
              kesimpulan: 'menunggu',
              jumlah_jenjang: 0,
              jenjang_kini: 2,
              jenjang_disetujui: 1,
              jumlah_jenjang_belum_diketahui: true,
              selesai: false,
              sudah_saya_putuskan: true,
              keputusan: [],
            },
          }),
        ]),
      ),
    )
    renderPage()

    expect(await screen.findByText('sudah diputuskan')).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /Beri keputusan komite/ }),
    ).not.toBeInTheDocument()
  })
})

describe('keadaan kosong', () => {
  // Inbox kosong punya DUA sebab yang sangat berbeda, dan keduanya tidak dapat dibedakan
  // dari tabel kosong. Operator yang dipakai menyaring karena itu ditampilkan.
  it('menyebutkan operator yang dipakai menyaring saat tidak ada kasus', async () => {
    stubFetch(() =>
      jsonResponse(200, listBody([], { ringkasan: { outstanding: 0, diterima: 0, ditolak: 0 } })),
    )
    renderPage()

    expect(await screen.findByText(/Tidak ada kasus yang menunggu keputusan Anda/)).toBeInTheDocument()
    expect(screen.getByText('ELLENSUPRIYATI')).toBeInTheDocument()
  })

  it('menjelaskan kegagalan memuat tanpa membocorkan rincian internal', async () => {
    stubFetch(() =>
      jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' }),
    )
    renderPage()

    expect(await screen.findByText('Daftar kasus komite gagal dimuat')).toBeInTheDocument()
  })
})
