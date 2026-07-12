#!/usr/bin/env bash
set -euo pipefail

# ==========================================================
# Auth Workflow Tests
# Tests: register, login (before verification), send
# verification email, verify (magic link), login (after),
# logout, edit credentials
# ==========================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

# ---- Random test user ----
SUFFIX=$(date +%s)
TEST_EMAIL="testuser_${SUFFIX}@example.com"
TEST_PASSWORD="Password123"
TEST_NAME="Test User ${SUFFIX}"
TEST_USERNAME="testuser_${SUFFIX}"

COOKIE_JAR="/tmp/vodzilla_auth_test_cookies.txt"

# ---- Main ----

main() {
    rm -f "$COOKIE_JAR"
    check_deps
    check_goose
    reset_db
    trap stop_server EXIT
    start_server

    echo ""
    echo -e "${BOLD}==============================${NC}"
    echo -e "${BOLD}  Auth Workflow Tests${NC}"
    echo -e "${BOLD}==============================${NC}"
    echo "  Server:   $BASE_URL"
    echo "  Papercut: $PAPERCUT_URL"
    echo "  User:     $TEST_EMAIL / $TEST_USERNAME"
    echo ""

    check_connectivity

    # ================================================================
    # 1. REGISTER
    # ================================================================
    echo -e "${BOLD}--- Register ---${NC}"

    api POST /auth/register "{}"
    expect "empty body" 400 InvalidData

    api POST /auth/register '{"password":"Password123","name":"Test","username":"test"}'
    expect "missing email" 400 InvalidData

    api POST /auth/register '{"email":"test@example.com","name":"Test","username":"test"}'
    expect "missing password" 400 InvalidData

    api POST /auth/register '{"email":"test@example.com","password":"Password123","username":"test"}'
    expect "missing name" 400 InvalidData

    api POST /auth/register '{"email":"test@example.com","password":"Password123","name":"Test"}'
    expect "missing username" 400 InvalidData

    api POST /auth/register '{"email":"bad","password":"Password123","name":"Test","username":"test"}'
    expect "invalid email" 400 InvalidData

    api POST /auth/register '{"email":"test@example.com","password":"short","name":"Test","username":"test"}'
    expect "short password" 400 InvalidData

    api POST /auth/register '{"email":"test@example.com","password":"Password123","name":"","username":"test"}'
    expect "empty name" 400 InvalidData

    api POST /auth/register '{"email":"test@example.com","password":"Password123","name":"Test","username":"bad user!"}'
    expect "invalid username chars" 400 InvalidData

    local register_data
    register_data=$(cat <<EOF
{
  "email": "$TEST_EMAIL",
  "password": "$TEST_PASSWORD",
  "name": "$TEST_NAME",
  "username": "$TEST_USERNAME"
}
EOF
)
    api POST /auth/register "$register_data"
    expect "valid registration" 200

    api POST /auth/register "$register_data"
    expect "duplicate email" 409 EmailConflict

    api POST /auth/register '{"email":"other@example.com","password":"Password123","name":"Other","username":"'$TEST_USERNAME'"}'
    expect "duplicate username" 409 UsernameConflict

    # ================================================================
    # 2. LOGIN BEFORE VERIFICATION  (must fail with 403)
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Login Before Verification ---${NC}"

    api POST /auth/login '{"email":"wrong@example.com","password":"Password123"}'
    expect "wrong email" 401 Unauthorized

    api POST /auth/login "{\"email\":\"$TEST_EMAIL\",\"password\":\"WrongPassword123\"}"
    expect "wrong password" 401 Unauthorized

    api POST /auth/login "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"
    expect "correct creds but unverified" 403 EmailNotVerified

    # ================================================================
    # 3. SEND VERIFICATION EMAIL
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Send Verification Email ---${NC}"

    api POST /auth/verification_email "{}"
    expect "empty body" 400 InvalidData

    api POST /auth/verification_email '{"baseUrl":"http://localhost:8080/auth/verification_email/verify"}'
    expect "missing email" 400 InvalidData

    api POST /auth/verification_email "{\"email\":\"$TEST_EMAIL\"}"
    expect "missing baseUrl" 400 InvalidData

    api POST /auth/verification_email '{"email":"bad","baseUrl":"http://localhost:8080/auth/verification_email/verify"}'
    expect "invalid email" 400 InvalidData

    api POST /auth/verification_email "{\"email\":\"$TEST_EMAIL\",\"baseUrl\":\"not-a-url\"}"
    expect "invalid baseUrl" 400 InvalidData

    api POST /auth/verification_email '{"email":"nonexistent@example.com","baseUrl":"http://localhost:8080/auth/verification_email/verify"}'
    expect "non-existent email" 404 UserNotFound

    api POST /auth/verification_email "{\"email\":\"$TEST_EMAIL\",\"baseUrl\":\"$BASE_URL/auth/verification_email/verify\"}"
    expect "valid request" 200

    # ================================================================
    # 4. VERIFY EMAIL (via browser magic link)
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Verify Email ---${NC}"

    # Test invalid verify requests first
    api GET /auth/verification_email/verify
    expect "missing token query" 400 InvalidRequest

    api GET "/auth/verification_email/verify?token=bad-token-123"
    expect "invalid token" 404 TokenNotFound

    # Prompt user to click the magic link in browser
    wait_for_magic_link_click

    # ================================================================
    # 5. LOGIN AFTER VERIFICATION
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Login After Verification ---${NC}"

    api POST /auth/login "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"
    expect "login with verified email" 200

    # Extract CSRF token from cookie jar
    CSRF_TOKEN=""
    if grep -q csrf_token "$COOKIE_JAR" 2>/dev/null; then
        CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" | awk '{print $NF}')
        echo -e "  CSRF: ${CSRF_TOKEN:0:20}..."
    else
        echo -e "  ${YELLOW}Warning: csrf_token not found in cookies${NC}"
    fi

    # ================================================================
    # 6. LOGOUT
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Logout ---${NC}"

    api POST /auth/logout "" true
    expect "logout" 200

    # Verify session is dead
    api POST /auth/logout "" true
    expect "logout again (dead session)" 401 Unauthorized

    # ================================================================
    # 7. EDIT CREDENTIALS
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Edit Credentials ---${NC}"

    # Login fresh
    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api POST /auth/login "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"
    if [ "$_API_STATUS" = 200 ]; then
        CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" | awk '{print $NF}' 2>/dev/null || echo "")
    fi

    api PUT /auth/credentials "{}" true
    expect "empty body" 400 InvalidData

    api PUT /auth/credentials '{"email":"bad","password":"NewPassword123"}' true
    expect "invalid email" 400 InvalidData

    api PUT /auth/credentials '{"email":"'$TEST_EMAIL'","password":""}' true
    expect "empty password" 400 InvalidData

    # Valid edit
    local NEW_EMAIL="new_${TEST_EMAIL}"
    api PUT /auth/credentials "{\"email\":\"$NEW_EMAIL\",\"password\":\"NewPassword456\"}" true
    expect "valid edit" 200

    # Login with new credentials
    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api POST /auth/login "{\"email\":\"$NEW_EMAIL\",\"password\":\"NewPassword456\"}"
    expect "login with new credentials" 200

    # Login with old credentials should fail
    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api POST /auth/login "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"
    expect "login with old credentials (rejected)" 401 Unauthorized

    # ================================================================
    # CLEANUP
    # ================================================================
    stop_server
    reset_db

    # ================================================================
    # SUMMARY
    # ================================================================
    print_summary
}

main "$@"
