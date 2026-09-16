/* Membangun docs/Steering/STEERING.md (dokumen gabungan) + STEERING.html
   dari berkas sumber di docs/Steering/.

   Jalankan: node docs/tools/build-steering.js "<root proyek>" ["<path STEERING.html>"]

   STEERING.md adalah HASIL BANGUNAN. Jangan disunting langsung — suntingan akan
   hilang pada pembangunan berikutnya. Yang disunting adalah bab sumbernya.
   Lihat docs/tools/README.md.                                                  */

const fs = require('fs');
const path = require('path');
const os = require('os');
const { mdToHtml, wrapHtml } = require('./md2html');

const ROOT = process.argv[2];
const SRC = path.join(ROOT, 'docs', 'Steering');
const OUT_MD = path.join(SRC, 'STEERING.md');
const OUT_HTML = process.argv[3] || path.join(os.tmpdir(), 'STEERING.html');

const VERSI = '2.0';
const TANGGAL = '14 September 2026';

/* Bab bernomor, berurutan. `judul` yang dipakai adalah judul bab di dokumen
   gabungan — bukan judul berkas sumbernya.                                     */
const BAB = [
  { file: '02-BUSINESS-UNDERSTANDING.md',        judul: 'Pemahaman Bisnis (Business Understanding)' },
  { file: '03-CURRENT-ARCHITECTURE.md',          judul: 'Arsitektur Saat Ini (As-Is)' },
  { file: '04-FUTURE-ARCHITECTURE.md',           judul: 'Arsitektur Masa Depan (To-Be)' },
  { file: '05-DOMAIN-MODEL.md',                  judul: 'Domain Model dan Bounded Context' },
  { file: '06-MODULE-BREAKDOWN.md',              judul: 'Module Breakdown dan Urutan Migrasi' },
  { file: '07-MIGRATION-STRATEGY.md',            judul: 'Strategi Migrasi' },
  { file: '08-TECHNICAL-STRATEGY.md',            judul: 'Strategi Teknis, Struktur Folder, dan Standar Coding' },
  { file: '09-DATABASE-STRATEGY.md',             judul: 'Strategi Database' },
  { file: '10-API-STRATEGY.md',                  judul: 'Strategi API' },
  { file: '11-SECURITY.md',                      judul: 'Keamanan, Autentikasi, dan Otorisasi' },
  { file: '12-CROSSCUTTING.md',                  judul: 'Error Handling, Logging, Konfigurasi, dan Observability' },
  { file: '13-DEPLOYMENT.md',                    judul: 'Strategi Deployment dan Infrastruktur' },
  { file: '14-TESTING-STRATEGY.md',              judul: 'Strategi Testing' },
  { file: '15-NFR-PERFORMANCE-SCALABILITY.md',   judul: 'Non-Functional Requirements, Performa, Skalabilitas, dan Maintainability' },
  { file: '16-RISK-ANALYSIS.md',                 judul: 'Analisis Risiko' },
  { file: '17-FUTURE-ENHANCEMENT.md',            judul: 'Future Enhancement' },
];

/* Lampiran — seluruhnya salinan verbatim berkas sumbernya. */
const LAMPIRAN = [
  { huruf: 'A', file: 'CONTEXT.md',                 judul: 'Glossary Domain (Ubiquitous Language)' },
  { huruf: 'B', file: '00-DECISION-LOG.md',         judul: 'Decision Log' },
  { huruf: 'C', file: '01-FRONTEND-ANALYSIS.md',    judul: 'Analisis Pemilihan Frontend' },
  { huruf: 'D', file: '18-SKILLS-USAGE-LOG.md',     judul: 'Catatan Penggunaan Matt Pocock Skills' },
  { huruf: 'E', file: '19-GAP-EXPORT-DETAIL.md',    judul: 'Rincian Gap Export' },
  { huruf: 'F', file: '20-DETAIL-KOMITE-DBLINK.md', judul: 'Rincian Penjenjangan Komite dan DB Link' },
  { huruf: 'G', file: '22-INVENTARIS-HARNESS.md',   judul: 'Inventaris 74 Harness' },
  { huruf: 'H', file: '23-STATUS-USULAN-REVISI.md', judul: 'Status 40 Usulan Revisi' },
];

const RIWAYAT = '21-RIWAYAT-REVISI.md';

/* ---------- pembantu ---------- */

function baca(nama) {
  const p = path.join(SRC, nama);
  if (!fs.existsSync(p)) throw new Error('Berkas sumber tidak ada: ' + nama);
  return fs.readFileSync(p, 'utf8').replace(/\r\n/g, '\n');
}

/* Buang baris judul `# ...` paling atas beserta baris kosong sesudahnya —
   judulnya diganti judul bab/lampiran yang dihasilkan.                         */
function isiTanpaJudul(teks) {
  const lines = teks.split('\n');
  let i = 0;
  while (i < lines.length && /^\s*$/.test(lines[i])) i++;
  if (i < lines.length && /^#\s+/.test(lines[i])) i++;
  while (i < lines.length && /^\s*$/.test(lines[i])) i++;
  return lines.slice(i).join('\n').trim();
}

/* Cegah tabrakan tingkat heading: di dalam dokumen gabungan, `#` hanya dipakai
   untuk judul bab dan lampiran. Heading `#` di dalam berkas sumber diturunkan.  */
function turunkanH1(teks) {
  return teks.split('\n').map(l => (/^#\s+/.test(l) ? '#' + l : l)).join('\n');
}

function anchor(text) {
  return text.toLowerCase()
    .replace(/[`*_[\]()]/g, '')
    .replace(/[^a-z0-9 \-]/g, '')
    .trim().replace(/\s+/g, '-');
}

/* ---------- susun ---------- */

const md = [];
const P = s => md.push(s);

P('# STEERING DOCUMENT');
P('');
P('## Migrasi Aplikasi CLAIM PNC');
P('### Dari Pega PRPC 8.3 ke Golang + React + PostgreSQL');
P('');
P('| | |');
P('|---|---|');
P('| **Status** | Draft v' + VERSI + ' — Menunggu Review |');
P('| **Owner** | PT. Asuransi Sinar Mas — Claim PNC Migration Project |');
P('| **Tanggal** | ' + TANGGAL + ' |');
P('| **Versi** | ' + VERSI + ' |');
P('| **Klasifikasi** | CONFIDENTIAL |');
P('| **Traceability** | Seluruh keputusan bersumber dari Lampiran B — Decision Log (`D-01` … `D-72`) |');
P('| **Revisi** | v' + VERSI + ' — menyerap 41 keputusan Sesi 3, 14 temuan verifikasi bukti, dan 29 ADR. Ringkasan perubahan ada di bab **Riwayat Revisi** |');
P('| **Basis Teknis** | Seluruh klaim teknis bersumber dari pembacaan langsung export rule Pega: **2.634 berkas XML**, ditambah 55 `.prc` dan 8 `.fnc` |');
P('| **Dokumen terkait** | `docs/ADR/` — 29 Architecture Decision Record · `docs/BRD/BRD.md` · `docs/ticketing/` |');
P('');
P('— CONFIDENTIAL —');
P('');
P('Dokumen ini adalah blueprint migrasi. Tidak berisi source code, API schema, atau database');
P('migration script — hal tersebut disusun setelah Steering ini disetujui.');
P('');
P('> **Dokumen ini dibangun otomatis** dari berkas sumber di `docs/Steering/` oleh');
P('> `docs/tools/build-steering.js`. **Jangan disunting langsung** — suntingan akan hilang pada');
P('> pembangunan berikutnya. Yang disunting adalah bab sumbernya.');
P('');
P('---');
P('');

/* daftar isi */
P('## Daftar Isi');
P('');
P('**Riwayat Revisi** — apa yang berubah sejak v1.0 dan atas dasar apa');
P('');
BAB.forEach((b, idx) => {
  P((idx + 1) + '. [' + b.judul + '](#' + anchor((idx + 1) + '. ' + b.judul) + ')');
});
P('');
LAMPIRAN.forEach(l => {
  P('- [Lampiran ' + l.huruf + ' — ' + l.judul + '](#' + anchor('Lampiran ' + l.huruf + ' — ' + l.judul) + ')');
});
P('');
P('---');
P('');

/* riwayat revisi — di depan, tanpa nomor bab */
P(turunkanH1(baca(RIWAYAT)));
P('');
P('---');
P('');

/* bab bernomor */
BAB.forEach((b, idx) => {
  P('# ' + (idx + 1) + '. ' + b.judul);
  P('');
  P(turunkanH1(isiTanpaJudul(baca(b.file))));
  P('');
});

/* lampiran */
LAMPIRAN.forEach(l => {
  P('# Lampiran ' + l.huruf + ' — ' + l.judul);
  P('');
  P(turunkanH1(isiTanpaJudul(baca(l.file))));
  P('');
});

const teks = md.join('\n').replace(/\n{3,}/g, '\n\n') + '\n';
fs.writeFileSync(OUT_MD, teks, 'utf8');

const html = wrapHtml('STEERING DOCUMENT — Migrasi Aplikasi CLAIM PNC', mdToHtml(teks));
fs.writeFileSync(OUT_HTML, html, 'utf8');

console.log('MD   : ' + OUT_MD + '  (' + teks.split('\n').length + ' baris)');
console.log('HTML : ' + OUT_HTML);
console.log('Bab  : ' + BAB.length + ' · Lampiran: ' + LAMPIRAN.length);
