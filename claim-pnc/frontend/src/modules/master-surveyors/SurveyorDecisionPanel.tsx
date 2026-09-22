import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { Surveyor } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useDecideSurveyor } from './api'

type Props = {
  surveyor: Surveyor
  onClose: () => void
}

/**
 * Panel keputusan komite atas satu surveyor.
 *
 * # Asalnya di sistem lama
 *
 * Kedua tombolnya ada di tab "Komite Approval"
 * (`Section/BrowseDetailSuveryorsKomite-Section.xml`) dan memanggil activity yang SAMA
 * dengan tombol Simpan, dibedakan satu parameter:
 *
 *     Approve  →  CNMInsertDetailSurveyors_act( approval = "1" )
 *     Reject   →  CNMInsertDetailSurveyors_act( approval = "2" )
 *     Simpan   →  CNMInsertDetailSurveyors_act( approval = "0" )
 *
 * Tab Waiting, Approve, dan Reject TIDAK punya kedua tombol ini — ketiganya hanya
 * menampilkan. Panel ini karena itu hanya dibuka dari baris yang masih menunggu.
 *
 * # Dua penolakan yang datang dari server, dan tidak diduga di sini
 *
 * Keputusan dari orang yang bukan komite yang ditunjuk ditolak `403`, dan keputusan kedua
 * atas baris yang sama ditolak `409`. Keduanya TIDAK diperiksa di layar: kewenangan yang
 * hanya ditegakkan di peramban bukan kewenangan, dan keadaan barisnya dapat berubah
 * antara saat daftar dimuat dan saat tombol ditekan.
 */
export function SurveyorDecisionPanel({ surveyor, onClose }: Props) {
  const decide = useDecideSurveyor()
  const [note, setNote] = useState('')

  async function submit(status: '1' | '2') {
    await decide.mutateAsync({ id: surveyor.id, status, catatan: note.trim() })
    onClose()
  }

  return (
    <section
      aria-labelledby="judul-keputusan-surveyor"
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 id="judul-keputusan-surveyor" className="text-lg font-semibold text-slate-900">
        Keputusan Komite
      </h2>
      <p className="mt-1 text-sm text-slate-600">
        Surveyor <span className="font-medium text-slate-900">{surveyor.nama}</span> menunggu
        keputusan. Keputusan yang sudah diambil tidak dapat diubah dari layar ini.
      </p>

      {surveyor.komite && (
        <p className="mt-2 text-xs text-slate-500">
          Komite yang ditunjuk:{' '}
          <span className="font-mono text-slate-700">{surveyor.komite}</span>
        </p>
      )}

      <div className="mt-4">
        <label htmlFor="catatan-komite" className="block text-sm font-medium text-slate-700">
          Catatan keputusan
        </label>
        <textarea
          id="catatan-komite"
          value={note}
          onChange={(e) => setNote(e.target.value)}
          rows={3}
          className="mt-1.5 w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
          placeholder="Alasan menyetujui atau menolak."
        />
        <p className="mt-1 text-xs text-slate-500">
          Catatan tersimpan bersama keputusan dan menjadi jejak audit — satu-satunya
          kontrol pengimbang yang ada, karena tidak ada pemisahan tugas formal.
        </p>
      </div>

      {decide.isError && <DecisionErrorMessage error={decide.error} />}

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Button
          type="button"
          tone="utama"
          onClick={() => void submit('1')}
          disabled={decide.isPending}
        >
          {decide.isPending ? 'Menyimpan…' : 'Setujui'}
        </Button>
        <Button
          type="button"
          tone="kedua"
          onClick={() => void submit('2')}
          disabled={decide.isPending}
        >
          Tolak
        </Button>
        <Button type="button" tone="halus" onClick={onClose} disabled={decide.isPending}>
          Batal
        </Button>
      </div>
    </section>
  )
}

function DecisionErrorMessage({ error }: { error: unknown }) {
  const message = decisionMessage(error)
  return (
    <div className="mt-4">
      <ErrorMessage
        title={message.title}
        description={message.description}
        tone={message.tone}
      />
    </div>
  )
}

function decisionMessage(error: unknown): {
  title: string
  description: string
  tone: ErrorTone
} {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Keputusan belum tersimpan. Periksa koneksi lalu coba lagi.',
      tone: 'gangguan',
    }
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case 'bukan_komite_yang_ditunjuk':
        return {
          title: 'Anda bukan komite yang ditunjuk',
          description:
            'Surveyor ini menunggu keputusan komite lain. Tab "Antrean Komite Saya" hanya menampilkan yang menjadi tanggung jawab Anda.',
          tone: 'penolakan',
        }
      case 'keputusan_komite_sudah_diambil':
        return {
          title: 'Keputusan sudah pernah diambil',
          description:
            'Komite lain mungkin baru saja memutuskannya. Tekan Refresh untuk melihat keadaan terbaru.',
          tone: 'penolakan',
        }
      default:
        return { title: 'Keputusan gagal disimpan', description: error.message, tone: 'gangguan' }
    }
  }

  return {
    title: 'Keputusan gagal disimpan',
    description: 'Terjadi kesalahan pada sistem. Coba lagi.',
    tone: 'gangguan',
  }
}
