#!/usr/bin/env python3
"""
Generate minimal valid test files for M9 E2E verification.

Outputs (same directory as this script):
  compliant.pdf     — 2 pages, ~350 words  (passes min_pages=2, min_word_count=300, required_types=[pdf])
  single-page.pdf   — 1 page               (fails min_pages=2)
  test.docx         — 1-page DOCX          (fails required_types=[pdf])

Usage:
  python3 scripts/gen-test-pdfs.py
"""
import io
import os
import sys
import zipfile

OUT_DIR = os.path.dirname(os.path.abspath(__file__))

# ~400 unique words for seeding content
_LOREM = (
    "lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor "
    "incididunt ut labore et dolore magna aliqua ut enim ad minim veniam quis nostrud "
    "exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat duis aute irure "
    "dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur "
    "excepteur sint occaecat cupidatat non proident sunt in culpa qui officia deserunt mollit "
    "anim id est laborum perspiciatis unde omnis iste natus error sit voluptatem accusantium "
    "doloremque laudantium totam rem aperiam eaque ipsa quae ab illo inventore veritatis quasi "
    "architecto beatae vitae dicta sunt explicabo nemo enim ipsam voluptatem quia voluptas sit "
    "aspernatur aut odit fugit quia consequuntur magni dolores eos qui ratione voluptatem sequi "
    "nesciunt neque porro quisquam est qui dolorem adipisci velit quia non numquam eius modi "
    "tempora incidunt dolore magnam aliquam quaerat ut ad minima veniam nostrum exercitationem "
    "ullam corporis suscipit laboriosam nisi aliquid commodi consequatur quis autem vel eum iure "
    "reprehenderit ea voluptate velit esse quam nihil molestiae consequatur illum dolorem eum "
    "fugiat quo voluptas nulla at vero eos accusamus iusto odio dignissimos ducimus blanditiis "
    "praesentium voluptatum deleniti atque corrupti dolores molestias excepturi sint occaecati "
    "cupiditate provident similique officia mollitia fuga harum rerum facilis expedita distinctio "
    "nam libero tempore cum soluta nobis eligendi optio cumque nihil impedit quo minus placeat "
    "facere possimus omnis voluptas assumenda repellendus temporibus autem quibusdam officiis "
    "debitis rerum necessitatibus saepe eveniet voluptates repudiandae recusandae itaque earum"
).split()

WORDS = (_LOREM * 5)  # repeat to have enough


# ─── Minimal PDF builder ─────────────────────────────────────────────────────

class _PDF:
    """Builds a minimal but pdfcpu-compatible PDF in memory."""

    def __init__(self):
        self._buf = bytearray()
        self._off: dict[int, int] = {}
        self._extend(b"%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

    def _extend(self, data: bytes | bytearray | str):
        if isinstance(data, str):
            data = data.encode("latin-1")
        self._buf.extend(data)

    def _pos(self) -> int:
        return len(self._buf)

    def add_obj(self, oid: int, raw):
        self._off[oid] = self._pos()
        self._extend(f"{oid} 0 obj\n")
        self._extend(raw)
        self._extend(b"\nendobj\n")

    def add_stream(self, oid: int, content: str | bytes):
        b = content.encode("latin-1") if isinstance(content, str) else content
        self.add_obj(
            oid,
            f"<< /Length {len(b)} >>\nstream\n".encode("latin-1") + b + b"\nendstream",
        )

    def finish(self, root_id: int) -> bytes:
        xref_pos = self._pos()
        max_id = max(self._off)
        self._extend(f"xref\n0 {max_id + 1}\n")
        self._extend(b"0000000000 65535 f \n")
        for i in range(1, max_id + 1):
            if i in self._off:
                self._extend(f"{self._off[i]:010d} 00000 n \n")
            else:
                self._extend(b"0000000000 65535 f \n")
        self._extend(f"trailer\n<< /Size {max_id + 1} /Root {root_id} 0 R >>\n")
        self._extend(f"startxref\n{xref_pos}\n%%EOF\n")
        return bytes(self._buf)


def _page_content(words: list[str]) -> str:
    """Build a PDF content stream with the given words.

    Uses BT/ET with Td positioning so pdfcpu ExtractContentFile produces
    content streams that parsePDFContentStream can harvest word tokens from.
    """
    lines: list[str] = []
    for i in range(0, len(words), 9):
        chunk = " ".join(words[i : i + 9])
        # Escape PDF string special chars
        chunk = chunk.replace("\\", "\\\\").replace("(", r"\(").replace(")", r"\)")
        lines.append(chunk)

    parts = ["BT", "/F1 10 Tf", "72 750 Td", "14 TL"]
    for line in lines:
        parts.append(f"({line}) Tj T*")
    parts.append("ET")
    return "\n".join(parts) + "\n"


def make_pdf(num_pages: int, words_per_page: int = 180) -> bytes:
    """Return bytes for a valid multi-page PDF with embedded text.

    Object layout:
      1  Catalog
      2  Pages
      3+2i  Page i
      4+2i  Content stream i
      3+2*num_pages  Font (Helvetica)
    """
    pdf = _PDF()

    font_id = 3 + 2 * num_pages
    page_ids = [3 + 2 * i for i in range(num_pages)]
    content_ids = [4 + 2 * i for i in range(num_pages)]

    # Catalog
    pdf.add_obj(1, "<< /Type /Catalog /Pages 2 0 R >>")

    # Pages dictionary
    kids = " ".join(f"{p} 0 R" for p in page_ids)
    pdf.add_obj(2, f"<< /Type /Pages /Kids [{kids}] /Count {num_pages} >>")

    for i in range(num_pages):
        w = WORDS[i * words_per_page : (i + 1) * words_per_page]
        if len(w) < words_per_page:
            w = (w * (words_per_page // len(w) + 1))[:words_per_page]
        content = _page_content(w)
        pdf.add_stream(content_ids[i], content)
        pdf.add_obj(
            page_ids[i],
            (
                f"<< /Type /Page /Parent 2 0 R "
                f"/MediaBox [0 0 612 792] "
                f"/Contents {content_ids[i]} 0 R "
                f"/Resources << /Font << /F1 {font_id} 0 R >> >> >>"
            ),
        )

    # Type1 font (no embedding needed — Helvetica is a standard PDF font)
    pdf.add_obj(
        font_id,
        "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
    )

    return pdf.finish(1)


# ─── DOCX builder ────────────────────────────────────────────────────────────

def make_docx(pages: int = 1, words: int = 500) -> bytes:
    """Return bytes for a minimal but valid DOCX with populated app.xml."""
    word_list = (WORDS * 5)[:words]
    paragraphs = []
    for i in range(0, len(word_list), 50):
        chunk = " ".join(word_list[i : i + 50])
        paragraphs.append(f"<w:p><w:r><w:t>{chunk}</w:t></w:r></w:p>")

    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.writestr(
            "[Content_Types].xml",
            '<?xml version="1.0" encoding="UTF-8"?>\n'
            '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
            '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
            '<Default Extension="xml" ContentType="application/xml"/>'
            '<Override PartName="/word/document.xml"'
            ' ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>'
            "</Types>",
        )
        zf.writestr(
            "_rels/.rels",
            '<?xml version="1.0" encoding="UTF-8"?>\n'
            '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
            '<Relationship Id="rId1"'
            ' Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"'
            ' Target="word/document.xml"/>'
            "</Relationships>",
        )
        zf.writestr(
            "docProps/app.xml",
            f'<?xml version="1.0" encoding="UTF-8"?>\n'
            f'<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties">'
            f"<Pages>{pages}</Pages>"
            f"<Words>{words}</Words>"
            f"<Characters>{words * 5}</Characters>"
            f"</Properties>",
        )
        zf.writestr(
            "docProps/core.xml",
            '<?xml version="1.0" encoding="UTF-8"?>\n'
            '<cp:coreProperties'
            ' xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"'
            ' xmlns:dc="http://purl.org/dc/elements/1.1/"'
            ' xmlns:dcterms="http://purl.org/dc/terms/"'
            ' xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">'
            "<dc:creator>Test Author</dc:creator>"
            '<dcterms:created xsi:type="dcterms:W3CDTF">2026-01-01T00:00:00Z</dcterms:created>'
            "</cp:coreProperties>",
        )
        zf.writestr(
            "word/document.xml",
            '<?xml version="1.0" encoding="UTF-8"?>\n'
            '<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">'
            "<w:body>"
            + "\n".join(paragraphs)
            + "</w:body></w:document>",
        )
        zf.writestr(
            "word/_rels/document.xml.rels",
            '<?xml version="1.0" encoding="UTF-8"?>\n'
            '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
            "</Relationships>",
        )
    return buf.getvalue()


# ─── Entry point ─────────────────────────────────────────────────────────────

def main():
    files = {
        "compliant.pdf": make_pdf(num_pages=2, words_per_page=180),
        "single-page.pdf": make_pdf(num_pages=1, words_per_page=40),
        "test.docx": make_docx(pages=1, words=500),
        "three-page.pdf": make_pdf(num_pages=3, words_per_page=120),
    }

    for name, data in files.items():
        path = os.path.join(OUT_DIR, name)
        with open(path, "wb") as f:
            f.write(data)
        print(f"  created {path}  ({len(data)} bytes)")

    print(f"\nGenerated {len(files)} test files in {OUT_DIR}/")


if __name__ == "__main__":
    main()
