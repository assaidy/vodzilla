#!/usr/bin/env bash
set -euo pipefail

# ==========================================================
# Profile Workflow Tests
# Tests: get profiles, get by username/id, edit profile,
# avatar upload generation, delete avatar, delete profile
# ==========================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

# ---- Random test user ----
SUFFIX=$(date +%s)
EMAIL="protest_${SUFFIX}@example.com"
PASSWORD="Password123"
NAME="Profile Tester ${SUFFIX}"
USERNAME="protest_${SUFFIX}"

EMAIL2="protest2_${SUFFIX}@example.com"
NAME2="Profile Tester 2 ${SUFFIX}"
USERNAME2="protest2_${SUFFIX}"

COOKIE_JAR="/tmp/vodzilla_profile_test_cookies.txt"

relogin() {
    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api POST /auth/login "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
    if [ "$_API_STATUS" = 200 ]; then
        CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" | awk '{print $NF}')
    fi
}

main() {
    rm -f "$COOKIE_JAR"
    check_deps
    check_goose
    reset_db
    trap stop_server EXIT
    start_server

    echo ""
    echo -e "${BOLD}==============================${NC}"
    echo -e "${BOLD}  Profile Workflow Tests${NC}"
    echo -e "${BOLD}==============================${NC}"
    echo "  Server: $BASE_URL"
    echo "  User:   $EMAIL / $USERNAME"
    echo ""

    check_connectivity

    # ================================================================
    # 1. AUTH SETUP
    # ================================================================
    auth_setup "$EMAIL" "$PASSWORD" "$NAME" "$USERNAME"

    # Grab our user ID
    api GET /profiles "" false
    expect "get own profile" 200
    local MY_ID
    MY_ID=$(echo "$_API_BODY" | jq -r '.id // empty' 2>/dev/null || echo "")

    # ================================================================
    # 2. GET PROFILE (own)
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Get Own Profile ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api GET /profiles "" false
    expect "no session" 401 Unauthorized

    relogin

    api GET /profiles "" false
    expect "own profile" 200

    if [ -z "$MY_ID" ]; then
        MY_ID=$(echo "$_API_BODY" | jq -r '.id // empty' 2>/dev/null || echo "")
    fi

    # ================================================================
    # 3. GET PROFILE BY USERNAME
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Get Profile By Username ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api GET "/profiles/usernames/$USERNAME" "" false
    expect "no session" 401 Unauthorized

    relogin

    api GET "/profiles/usernames/$USERNAME" "" false
    expect "by username" 200

    api GET "/profiles/usernames/nonexistent_user_abc" "" false
    expect "non-existent username" 404 UserNotFound

    # ================================================================
    # 4. GET PROFILE BY ID
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Get Profile By ID ---${NC}"

    api GET "/profiles/id/not-a-uuid" "" false
    expect "invalid UUID" 400 InvalidRequest

    api GET "/profiles/id/00000000-0000-0000-0000-000000000000" "" false
    expect "non-existent ID" 404 UserNotFound

    if [ -n "$MY_ID" ]; then
        api GET "/profiles/id/$MY_ID" "" false
        expect "by ID" 200
    fi

    # ================================================================
    # 5. EDIT PROFILE
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Edit Profile ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api PUT /profiles '{"name":"x","username":"y"}' false
    expect "no session" 401 Unauthorized

    relogin
    api PUT /profiles '{"name":"x","username":"y"}' false
    expect "no CSRF" 401 Unauthorized

    relogin

    api PUT /profiles "{}" true
    expect "empty body" 400 InvalidData

    api PUT /profiles '{"name":"","username":"newname","bio":""}' true
    expect "empty name" 400 InvalidData

    api PUT /profiles '{"name":"New Name","username":"","bio":""}' true
    expect "empty username" 400 InvalidData

    api PUT /profiles '{"name":"New Name","username":"bad user!","bio":""}' true
    expect "invalid username chars" 400 InvalidData

    local NEW_NAME="Updated ${SUFFIX}"
    local NEW_USERNAME="proupdated_${SUFFIX}"
    api PUT /profiles "{\"name\":\"$NEW_NAME\",\"username\":\"$NEW_USERNAME\",\"bio\":\"This is my bio\"}" true
    expect "valid edit" 200

    # Verify changes via GET
    api GET /profiles "" false
    expect "verify edited profile" 200
    local got_name
    got_name=$(echo "$_API_BODY" | jq -r '.name // empty' 2>/dev/null || echo "")
    if [ "$got_name" != "$NEW_NAME" ]; then
        TESTS=$((TESTS + 1)); FAIL=$((FAIL + 1))
        echo -e "  ${RED}✗${NC} name update persisted (want $NEW_NAME, got $got_name)"
    else
        TESTS=$((TESTS + 1)); PASS=$((PASS + 1))
        echo -e "  ${GREEN}✓${NC} name update persisted"
    fi

    # Register second user and try to take their username
    local rdata2
    rdata2=$(cat <<EOF
{"email":"$EMAIL2","password":"$PASSWORD","name":"$NAME2","username":"$USERNAME2"}
EOF
)
    api POST /auth/register "$rdata2"
    expect "register second user" 200

    api PUT /profiles "{\"name\":\"$NEW_NAME\",\"username\":\"$USERNAME2\",\"bio\":\"\"}" true
    expect "duplicate username" 409 UsernameConflict

    # ================================================================
    # 6. AVATAR UPLOAD GENERATION
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Avatar Upload Generation ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api PUT /profiles/avatar '{"contentType":"image/png","fileSize":102400}' false
    expect "no session" 401 Unauthorized

    relogin
    api PUT /profiles/avatar '{"contentType":"image/png","fileSize":102400}' false
    expect "no CSRF generate" 401 Unauthorized

    relogin

    api PUT /profiles/avatar '{"contentType":"","fileSize":0}' true
    expect "invalid contentType" 400 InvalidData

    api PUT /profiles/avatar '{"contentType":"text/plain","fileSize":1024}' true
    expect "non-image contentType" 400 InvalidData

    api PUT /profiles/avatar "{\"contentType\":\"image/png\",\"fileSize\":$((3*1024*1024))}" true
    expect "file too large" 400 InvalidData

    api PUT /profiles/avatar '{"contentType":"image/png","fileSize":102400}' true
    expect "valid generate" 200

    # ================================================================
    # 7. DELETE AVATAR
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Delete Avatar ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api DELETE /profiles/avatar "" false
    expect "no session" 401 Unauthorized

    relogin
    api DELETE /profiles/avatar "" false
    expect "no CSRF delete avatar" 401 Unauthorized

    relogin

    api DELETE /profiles/avatar "" true
    expect "no avatar" 404 AvatarNotFound

    # ================================================================
    # 8. DELETE PROFILE
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Delete Profile ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api DELETE /profiles "" false
    expect "no session" 401 Unauthorized

    relogin
    api DELETE /profiles "" false
    expect "no CSRF delete profile" 401 Unauthorized

    relogin

    api DELETE /profiles "" true
    expect "delete profile" 200

    # Verify dead
    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api POST /auth/login "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" false
    expect "login after delete" 401 Unauthorized

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
