/* Pengubah Markdown -> HTML untuk subset yang dipakai dokumen proyek ini.
   Dipakai bersama oleh build-adr.js dan build-steering.js.

   Ditangani: heading, tabel pipa, blok kode berpagar, kutipan, daftar berurut dan
   tak berurut, **tebal**, *miring*, `kode`, dan tautan.
   Tidak ditangani: HTML mentah, gambar, footnote, definition list.          */

function esc(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function inline(s) {
  let out = esc(s);
  out = out.replace(/`([^`]+)`/g, (m, c) => '<code>' + c + '</code>');
  out = out.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  out = out.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>');
  out = out.replace(/~~([^~]+)~~/g, '<s>$1</s>');
  // tautan internal (#anchor) kehilangan maknanya di .docx -> tinggalkan teksnya saja
  out = out.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (m, t, h) => h.charAt(0) === '#' ? t : '<a href="' + h + '">' + t + '</a>');
  return out;
}

/* H1 pertama adalah judul dokumen (tanpa page break); H1 berikutnya adalah bab
   dan selalu memulai halaman baru. State ini hidup per pemanggilan tingkat atas. */
let seenH1 = false;

/* Memecah satu baris tabel Markdown menjadi sel, MENGHORMATI pipa ter-escape.

   Sebelumnya dipakai split("|") biasa. Itu memecah juga pada | di dalam sel,
   sehingga baris yang memuat operator SQL || (ditulis || di Markdown) pecah
   menjadi kolom tambahan DAN isinya terpotong. Terdeteksi pada dua tabel:
   docs/Steering/00-DECISION-LOG.md:460 dan docs/BRD/BRD.md:263.                */
function belahSel(L) {
  const sel = [];
  let cur = "";
  for (let k = 0; k < L.length; k++) {
    if (L[k] === "\\" && L[k + 1] === "|") { cur += "|"; k++; continue; }
    if (L[k] === "|") { sel.push(cur); cur = ""; continue; }
    cur += L[k];
  }
  sel.push(cur);
  return sel.slice(1, -1).map(c => c.trim());
}
function mdToHtml(src, nested) {
  if (!nested) seenH1 = false;
  const lines = src.split('\n');
  const out = [];
  let i = 0;
  while (i < lines.length) {
    const L = lines[i];

    if (/^```/.test(L)) {
      const buf = [];
      i++;
      while (i < lines.length && !/^```/.test(lines[i])) { buf.push(esc(lines[i])); i++; }
      i++;
      out.push('<pre><code>' + buf.join('\n') + '</code></pre>');
      continue;
    }
    if (/^---+\s*$/.test(L)) { out.push('<hr/>'); i++; continue; }

    const h = L.match(/^(#{1,6})\s+(.*)$/);
    if (h) {
      const lv = h[1].length;
      let cls = '';
      if (lv === 1) { cls = seenH1 ? ' class="chap"' : ' class="doc-title"'; seenH1 = true; }
      else if (lv === 2) cls = ' class="grp"';
      out.push('<h' + lv + cls + '>' + inline(h[2]) + '</h' + lv + '>');
      i++; continue;
    }

    if (/^\|/.test(L) && i + 1 < lines.length && /^\|[\s:\-|]+\|\s*$/.test(lines[i + 1])) {
      const head = belahSel(L);
      i += 2;
      const rows = [];
      while (i < lines.length && /^\|/.test(lines[i])) {
        rows.push(belahSel(lines[i]));
        i++;
      }
      const blank = head.every(c => c === '');
      const pct = blank ? [26, 74] : lebarKolom(head, rows);
      let t = '<table class="' + (blank ? 'meta' : 'grid') + '" width="100%" style="table-layout:fixed">';
      if (!blank) {
        t += '<thead><tr>' + head.map((c, j) => '<th width="' + (pct[j] || 0) + '%">' + penggal(inline(c)) + '</th>').join('') + '</tr></thead>';
      }
      t += '<tbody>' + rows.map(r => '<tr>' + r.map((c, j) => '<td width="' + (pct[j] || 0) + '%">' + penggal(inline(c)) + '</td>').join('') + '</tr>').join('') + '</tbody></table>';
      out.push(t);
      continue;
    }

    if (/^>\s?/.test(L)) {
      const buf = [];
      while (i < lines.length && /^>\s?/.test(lines[i])) { buf.push(lines[i].replace(/^>\s?/, '')); i++; }
      out.push('<div class="note">' + mdToHtml(buf.join('\n'), true) + '</div>');
      continue;
    }

    if (/^\s*\d+\.\s+/.test(L)) {
      const buf = [];
      while (i < lines.length && (/^\s*\d+\.\s+/.test(lines[i]) || /^\s{3,}\S/.test(lines[i]))) {
        if (/^\s*\d+\.\s+/.test(lines[i])) buf.push(lines[i].replace(/^\s*\d+\.\s+/, ''));
        else buf[buf.length - 1] += ' ' + lines[i].trim();
        i++;
      }
      out.push('<ol>' + buf.map(b => '<li>' + inline(b) + '</li>').join('') + '</ol>');
      continue;
    }

    if (/^\s*[-*]\s+/.test(L)) {
      const buf = [];
      while (i < lines.length && (/^\s*[-*]\s+/.test(lines[i]) || /^\s{3,}\S/.test(lines[i]))) {
        if (/^\s*[-*]\s+/.test(lines[i])) buf.push(lines[i].replace(/^\s*[-*]\s+/, ''));
        else buf[buf.length - 1] += ' ' + lines[i].trim();
        i++;
      }
      out.push('<ul>' + buf.map(b => '<li>' + inline(b) + '</li>').join('') + '</ul>');
      continue;
    }

    if (/^\s*$/.test(L)) { i++; continue; }

    const buf = [];
    while (i < lines.length && !/^\s*$/.test(lines[i]) && !/^[#>|`-]/.test(lines[i]) && !/^\s*[-*\d]+[.\s]/.test(lines[i])) {
      buf.push(lines[i]); i++;
    }
    if (buf.length) out.push('<p>' + inline(buf.join(' ')) + '</p>');
    else { out.push('<p>' + inline(L) + '</p>'); i++; }
  }
  return out.join('\n');
}

const CSS = `
@page { size: A4; margin: 2.2cm 2.0cm 2.2cm 2.0cm; }
body { font-family: "Segoe UI", Calibri, Arial, sans-serif; font-size: 10.5pt; color: #1a1a1a; line-height: 1.45; }
h1.doc-title { font-size: 26pt; color: #0b3d6b; margin: 0 0 4pt 0; border-bottom: 3pt solid #0b3d6b; padding-bottom: 8pt; }
h1.chap { font-size: 22pt; color: #0b3d6b; margin: 0 0 10pt 0; border-bottom: 2.5pt solid #0b3d6b; padding-bottom: 6pt; page-break-before: always; page-break-after: avoid; }
h2.grp { font-size: 17pt; color: #0b3d6b; margin: 22pt 0 8pt 0; padding: 6pt 0 4pt 0; border-bottom: 1.5pt solid #c3d4e3; page-break-after: avoid; }
h3 { font-size: 13.5pt; color: #14507f; margin: 16pt 0 6pt 0; padding: 4pt 8pt; background: #eef4f9; border-left: 4pt solid #0b3d6b; page-break-after: avoid; }
h4 { font-size: 11.5pt; color: #0b3d6b; margin: 12pt 0 4pt 0; page-break-after: avoid; }
h5 { font-size: 10.5pt; color: #444; margin: 9pt 0 3pt 0; font-style: italic; page-break-after: avoid; }
p { margin: 5pt 0; text-align: justify; }
code { font-family: Consolas, "Courier New", monospace; font-size: 9.5pt; background: #f3f4f6; color: #8a2b2b; padding: 0 2pt; }
pre { background: #f7f8fa; border: 0.5pt solid #d8dde3; padding: 7pt; margin: 7pt 0; page-break-inside: avoid; }
pre code { background: transparent; color: #1a1a1a; font-size: 9pt; }
table { border-collapse: collapse; width: 100%; table-layout: auto; margin: 8pt 0; font-size: 9.5pt; }
table.grid th { background: #0b3d6b; color: #ffffff; text-align: left; padding: 5pt 6pt; border: 0.5pt solid #0b3d6b; font-weight: bold; }
table.grid td { padding: 4pt 6pt; border: 0.5pt solid #c3d4e3; vertical-align: top; word-wrap: break-word; overflow-wrap: break-word; }
table.grid tr:nth-child(even) td { background: #f6f9fc; }
table.meta { border: none; margin: 6pt 0 10pt 0; }
table.meta td { padding: 3pt 6pt; border: none; border-bottom: 0.5pt solid #e2e8ee; vertical-align: top; }
table.meta td:first-child { width: 26%; background: #f3f7fa; font-weight: bold; color: #14507f; }
div.note { background: #fff8e6; border-left: 4pt solid #e0a800; padding: 6pt 10pt; margin: 8pt 0; page-break-inside: avoid; }
div.note p { margin: 3pt 0; }
hr { border: none; border-top: 0.5pt solid #d8dde3; margin: 14pt 0; }
ul, ol { margin: 5pt 0 5pt 18pt; padding: 0; }
li { margin: 3pt 0; text-align: justify; }
a { color: #14507f; }
`;


/* Menyisipkan titik-penggal tak terlihat (U+200B) ke DALAM SEL TABEL saja.

   Word tidak memenggal kata panjang. Satu token seperti
   "ASM-FW-GCNMFW-Work-ReceiveDocument" atau "docs/Steering/00-DECISION-LOG.md:605"
   memaksa kolomnya melebar, tabelnya melewati batas cetak, dan tulisannya
   terpotong. Terukur pada ADR.docx sebelum perbaikan: 11 dari 64 tabel melebihi
   lebar cetak, terparah 896pt versus 415pt — dan tabel terparah itu hanya
   2 kolom, jadi sebabnya memang token panjang, bukan banyaknya kolom.

   Hanya token sepanjang >= 16 karakter tanpa spasi yang disentuh, dan
   penggalnya ditaruh SESUDAH pemisah alami ( / _ . : - ) supaya tidak memotong
   kata di tengah. U+200B tidak tercetak dan tidak menambah lebar.

   Dijalankan SETELAH inline(), tetapi hanya pada teks di antara tag — pola
   >([^<]+)< menjamin isi atribut dan nama tag tidak tersentuh.               */
/* Membagi lebar kolom tabel dalam PERSEN, lalu menuliskannya sebagai atribut
   width pada <table> dan tiap sel.

   Kenapa lewat HTML dan bukan lewat Word: diuji langsung, Word MENOLAK
   me-reflow tabel hasil impor HTML. AutoFitBehavior(wdAutoFitContent maupun
   wdAutoFitWindow), Columns.Width, Cell.PreferredWidth, pengubahan view ke
   Print, dan Repaginate() — semuanya tidak mengubah lebar sedikit pun. Sebuah
   tabel 2 kolom berisi kata "Accepted" tetap memakai kolom 946pt pada halaman
   yang area teksnya hanya 415pt, sehingga tulisannya terpotong.

   Atribut width dalam persen dihormati importer HTML Word, jadi lebar ditetapkan
   SEBELUM Word sempat menghitungnya sendiri.

   Pembagiannya sebanding dengan panjang isi tiap kolom, dengan batas bawah 7%
   supaya kolom sempit tetap terbaca, dan batas atas 55% supaya satu kolom tidak
   menelan seluruh tabel. Totalnya selalu dinormalkan menjadi 100%.            */
function lebarKolom(head, rows) {
  const n = Math.max(head.length, ...rows.map(r => r.length));
  const bobot = new Array(n).fill(0);
  for (let c = 0; c < n; c++) {
    let maks = (head[c] || "").length;
    let jml = 0, cacah = 0;
    for (const r of rows) { const L = (r[c] || "").length; jml += L; cacah++; if (L > maks) maks = L; }
    // rata-rata menahan pengaruh satu sel panjang; akar meredam selisih ekstrem
    const rata = cacah ? jml / cacah : 0;
    bobot[c] = Math.sqrt(Math.max(4, rata * 0.7 + maks * 0.3));
  }
  const tot = bobot.reduce((a, b) => a + b, 0) || 1;
  let pct = bobot.map(b => (b / tot) * 100);
  const MIN = 7, MAKS = 55;
  pct = pct.map(v => Math.min(MAKS, Math.max(MIN, v)));
  const tot2 = pct.reduce((a, b) => a + b, 0);
  return pct.map(v => Math.round((v / tot2) * 1000) / 10);
}
function penggal(html) {
  return html.replace(/>([^<]+)</g, function (m, teks) {
    return '>' + teks.replace(/\S{16,}/g, function (tok) {
      return tok.replace(/([\/_.:-])(?=.)/g, '$1\u200b');
    }) + '<';
  });
}

function wrapHtml(title, bodyHtml) {
  return '<!DOCTYPE html>\n<html lang="id"><head><meta charset="utf-8"/>' +
    '<title>' + esc(title) + '</title><style>' + CSS + '</style></head><body>\n' +
    bodyHtml + '\n</body></html>\n';
}

module.exports = { mdToHtml, wrapHtml, CSS, penggal };
