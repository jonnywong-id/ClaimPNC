import { useState } from 'react'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { Button } from '@/components/Button'

type Props = {
  /** Endpoint pembuat klaim, mis. `/api/inbox-claim-treaty-prop/klaim`. */
  path: string
  label: string
  /** Pesan bila server ternyata SUDAH dapat membuat klaim. */
  readyNotice: string
  describe: (error: unknown) => string
  /** Kelas jarak antara tombol dan pesannya. */
  gapClassName: string
}

/**
 * Tombol pembuat klaim yang DIGAMBAR meski belum membuat apa pun.
 *
 * Di sistem lama ia membuat objek kerja baru di tabel yang selama masa paralel masih
 * dimiliki Pega (`P-1`). Menyembunyikannya membuat pengguna mengira fiturnya hilang;
 * menghidupkannya membuat dua sistem menulis tabel yang sama. Yang dilakukan tombol ini
 * adalah bertanya ke server lalu menampilkan jawabannya — sehingga alasannya datang dari
 * satu tempat, dan hilang dengan sendirinya begitu kepemilikan tabelnya berpindah.
 */
export function CreateClaimButton({ path, label, readyNotice, describe, gapClassName }: Readonly<Props>) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const [notice, setNotice] = useState('')
  const [asking, setAsking] = useState(false)

  async function ask() {
    setAsking(true)
    try {
      await callAPI(path, {
        metode: 'POST',
        token,
        portal,
      })
      // Jalur ini tercapai hanya bila server SUDAH dapat membuat klaim. Selama itu belum
      // terjadi, ia tidak pernah berjalan — dan bila kelak berjalan, pesan ini yang
      // pertama memberi tahu bahwa perilakunya berubah.
      setNotice(readyNotice)
    } catch (error) {
      setNotice(describe(error))
    } finally {
      setAsking(false)
    }
  }

  return (
    <div className={`flex flex-col items-end ${gapClassName}`}>
      <Button tone="utama" onClick={ask} disabled={asking || portal === null}>
        {label}
      </Button>
      {notice !== '' && (
        <output className="block max-w-md text-right text-xs text-amber-900">
          {notice}
        </output>
      )}
    </div>
  )
}
