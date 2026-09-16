/* Membangun docs/ADR.md (gabungan) + ADR.html (berkas antara untuk dikonversi ke .docx)
   dari docs/ADR/README.md dan docs/ADR/00NN-*.md

   Jalankan: node docs/tools/build-adr.js "<root proyek>" ["<path ADR.html>"]

   ADR.html bersifat sementara; secara bawaan ditulis ke folder temp sistem agar
   tidak ikut ter-commit. Lihat docs/tools/README.md.                          */

const fs = require('fs');
const path = require('path');
const os = require('os');

const ROOT = process.argv[2];
const ADRDIR = path.join(ROOT, 'docs', 'ADR');
const OUT_MD = path.join(ROOT, 'docs', 'ADR', 'ADR.md');
const OUT_HTML = process.argv[3] || path.join(os.tmpdir(), 'ADR.html');

const TANGGAL = '2026-09-14';

const GROUPS = [
  { key: 'A', judul: 'Fondasi & Topologi', nos: ['0001', '0002', '0003', '0004', '0005', '0006'],
    intro: 'Keputusan yang menentukan bentuk aplikasi secara keseluruhan: bahasa, bentuk artefak, tempat ia berjalan, cara ia menggantikan sistem lama, dan di mana batas kepemilikan datanya.' },
  { key: 'B', judul: 'Data, Integrasi, dan Artefak', nos: ['0007', '0008', '0009', '0010', '0011', '0012', '0013'],
    intro: 'Keputusan tentang cara data ditulis, dibaca, dipertukarkan dengan sistem lain, dan diterbitkan sebagai dokumen.' },
  { key: 'C', judul: 'Aturan Bisnis Bernilai Uang dan Kewenangan', nos: ['0014', '0015', '0016', '0017', '0018', '0019', '0020'],
    intro: 'Keputusan yang langsung menentukan berapa nilai sebuah klaim, siapa yang berwenang menyetujuinya, dan bagaimana pekerjaan dibagikan. Kelompok paling sensitif dalam dokumen ini.' },
  { key: 'D', judul: 'Mekanisme Pega yang Harus Diganti', nos: ['0021', '0022'],
    intro: 'Dua mekanisme platform yang tidak punya padanan otomatis di luar Pega, dan keduanya belum dapat diputuskan.' },
  { key: 'E', judul: 'Keamanan, Konfigurasi, dan Audit', nos: ['0023', '0024', '0025', '0026'],
    intro: 'Keputusan tentang siapa boleh melakukan apa, di mana nilai dan rahasia disimpan, dan apa yang tercatat.' },
  { key: 'F', judul: 'Mutu dan Data Uji', nos: ['0027', '0028', '0029'],
    intro: 'Keputusan tentang cara sebuah modul dinyatakan lulus, dan lingkungan tempat pembuktiannya dijalankan.' },
];

/* ---------- baca dan uraikan ---------- */

function anchor(text) {
  return text.toLowerCase()
    .replace(/[`*_[\]()]/g, '')
    .replace(/[^a-z0-9 \-]/g, '')
    .trim().replace(/\s+/g, '-');
}

const HEADER_KEYS = ['Status', 'Tanggal keputusan', 'Tanggal dokumen', 'Sifat', 'Pemilik keputusan', 'Jejak bukti', 'Terkait'];

function parseAdr(file) {
  const raw = fs.readFileSync(path.join(ADRDIR, file), 'utf8').replace(/\r\n/g, '\n');
  const lines = raw.split('\n');
  const m = lines[0].match(/^#\s*(\d{4})\s*—\s*(.+)$/);
  if (!m) throw new Error('Judul tidak terbaca: ' + file);
  const no = m[1], judul = m[2].trim();

  const header = {};
  let i = 1;
  for (; i < lines.length; i++) {
    const L = lines[i];
    if (/^##\s/.test(L) || /^>\s/.test(L)) break;
    const km = L.match(/^([A-Za-z ]+):\s*(.*)$/);
    if (km && HEADER_KEYS.indexOf(km[1].trim()) >= 0) {
      let key = km[1].trim(), val = km[2];
      // "Tanggal keputusan: X    Tanggal dokumen: Y" dalam satu baris
      const split = val.match(/^(.*?)\s{2,}Tanggal dokumen:\s*(.*)$/);
      if (key === 'Tanggal keputusan' && split) {
        header['Tanggal keputusan'] = split[1].trim();
        header['Tanggal dokumen'] = split[2].trim();
      } else {
        header[key] = val.trim();
      }
    }
  }
  const body = lines.slice(i).join('\n').trim();

  const status = (header['Status'] || '').trim();
  const oq = extractSection(body, 'Pertanyaan terbuka');
  return { file, no, judul, header, body, status, openQuestions: oq };
}

function extractSection(body, name) {
  const re = new RegExp('^##\\s+' + name + '\\s*$', 'm');
  const mm = body.match(re);
  if (!mm) return '';
  const start = body.indexOf(mm[0]) + mm[0].length;
  const rest = body.slice(start);
  const nxt = rest.search(/^##\s+/m);
  return (nxt === -1 ? rest : rest.slice(0, nxt)).trim();
}

/* naikkan level heading: ## -> ####, ### -> ##### */
function shiftHeadings(body) {
  return body.split('\n').map(l => {
    if (/^###\s/.test(l)) return '##' + l;
    if (/^##\s/.test(l)) return '##' + l;
    return l;
  }).join('\n');
}

const files = fs.readdirSync(ADRDIR).filter(f => /^\d{4}-.*\.md$/.test(f)).sort();
const adrs = files.map(parseAdr);
const byNo = {};
adrs.forEach(a => { byNo[a.no] = a; });

/* index dari README */
const readme = fs.readFileSync(path.join(ADRDIR, 'README.md'), 'utf8').replace(/\r\n/g, '\n');
const indexRows = readme.split('\n').filter(l => /^\|\s*\[\d{4}\]/.test(l)).map(l => {
  const cells = l.split('|').slice(1, -1).map(c => c.trim());
  const no = cells[0].match(/\[(\d{4})\]/)[1];
  return { no: no, judul: cells[1], status: cells[2], jenis: cells[3], modul: cells[4], pemilik: cells[5] };
});

const nAcc = adrs.filter(a => /Accepted/.test(a.status)).length;
const nProp = adrs.filter(a => /Proposed/.test(a.status)).length;

/* ---------- susun markdown gabungan ---------- */

const md = [];
const P = s => md.push(s);

P('# ADR — Migrasi Aplikasi Claim PNC ke Golang');
P('');
P('**Architecture Decision Records · Dokumen Gabungan**');
P('');
P('| | |');
P('|---|---|');
P('| **Dokumen** | Architecture Decision Records (ADR) — Migrasi Claim PNC dari Pega PRPC 8.3 ke Golang + React + PostgreSQL |');
P('| **Tanggal dokumen** | ' + TANGGAL + ' |');
P('| **Pemilik keputusan** | Work Owner |');
P('| **Jumlah keputusan** | **' + adrs.length + '** — ' + nAcc + ' `Accepted`, ' + nProp + ' `Proposed` |');
P('| **Sumber keputusan** | `docs/Steering/00-DECISION-LOG.md` (70 entri, D-01…D-70) |');
P('| **Sumber bukti** | `docs/verifikasi-bukti-adr.md` · export rule Pega (baca-saja) |');
P('| **Sifat seluruh ADR** | `retrospective` — keputusan diambil 2026-09-07 … 2026-09-14, dokumen ditulis ' + TANGGAL + ' |');
P('');
P('> **Dokumen ini dibangun otomatis** dari `docs/ADR/README.md` dan `docs/ADR/00NN-*.md`.');
P('> Berkas sumber tetap menjadi acuan; dokumen gabungan ini untuk dibaca dan diedarkan.');
P('');
P('---');
P('');

/* daftar isi */
P('## Daftar Isi');
P('');
P('1. [Ringkasan Eksekutif](#1-ringkasan-eksekutif)');
P('2. [Index Keputusan](#2-index-keputusan)');
P('3. [Kosakata Status](#3-kosakata-status)');
P('4. [Hubungan dengan Dokumen Lain](#4-hubungan-dengan-dokumen-lain)');
let secNo = 4;
GROUPS.forEach(g => {
  secNo++;
  P(secNo + '. [Kelompok ' + g.key + ' — ' + g.judul + '](#' + anchor(secNo + ' Kelompok ' + g.key + ' — ' + g.judul) + ')');
  g.nos.forEach(n => {
    const a = byNo[n];
    P('    - [' + n + ' — ' + a.judul + '](#' + anchor(n + ' — ' + a.judul) + ')');
  });
});
P('');
P('- [Lampiran A — Template ADR Kosong](#lampiran-a--template-adr-kosong)');
P('- [Lampiran B — Aturan Menulis yang Mengikat](#lampiran-b--aturan-menulis-yang-mengikat)');
P('- [Lampiran C — Rekapitulasi Seluruh Pertanyaan Terbuka](#lampiran-c--rekapitulasi-seluruh-pertanyaan-terbuka)');
P('');
P('---');
P('');

/* 1. ringkasan eksekutif */
P('## 1. Ringkasan Eksekutif');
P('');
P('### 1.1 Apa isi dokumen ini');
P('');
P('Satu berkas ADR = satu keputusan yang **mahal dibalik**. Uji kelayakan yang dipakai menyaring:');
P('*kalau keputusan ini dibalik enam bulan lagi, adakah arsitektur yang harus dibongkar besar atau');
P('risiko bisnis yang muncul?* Dari **70 keputusan** di Decision Log, **' + adrs.length + ' lolos** dan 41 tidak —');
P('yang tidak lolos tetap hidup sebagai `D-nn` di `docs/Steering/00-DECISION-LOG.md`.');
P('');
P('### 1.2 Papan status');
P('');
P('| Kelompok | Jumlah | `Accepted` | `Proposed` |');
P('|---|---:|---:|---:|');
GROUPS.forEach(g => {
  const acc = g.nos.filter(n => /Accepted/.test(byNo[n].status)).length;
  const pro = g.nos.filter(n => /Proposed/.test(byNo[n].status)).length;
  P('| **' + g.key + '** — ' + g.judul + ' | ' + g.nos.length + ' | ' + acc + ' | ' + pro + ' |');
});
P('| **Total** | **' + adrs.length + '** | **' + nAcc + '** | **' + nProp + '** |');
P('');
P('### 1.3 Lima keputusan yang belum diambil, dan apa yang terhalang');
P('');
P('Kelima ADR berikut berstatus `Proposed`: memuat konteks, opsi, dan pemilik keputusan — **tanpa');
P('bagian `Keputusan`**. Tidak satu pun boleh dijadikan dasar implementasi.');
P('');
P('| ADR | Yang belum diputuskan | Pemilik keputusan | Menghalangi |');
P('|---|---|---|---|');
P('| **0013** | Pengganti pola hapus-lalu-sisip-ulang pada konversi klaim | Work Owner + Lead Engineer | tiket `B-2` |');
P('| **0021** | 8 Ticket rule custom hilang; 14 dari 17 nama tanpa pemicu | Tim Pega → Work Owner | `B-7`, `B-11`, `B-13`, `B-14` |');
P('| **0022** | Mekanisme penjadwal saat aplikasi hidup di dua instans; nasib `AutoAcceptKomite` | Lead Engineer + Infra · Work Owner | `S-6` |');
P('| **0024** | Kontrak API HCC/HCQ — nol jejak di export | Tim HCC/HCQ → Work Owner | `F-3`, dan login seluruh aplikasi |');
P('| **0025** | Ke mana rahasia dipindahkan dan siapa pemiliknya (`D-40` OPEN) | Tim Infra/Security | `F-4`, `F-5`, seluruh deployment |');
P('');
P('### 1.4 Keputusan yang menyupersede dokumen lain');
P('');
P('| ADR | Menyupersede | Keterangan |');
P('|---|---|---|');
P('| **0012** | `D-65` | Soft delete menyeluruh menggantikan kebijakan penghapusan yang mengikuti sistem lama |');
P('| **0028** | **`BRD §21.4`** untuk `B-7`, `B-9`, `B-10`, `B-12` | Pasalnya dicabut lewat ADR; **BRD tidak diedit** — usulan revisinya menunggu persetujuan Work Owner |');
P('| **0014** | membatasi cakupan `D-52` | `TYPE_KOMITE` adalah pita nilai **hanya** di lini Non-MBU |');
P('| **0009** | bagian format pada `D-22` | Format nomor klaim `PNCN-xxxx` direvisi menjadi `PNCN.YY.xxxx` oleh `D-71`; sisa isi `D-22` tetap berlaku |');
P('');
P('### 1.5 Tiga keputusan paling berisiko');
P('');
P('| ADR | Mengapa paling berisiko |');
P('|---|---|');
P('| **0004** Database bersama, penulis tunggal per tabel | Satu-satunya keputusan yang dapat **merusak data produksi** bila salah; mengikat seluruh masa paralel dan hampir mustahil dibalik setelah gelombang 1 berjalan |');
P('| **0007** Stored procedure naik ke Go | Menyentuh 62 procedure + 12 dependensi, memindahkan kepemilikan transaksi, dan menjadi dasar `B-4`/`B-9` dapat dibuat atomik |');
P('| **0014** Komite kumulatif per lini | Menentukan **siapa berwenang menyetujui uang**; mekanismenya terbukti dua kali hampir salah dibaca selama analisis |');
P('');
P('---');
P('');

/* 2. index */
P('## 2. Index Keputusan');
P('');
P('Kolom `Jenis` dan `Modul terdampak` berada di tabel ini, bukan di dalam berkas ADR — sesuai `D-31`.');
P('');
P('| # | Judul | Status | Jenis | Modul terdampak | Pemilik keputusan |');
P('|---|---|---|---|---|---|');
indexRows.forEach(r => {
  const a = byNo[r.no];
  P('| [' + r.no + '](#' + anchor(r.no + ' — ' + a.judul) + ') | ' + r.judul + ' | ' + r.status + ' | ' + r.jenis + ' | ' + r.modul + ' | ' + r.pemilik + ' |');
});
P('');
P('---');
P('');

/* 3. kosakata status */
P('## 3. Kosakata Status');
P('');
P('| Status | Artinya |');
P('|---|---|');
P('| `Accepted` | Work Owner sudah memutuskan; keputusan mengikat tiket dan implementasi |');
P('| `Proposed` | **Belum** diputuskan. Berkasnya memuat konteks, opsi, pemilik keputusan, dan apa yang diblokir — **tanpa** bagian `Keputusan`. Tidak boleh dijadikan dasar implementasi |');
P('| `Superseded by ADR-XXXX` | Digantikan ADR lain; isinya dibiarkan utuh sebagai jejak |');
P('| `Deprecated` | Tidak lagi berlaku dan tidak diganti |');
P('');
P('---');
P('');

/* 4. hubungan dokumen */
P('## 4. Hubungan dengan Dokumen Lain');
P('');
P('| Dokumen | Perannya |');
P('|---|---|');
P('| `docs/Steering/00-DECISION-LOG.md` | Sumber seluruh `D-nn`. ADR **tidak** menggantikannya |');
P('| `docs/Steering/CONTEXT.md` | Kamus istilah. ADR memakai istilahnya, tidak mendefinisikan ulang |');
P('| `docs/verifikasi-bukti-adr.md` | Bukti Fase 1 — `berkas:baris`, angka, dan kontradiksi `K-nn` |');
P('| `docs/BRD.md` | Sumber `FR-xx` |');
P('| `docs/ticketing/` | Tiket pekerjaan; setiap tiket merujuk ADR yang mengikatnya |');
P('');
P('> **Peringatan istilah.** Folder `Ticket/` pada export Pega berisi **Ticket rule** — mekanisme');
P('> lompatan lateral (lihat ADR 0021). Kata **"tiket"** di seluruh dokumen proyek ini selalu berarti');
P('> **tiket pekerjaan** di `docs/ticketing/`.');
P('');
P('---');
P('');

/* 5..10 kelompok */
let sn = 4;
GROUPS.forEach(g => {
  sn++;
  P('## ' + sn + '. Kelompok ' + g.key + ' — ' + g.judul);
  P('');
  P(g.intro);
  P('');
  P('| # | Judul | Status |');
  P('|---|---|---|');
  g.nos.forEach(n => {
    const a = byNo[n];
    P('| ' + n + ' | ' + a.judul + ' | ' + a.status + ' |');
  });
  P('');
  g.nos.forEach(n => {
    const a = byNo[n];
    P('### ' + n + ' — ' + a.judul);
    P('');
    P('| | |');
    P('|---|---|');
    HEADER_KEYS.forEach(k => {
      // pemisah " | " di dalam nilai header akan merusak tabel dua kolom -> jadikan "·"
      if (a.header[k]) P('| **' + k + '** | ' + a.header[k].replace(/\s*\|\s*/g, ' · ') + ' |');
    });
    P('');
    P(shiftHeadings(a.body));
    P('');
  });
  P('---');
  P('');
});

/* lampiran */
P('## Lampiran A — Template ADR Kosong');
P('');
P('```markdown');
P('# NNNN — <judul keputusan, kalimat aktif>');
P('');
P('Status: Proposed | Accepted | Superseded by ADR-XXXX | Deprecated');
P('Tanggal keputusan: YYYY-MM-DD    Tanggal dokumen: YYYY-MM-DD');
P('Sifat: original | retrospective');
P('Pemilik keputusan: <peran, mis. Work Owner / Lead Engineer / Tim Infra>');
P('Jejak bukti: D-nn | FR-xx | R-nn | path/berkas:baris | nama objek DB');
P('Terkait: CONTEXT.md#<istilah>, ADR-XXXX, modul <kode>');
P('');
P('## Konteks');
P('## Opsi yang dipertimbangkan');
P('## Keputusan');
P('## Rationale');
P('## Konsekuensi');
P('### Positif');
P('### Negatif / utang teknis');
P('### Risiko yang diterima secara sadar');
P('## Pertanyaan terbuka');
P('```');
P('');
P('---');
P('');
P('## Lampiran B — Aturan Menulis yang Mengikat');
P('');
P('1. Satu berkas = satu keputusan yang mahal dibalik.');
P('2. `Sifat: retrospective` wajib ditulis terbuka bila keputusan mendahului dokumennya.');
P('3. **Bagian `Negatif / utang teknis` tidak boleh kosong.** ADR yang hanya memuat keuntungan adalah dokumen jualan, bukan ADR.');
P('4. Jangan menulis `Keputusan` untuk hal yang belum diputuskan Work Owner — pakai `Proposed`, sebutkan pemilik keputusan dan apa yang diblokir.');
P('5. Bila ADR bertentangan dengan Steering atau BRD, tulis **`Menyupersede …`** secara eksplisit dan ajukan revisinya sebagai pertanyaan terbuka. Jangan mengedit Steering diam-diam.');
P('6. Nilai sensitif tidak pernah ditulis: kredensial, kunci API, hostname/IP produksi, dan data nasabah dirujuk dengan `berkas:baris` + nama elemen saja. Alamat email disamarkan (`D-69`).');
P('');
P('---');
P('');
P('## Lampiran C — Rekapitulasi Seluruh Pertanyaan Terbuka');
P('');
P('Dikumpulkan otomatis dari bagian **Pertanyaan terbuka** setiap ADR. Ini daftar kerja untuk');
P('Work Owner dan pihak luar — bukan bagian dari keputusan yang sudah diambil.');
P('');
adrs.forEach(a => {
  if (!a.openQuestions) return;
  P('### ' + a.no + ' — ' + a.judul);
  P('');
  P(a.openQuestions);
  P('');
});

fs.writeFileSync(OUT_MD, md.join('\n').replace(/\n{3,}/g, '\n\n') + '\n', 'utf8');
console.log('MD  : ' + OUT_MD);

/* ---------- markdown -> HTML ---------- */

const { mdToHtml, wrapHtml } = require('./md2html');

const html = wrapHtml('ADR — Migrasi Aplikasi Claim PNC ke Golang',
  mdToHtml(fs.readFileSync(OUT_MD, 'utf8')));

fs.writeFileSync(OUT_HTML, html, 'utf8');
console.log('HTML: ' + OUT_HTML);
console.log('ADR : ' + adrs.length + ' (' + nAcc + ' Accepted, ' + nProp + ' Proposed)');
