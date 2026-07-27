#!/usr/bin/env bash
#
# localvoice.sh — set up and run the local MLX voice stack for videoascode.
#
# Provides local, offline TTS + STT for `vac ... --local`:
#   - F5-TTS MLX  (TTS)  -> unix:///tmp/omnivoice-f5tts.sock
#   - Whisper MLX (STT)  -> unix:///tmp/omnivoice-whisper.sock
#
# Both run as Python/MLX gRPC servers and REQUIRE Apple Silicon (arm64).
# The MLX wheels are arm64-only, so this script always launches Python under
# `arch -arm64`, which matters when the calling shell is running under Rosetta.
#
# The server sources live in the omnivoice-core module; this script locates
# that module via `go list` so it works regardless of where it is checked out.
#
# Usage:
#   scripts/localvoice.sh setup     # create arm64 venv, install deps, gen protos
#   scripts/localvoice.sh start     # start both servers (foreground: Ctrl-C stops)
#   scripts/localvoice.sh start -d  # start both servers in the background
#   scripts/localvoice.sh stop      # stop background servers
#   scripts/localvoice.sh status    # show whether the sockets are live
#   scripts/localvoice.sh up        # setup (if needed) + start -d
#
# Env overrides:
#   PYTHON            Universal2/arm64 python3 to build the venv (default: autodetect)
#   VENV_DIR          venv location (default: <repo>/.localvoice-venv)
#   WHISPER_MODEL     Whisper model (default: large-v3-turbo)
#   F5TTS_SOCK        (default: /tmp/omnivoice-f5tts.sock)
#   WHISPER_SOCK      (default: /tmp/omnivoice-whisper.sock)
#
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VENV_DIR="${VENV_DIR:-$REPO_DIR/.localvoice-venv}"
LOG_DIR="${LOG_DIR:-$REPO_DIR/.localvoice-logs}"
WHISPER_MODEL="${WHISPER_MODEL:-large-v3-turbo}"
F5TTS_SOCK="${F5TTS_SOCK:-/tmp/omnivoice-f5tts.sock}"
WHISPER_SOCK="${WHISPER_SOCK:-/tmp/omnivoice-whisper.sock}"

VPY="$VENV_DIR/bin/python"

die() { echo "error: $*" >&2; exit 1; }
info() { echo "==> $*"; }

require_apple_silicon() {
  # The hardware must be arm64 even if this shell is x86_64 under Rosetta.
  if ! arch -arm64 /usr/bin/true 2>/dev/null; then
    die "Apple Silicon (arm64) is required; 'arch -arm64' is unavailable on this machine."
  fi
}

core_dir() {
  # Resolve the omnivoice-core module directory (respects go.work / replace).
  (cd "$REPO_DIR" && go list -m -f '{{.Dir}}' github.com/plexusone/omnivoice-core) \
    || die "could not locate github.com/plexusone/omnivoice-core via 'go list -m'"
}

find_python() {
  if [[ -n "${PYTHON:-}" ]]; then echo "$PYTHON"; return; fi
  # Prefer a universal2 / arm64-capable python3. The python.org framework
  # builds are universal2 and run native arm64 under `arch -arm64`.
  local candidates=(
    /opt/homebrew/bin/python3
    /Library/Frameworks/Python.framework/Versions/3.14/bin/python3
    /Library/Frameworks/Python.framework/Versions/3.12/bin/python3
    /Library/Frameworks/Python.framework/Versions/3.11/bin/python3
    "$(command -v python3 || true)"
  )
  local p
  for p in "${candidates[@]}"; do
    [[ -n "$p" && -x "$p" ]] || continue
    if arch -arm64 "$p" -c 'import platform,sys; sys.exit(0 if platform.machine()=="arm64" else 1)' 2>/dev/null; then
      echo "$p"; return
    fi
  done
  die "no arm64-capable python3 found; install one (e.g. 'brew install python') or set PYTHON=/path/to/python3"
}

cmd_setup() {
  require_apple_silicon
  local core; core="$(core_dir)"
  local f5_srv="$core/providers/f5tts-mlx/server"
  local wh_srv="$core/providers/whisper-mlx/server"
  [[ -f "$f5_srv/f5tts_server.py" ]] || die "f5tts server not found at $f5_srv"
  [[ -f "$wh_srv/whisper_server.py" ]] || die "whisper server not found at $wh_srv"

  if [[ ! -x "$VPY" ]]; then
    local py; py="$(find_python)"
    info "creating arm64 venv at $VENV_DIR (from $py)"
    arch -arm64 "$py" -m venv "$VENV_DIR"
  fi

  info "installing MLX + gRPC deps (arm64) into venv"
  arch -arm64 "$VPY" -m pip install --quiet --upgrade pip wheel setuptools
  arch -arm64 "$VPY" -m pip install --quiet \
    "grpcio>=1.70.0" "grpcio-tools>=1.70.0" \
    "mlx>=0.22.0" "f5-tts-mlx>=0.2.0" "mlx-whisper>=0.4.0" \
    "soundfile>=0.13.0" "numpy>=2.0.0"

  info "generating F5-TTS proto stubs (localtts)"
  arch -arm64 "$VPY" -m grpc_tools.protoc \
    -I"$core/proto/localtts/v1" \
    --python_out="$f5_srv" --grpc_python_out="$f5_srv" \
    "$core/proto/localtts/v1/localtts.proto"
  if [[ -f "$f5_srv/localtts/v1/localtts_pb2.py" ]]; then
    mv "$f5_srv/localtts/v1/"localtts_pb2*.py "$f5_srv/"; rm -rf "$f5_srv/localtts"
  fi
  _fix_grpc_import "$f5_srv/localtts_pb2_grpc.py" "from localtts.v1 import localtts_pb2" "import localtts_pb2"

  info "generating Whisper proto stubs (localstt)"
  arch -arm64 "$VPY" -m grpc_tools.protoc \
    -I"$core/proto/localstt/v1" \
    --python_out="$wh_srv" --grpc_python_out="$wh_srv" \
    "$core/proto/localstt/v1/localstt.proto"
  if [[ -f "$wh_srv/localstt/v1/localstt_pb2.py" ]]; then
    mv "$wh_srv/localstt/v1/"localstt_pb2*.py "$wh_srv/"; rm -rf "$wh_srv/localstt"
  fi
  _fix_grpc_import "$wh_srv/localstt_pb2_grpc.py" "from localstt.v1 import localstt_pb2" "import localstt_pb2"

  info "setup complete"
}

_fix_grpc_import() {
  local file="$1" from="$2" to="$3"
  [[ -f "$file" ]] || return 0
  sed -i '' "s#$from#$to#" "$file" 2>/dev/null || sed -i "s#$from#$to#" "$file"
}

cmd_start() {
  require_apple_silicon
  [[ -x "$VPY" ]] || die "venv missing; run: $0 setup"
  local core; core="$(core_dir)"
  local f5_srv="$core/providers/f5tts-mlx/server"
  local wh_srv="$core/providers/whisper-mlx/server"
  [[ -f "$f5_srv/localtts_pb2.py" ]] || die "F5-TTS proto stubs missing; run: $0 setup"
  [[ -f "$wh_srv/localstt_pb2.py" ]] || die "Whisper proto stubs missing; run: $0 setup"

  rm -f "$F5TTS_SOCK" "$WHISPER_SOCK"
  mkdir -p "$LOG_DIR"

  local detach=0
  [[ "${1:-}" == "-d" || "${1:-}" == "--detach" ]] && detach=1

  if [[ $detach -eq 1 ]]; then
    info "starting F5-TTS MLX server (background) -> $F5TTS_SOCK"
    ( cd "$f5_srv" && nohup arch -arm64 "$VPY" f5tts_server.py \
        --socket "$F5TTS_SOCK" --auto-load >"$LOG_DIR/f5tts.log" 2>&1 & echo $! >"$LOG_DIR/f5tts.pid" )
    info "starting Whisper MLX server (background, model=$WHISPER_MODEL) -> $WHISPER_SOCK"
    ( cd "$wh_srv" && nohup arch -arm64 "$VPY" whisper_server.py \
        --socket "$WHISPER_SOCK" --model "$WHISPER_MODEL" >"$LOG_DIR/whisper.log" 2>&1 & echo $! >"$LOG_DIR/whisper.pid" )
    info "waiting for sockets..."
    _wait_socket "$F5TTS_SOCK" 120 || die "F5-TTS server did not come up; see $LOG_DIR/f5tts.log"
    _wait_socket "$WHISPER_SOCK" 30 || die "Whisper server did not come up; see $LOG_DIR/whisper.log"
    info "both servers up. logs in $LOG_DIR"
    info "stop with: $0 stop"
  else
    info "starting Whisper MLX server (background, model=$WHISPER_MODEL) -> $WHISPER_SOCK"
    ( cd "$wh_srv" && nohup arch -arm64 "$VPY" whisper_server.py \
        --socket "$WHISPER_SOCK" --model "$WHISPER_MODEL" >"$LOG_DIR/whisper.log" 2>&1 & echo $! >"$LOG_DIR/whisper.pid" )
    info "starting F5-TTS MLX server (foreground; Ctrl-C to stop both) -> $F5TTS_SOCK"
    trap 'cmd_stop' INT TERM
    ( cd "$f5_srv" && exec arch -arm64 "$VPY" f5tts_server.py --socket "$F5TTS_SOCK" --auto-load )
  fi
}

_wait_socket() {
  local sock="$1" tries="${2:-60}" i=0
  while [[ $i -lt $tries ]]; do
    [[ -S "$sock" ]] && return 0
    sleep 1; i=$((i+1))
  done
  return 1
}

cmd_stop() {
  local stopped=0 f
  for f in f5tts whisper; do
    if [[ -f "$LOG_DIR/$f.pid" ]]; then
      kill "$(cat "$LOG_DIR/$f.pid")" 2>/dev/null && stopped=1
      rm -f "$LOG_DIR/$f.pid"
    fi
  done
  pkill -f "f5tts_server.py" 2>/dev/null && stopped=1 || true
  pkill -f "whisper_server.py" 2>/dev/null && stopped=1 || true
  rm -f "$F5TTS_SOCK" "$WHISPER_SOCK"
  [[ $stopped -eq 1 ]] && info "servers stopped" || info "no servers were running"
}

cmd_status() {
  local s
  for s in "F5-TTS:$F5TTS_SOCK" "Whisper:$WHISPER_SOCK"; do
    local name="${s%%:*}" sock="${s#*:}"
    if [[ -S "$sock" ]]; then echo "  $name  UP    ($sock)"; else echo "  $name  down  ($sock)"; fi
  done
}

cmd_up() {
  [[ -x "$VPY" && -f "$(core_dir)/providers/whisper-mlx/server/localstt_pb2.py" ]] || cmd_setup
  cmd_start -d
}

case "${1:-}" in
  setup)  cmd_setup ;;
  start)  shift; cmd_start "${1:-}" ;;
  stop)   cmd_stop ;;
  status) cmd_status ;;
  up)     cmd_up ;;
  *) cat >&2 <<EOF
usage: $0 {setup|start [-d]|stop|status|up}

  setup      create arm64 venv, install MLX deps, generate proto stubs
  start      run both servers in the foreground (Ctrl-C stops both)
  start -d   run both servers in the background
  stop       stop background servers
  status     show whether the gRPC sockets are live
  up         setup if needed, then start -d
EOF
    exit 2 ;;
esac
