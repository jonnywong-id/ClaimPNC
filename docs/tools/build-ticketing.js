/* Membangun docs/ticketing/TICKETING.md (gabungan) + TICKETING.html (berkas antara
   untuk dikonversi ke .docx) dari docs/ticketing/README.md, setiap
   docs/ticketing/<modul>/spec.md + issues/NN-*.md, dan INVENTARIS-HARNESS.md.

   Jalankan: node docs/tools/build-ticketing.js "<root proyek>" ["<path TICKETING.html>"]

   TICKETING.html bersifat sementara; secara bawaan ditulis ke folder temp sistem
   agar tidak ikut ter-commit. Lihat docs/tools/README.md.                      */

const fs = require('fs');
const path = require('path');
const os = require('os');

const ROOT = process.argv[2];
const TDIR = path.join(ROOT, 'docs', 'ticketing');
const OUT_MD = path.join(TDIR, 'TICKETING.md');
const OUT_HTML = process.argv[3] || path.join(os.tmpdir(), 'TICKETING.html');

const TANGGAL = new Date().toISOString().slice(0, 10);

/* urutan kelompok modul mengikuti gelombang di 06-MODULE-BREAKDOWN.md */
const KELOMPOK = [
  { kode: 'F', judul: 'Modul Fondasi' },
  { kode: 'B', judul: 'Modul Bisnis Inti' },
  { kode: 'S', judul: 'Modul Pendukung' },
  { kode: 'U', judul: 'Modul Frontend' },
];

const baca = f => fs.readFileSync(f, 'utf8').replace(/\r\n/g, '\n');

/* turunkan heading sebanyak n tingkat (maks. h6) */
function turunkan(body, n) {
  return body.split('\n').map(l => {
    const m = l.match(/^(#{1,6})(\s.*)$/);
    if (!m) return l;
    return '#'.repeat(Math.min(6, m[1].length + n)) + m[2];
  }).join('\n');
}

/* buang baris heading H1 pertama */
function tanpaH1(body) {
  return body.replace(/^\s*#\s[^\n]*\n/, '');
}

/* kotak centang Markdown -> simbol, karena md2html tidak mengenalnya */
function centang(body) {
  return body.replace(/^(\s*)- \[ \] /gm, '$1- ☐ ').replace(/^(\s*)- \[x\] /gim, '$1- ☑ ');
}

const selAman = s => s.replace(/\|/g, '\\|');

/* ---------- uraikan satu tiket ---------- */

function parseTiket(file) {
  let raw = baca(file);
  const fm = {};
  const mFm = raw.match(/^---\n([\s\S]*?)\n---\n/);
  if (mFm) {
    mFm[1].split('\n').forEach(l => {
      const km = l.match(/^(\w+):\s*(.*)$/);
      if (km) fm[km[1]] = km[2].replace(/^"(.*)"$/, '$1').trim();
    });
    raw = raw.slice(mFm[0].length);
  }
  const lines = raw.replace(/^\s+/, '').split('\n');
  const judul = (lines[0].match(/^#\s+(.*)$/) || [, fm.title || path.basename(file)])[1].trim();

  /* baris "Kunci: nilai" di antara judul dan heading ## pertama -> tabel meta */
  const meta = [];
  let i = 1;
  for (; i < lines.length; i++) {
    const L = lines[i];
    if (/^##\s/.test(L)) break;
    if (!L.trim()) continue;
    L.split(/\s{2,}(?=[A-Z][A-Za-z ]{1,40}:\s)/).forEach(bag => {
      const km = bag.match(/^([A-Z][A-Za-z0-9 ]{1,40}):\s*(.*)$/);
      if (km) meta.push([km[1].trim(), km[2].trim()]);
      else if (meta.length) meta[meta.length - 1][1] += ' ' + bag.trim();
    });
  }
  const body = lines.slice(i).join('\n').trim();
  const status = (meta.find(m => m[0] === 'Status') || [, ''])[1];
  return { file, judul, fm, meta, body, status };
}

/* ---------- kumpulkan modul ---------- */

function kunciModul(nama) {
  const m = nama.match(/^([FBSU])-(\d+)-/);
  return m ? { kel: m[1], no: +m[2] } : null;
}

const modul = fs.readdirSync(TDIR, { withFileTypes: true })
  .filter(d => d.isDirectory() && kunciModul(d.name))
  .map(d => {
    const dir = path.join(TDIR, d.name);
    const k = kunciModul(d.name);
    const specPath = path.join(dir, 'spec.md');
    const spec = fs.existsSync(specPath) ? baca(specPath) : '';
    const judul = (spec.match(/^#\s+(.*)$/m) || [, d.name])[1].trim();
    const idir = path.join(dir, 'issues');
    const tiket = fs.existsSync(idir)
      ? fs.readdirSync(idir).filter(f => /\.md$/.test(f)).sort().map(f => parseTiket(path.join(idir, f)))
      : [];
    return { nama: d.name, kel: k.kel, no: k.no, judul, spec, tiket };
  })
  .sort((a, b) => KELOMPOK.findIndex(g => g.kode === a.kel) - KELOMPOK.findIndex(g => g.kode === b.kel) || a.no - b.no);

const semuaTiket = modul.reduce((a, m) => a.concat(m.tiket), []);
const hitungStatus = {};
semuaTiket.forEach(t => { const s = t.status || '(tanpa status)'; hitungStatus[s] = (hitungStatus[s] || 0) + 1; });

/* ---------- susun markdown gabungan ---------- */

const md = [];
const P = s => md.push(s);

P('# Ticketing — Migrasi Aplikasi Claim PNC');
P('');
P('**Papan Tiket Pekerjaan · Dokumen Gabungan**');
P('');
P('| | |');
P('|---|---|');
P('| **Dokumen** | Ticketing migrasi Claim PNC dari Pega PRPC 8.3 ke Golang + React + PostgreSQL |');
P('| **Tanggal dibangun** | ' + TANGGAL + ' |');
P('| **Jumlah modul** | **' + modul.length + '** |');
P('| **Jumlah tiket** | **' + semuaTiket.length + '** — ' +
  Object.keys(hitungStatus).sort().map(s => hitungStatus[s] + ' `' + s + '`').join(' · ') + ' |');
P('| **Sumber** | `docs/ticketing/README.md` · `docs/ticketing/<modul>/spec.md` · `docs/ticketing/<modul>/issues/*.md` · `docs/ticketing/INVENTARIS-HARNESS.md` |');
P('| **Klasifikasi** | CONFIDENTIAL |');
P('');
P('> **Dokumen ini dibangun otomatis** oleh `docs/tools/build-ticketing.js`. Jangan disunting');
P('> langsung — suntingan hilang pada pembangunan berikutnya. Yang disunting adalah berkas sumbernya');
P('> di `docs/ticketing/`.');
P('');
P('---');
P('');

/* daftar isi */
P('## Daftar Isi');
P('');
P('1. Papan Tiket — ringkasan, status, dependency, traceability, jadwal');
let bab = 1;
KELOMPOK.forEach(g => {
  modul.filter(m => m.kel === g.kode).forEach(m => {
    bab++;
    P(bab + '. ' + m.judul + ' — ' + m.tiket.length + ' tiket');
  });
});
P('');
P('- Lampiran A — Inventaris 74 Harness');
P('- Lampiran B — Indeks Seluruh Tiket');
P('');

/* bab 1: README */
P('# 1. Papan Tiket');
P('');
P(centang(turunkan(tanpaH1(baca(path.join(TDIR, 'README.md'))), 0)));
P('');

/* bab per modul */
bab = 1;
KELOMPOK.forEach(g => {
  modul.filter(m => m.kel === g.kode).forEach(m => {
    bab++;
    P('# ' + bab + '. ' + m.judul);
    P('');
    P('*' + g.judul + ' · folder `docs/ticketing/' + m.nama + '/`*');
    P('');
    if (m.spec) {
      P('## Spesifikasi Modul');
      P('');
      P(centang(turunkan(tanpaH1(m.spec), 1)));
      P('');
    }
    P('## Daftar Tiket');
    P('');
    if (!m.tiket.length) {
      P('*Belum ada tiket.*');
      P('');
    } else if (!/TKT-/.test(m.spec)) {
      /* ringkasan hanya bila spec.md belum memuat daftar tiketnya sendiri */
      P('| Tiket | Status | Prioritas |');
      P('|---|---|---|');
      m.tiket.forEach(t => {
        const pr = ((t.fm.labels || '').match(/prioritas::([\w-]+)/) || [, '—'])[1];
        P('| ' + selAman(t.judul) + ' | ' + selAman(t.status || '—') + ' | ' + pr + ' |');
      });
      P('');
    }
    m.tiket.forEach(t => {
      P('### ' + t.judul);
      P('');
      P('| | |');
      P('|---|---|');
      t.meta.forEach(([k, v]) => P('| **' + k + '** | ' + selAman(v) + ' |'));
      if (t.fm.labels) P('| **Label** | ' + selAman(t.fm.labels.replace(/^\[|\]$/g, '').split(/\s*,\s*/).map(x => '`' + x + '`').join(' ')) + ' |');
      if (t.fm.milestone) P('| **Milestone** | ' + selAman(t.fm.milestone) + ' |');
      P('| **Berkas sumber** | `' + path.relative(path.join(ROOT, 'docs'), t.file).replace(/\\/g, '/') + '` |');
      P('');
      P(centang(turunkan(t.body, 2)));
      P('');
    });
  });
});

/* lampiran */
const inv = path.join(TDIR, 'INVENTARIS-HARNESS.md');
if (fs.existsSync(inv)) {
  P('# Lampiran A — Inventaris 74 Harness');
  P('');
  P(turunkan(tanpaH1(baca(inv)), 0));
  P('');
}

P('# Lampiran B — Indeks Seluruh Tiket');
P('');
P('| Modul | Tiket | Status |');
P('|---|---|---|');
modul.forEach(m => m.tiket.forEach(t => {
  P('| ' + m.nama.match(/^[FBSU]-\d+/)[0] + ' | ' + selAman(t.judul) + ' | ' + selAman(t.status || '—') + ' |');
}));
P('');

fs.writeFileSync(OUT_MD, md.join('\n').replace(/\n{3,}/g, '\n\n') + '\n', 'utf8');
console.log('MD   : ' + OUT_MD);

/* ---------- markdown -> HTML ---------- */

const { mdToHtml, wrapHtml } = require('./md2html');
const html = wrapHtml('Ticketing — Migrasi Aplikasi Claim PNC', mdToHtml(fs.readFileSync(OUT_MD, 'utf8')));
fs.writeFileSync(OUT_HTML, html, 'utf8');
console.log('HTML : ' + OUT_HTML);
console.log('Modul: ' + modul.length + ' · Tiket: ' + semuaTiket.length);
console.log('Status: ' + JSON.stringify(hitungStatus));
