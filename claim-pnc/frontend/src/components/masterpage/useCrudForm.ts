import { useState } from 'react'

type Mutation<TVariables> = {
  reset: () => void
  isPending: boolean
  error: unknown
  mutate: (variables: TVariables, options?: { onSuccess?: () => void }) => void
}

/**
 * Keadaan form tambah/ubah bersama layar master yang menyimpan lewat dua mutasi — satu
 * untuk menambah, satu untuk mengubah.
 *
 * Keadaannya memikul tiga hal sekaligus — tertutup, tambah, atau baris yang sedang disunting.
 * Menyimpannya sebagai dua state terpisah (`isOpen` + `edited`) membuka keadaan yang tidak
 * masuk akal: terbuka tanpa mode.
 *
 * Setiap kali form dibuka atau ditutup, galat kedua mutasi dibersihkan supaya pesan gagal
 * dari percobaan sebelumnya tidak terbawa ke form berikutnya.
 */
export function useCrudForm<TRow, TCreate, TUpdate>(
  create: Mutation<TCreate>,
  update: Mutation<TUpdate>,
  /** Membentuk variabel mutasi ubah dari baris yang sedang disunting dan isiannya. */
  toUpdate: (row: TRow, input: TCreate) => TUpdate,
) {
  // null = tertutup; { row: null } = sedang menambah; { row } = sedang mengubah.
  const [state, setState] = useState<{ row: TRow | null } | null>(null)

  function resetBoth() {
    create.reset()
    update.reset()
  }

  function closeForm() {
    resetBoth()
    setState(null)
  }

  const openedRow = state === null ? null : state.row

  return {
    isOpen: state !== null,
    openedRow,
    isSaving: create.isPending || update.isPending,
    saveError: openedRow ? update.error : create.error,
    openCreate() {
      resetBoth()
      setState({ row: null })
    },
    openEdit(row: TRow) {
      resetBoth()
      setState({ row })
    },
    closeForm,
    /**
     * Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
     * membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
     * diketik, itu berarti mengetik ulang dari awal.
     */
    submit(input: TCreate) {
      if (openedRow) {
        update.mutate(toUpdate(openedRow, input), { onSuccess: closeForm })
        return
      }
      create.mutate(input, { onSuccess: closeForm })
    },
  }
}
