import type { Account } from '@/api/types'

type Props = {
  rows: Account[]
  loading: boolean
  /** Tindakan per baris; kosong berarti tabel hanya dibaca. */
  aksi?: ((rekening: Account) => React.ReactNode) | undefined
}

/**
 * TabelRekening menampilkan daftar rekening.
 *
 * Kelima tab layar memakai tabel yang sama; yang berbeda hanya saringan dan tindakan
 * per barisnya. Membuat lima tabel untuk lima tab berarti lima tempat yang harus
 * diubah setiap kali satu kolom bertambah — persis yang terjadi di sistem lama, tempat
 * kelima tabnya adalah lima section terpisah.
 *
 * Komponen tabel baku milik seluruh aplikasi adalah TKT-U2-001, yang belum ada. Saat
 * ia ada, tabel ini diganti dengannya; kolom dan tindakannya sudah dipisahkan supaya
 * penggantian itu tidak menyentuh isi layar.
 */
export function AccountTable({ rows, loading, aksi }: Props) {
  if (loading) {
    return <p className="py-8 text-center text-sm text-slate-500">Memuat data rekening…</p>
  }
  if (rows.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-slate-500">
        Tidak ada rekening yang cocok dengan search Anda.
      </p>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[56rem] border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-300 text-left text-xs uppercase tracking-wide text-slate-500">
            <th scope="col" className="py-2 pr-4 font-medium">No Rekening</th>
            <th scope="col" className="py-2 pr-4 font-medium">Nama Pemilik</th>
            <th scope="col" className="py-2 pr-4 font-medium">Bank</th>
            <th scope="col" className="py-2 pr-4 font-medium">Cabang</th>
            <th scope="col" className="py-2 pr-4 font-medium">Tipe</th>
            <th scope="col" className="py-2 pr-4 font-medium">Status</th>
            <th scope="col" className="py-2 pr-4 font-medium">Kasir</th>
            {aksi && <th scope="col" className="py-2 font-medium">Tindakan</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={`${r.kode_bank}-${r.nomor_rekening}`} className="border-b border-slate-200">
              <td className="py-2 pr-4 font-mono">{r.nomor_rekening}</td>
              <td className="py-2 pr-4">{r.nama_pemilik}</td>
              <td className="py-2 pr-4">{r.nama_bank}</td>
              <td className="py-2 pr-4">{r.cabang_bank}</td>
              <td className="py-2 pr-4">{r.tipe_rekening}</td>
              <td className="py-2 pr-4">
                <StatusBadge rekening={r} />
              </td>
              <td className="py-2 pr-4">
                <CashierNote rekening={r} />
              </td>
              {aksi && <td className="py-2">{aksi(r)}</td>}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/**
 * LencanaStatus menampilkan posisi persetujuan.
 *
 * Rekening yang sudah disetujui tetapi dinonaktifkan ditandai terpisah: tanpa itu,
 * layar menampilkannya sebagai "Komite Approve" dan petugas mengira rekening itu masih
 * dapat dipakai membayar klaim.
 */
function StatusBadge({ rekening }: { rekening: Account }) {
  const color =
    rekening.status === '1'
      ? 'bg-green-100 text-green-800'
      : rekening.status === '2'
        ? 'bg-red-100 text-red-800'
        : 'bg-amber-100 text-amber-900'

  return (
    <span className="inline-flex flex-col gap-1">
      <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${color}`}>
        {rekening.status_label || 'Tidak dikenal'}
      </span>
      {rekening.status === '1' && !rekening.aktif && (
        <span className="text-xs text-slate-500">nonaktif — tidak dapat dipakai</span>
      )}
    </span>
  )
}

/**
 * KeteranganKasir menampilkan hasil pendaftaran ke sistem Kasir.
 *
 * Kegagalannya ditampilkan, bukan disembunyikan: rekening yang disetujui komite tetapi
 * gagal didaftarkan ke Kasir akan menahan pembayaran, dan satu-satunya orang yang dapat
 * menindaklanjutinya adalah petugas yang melihat layar ini.
 */
function CashierNote({ rekening }: { rekening: Account }) {
  if (rekening.status_layanan === '') {
    return <span className="text-xs text-slate-400">—</span>
  }
  if (rekening.status_layanan === 'BERHASIL') {
    return (
      <span className="text-xs text-slate-600">
        Terdaftar{rekening.id_rekening_kasir && ` · ${rekening.id_rekening_kasir}`}
      </span>
    )
  }
  return (
    <span className="text-xs text-red-700">
      Gagal{rekening.respons_kasir && ` · ${rekening.respons_kasir}`}
    </span>
  )
}
