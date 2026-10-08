import type { ReactNode } from 'react'

import type { Column } from '@/components/DataTable'

/** Satu kolom yang bentuknya ditetapkan server. */
export type ServerColumn = {
  kunci: string
  judul: string
}

/**
 * serverColumns menyusun kolom tabel dari daftar kolom yang dikirim server.
 *
 * Daftar kolom datang dari server karena tiap tab punya kolom yang berbeda, dan daftar
 * itu hasil pembacaan export Pega yang tercatat di backend — menyalinnya ke layar berarti
 * daftar yang sama hidup di dua tempat.
 *
 * Kolom yang tercantum di `renders` tetap punya `value` berupa teks polos. `Column.value`
 * adalah yang dicari dan diurutkan, sedangkan `render` yang dilihat; menyatukannya akan
 * membuat pengurutan menelusuri markup tautannya, bukan nomornya.
 */
export function serverColumns<T, C extends ServerColumn>(
  kolom: C[],
  text: (row: T, column: C) => string,
  renders: Partial<Record<string, (row: T) => ReactNode>> = {},
): Column<T>[] {
  return kolom.map((column) => {
    const base: Column<T> = {
      key: column.kunci,
      title: column.judul,
      value: (row) => text(row, column),
    }

    const render = renders[column.kunci]
    if (render) {
      return { ...base, render }
    }
    return base
  })
}

/**
 * columnsWithAction menyusun kolom dari server lalu menambahkan kolom aksi di ujung.
 *
 * Kolom aksi ditambahkan layar, bukan disebut server: ia bukan DATA melainkan kontrol, dan
 * backend tidak tahu apa pun tentang rute antarmuka. `decorate` menambahkan sifat tampilan
 * per kolom (mis. rata kanan untuk angka).
 */
export function columnsWithAction<T, C extends ServerColumn>(
  kolom: C[],
  text: (row: T, column: C) => string,
  action: (row: T) => ReactNode,
  decorate?: (column: C) => Partial<Column<T>>,
): Column<T>[] {
  const columns: Column<T>[] = kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => text(row, column),
    ...decorate?.(column),
  }))

  columns.push(actionColumn(action))

  return columns
}

/**
 * actionColumn menyusun kolom aksi tanpa judul di ujung kanan tabel.
 *
 * Isinya tombol, sehingga ia tidak layak diurutkan dan tidak punya teks untuk dicari.
 */
export function actionColumn<T>(render: (row: T) => ReactNode): Column<T> {
  return {
    key: 'aksi',
    title: '',
    value: () => '',
    render,
    noSort: true,
    alignRight: true,
  }
}
