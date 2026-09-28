import { useRef, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useDokumenPenunjang, useUnggahDokumen } from './api'
import { FITUR_DOKUMEN_PENUNJANG_AKTIF } from './fitur'
import type { DokumenPenunjang } from './types'

/**
 * Panel "Unggah File Penunjang", dipakai KEDUA layar Open Protection.
 *
 * # Kenapa satu komponen, bukan satu per layar
 *
 * Input Req Protection dan Inbox Accept Open Protection mengunggah hal yang sama ke klaim
 * yang sama. Dua salinan akan berbeda diam-diam begitu salah satunya diperbaiki, dan
 * bedanya baru terlihat sebagai berkas yang tidak muncul di layar sebelah.
 *
 * Di Pega pun demikian: keduanya memanggil flow action yang SAMA, `SetUploadDocPUCL`
 * (`Section/InputProtectionSection-Section.xml:8634`, `:8777` dan
 * `Section/AcceptProtectionSection-Section.xml:8812`, `:8906`).
 *
 * # Ia hidup di modulnya sendiri, bukan di salah satu modul proteksi
 *
 * Modul fitur TIDAK boleh mengimpor dari modul fitur lain
 * (`08-TECHNICAL-STRATEGY.md` §3 aturan 1). Menaruhnya di `input-req-protection` lalu
 * mengimpornya dari `inbox-accept-open-protection` melanggar itu; menaruhnya di modulnya
 * sendiri tidak.
 */
export function DokumenPenunjangPanel({
  nomorKlaim,
  readOnly = false,
}: {
  /** Kosong berarti klaim belum dipilih — panel menjelaskan itu alih-alih tampil kosong. */
  nomorKlaim: string | null

  /**
   * Menyembunyikan tombol unggah, daftarnya tetap terbaca.
   *
   * Dipakai ketika permintaan proteksinya sudah diputuskan: dokumennya tetap perlu dibuka
   * untuk ditinjau, tetapi menambah berkas baru pada permintaan yang sudah selesai tidak
   * lagi bermakna.
   */
  readOnly?: boolean
}) {
  const nomor = nomorKlaim?.trim() ?? ''

  // Kedua hook tetap dipanggil meski fiturnya mati — aturan hook melarang pemanggilan
  // bersyarat. Yang dimatikan adalah PERMINTAANNYA: nomor `null` membuat `enabled` bernilai
  // salah, sehingga tidak ada satu pun tembakan ke server.
  const nomorAktif = FITUR_DOKUMEN_PENUNJANG_AKTIF && nomor !== '' ? nomor : null
  const daftar = useDokumenPenunjang(nomorAktif)
  const unggah = useUnggahDokumen(nomorAktif)
  const inputRef = useRef<HTMLInputElement>(null)
  const [pesanGagal, setPesanGagal] = useState<string | null>(null)
  const [nadaGagal, setNadaGagal] = useState<'penolakan' | 'gangguan'>('penolakan')

  // `?? []` bukan kehati-hatian berlebih. Badan respons datang dari jaringan, dan `as T`
  // pada `callAPI` TIDAK memeriksa apa pun saat berjalan — catatan yang sudah tertulis di
  // `src/api/client.ts`. Respons yang kehilangan `data` karena itu akan membuat layar ini
  // galat total, bukan menampilkan daftar kosong.
  const isi = daftar.data?.data ?? []

  async function pilihBerkas(berkas: File | null) {
    if (!berkas || nomor === '') return
    setPesanGagal(null)

    try {
      await unggah.mutateAsync({ nomorKlaim: nomor, berkas })
    } catch (error) {
      if (error instanceof APIError) {
        // Nada dipilih dari KODE, bukan dari status: "tersimpan_sebagian" berstatus 500
        // tetapi bukan gangguan yang boleh diulang — pesannya justru melarang mengulang.
        setNadaGagal(
          error.kode === 'layanan_penyimpanan_gagal' ? 'gangguan' : 'penolakan',
        )
        setPesanGagal(error.message)
      } else {
        setNadaGagal('gangguan')
        setPesanGagal('Berkas gagal diunggah. Periksa sambungan lalu coba lagi.')
      }
    } finally {
      // Nilai input DIKOSONGKAN, berhasil maupun gagal. Tanpa ini, memilih berkas yang
      // SAMA dua kali berturut-turut tidak memicu `change` sama sekali — dan pengguna
      // melihat tombolnya seperti tidak bekerja.
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <section className="mt-6 rounded-xl border border-slate-200 bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-slate-800">Unggah File Penunjang</h3>

        {!FITUR_DOKUMEN_PENUNJANG_AKTIF && (
          // Tombolnya TAMPIL tetapi mati, bukan disembunyikan.
          //
          // Menyembunyikannya akan membuat layar ini tampak tidak punya fitur unggah sama
          // sekali, dan pertanyaan "kok hilang?" berulang. Tombol mati beserta sebabnya
          // menyatakan hal yang benar: fiturnya ada, layanannya yang belum.
          //
          // `disabled` — bukan tombol hidup yang tidak melakukan apa-apa. Tombol yang
          // ditekan tanpa akibat terbaca sebagai rusak, dan pengguna akan menekannya
          // berkali-kali sebelum melaporkannya.
          <Button tone="kedua" disabled title="Layanan penyimpanan dokumen belum tersedia">
            Pilih berkas
          </Button>
        )}

        {FITUR_DOKUMEN_PENUNJANG_AKTIF && !readOnly && nomor !== '' && (
          <>
            <input
              ref={inputRef}
              type="file"
              className="sr-only"
              // `aria-label`, bukan `id` + `<label htmlFor>`: input ini disembunyikan dan
              // dipicu tombol di sebelahnya, sehingga tidak ada teks di layar yang layak
              // menjadi labelnya. Tanpa nama sama sekali, pembaca layar mengumumkannya
              // hanya sebagai "file upload".
              aria-label="Pilih berkas untuk diunggah"
              onChange={(e) => void pilihBerkas(e.target.files?.[0] ?? null)}
              disabled={unggah.isPending}
            />
            <Button
              tone="kedua"
              onClick={() => inputRef.current?.click()}
              disabled={unggah.isPending}
            >
              {unggah.isPending ? 'Mengunggah…' : 'Pilih berkas'}
            </Button>
          </>
        )}
      </div>

      {!FITUR_DOKUMEN_PENUNJANG_AKTIF ? (
        <p className="mt-3 text-sm text-slate-600">
          Layanan penyimpanan dokumen belum tersedia, sehingga berkas belum dapat diunggah
          maupun ditampilkan di sini.
        </p>
      ) : nomor === '' ? (
        <p className="mt-3 text-sm text-slate-600">
          Pilih No Klaim lebih dulu. Dokumen penunjang menempel pada klaim, bukan pada
          permintaan proteksinya.
        </p>
      ) : (
        <>
          {pesanGagal && (
            <div className="mt-3">
              <ErrorMessage
                tone={nadaGagal}
                title="Berkas tidak tersimpan"
                description={pesanGagal}
              />
            </div>
          )}

          {daftar.isPending && (
            <p className="mt-3 text-sm text-slate-500">Memuat daftar dokumen…</p>
          )}

          {daftar.isError && (
            <div className="mt-3">
              <ErrorMessage
                tone="gangguan"
                title="Daftar dokumen gagal dimuat"
                description="Muat ulang halaman untuk mencoba lagi."
              />
            </div>
          )}

          {daftar.isSuccess && isi.length === 0 && (
            <p className="mt-3 text-sm text-slate-600">
              Belum ada dokumen penunjang untuk klaim ini.
            </p>
          )}

          {isi.length > 0 && (
            <ul className="mt-3 divide-y divide-slate-100">
              {isi.map((dokumen) => (
                <BarisDokumen key={dokumen.id} dokumen={dokumen} />
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}

function BarisDokumen({ dokumen }: { dokumen: DokumenPenunjang }) {
  // Tautan hanya dipasang bila ada alamatnya DAN belum kedaluwarsa. Tautan mati yang
  // tetap dapat diklik membuat pengguna mengira berkasnya hilang; teks tanpa tautan
  // beserta sebabnya jauh lebih terbaca.
  const dapatDibuka = dokumen.url !== '' && !dokumen.kedaluwarsa

  return (
    <li className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-2">
      <span className="text-sm text-slate-800">
        {dapatDibuka ? (
          <a
            href={dokumen.url}
            target="_blank"
            rel="noreferrer"
            className="font-medium text-blue-700 underline underline-offset-2 hover:text-blue-800"
          >
            {dokumen.nama_berkas}
          </a>
        ) : (
          <span className="font-medium">{dokumen.nama_berkas}</span>
        )}
        {dokumen.jenis_dokumen && (
          <span className="ml-2 text-slate-500">· {dokumen.jenis_dokumen}</span>
        )}
      </span>

      <span className="text-xs text-slate-500">
        {dokumen.tanggal_unggah && <>Diunggah {dokumen.tanggal_unggah}</>}
        {dokumen.kedaluwarsa && (
          // Ditandai teks, bukan warna saja: pembeda yang hanya warna tidak terbaca
          // pengguna buta warna, dan yang dibedakan di sini adalah dapat-tidaknya dibuka.
          <span className="ml-2 font-medium text-amber-700">· tautan kedaluwarsa</span>
        )}
        {!dokumen.kedaluwarsa && dokumen.url === '' && (
          <span className="ml-2 font-medium text-slate-600">· tautan belum siap</span>
        )}
      </span>
    </li>
  )
}
