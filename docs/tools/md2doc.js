/* Mengubah satu berkas Markdown menjadi HTML bergaya, siap dikonversi ke .docx
   oleh html2docx.ps1.

   Jalankan: node docs/tools/md2doc.js "<berkas.md>" "<keluaran.html>" "<judul>"

   Dipakai untuk dokumen yang tidak punya generator sendiri — mis. BRD.md.
   STEERING.md dan ADR.md dibangun oleh build-steering.js dan build-adr.js, yang
   sudah menghasilkan HTML-nya sendiri.                                         */

const fs = require('fs');
const path = require('path');
const os = require('os');
const { mdToHtml, wrapHtml } = require('./md2html');

const SRC = process.argv[2];
const OUT = process.argv[3] || path.join(os.tmpdir(), path.basename(SRC, '.md') + '.html');
const JUDUL = process.argv[4] || path.basename(SRC, '.md');

if (!SRC || !fs.existsSync(SRC)) {
  console.error('Berkas sumber tidak ditemukan: ' + SRC);
  process.exit(1);
}

const teks = fs.readFileSync(SRC, 'utf8').replace(/\r\n/g, '\n');
fs.writeFileSync(OUT, wrapHtml(JUDUL, mdToHtml(teks)), 'utf8');

console.log('MD   : ' + SRC + '  (' + teks.split('\n').length + ' baris)');
console.log('HTML : ' + OUT);
