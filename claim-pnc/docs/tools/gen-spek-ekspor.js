// Menyusun spesifikasi berkas ekspor KPI dari export Pega, apa adanya.
//
// Sumbernya `pyStepsCallParams` pada pemanggilan `pxConvertResultsToCSV` — di situlah
// baris judul (`CSVPropHeaders`), daftar properti (`CSVProperties`), dan nama berkas
// (`FileName`) ditetapkan pemanggilnya.
const fs = require('fs')
const path = require('path')

const akar = process.argv[2]
const tujuan = process.argv[3]

const aktivitas = [
  ['PNCReportKPIAdjuster_act', 'tab KPI Adjuster'],
  ['PNCReportKPIAdmin_Act', 'tab KPI Admin'],
  ['PNCReportKPI_act', 'tab KPI PIC Teknik'],
  ['EksportDataAllKPIPICKlaim', 'tab KPI PIC Teknik - Pilih Data KPI'],
  ['ExportKPILoginAdjuster', 'KPI Login Adjuster (di luar lingkup layar ini)'],
]

function urai(berkas) {
  const mentah = fs.readFileSync(berkas, 'utf8')
  return mentah
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&amp;/g, '&')
}

function daftar(teks) {
  return teks
    .trim()
    .replace(/"/g, '')
    .split(',')
    .map((s) => s.trim())
    .filter((s) => s !== '')
}

const keluar = []
keluar.push('# Spesifikasi berkas ekspor Report KPI — menurut Pega')
keluar.push('')
keluar.push('> **Dibangun otomatis** dari `Activity/*.xml` oleh `docs/tools/gen-spek-ekspor.js`.')
keluar.push('> Jangan disunting dengan tangan.')
keluar.push('')
keluar.push('Seluruh ekspor KPI di Pega berbentuk **CSV**, dibuat `pxConvertResultsToCSV`.')
keluar.push('Baris judulnya TIDAK diturunkan dari alias SQL melainkan ditetapkan pemanggil lewat')
keluar.push('parameter `CSVPropHeaders`, dan kolom yang ditulis dipilih lewat `CSVProperties`.')
keluar.push('')
keluar.push('Akibatnya dua hal yang mudah keliru:')
keluar.push('')
keluar.push('1. **Jumlah kolom berkas tidak sama dengan jumlah alias kuerinya.** Kueri Progress')
keluar.push('   mengembalikan 43 alias; berkasnya 32 kolom.')
keluar.push('2. **Judul kolomnya bahasa manusia**, bukan alias. `No Klaim`, bukan `CLAIMNO`.')
keluar.push('')

for (const [nama, keterangan] of aktivitas) {
  const berkas = path.join(akar, 'Activity', nama + '-Act.xml')
  if (!fs.existsSync(berkas)) continue

  const isi = urai(berkas)
  const blok = [...isi.matchAll(/<pyStepsCallParams>([\s\S]*?)<\/pyStepsCallParams>/g)].map(
    (m) => m[1],
  )

  const ekspor = []
  for (const b of blok) {
    const judul = b.match(/<CSVPropHeaders>([\s\S]*?)<\/CSVPropHeaders>/)
    if (!judul) continue
    const properti = b.match(/<CSVProperties>([\s\S]*?)<\/CSVProperties>/)
    const berkasNama = b.match(/<FileName>([\s\S]*?)<\/FileName>/)
    ekspor.push({
      judul: daftar(judul[1]),
      properti: properti ? daftar(properti[1]) : [],
      berkas: berkasNama ? berkasNama[1].trim() : '(tidak disetel)',
    })
  }
  if (ekspor.length === 0) continue

  keluar.push('## `' + nama + '` — ' + keterangan)
  keluar.push('')
  for (const e of ekspor) {
    keluar.push('### Berkas: ' + e.berkas)
    keluar.push('')
    keluar.push('**' + e.judul.length + ' kolom.**')
    keluar.push('')
    keluar.push('| # | Judul kolom | Properti sumber |')
    keluar.push('|---:|---|---|')
    for (let i = 0; i < e.judul.length; i++) {
      keluar.push('| ' + (i + 1) + ' | ' + e.judul[i] + ' | `' + (e.properti[i] ?? '') + '` |')
    }
    keluar.push('')
  }
}

fs.writeFileSync(tujuan, keluar.join('\n') + '\n', 'utf8')
console.log('ditulis: ' + tujuan)
