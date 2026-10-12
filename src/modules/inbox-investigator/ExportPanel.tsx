import { useState } from 'react'

import { Button } from '@/components/Button'
import { DateField } from '@/components/DateField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { SelectField } from '@/components/SelectField'

import { useExportInvestigation } from './api'

/**
 * Pilihan dropdown "Pilih Investigation".
 *
 * Kedua NILAINYA — `"1"` dan `"0"` — terbaca dari data: `SurveyList[0].IsInvestigated` pada
 * dokumen klaim hanya pernah berisi keduanya. Nilai itulah yang dibandingkan syarat
 * penyalinan di `Activity/ExportDataInvestigator-Act.xml:2112`, dan nilai itu pula yang
 * dikirim ke server.
 *
 * LABELNYA ditambahkan di sini, dan itu dinyatakan: rule Property `IsInvestigated` termasuk
 * ±242 rule yang hilang dari export (`R-16`), sehingga tidak ada artefak yang memberi tahu
 * kata apa yang dibaca pengguna di dropdown lama. Yang dipilih adalah kata yang menyebut
 * artinya, bukan kodenya — dropdown berisi "1" dan "0" menuntut pengguna menghafal sesuatu
 * yang tidak perlu dihafal.
 */
const INVESTIGATION_OPTIONS = [
  { value: '1', label: 'Sudah diinvestigasi' },
  { value: '0', label: 'Belum diinvestigasi' },
]

/**
 * ExportPanel adalah tombol **Export Data Investigation** beserta ketiga kendalinya.
 *
 * # Asalnya
 *
 * Ketiganya ada di kepala `Section/InputInvestigator_Section-Section.xml`, seluruhnya
 * ber-`pyVisible = ALWAYS`:
 *
 *	"Dari"                isian tanggal, `TempInvestigate.DateOfLoss`
 *	"Sampai"              isian tanggal, `TempInvestigate.DateReceived`
 *	"Pilih Investigation" dropdown,      `TempInvestigateChose.IsInvestigated`
 *
 * # Letaknya: DI ATAS daftar, mengikuti layar lama
 *
 * Ia panel tersendiri, bukan tombol di kepala halaman, karena ketiga kendalinya BUKAN
 * penyaring daftar: rentang tanggalnya menyaring `INVESTIGATOR_TF_DATE` — kolom yang tidak
 * dibaca grid sama sekali — dan berkasnya memuat klaim yang sudah selesai dan tidak lagi
 * tampil di layar.
 *
 * Panel ini sempat ditaruh di bawah daftar supaya kekeliruan itu lebih sulit terjadi.
 * Letaknya dikembalikan ke ATAS daftar atas permintaan Work Owner, karena di layar lama ia
 * memang di sana (`D-13`): urutan yang dikenal pengguna lebih berharga daripada penjagaan
 * yang kita karang sendiri.
 *
 * # Nama isiannya mengikuti layar lama, bukan artinya
 *
 * "Dari" dan "Sampai" dipakai apa adanya (`D-13`), meski property Pega di baliknya bernama
 * `DateOfLoss` dan `DateReceived` — dua nama yang menyesatkan, sebab yang disaring bukan
 * tanggal kejadian maupun tanggal terima dokumen. Yang diperbaiki adalah nama di dalam
 * kode, bukan kata yang dibaca pengguna.
 *
 * # TANPA satu pun teks yang tidak ada di Pega
 *
 * Dua paragraf sempat ditulis di sini — satu menjelaskan bahwa isi berkasnya berbeda dari
 * daftar, satu memperingatkan bahwa berkasnya memuat data medis — dan **keduanya dicabut**.
 * Layar lama tidak punya teks apa pun di tempat ini, dan tambahan yang alasannya masuk akal
 * tetap membuat layar baru terasa berbeda tanpa sebab.
 *
 * Yang ditulis untuk pengguna hanyalah pesan yang muncul ketika ada yang salah.
 */
export function ExportPanel() {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [investigated, setInvestigated] = useState('')
  const [problem, setProblem] = useState<string | null>(null)

  const exporting = useExportInvestigation()

  /*
    Ketiganya diperiksa DI LAYAR lebih dulu, meski server memeriksanya lagi.

    Bukan karena pemeriksaan server kurang: ia tetap yang menentukan. Yang dilakukan di sini
    hanyalah menjawab lebih cepat dan tanpa mengunduh apa pun — sebab galat yang datang
    sebagai berkas unduhan jauh lebih sulit dibaca pengguna daripada kalimat di bawah
    tombolnya.
  */
  function validate(): string | null {
    if (!from) return 'Isi tanggal "Dari" lebih dulu.'
    if (!to) return 'Isi tanggal "Sampai" lebih dulu.'
    if (to < from) return 'Tanggal "Sampai" tidak boleh lebih awal daripada "Dari".'
    if (!investigated) return 'Pilih salah satu pada "Pilih Investigation".'
    return null
  }

  function run() {
    const found = validate()
    setProblem(found)
    if (found) return
    exporting.mutate({ dari: from, sampai: to, investigasi: investigated })
  }

  const failure =
    exporting.error instanceof Error ? exporting.error.message : null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50/60 p-4">
      {/* Judulnya saja, tanpa kalimat penjelas.

          Keterangan "isinya berbeda dari daftar di atas" sempat ditulis di sini dan DICABUT:
          layar lama tidak punya teks apa pun di tempat ini, dan tambahan yang tidak ada di
          Pega membuat layar baru terasa berbeda tanpa sebab. Penjelasannya tinggal di
          dokumen, tempat ia memang dibaca. */}
      <h2 className="text-sm font-semibold text-slate-800">Export Data Investigation</h2>

      <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4 lg:items-end">
        <DateField
          id="ekspor-dari"
          label="Dari"
          value={from}
          onChange={setFrom}
          disabled={exporting.isPending}
        />
        <DateField
          id="ekspor-sampai"
          label="Sampai"
          value={to}
          onChange={setTo}
          disabled={exporting.isPending}
        />
        <SelectField
          id="ekspor-investigasi"
          label="Pilih Investigation"
          options={INVESTIGATION_OPTIONS}
          value={investigated}
          onChange={(event) => setInvestigated(event.target.value)}
          disabled={exporting.isPending}
        />
        <div>
          <Button onClick={run} disabled={exporting.isPending}>
            {exporting.isPending ? 'Menyiapkan…' : 'Export Data Investigation'}
          </Button>
        </div>
      </div>

      {problem && (
        <p className="mt-3 text-sm text-rose-700" role="alert">
          {problem}
        </p>
      )}

      {!problem && failure && (
        <div className="mt-3">
          <ErrorMessage
            title="Berkas investigasi tidak dapat diunduh"
            description={failure}
            tone="gangguan"
          />
        </div>
      )}

    </section>
  )
}
