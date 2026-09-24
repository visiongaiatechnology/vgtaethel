#!/usr/bin/env bash
# ==============================================================================
# STATUS: DIAMANT VGT SUPREME
# VGT AETHEL — LINUX SERVER INSTALLER & PASSWORD PROVISIONING SCRIPT
# ==============================================================================
set -euo pipefail

umask 077

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="${SCRIPT_DIR}/go-aethel"
BINARY_NAME="AETHEL-SERVER"
BINARY_PATH="${APP_DIR}/${BINARY_NAME}"
WORKSPACE_DIR="${APP_DIR}/vgt_workspace"
GO_FALLBACK_VERSION="1.24.2"

echo -e "${CYAN}${BOLD}"
echo "╔══════════════════════════════════════════════════════════════════════╗"
echo "║   🛡️  VGT AETHEL // LINUX SERVER INSTALLER (BETA V4)                 ║"
echo "║   Headless Server Runtime · Argon2id + AES-256-GCM Login Gate        ║"
echo "╚══════════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

if [[ ! -d "${APP_DIR}" ]]; then
    echo -e "${RED}[ERROR] Verzeichnis 'go-aethel' wurde nicht gefunden (${APP_DIR}).${NC}"
    exit 1
fi

# 1. Detect OS & Architecture
OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_RAW="$(uname -m)"
case "${ARCH_RAW}" in
    x86_64|amd64) GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *)
        echo -e "${RED}[ERROR] Nicht unterstützte CPU-Architektur: ${ARCH_RAW}${NC}"
        exit 1
        ;;
esac

echo -e "${CYAN}[1/5] Systemprüfung:${NC} OS=${OS_NAME} · ARCH=${GOARCH}"

# 2. Check / Install Go Toolchain
ensure_go_toolchain() {
    if command -v go >/dev/null 2>&1; then
        local ver
        ver="$(go version | awk '{print $3}' | sed 's/go//')"
        local major minor
        major="$(echo "${ver}" | cut -d. -f1)"
        minor="$(echo "${ver}" | cut -d. -f2)"
        if [[ "${major}" -gt 1 ]] || { [[ "${major}" -eq 1 ]] && [[ "${minor}" -ge 23 ]]; }; then
            echo -e "${GREEN}  ✓ Go Toolchain gefunden: go${ver}${NC}"
            return 0
        fi
        echo -e "${YELLOW}  ! Gefundene Go-Version (go${ver}) ist älter als 1.23.${NC}"
    else
        echo -e "${YELLOW}  ! Go Toolchain ist noch nicht installiert.${NC}"
    fi

    echo -e "${CYAN}  → Installiere offizielle Go ${GO_FALLBACK_VERSION} Toolchain für linux-${GOARCH}...${NC}"
    local tarball="go${GO_FALLBACK_VERSION}.linux-${GOARCH}.tar.gz"
    local url="https://go.dev/dl/${tarball}"
    local tmp_dir
    tmp_dir="$(mktemp -d)"
    trap 'rm -rf "${tmp_dir}"' EXIT

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "${url}" -o "${tmp_dir}/${tarball}"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "${tmp_dir}/${tarball}" "${url}"
    else
        echo -e "${RED}[ERROR] Weder 'curl' noch 'wget' gefunden. Bitte Go >= 1.23 oder curl installieren.${NC}"
        exit 1
    fi

    local install_base="/usr/local"
    if [[ "${EUID}" -ne 0 ]] && ! sudo -n true 2>/dev/null; then
        install_base="${HOME}/.local"
        mkdir -p "${install_base}"
        rm -rf "${install_base}/go"
        tar -C "${install_base}" -xzf "${tmp_dir}/${tarball}"
    else
        if [[ "${EUID}" -eq 0 ]]; then
            rm -rf /usr/local/go
            tar -C /usr/local -xzf "${tmp_dir}/${tarball}"
        else
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "${tmp_dir}/${tarball}"
        fi
    fi

    export PATH="${install_base}/go/bin:${PATH}"
    if ! command -v go >/dev/null 2>&1; then
        echo -e "${RED}[ERROR] Go Installation fehlgeschlagen.${NC}"
        exit 1
    fi
    echo -e "${GREEN}  ✓ Go erfolgreich bereitgestellt: $(go version)${NC}"
}

if [[ -x "${BINARY_PATH}" ]] && [[ "${1:-}" != "--rebuild" ]]; then
    echo -e "${GREEN}  ✓ Vorab kompiliertes Release-Binary gefunden: ${BINARY_PATH}${NC}"
else
    ensure_go_toolchain
fi

# 3. Port & Operator Password Prompt
echo -e "\n${CYAN}[2/5] Server-Netzwerk & Login-Schutz konfigurieren:${NC}"

SERVER_PORT="${AETHEL_PORT:-}"
if [[ -z "${SERVER_PORT}" ]]; then
    if [[ -t 0 ]]; then
        read -r -p "  Server-Port [Standard: 8080]: " INPUT_PORT
        SERVER_PORT="${INPUT_PORT:-8080}"
    else
        SERVER_PORT="8080"
    fi
fi

if ! [[ "${SERVER_PORT}" =~ ^[0-9]+$ ]] || [[ "${SERVER_PORT}" -lt 1 ]] || [[ "${SERVER_PORT}" -gt 65535 ]]; then
    echo -e "${RED}[ERROR] Ungültiger Port: ${SERVER_PORT} (erlaubt: 1-65535).${NC}"
    exit 1
fi

OPERATOR_PASSWORD="${AETHEL_ADMIN_PASSWORD:-}"
if [[ -z "${OPERATOR_PASSWORD}" ]]; then
    if [[ ! -t 0 ]]; then
        echo -e "${RED}[ERROR] Nicht-interaktive Installation erfordert AETHEL_ADMIN_PASSWORD (mind. 10 Zeichen).${NC}"
        exit 1
    fi
    while true; do
        read -r -s -p "  Neues Operator-Passwort für Web-Login (mind. 10 Zeichen): " PW1
        echo ""
        if [[ "${#PW1}" -lt 10 ]]; then
            echo -e "${YELLOW}  ! Das Passwort muss mindestens 10 Zeichen lang sein.${NC}"
            continue
        fi
        read -r -s -p "  Operator-Passwort bestätigen: " PW2
        echo ""
        if [[ "${PW1}" != "${PW2}" ]]; then
            echo -e "${YELLOW}  ! Die Passwörter stimmen nicht überein. Bitte erneut eingeben.${NC}"
            continue
        fi
        OPERATOR_PASSWORD="${PW1}"
        unset PW1 PW2
        break
    done
else
    if [[ "${#OPERATOR_PASSWORD}" -lt 10 ]]; then
        echo -e "${RED}[ERROR] AETHEL_ADMIN_PASSWORD muss mindestens 10 Zeichen lang sein.${NC}"
        exit 1
    fi
fi

# 4. Compile or Verify Standalone Headless Server Binary
mkdir -p "${WORKSPACE_DIR}"
chmod 700 "${WORKSPACE_DIR}"

if [[ -x "${BINARY_PATH}" ]] && [[ "${1:-}" != "--rebuild" ]]; then
    echo -e "\n${CYAN}[3/5] Nutze vorab kompiliertes AETHEL Server-Binary...${NC}"
    chmod 750 "${BINARY_PATH}"
    echo -e "${GREEN}  ✓ Binary bereit: ${BINARY_PATH}${NC}"
else
    echo -e "\n${CYAN}[3/5] Kompiliere AETHEL Server-Binary (CGO_ENABLED=0, eingebettetes Frontend)...${NC}"
    (
        cd "${APP_DIR}"
        CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w" -o "${BINARY_PATH}" .
    )
    chmod 750 "${BINARY_PATH}"
    echo -e "${GREEN}  ✓ Binary kompiliert: ${BINARY_PATH}${NC}"
fi

# 5. Provision Password into Sealed Authority Store
echo -e "\n${CYAN}[4/5] Versiegele Operator-Passwort (Argon2id + AES-256-GCM)...${NC}"
(
    cd "${APP_DIR}"
    printf '%s\n' "${OPERATOR_PASSWORD}" | "${BINARY_PATH}" --set-password-stdin
)
unset OPERATOR_PASSWORD

# 6. Create start-server.sh Helper & Optional systemd Service
echo -e "\n${CYAN}[5/5] Erstelle Start-Skript & optionalen systemd-Dienst...${NC}"

START_SCRIPT="${SCRIPT_DIR}/start-server.sh"
cat > "${START_SCRIPT}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
cd "${APP_DIR}"
exec "${BINARY_PATH}" --server --addr "0.0.0.0:${SERVER_PORT}" "\$@"
EOF
chmod 750 "${START_SCRIPT}"
echo -e "${GREEN}  ✓ Start-Skript erstellt: ${START_SCRIPT}${NC}"

INSTALL_SYSTEMD="n"
if [[ "${AETHEL_INSTALL_SYSTEMD:-}" == "1" ]] || [[ "${AETHEL_INSTALL_SYSTEMD:-}" == "true" ]]; then
    INSTALL_SYSTEMD="y"
elif [[ -t 0 ]] && command -v systemctl >/dev/null 2>&1; then
    read -r -p "  Soll AETHEL als systemd-Hintergrunddienst eingerichtet & gestartet werden? [J/n]: " RESP_SYSTEMD
    RESP_SYSTEMD="${RESP_SYSTEMD:-J}"
    if [[ "${RESP_SYSTEMD}" =~ ^[JjYy]$ ]]; then
        INSTALL_SYSTEMD="y"
    fi
fi

SERVICE_ACTIVE="false"
if [[ "${INSTALL_SYSTEMD}" == "y" ]] && command -v systemctl >/dev/null 2>&1; then
    RUN_USER="$(id -un)"
    RUN_GROUP="$(id -gn)"
    UNIT_FILE="/etc/systemd/system/aethel.service"
    UNIT_CONTENT="[Unit]
Description=VGT AETHEL Sovereign Intelligence Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${RUN_USER}
Group=${RUN_GROUP}
WorkingDirectory=${APP_DIR}
Environment=AETHEL_SERVER_MODE=1
Environment=AETHEL_BIND_ADDR=0.0.0.0:${SERVER_PORT}
ExecStart=${BINARY_PATH} --server --addr 0.0.0.0:${SERVER_PORT}
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
"
    if [[ "${EUID}" -eq 0 ]]; then
        printf '%s' "${UNIT_CONTENT}" > "${UNIT_FILE}"
        systemctl daemon-reload
        systemctl enable --now aethel.service
        SERVICE_ACTIVE="true"
    elif command -v sudo >/dev/null 2>&1; then
        printf '%s' "${UNIT_CONTENT}" | sudo tee "${UNIT_FILE}" >/dev/null
        sudo systemctl daemon-reload
        sudo systemctl enable --now aethel.service
        SERVICE_ACTIVE="true"
    else
        echo -e "${YELLOW}  ! Keine Root-/sudo-Rechte für /etc/systemd/system. Nutze ./start-server.sh zum Starten.${NC}"
    fi
fi

SERVER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || echo "127.0.0.1")"
if [[ -z "${SERVER_IP}" ]]; then
    SERVER_IP="127.0.0.1"
fi

echo -e "\n${GREEN}${BOLD}╔══════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}${BOLD}║   ✅ VGT AETHEL SERVER INSTALLATION ERFOLGREICH ABGESCHLOSSEN        ║${NC}"
echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════════════════════════════════╝${NC}"
echo -e "  ${BOLD}Web-Oberfläche:${NC}      ${CYAN}http://${SERVER_IP}:${SERVER_PORT}${NC}"
echo -e "  ${BOLD}Login-Schutz:${NC}        ${GREEN}AKTIV (Argon2id + AES-256-GCM Session Gate)${NC}"
echo -e "  ${BOLD}Workspace-Pfad:${NC}      ${WORKSPACE_DIR}"
echo -e "  ${BOLD}Passwort ändern:${NC}     ${BINARY_PATH} --set-password-stdin"
if [[ "${SERVICE_ACTIVE}" == "true" ]]; then
    echo -e "  ${BOLD}Systemd-Status:${NC}      ${GREEN}AKTIV (systemctl status aethel)${NC}"
    echo -e "  ${BOLD}Live-Logs:${NC}           journalctl -u aethel -f"
else
    echo -e "  ${BOLD}Server starten:${NC}      ${CYAN}./start-server.sh${NC}"
fi
echo ""
