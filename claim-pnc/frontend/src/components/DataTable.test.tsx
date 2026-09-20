import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { DataTable, pageWindow, type Column } from './DataTable'

type Row = { id: string; nama: string }

/** Membuat sejumlah baris bernomor urut, supaya isi tiap halaman dapat diperiksa. */
function rowsOf(count: number): Row[] {
  return Array.from({ length: count }, (_, index) => ({
    id: String(index + 1),
    nama: `Baris ${String(index + 1).padStart(3, '0')}`,
  }))
}

const columns: Column<Row>[] = [
  { key: 'nama', title: 'Nama', value: (row) => row.nama },
  { key: 'id', title: 'ID', value: (row) => row.id },
]

function show(rows: Row[], pageSize?: number) {
  return render(
    <DataTable
      columns={columns}
      rows={rows}
      rowKey={(row) => row.id}
      searchLabel="Cari"
      {...(pageSize === undefined ? {} : { pageSize })}
    />,
  )
}

/** Mengambil bilah nomor halaman. */
function pager() {
  return screen.getByRole('navigation', { name: 'Halaman tabel' })
}

// ============================================================================
// Tanpa pageSize — perilaku lama tidak boleh berubah sedikit pun
// ============================================================================

describe('tanpa paginasi', () => {
  /*
    Ini uji yang paling penting di berkas ini.

    Sembilan layar master yang sudah selesai memakai DataTable TANPA prop pageSize, dan
    Isolasi Protektif melarang perilakunya berubah. Bila bawaan komponen ini kelak
    diam-diam diubah menjadi berpaginasi, uji inilah yang gagal — bukan uji milik modul
    yang sudah dinyatakan selesai.
  */
  it('menggambar SELURUH baris bila pageSize tidak diisi', () => {
    show(rowsOf(40))

    expect(screen.getByText('Baris 001')).toBeInTheDocument()
    expect(screen.getByText('Baris 040')).toBeInTheDocument()
  })

  it('tidak menggambar paginator sama sekali', () => {
    show(rowsOf(40))

    expect(screen.queryByRole('navigation', { name: 'Halaman tabel' })).not.toBeInTheDocument()
    expect(screen.queryByText(/Menampilkan/)).not.toBeInTheDocument()
  })
})

// ============================================================================
// Dengan pageSize — meniru pyGridPaginator
// ============================================================================

describe('paginasi', () => {
  // `pyPageSize = 20` pada Master Supplier: halaman pertama memuat dua puluh baris
  // pertama, dan baris ke-21 tidak tergambar sama sekali.
  it('hanya menggambar baris halaman yang sedang dibuka', () => {
    show(rowsOf(57), 20)

    expect(screen.getByText('Baris 001')).toBeInTheDocument()
    expect(screen.getByText('Baris 020')).toBeInTheDocument()
    expect(screen.queryByText('Baris 021')).not.toBeInTheDocument()
  })

  it('menyebutkan rentang baris yang sedang tampil', () => {
    show(rowsOf(57), 20)

    expect(screen.getByText(/1–20/)).toBeInTheDocument()
    expect(screen.getByText(/dari 57 baris/)).toBeInTheDocument()
  })

  it('berpindah ke halaman berikutnya lewat nomor', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    await user.click(within(pager()).getByRole('button', { name: 'Halaman 2' }))

    expect(screen.getByText('Baris 021')).toBeInTheDocument()
    expect(screen.queryByText('Baris 020')).not.toBeInTheDocument()
    expect(screen.getByText(/21–40/)).toBeInTheDocument()
  })

  it('berpindah lewat tombol berikutnya dan sebelumnya', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    await user.click(within(pager()).getByRole('button', { name: 'Halaman berikutnya' }))
    expect(screen.getByText('Baris 021')).toBeInTheDocument()

    await user.click(within(pager()).getByRole('button', { name: 'Halaman sebelumnya' }))
    expect(screen.getByText('Baris 001')).toBeInTheDocument()
  })

  // Halaman terakhir yang tidak penuh menampilkan sisanya apa adanya — 57 baris pada
  // ukuran 20 berarti halaman ketiga memuat 17 baris.
  it('menggambar sisa baris pada halaman terakhir', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    await user.click(within(pager()).getByRole('button', { name: 'Halaman 3' }))

    expect(screen.getByText('Baris 041')).toBeInTheDocument()
    expect(screen.getByText('Baris 057')).toBeInTheDocument()
    expect(screen.getByText(/41–57/)).toBeInTheDocument()
  })

  // Ujung daftar tidak dapat dilewati. Tombol yang tetap dapat ditekan di ujung akan
  // menghasilkan halaman kosong tanpa penjelasan.
  it('mematikan tombol di kedua ujung', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    expect(within(pager()).getByRole('button', { name: 'Halaman sebelumnya' })).toBeDisabled()
    expect(within(pager()).getByRole('button', { name: 'Halaman berikutnya' })).toBeEnabled()

    await user.click(within(pager()).getByRole('button', { name: 'Halaman 3' }))

    expect(within(pager()).getByRole('button', { name: 'Halaman sebelumnya' })).toBeEnabled()
    expect(within(pager()).getByRole('button', { name: 'Halaman berikutnya' })).toBeDisabled()
  })

  // Halaman yang sedang dibuka ditandai `aria-current`, bukan hanya warna — warna tidak
  // terbaca pembaca layar, dan tidak terbedakan oleh yang buta warna merah-hijau.
  it('menandai halaman yang sedang dibuka untuk pembaca layar', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    expect(within(pager()).getByRole('button', { name: 'Halaman 1' })).toHaveAttribute(
      'aria-current',
      'page',
    )

    await user.click(within(pager()).getByRole('button', { name: 'Halaman 2' }))

    expect(within(pager()).getByRole('button', { name: 'Halaman 2' })).toHaveAttribute(
      'aria-current',
      'page',
    )
    expect(within(pager()).getByRole('button', { name: 'Halaman 1' })).not.toHaveAttribute(
      'aria-current',
    )
  })

  // Satu halaman tidak perlu tombol; ringkasan barisnya tetap berguna dan tetap tampil.
  it('menyembunyikan tombol bila halamannya hanya satu', () => {
    show(rowsOf(5), 20)

    expect(screen.queryByRole('navigation', { name: 'Halaman tabel' })).not.toBeInTheDocument()
    expect(screen.getByText(/1–5/)).toBeInTheDocument()
  })

  // Tabel kosong tidak menggambar paginator: keadaan kosong sudah punya tampilannya
  // sendiri, dan "Menampilkan 0–0 dari 0 baris" tidak memberi tahu apa pun.
  it('tidak menggambar paginator saat tidak ada baris', () => {
    show([], 20)

    expect(screen.queryByText(/Menampilkan/)).not.toBeInTheDocument()
    expect(screen.getByText('Belum ada data.')).toBeInTheDocument()
  })
})

// ============================================================================
// Paginasi berjalan SESUDAH pencarian dan pengurutan
// ============================================================================

describe('paginasi bersama pencarian dan pengurutan', () => {
  /*
    Urutannya menentukan, dan salah urutan menghasilkan cacat yang sulit dikenali:
    memotong halaman LEBIH DULU akan membuat pencarian hanya menemukan baris yang
    kebetulan ada di halaman yang sedang dibuka.

    Grid Pega pun menyaring lebih dulu — paginatornya memotong page list klipboard yang
    sudah tersaring, bukan meminta halaman ke server.
  */
  it('mencari di SELURUH baris, bukan hanya halaman yang tampil', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    // Baris 045 ada di halaman ketiga, jauh dari halaman yang sedang dibuka.
    await user.type(screen.getByPlaceholderText('Cari'), 'Baris 045')

    expect(screen.getByText('Baris 045')).toBeInTheDocument()
    expect(screen.getByText(/1–1/)).toBeInTheDocument()
  })

  it('kembali ke halaman pertama saat kata kunci berubah', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    await user.click(within(pager()).getByRole('button', { name: 'Halaman 3' }))
    expect(screen.getByText('Baris 041')).toBeInTheDocument()

    // "Baris 0" cocok dengan seluruh 57 baris, sehingga halamannya tetap tiga — yang
    // diuji adalah posisinya kembali ke halaman pertama, bukan jumlah halamannya.
    await user.type(screen.getByPlaceholderText('Cari'), 'Baris 0')

    expect(screen.getByText('Baris 001')).toBeInTheDocument()
    expect(screen.getByText(/1–20/)).toBeInTheDocument()
  })

  it('kembali ke halaman pertama saat urutan berubah', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    await user.click(within(pager()).getByRole('button', { name: 'Halaman 3' }))
    expect(screen.getByText(/41–57/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Nama/ }))

    expect(screen.getByText(/1–20/)).toBeInTheDocument()
  })

  // Mengurutkan menurun membuat baris terakhir berpindah ke halaman pertama. Ini yang
  // membuktikan paginasi memotong daftar yang SUDAH diurutkan.
  it('memotong daftar yang sudah diurutkan', async () => {
    const user = userEvent.setup()
    show(rowsOf(57), 20)

    const judul = screen.getByRole('button', { name: /Nama/ })
    await user.click(judul) // menaik
    await user.click(judul) // menurun

    expect(screen.getByText('Baris 057')).toBeInTheDocument()
    expect(screen.queryByText('Baris 001')).not.toBeInTheDocument()
  })
})

// ============================================================================
// Pemilihan nomor halaman yang digambar
// ============================================================================

describe('pageWindow', () => {
  it('menggambar seluruh nomor bila tujuh halaman atau kurang', () => {
    expect(pageWindow(1, 7)).toEqual([1, 2, 3, 4, 5, 6, 7])
    expect(pageWindow(3, 3)).toEqual([1, 2, 3])
  })

  it('menyisipkan sela di tengah pada daftar panjang', () => {
    expect(pageWindow(1, 20)).toEqual([1, 2, 'sela', 20])
    expect(pageWindow(10, 20)).toEqual([1, 'sela', 9, 10, 11, 'sela', 20])
    expect(pageWindow(20, 20)).toEqual([1, 'sela', 19, 20])
  })

  // Halaman pertama dan terakhir SELALU tergambar, berapa pun posisi sekarang: keduanya
  // tujuan yang paling sering dipakai, dan mencapainya tidak boleh menuntut beberapa kali
  // menekan tombol berikutnya.
  it('selalu menggambar halaman pertama dan terakhir', () => {
    for (const current of [1, 2, 5, 15, 19, 20]) {
      const window = pageWindow(current, 20)
      expect(window[0]).toBe(1)
      expect(window[window.length - 1]).toBe(20)
    }
  })

  // Tidak ada sela yang menyembunyikan tepat satu nomor: menggambar "…" untuk satu
  // halaman memakan ruang yang sama dengan nomornya sendiri, dan menghilangkan tujuan
  // yang dapat dicapai sekali tekan.
  it('tidak memakai sela untuk menyembunyikan satu nomor saja', () => {
    expect(pageWindow(4, 20)).toEqual([1, 'sela', 3, 4, 5, 'sela', 20])
    expect(pageWindow(3, 8)).toEqual([1, 2, 3, 4, 'sela', 8])
  })
})
