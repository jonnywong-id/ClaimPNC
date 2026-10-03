import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { DataTable, type Column } from './DataTable'

/** Uji tambahan DataTable: keadaan layar, pencarian server, bilah halaman, baris terbuka. */

type Row = { id: string; nama: string; kota: string }

const ROWS: Row[] = [
  { id: '1', nama: 'Beta', kota: '' },
  { id: '2', nama: 'Alfa', kota: 'Jakarta' },
  { id: '3', nama: 'Gamma', kota: 'Bandung' },
]

const columns: Column<Row>[] = [
  { key: 'nama', title: 'Nama', value: (row) => row.nama, width: '8rem' },
  { key: 'kota', title: 'Kota', value: (row) => row.kota },
  { key: 'aksi', title: 'Aksi', value: () => '', noSort: true, alignRight: true, render: (row) => <span>aksi-{row.id}</span> },
]

function bodyNames() {
  return within(screen.getByRole('table'))
    .getAllByRole('row')
    .slice(1)
    .map((row) => row.textContent)
}

describe('kepala, keadaan memuat, galat, dan kosong', () => {
  it('menggambar judul, keterangan, tombol, dan nama tabel', () => {
    render(
      <DataTable
        columns={columns}
        rows={ROWS}
        rowKey={(r) => r.id}
        title="Daftar Uji"
        description="Keterangan uji."
        label="Tabel uji"
        actions={<button type="button">Tambah</button>}
        dense
      />,
    )

    expect(screen.getByRole('heading', { name: 'Daftar Uji' })).toBeInTheDocument()
    expect(screen.getByText('Keterangan uji.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'Tabel uji' })).toBeInTheDocument()
    // Sel kosong digambar tanda pisah; kolom aksi rata kanan dan tidak dapat diurutkan.
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)
    expect(screen.getByRole('columnheader', { name: 'Aksi' }).className).toContain('text-right')
    expect(screen.getByRole('columnheader', { name: 'Nama' }).className).toContain('px-3')
  })

  it('menampilkan kerangka memuat dan galat menggantikan isi', () => {
    const { rerender } = render(
      <DataTable columns={columns} rows={ROWS} rowKey={(r) => r.id} isLoading />,
    )
    expect(screen.getByText('Memuat data…')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()

    rerender(
      <DataTable
        columns={columns}
        rows={ROWS}
        rowKey={(r) => r.id}
        error={<p>Galat uji</p>}
      />,
    )
    expect(screen.getByText('Galat uji')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('menyembunyikan kotak cari saat tidak dapat dicari', () => {
    render(<DataTable columns={columns} rows={ROWS} rowKey={(r) => r.id} searchable={false} />)
    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument()
  })

  it('menyarankan kata kunci lebih pendek saat pencarian tidak cocok', async () => {
    const user = userEvent.setup()
    render(<DataTable columns={columns} rows={ROWS} rowKey={(r) => r.id} />)

    await user.type(screen.getByRole('searchbox'), 'Zeta')
    expect(screen.getByText('Tidak ada baris yang cocok dengan “Zeta”.')).toBeInTheDocument()
    expect(screen.getByText('Coba kata kunci yang lebih pendek.')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('0 dari 3 baris cocok.')
  })

  it('tetap menggambar kepala kolom saat kosong bila diminta', async () => {
    const user = userEvent.setup()
    render(
      <DataTable
        columns={columns}
        rows={ROWS}
        rowKey={(r) => r.id}
        showHeaderWhenEmpty
        emptyMessage="Tidak ada data."
      />,
    )

    await user.type(screen.getByRole('searchbox'), 'Zeta')
    expect(screen.getByRole('columnheader', { name: 'Nama' })).toBeInTheDocument()
    expect(screen.getByText('Tidak ada baris yang cocok dengan “Zeta”.')).toBeInTheDocument()
  })

  it('menulis pesan kosong bawaan di dalam tabel saat tidak ada baris', () => {
    render(
      <DataTable
        columns={columns}
        rows={[]}
        rowKey={(r) => r.id}
        showHeaderWhenEmpty
        emptyMessage="Tidak ada data."
      />,
    )
    expect(screen.getByRole('columnheader', { name: 'Kota' })).toBeInTheDocument()
    expect(screen.getByText('Tidak ada data.')).toBeInTheDocument()
    expect(screen.queryByText('Coba kata kunci yang lebih pendek.')).not.toBeInTheDocument()
  })
})

describe('pengurutan', () => {
  it('menaik, menurun, lalu kembali ke urutan asli pada klik ketiga', async () => {
    const user = userEvent.setup()
    render(<DataTable columns={columns} rows={ROWS} rowKey={(r) => r.id} />)

    const header = screen.getByRole('button', { name: 'Nama' })
    await user.click(header)
    expect(screen.getByRole('columnheader', { name: 'Nama' })).toHaveAttribute('aria-sort', 'ascending')
    expect(bodyNames()[0]).toContain('Alfa')

    await user.click(header)
    expect(screen.getByRole('columnheader', { name: 'Nama' })).toHaveAttribute('aria-sort', 'descending')
    expect(bodyNames()[0]).toContain('Gamma')

    await user.click(header)
    expect(screen.getByRole('columnheader', { name: 'Nama' })).toHaveAttribute('aria-sort', 'none')
    expect(bodyNames()[0]).toContain('Beta')
  })

  it('mengabaikan urutan bila kolomnya hilang setelah digambar ulang', async () => {
    const user = userEvent.setup()
    const { rerender } = render(<DataTable columns={columns} rows={ROWS} rowKey={(r) => r.id} />)

    await user.click(screen.getByRole('button', { name: 'Nama' }))
    rerender(<DataTable columns={columns.slice(1)} rows={ROWS} rowKey={(r) => r.id} />)

    expect(bodyNames()[0]).toContain('aksi-1')
  })
})

describe('pencarian server', () => {
  function Harness({ onChange }: { onChange: (value: string) => void }) {
    const [value, setValue] = useState('')
    return (
      <DataTable
        columns={columns}
        rows={ROWS}
        rowKey={(r) => r.id}
        serverSearch={{
          value,
          onChange: (next) => {
            setValue(next)
            onChange(next)
          },
          matchCount: value === '' ? undefined : 42,
        }}
      />
    )
  }

  it('tidak menyaring maupun mengurutkan, dan menyebut jumlah dari server', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(<Harness onChange={onChange} />)

    // Pengurutan dimatikan: judul kolom berupa teks biasa.
    expect(screen.queryByRole('button', { name: 'Nama' })).not.toBeInTheDocument()
    await user.type(screen.getByRole('searchbox'), 'Z')

    expect(onChange).toHaveBeenLastCalledWith('Z')
    expect(bodyNames()).toHaveLength(3)
    expect(screen.getByRole('status')).toHaveTextContent('42 baris cocok.')
  })

  it('memakai jumlah baris yang tampil bila server tidak menyebutnya', async () => {
    const user = userEvent.setup()
    function NoCount() {
      const [value, setValue] = useState('')
      return (
        <DataTable
          columns={columns}
          rows={ROWS}
          rowKey={(r) => r.id}
          serverSearch={{ value, onChange: setValue }}
        />
      )
    }
    render(<NoCount />)

    await user.type(screen.getByRole('searchbox'), 'a')
    expect(screen.getByRole('status')).toHaveTextContent('3 baris cocok.')
  })
})

describe('bilah halaman server', () => {
  function show(page: number, totalPage: number, total: number, isLoading = false) {
    const onPageChange = vi.fn()
    render(
      <DataTable
        columns={columns}
        rows={ROWS}
        rowKey={(r) => r.id}
        pagination={{ page, size: 3, total, totalPage, onPageChange, isLoading }}
      />,
    )
    return { onPageChange, nav: screen.getByRole('navigation', { name: 'Navigasi halaman' }) }
  }

  it('menyebut rentang baris dan berpindah ke keempat arah', async () => {
    const user = userEvent.setup()
    const { onPageChange, nav } = show(2, 4, 11)

    expect(within(nav).getByRole('status')).toHaveTextContent('Menampilkan 4–6 · Total Data : 11')
    expect(nav).toHaveTextContent('Hal. 2 dari 4')
    await user.click(within(nav).getByRole('button', { name: 'Halaman pertama' }))
    await user.click(within(nav).getByRole('button', { name: 'Halaman sebelumnya' }))
    await user.click(within(nav).getByRole('button', { name: 'Halaman berikutnya' }))
    await user.click(within(nav).getByRole('button', { name: 'Halaman terakhir' }))
    expect(onPageChange.mock.calls.map((call) => call[0])).toEqual([1, 1, 3, 4])
  })

  it('menulis total nol dan mematikan seluruh tombol saat tidak ada halaman', () => {
    const { nav } = show(1, 0, 0)

    expect(within(nav).getByRole('status')).toHaveTextContent('Total Data : 0')
    expect(nav).toHaveTextContent('Hal. 1 dari 1')
    for (const button of within(nav).getAllByRole('button')) expect(button).toBeDisabled()
  })

  it('mematikan tombol selama halaman lain dimuat', () => {
    const { nav } = show(2, 4, 11, true)
    for (const button of within(nav).getAllByRole('button')) expect(button).toBeDisabled()
  })
})

describe('baris yang dapat dibuka', () => {
  it('membuka satu baris saja, lewat klik maupun papan ketik', async () => {
    const user = userEvent.setup()
    render(
      <DataTable
        columns={columns}
        rows={ROWS}
        rowKey={(r) => r.id}
        expandedRow={(row) => (row.id === '3' ? null : <p>Rincian {row.nama}</p>)}
      />,
    )

    const rows = within(screen.getByRole('table')).getAllByRole('button', { name: /Beta|Alfa/ })
    expect(rows).toHaveLength(2)
    // Baris tanpa isi tidak dijadikan tombol.
    expect(screen.getByText('Gamma').closest('tr')).not.toHaveAttribute('role')

    await user.click(rows[0]!)
    expect(screen.getByText('Rincian Beta')).toBeInTheDocument()
    expect(rows[0]).toHaveAttribute('aria-expanded', 'true')

    rows[1]!.focus()
    await user.keyboard('{Enter}')
    expect(screen.queryByText('Rincian Beta')).not.toBeInTheDocument()
    expect(screen.getByText('Rincian Alfa')).toBeInTheDocument()

    await user.keyboard(' ')
    expect(screen.queryByText('Rincian Alfa')).not.toBeInTheDocument()

    // Tombol lain diabaikan.
    await user.keyboard('a')
    expect(screen.queryByText('Rincian Alfa')).not.toBeInTheDocument()

    await user.click(rows[0]!)
    await user.click(rows[0]!)
    expect(screen.queryByText('Rincian Beta')).not.toBeInTheDocument()
  })
})
