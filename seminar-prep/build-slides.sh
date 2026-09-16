#!/usr/bin/env bash
# 세미나 덱 하나를 파일 하나로 낸다. 그림을 HTML 안에 박아서 HTML 만 들고 다녀도 뜬다.
#
#   scripts/build-slides.sh                  seminar-slides.md -> seminar.html
#   scripts/build-slides.sh deck.md out.html
#
# marp 는 그림을 상대 경로로 걸어 둔다. 그대로 두면 HTML 만 옮겼을 때 그림이 깨진다.
# SVG 는 img 째로 <svg> 본문으로 갈아 끼우고 (벡터라 확대해도 안 깨진다),
# 나머지는 data URI 로 박는다.
set -euo pipefail

SRC="${1:-seminar-slides.md}"
OUT="${2:-seminar.html}"
MARP="${MARP:-$HOME/node_modules/.bin/marp}"

if [ ! -x "$MARP" ]; then
  MARP="$(command -v marp || true)"
fi
if [ -z "$MARP" ]; then
  echo "marp not found. set MARP=/path/to/marp" >&2
  exit 1
fi

TMP="$(mktemp -t slides-XXXXXX.html)"
trap 'rm -f "$TMP"' EXIT

# --no-stdin 이 없으면 marp 가 stdin 에서 마크다운이 올 줄 알고 멎는다
"$MARP" "$SRC" -o "$TMP" --html --no-stdin

python3 - "$TMP" "$OUT" <<'PY'
import base64, mimetypes, pathlib, re, sys

src_html, out_html = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
html = src_html.read_text(encoding="utf-8")
root = pathlib.Path.cwd()
svg_inlined = data_uris = 0
missing = []

IMG = re.compile(r"<img\b[^>]*?/?>", re.I)
ATTR = re.compile(r'(\w[\w-]*)\s*=\s*"([^"]*)"')


def local(url):
    if not url or url.startswith(("data:", "http:", "https:", "//")):
        return None
    p = (root / url).resolve()
    return p if p.is_file() else False


def as_data_uri(path):
    kind = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    return f"data:{kind};base64," + base64.b64encode(path.read_bytes()).decode("ascii")


def inline_svg(path, img_style):
    """SVG 파일 본문을 그대로 박는다. XML 선언과 DOCTYPE 은 떨어뜨린다."""
    text = path.read_text(encoding="utf-8")
    start = text.index("<svg")
    body = text[start:]
    # 여는 태그의 style 에 img 가 지시한 크기와 가운데 정렬을 얹는다
    open_end = body.index(">")
    head, rest = body[:open_end], body[open_end:]
    extra = f"{img_style.rstrip(chr(59))};display:block;margin:0 auto;".lstrip(";")
    if 'style="' in head:
        head = head.replace('style="', f'style="{extra}', 1)
    else:
        head = f'{head} style="{extra}"'
    return head + rest


def handle_img(m):
    global svg_inlined, data_uris
    tag = m.group(0)
    attrs = dict(ATTR.findall(tag))
    path = local(attrs.get("src", ""))
    if path is None:
        return tag
    if path is False:
        missing.append(attrs.get("src", ""))
        return tag
    if path.suffix.lower() == ".svg":
        svg_inlined += 1
        return inline_svg(path, attrs.get("style", ""))
    data_uris += 1
    return tag.replace(f'src="{attrs["src"]}"', f'src="{as_data_uri(path)}"')


html = IMG.sub(handle_img, html)


def handle_src(m):
    global data_uris
    path = local(m.group(2))
    if not path:
        return m.group(0)
    data_uris += 1
    return f'{m.group(1)}"{as_data_uri(path)}"'


html = re.sub(r'(\ssrc=)"([^"]+)"', handle_src, html)

out_html.write_text(html, encoding="utf-8")
size = out_html.stat().st_size
print(f"inlined {svg_inlined} svg + {data_uris} data uri  ->  {out_html} ({size/1024/1024:.2f} MB)")
for u in missing:
    print(f"  MISSING {u}", file=sys.stderr)
if missing:
    sys.exit(1)
PY
