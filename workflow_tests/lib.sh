# ==========================================================
# Shared test library
# Source this from workflow test scripts.
# ==========================================================

# TODO: testing: test other workflows: videos, playlists, watchalters, reactions, notifications, websockets, ...etc
# TODO: testing: we need to use different docker containers for testing and reset them for each test

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# ---- Load .env ----
if [ -f "$PROJECT_DIR/.env" ]; then
    set -a
    source <(grep -v '^#' "$PROJECT_DIR/.env" | sed 's/\s*=\s*/=/g')
    set +a
fi

PORT="${PORT:-8080}"
PAPERCUT_WEB_PORT="${PAPERCUT_WEB_PORT:-3000}"

BASE_URL="${APP_BASE_URL:-http://localhost:${PORT}}"
PAPERCUT_URL="http://localhost:${PAPERCUT_WEB_PORT}"
CURL_TIMEOUT=10

# ---- Colors ----
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# ---- Test counters (reset per script) ----
PASS=0
FAIL=0
TESTS=0
_API_STATUS=0
_API_BODY=""
CSRF_TOKEN=""
COOKIE_JAR=""
SERVER_PID=""

# ---- Dependencies ----

check_deps() {
    local missing=false
    for cmd in curl jq sed grep; do
        if ! command -v "$cmd" &>/dev/null; then
            echo -e "${RED}Missing dependency:${NC} $cmd"
            missing=true
        fi
    done
    if [ "$missing" = true ]; then
        exit 1
    fi
}

check_goose() {
    if ! command -v goose &>/dev/null; then
        echo -e "${RED}Missing dependency: goose${NC}"
        exit 1
    fi
}

reset_db() {
    echo -e "${YELLOW}Resetting database...${NC}"
    local services=("user" "video" "media" "reaction" "social" "notification")
    for s in "${services[@]}"; do
        local dir="$PROJECT_DIR/internals/services/$s/db/migrations"
        local table="${s}_goose_db_version"

        GOOSE_DRIVER="postgres" \
        GOOSE_DBSTRING="$POSTGRES_URL" \
            goose -dir "$dir" reset -table "$table" > /dev/null 2>&1

        GOOSE_DRIVER="postgres" \
        GOOSE_DBSTRING="$POSTGRES_URL" \
            goose -dir "$dir" up -table "$table" > /dev/null 2>&1

        echo -e "  ${GREEN}✓${NC} $s migrated"
    done
}

# ---- Server management ----

build_server() {
    echo -e "  ${YELLOW}Building server...${NC}"
    make -C "$PROJECT_DIR" build > /tmp/vodzilla_build.log 2>&1
    echo -e "  ${GREEN}✓${NC} build complete"
}

start_server() {
    build_server
    echo -e "  ${YELLOW}Starting server...${NC}"
    "$PROJECT_DIR/bin/server" > /tmp/vodzilla_server.log 2>&1 &
    SERVER_PID=$!
    for i in $(seq 1 30); do
        if curl -s -o /dev/null -w "%{http_code}" --connect-timeout 1 "$BASE_URL/health" 2>/dev/null | grep -q 200; then
            echo -e "  ${GREEN}✓${NC} server started (PID $SERVER_PID)"
            return
        fi
        sleep 1
    done
    echo -e "  ${RED}✗${NC} server failed to start"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
}

stop_server() {
    if [ -n "$SERVER_PID" ]; then
        echo -e "  ${YELLOW}Stopping server (PID $SERVER_PID)...${NC}"
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        SERVER_PID=""
        echo -e "  ${GREEN}✓${NC} server stopped"
    fi
}

check_connectivity() {
    echo -n "  Checking server connectivity... "
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$BASE_URL/health" 2>/dev/null || echo "failed")
    if [ "$status" = "200" ]; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}FAILED${NC} (server returned $status)"
        echo -e "  Make sure the server is running at $BASE_URL"
        exit 1
    fi

    echo -n "  Checking Papercut connectivity... "
    status=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$PAPERCUT_URL/health" 2>/dev/null || echo "failed")
    if [ "$status" = "200" ]; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${YELLOW}WARNING${NC} (Papercut returned $status)"
    fi
    echo ""
}

# ---- API helpers ----

api() {
    local method="$1"
    local path="$2"
    local data="${3:-}"
    local needs_csrf="${4:-false}"

    local args=(-s --max-time "$CURL_TIMEOUT" -X "$method" "$BASE_URL$path")

    if [ -n "$data" ]; then
        args+=(-H "Content-Type: application/json" -d "$data")
    fi

    if [ -f "$COOKIE_JAR" ]; then
        args+=(-b "$COOKIE_JAR")
    fi
    args+=(-c "$COOKIE_JAR")

    if [ "$needs_csrf" = "true" ] && [ -n "$CSRF_TOKEN" ]; then
        args+=(-H "X-CSRF-Token: $CSRF_TOKEN")
    fi

    local tmpfile
    tmpfile=$(mktemp)
    _API_STATUS=$(curl "${args[@]}" -o "$tmpfile" -w "%{http_code}" 2>/dev/null || echo "000")
    _API_BODY=$(cat "$tmpfile")
    rm -f "$tmpfile"
}

expect() {
    local description="$1"
    local expected_status="$2"
    local expected_kind="${3:-}"

    TESTS=$((TESTS + 1))
    local ok=true
    local reason=""

    if [ "$_API_STATUS" != "$expected_status" ]; then
        ok=false
        reason="status: want $expected_status, got $_API_STATUS"
    fi

    if [ -n "$expected_kind" ] && [ "$ok" = true ]; then
        local kind
        kind=$(echo "$_API_BODY" | jq -r '.kind // "none"' 2>/dev/null || echo "none")
        if [ "$kind" != "$expected_kind" ]; then
            ok=false
            reason="kind: want $expected_kind, got $kind"
        fi
    fi

    if [ "$ok" = true ]; then
        PASS=$((PASS + 1))
        echo -e "  ${GREEN}✓${NC} $description"
    else
        FAIL=$((FAIL + 1))
        echo -e "  ${RED}✗${NC} $description"
        echo -e "    ${RED}→ $reason${NC}"
        if [ -n "$_API_BODY" ]; then
            local snippet
            snippet=$(echo "$_API_BODY" | head -c 300)
            echo -e "    body: $snippet"
        fi
    fi
}

# ---- Auth setup (register + verify + login) ----

auth_setup() {
    local email="$1"
    local password="$2"
    local name="$3"
    local username="$4"

    echo -e "${BOLD}--- Auth Setup ---${NC}"

    local register_data
    register_data=$(cat <<EOF
{
  "email": "$email",
  "password": "$password",
  "name": "$name",
  "username": "$username"
}
EOF
)
    api POST /auth/register "$register_data"
    expect "register" 200

    api POST /auth/verification_email "{\"email\":\"$email\",\"baseUrl\":\"$BASE_URL/auth/verification_email/verify\"}"
    expect "send verification email" 200

    wait_for_magic_link_click

    api POST /auth/login "{\"email\":\"$email\",\"password\":\"$password\"}"
    expect "login" 200

    if grep -q csrf_token "$COOKIE_JAR" 2>/dev/null; then
        CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" | awk '{print $NF}')
    fi
}

# ---- Magic link prompt ----

wait_for_magic_link_click() {
    echo ""
    echo -e "  ${YELLOW}Email sent!${NC}"
    echo -e "  ${YELLOW}Open http://localhost:${PAPERCUT_WEB_PORT} in your browser,${NC}"
    echo -e "  ${YELLOW}find the verification email, click the magic link,${NC}"
    echo -e "  ${YELLOW}then press ENTER to continue.${NC}"
    read -p ""
    echo -e "  ${GREEN}✓${NC} Continuing..."
}

# ---- Summary ----

print_summary() {
    echo ""
    echo -e "${BOLD}========================================${NC}"
    if [ "$FAIL" -gt 0 ]; then
        echo -e "${BOLD}Results:${NC} ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC} ($TESTS total)"
        echo ""
        exit 1
    else
        echo -e "${BOLD}Results:${NC} ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC} ($TESTS total)"
        echo ""
    fi
}
