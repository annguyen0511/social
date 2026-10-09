#!/usr/bin/env bash
#
# Mở một quick tunnel của Cloudflare tới frontend, đọc URL ngẫu nhiên nó cấp,
# ghi thẳng vào FRONTEND_URL trong .envrc rồi chạy direnv allow. Nhờ vậy link
# kích hoạt trong email trỏ đúng tunnel mà không phải sửa tay mỗi phiên.
#
# Opens a Cloudflare quick tunnel to the frontend, reads the random URL it
# hands out, writes it into FRONTEND_URL in .envrc and runs direnv allow, so
# the activation links in email point at the tunnel without editing anything
# by hand each session.
set -euo pipefail

PORT="${PORT:-5173}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENVRC="$ROOT/.envrc"
LOG="$(mktemp)"
# cloudflared vẫn đọc ~/.cloudflared/config.yml ngay cả khi có --url, nên nó
# lấy nhầm cấu hình của named tunnel: URL trycloudflare in ra không được định
# tuyến và mọi request trả 404. Một file config rỗng buộc nó bỏ qua.
#
# cloudflared still reads ~/.cloudflared/config.yml even with --url, so it
# picks up the named tunnel settings: the trycloudflare URL it prints is never
# routed and every request 404s. An empty config file makes it ignore them.
CFG="$(mktemp)"

cleanup() { rm -f "$LOG" "$CFG"; kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

command -v cloudflared >/dev/null || {
  echo "cloudflared chưa được cài. Xem hướng dẫn cài ở README." >&2
  exit 1
}

echo "Đang mở tunnel tới http://localhost:$PORT ..."
cloudflared --config "$CFG" tunnel --url "http://localhost:$PORT" >"$LOG" 2>&1 &
CF_PID=$!

# cloudflared in URL ra sau vài giây. Vừa chờ vừa canh xem nó có chết không,
# để báo lỗi thật thay vì treo tới hết thời gian chờ.
URL=""
for _ in $(seq 1 45); do
  URL="$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOG" | head -1 || true)"
  [ -n "$URL" ] && break
  if ! kill -0 "$CF_PID" 2>/dev/null; then
    echo "cloudflared thoát sớm:" >&2; cat "$LOG" >&2; exit 1
  fi
  sleep 1
done

if [ -z "$URL" ]; then
  echo "Không đọc được URL sau 45 giây:" >&2; cat "$LOG" >&2; exit 1
fi

# Thay dòng FRONTEND_URL nếu đã có, thêm mới nếu chưa. Dùng python cho chắc,
# vì giá trị chứa dấu / dễ làm sed hiểu nhầm.
python3 - "$ENVRC" "$URL" <<'PY'
import pathlib, re, sys
path, url = pathlib.Path(sys.argv[1]), sys.argv[2]
line = f'export FRONTEND_URL="{url}"'
text = path.read_text() if path.exists() else ""
if re.search(r'^export FRONTEND_URL=.*$', text, flags=re.M):
    text = re.sub(r'^export FRONTEND_URL=.*$', line, text, flags=re.M)
else:
    text = text.rstrip("\n") + ("\n\n" if text.strip() else "") + line + "\n"
path.write_text(text)
PY

direnv allow "$ROOT" 2>/dev/null || true

cat <<EOF

  Tunnel:       $URL
  Đã ghi:       FRONTEND_URL trong .envrc  (direnv allow đã chạy)

  Nếu API đã chạy sẵn, phải khởi động lại nó: tiến trình đang chạy giữ
  FRONTEND_URL cũ trong bộ nhớ và direnv không sửa được tiến trình đã khởi động.

  Dùng 'make dev-public' thì cả hai lên cùng lúc, đúng thứ tự, khỏi lo.

  Ctrl-C để đóng tunnel.

EOF

# Nếu được truyền lệnh, chạy nó SAU khi .envrc đã có URL mới, với môi trường
# vừa nạp lại. Nhờ vậy API đọc đúng FRONTEND_URL ngay từ lúc khởi động và
# không còn phải nhớ khởi động lại.
#
# When given a command, run it AFTER .envrc holds the new URL, with the
# environment freshly loaded, so the API starts with the right FRONTEND_URL
# and nothing has to be restarted by hand.
if [ "$#" -gt 0 ]; then
  set -a; . "$ENVRC"; set +a
  "$@" &
fi

tail -f "$LOG" &
wait "$CF_PID"
