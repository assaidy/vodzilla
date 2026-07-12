#!/usr/bin/env bash
set -euo pipefail

# ==========================================================
# Social Workflow Tests
# Tests: follow, unfollow, follow counts, followers list,
# followeds list
# ==========================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

# ---- Users ----
SUFFIX=$(date +%s)
# User A — the follower (main actor)
A_EMAIL="soc_a_${SUFFIX}@example.com"
A_PASS="Password123"
A_NAME="Social A ${SUFFIX}"
A_USER="soc_a_${SUFFIX}"

# User B — the one being followed
B_EMAIL="soc_b_${SUFFIX}@example.com"
B_PASS="Password123"
B_NAME="Social B ${SUFFIX}"
B_USER="soc_b_${SUFFIX}"

COOKIE_JAR="/tmp/vodzilla_social_test_cookies.txt"
COOKIE_JAR_B="/tmp/vodzilla_social_test_cookies_b.txt"

relogin() {
    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api POST /auth/login "{\"email\":\"$A_EMAIL\",\"password\":\"$A_PASS\"}"
    if [ "$_API_STATUS" = 200 ]; then
        CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" | awk '{print $NF}')
    fi
}

main() {
    rm -f "$COOKIE_JAR" "$COOKIE_JAR_B"
    check_deps
    check_goose
    reset_db
    trap stop_server EXIT
    start_server

    echo ""
    echo -e "${BOLD}==============================${NC}"
    echo -e "${BOLD}  Social Workflow Tests${NC}"
    echo -e "${BOLD}==============================${NC}"
    echo "  Server: $BASE_URL"
    echo "  User A: $A_EMAIL / $A_USER"
    echo "  User B: $B_EMAIL / $B_USER"
    echo ""

    check_connectivity

    # ================================================================
    # SETUP
    # ================================================================
    echo -e "${BOLD}--- Setup ---${NC}"

    # Register + verify + login User A
    auth_setup "$A_EMAIL" "$A_PASS" "$A_NAME" "$A_USER"

    # Get A's ID
    api GET /profiles
    expect "get A's profile" 200
    local A_ID
    A_ID=$(echo "$_API_BODY" | jq -r '.id // empty')

    # Register User B (no verification needed for B to exist)
    local rdata_b
    rdata_b=$(cat <<EOF
{"email":"$B_EMAIL","password":"$B_PASS","name":"$B_NAME","username":"$B_USER"}
EOF
)
    api POST /auth/register "$rdata_b"
    expect "register User B" 200

    # Get B's ID via username lookup using A's session
    api GET "/profiles/usernames/$B_USER"
    expect "get B's profile by username" 200
    local B_ID
    B_ID=$(echo "$_API_BODY" | jq -r '.id // empty')

    echo "  A_ID=$A_ID"
    echo "  B_ID=$B_ID"
    echo ""

    # ================================================================
    # FOLLOW
    # ================================================================
    echo -e "${BOLD}--- Follow ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api POST "/follows/$B_ID"
    expect "no session" 401 Unauthorized

    relogin
    api POST "/follows/$B_ID" "" false
    expect "no CSRF" 401 Unauthorized

    relogin

    api POST "/follows/not-a-uuid" "" true
    expect "invalid user_id" 400 InvalidRequest

    api POST "/follows/00000000-0000-0000-0000-000000000000" "" true
    expect "non-existent user" 404 UserNotFound

    api POST "/follows/$A_ID" "" true
    expect "self-follow" 403 SelfFollowNotAllowed

    api POST "/follows/$B_ID" "" true
    expect "valid follow" 200

    api POST "/follows/$B_ID" "" true
    expect "duplicate follow" 409 AlreadyFollowing

    # ================================================================
    # FOLLOW COUNTS (after follow)
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Follow Counts ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api GET "/follows/$A_ID/counts"
    expect "no session" 401 Unauthorized

    relogin

    api GET "/follows/not-a-uuid/counts"
    expect "invalid UUID" 400 InvalidRequest

    api GET "/follows/00000000-0000-0000-0000-000000000000/counts"
    expect "non-existent user" 404 UserNotFound

    # A follows B → A's followeds = 1, B's followers = 1
    api GET "/follows/$A_ID/counts"
    expect "A's counts (after follow)" 200
    local a_followeds
    a_followeds=$(echo "$_API_BODY" | jq -r '.followeds // 0')
    if [ "$a_followeds" != "1" ]; then
        TESTS=$((TESTS + 1)); FAIL=$((FAIL + 1))
        echo -e "  ${RED}✗${NC} A's followeds = 1 (got $a_followeds)"
    else
        TESTS=$((TESTS + 1)); PASS=$((PASS + 1))
        echo -e "  ${GREEN}✓${NC} A's followeds = 1"
    fi

    api GET "/follows/$B_ID/counts"
    expect "B's counts (after follow)" 200
    local b_followers
    b_followers=$(echo "$_API_BODY" | jq -r '.followers // 0')
    if [ "$b_followers" != "1" ]; then
        TESTS=$((TESTS + 1)); FAIL=$((FAIL + 1))
        echo -e "  ${RED}✗${NC} B's followers = 1 (got $b_followers)"
    else
        TESTS=$((TESTS + 1)); PASS=$((PASS + 1))
        echo -e "  ${GREEN}✓${NC} B's followers = 1"
    fi

    # ================================================================
    # FOLLOWERS / FOLLOWEDS LISTS
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Followers / Followeds Lists ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api GET "/follows/$B_ID/followers"
    expect "followers: no session" 401 Unauthorized

    relogin

    api GET "/follows/not-a-uuid/followers"
    expect "followers: invalid UUID" 400 InvalidRequest

    api GET "/follows/00000000-0000-0000-0000-000000000000/followers"
    expect "followers: non-existent user" 404 UserNotFound

    api GET "/follows/$B_ID/followers?limit=15"
    expect "followers: valid" 200

    api GET "/follows/not-a-uuid/followeds"
    expect "followeds: invalid UUID" 400 InvalidRequest

    api GET "/follows/00000000-0000-0000-0000-000000000000/followeds"
    expect "followeds: non-existent user" 404 UserNotFound

    api GET "/follows/$A_ID/followeds?limit=15"
    expect "followeds: valid" 200

    # Pagination validation
    api GET "/follows/$B_ID/followers?limit=5"
    expect "followers: limit too small" 400 InvalidData

    api GET "/follows/$B_ID/followers?limit=200"
    expect "followers: limit too large" 400 InvalidData

    # ================================================================
    # UNFOLLOW
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Unfollow ---${NC}"

    rm -f "$COOKIE_JAR"
    CSRF_TOKEN=""
    api DELETE "/follows/$B_ID"
    expect "no session" 401 Unauthorized

    relogin
    api DELETE "/follows/$B_ID" "" false
    expect "no CSRF" 401 Unauthorized

    relogin

    api DELETE "/follows/not-a-uuid" "" true
    expect "invalid user_id" 400 InvalidRequest

    api DELETE "/follows/00000000-0000-0000-0000-000000000000" "" true
    expect "non-existent user" 404 UserNotFound

    # Already following B from the valid follow test above
    api DELETE "/follows/$B_ID" "" true
    expect "valid unfollow" 200

    api DELETE "/follows/$B_ID" "" true
    expect "unfollow again (not following)" 404 NotFollowing

    # ================================================================
    # FOLLOW COUNTS (after unfollow)
    # ================================================================
    echo ""
    echo -e "${BOLD}--- Follow Counts (after unfollow) ---${NC}"

    api GET "/follows/$A_ID/counts"
    expect "A's counts (after unfollow)" 200
    local a_followeds2
    a_followeds2=$(echo "$_API_BODY" | jq -r '.followeds // 0')
    if [ "$a_followeds2" != "0" ]; then
        TESTS=$((TESTS + 1)); FAIL=$((FAIL + 1))
        echo -e "  ${RED}✗${NC} A's followeds = 0 (got $a_followeds2)"
    else
        TESTS=$((TESTS + 1)); PASS=$((PASS + 1))
        echo -e "  ${GREEN}✓${NC} A's followeds = 0"
    fi

    api GET "/follows/$B_ID/counts"
    expect "B's counts (after unfollow)" 200
    local b_followers2
    b_followers2=$(echo "$_API_BODY" | jq -r '.followers // 0')
    if [ "$b_followers2" != "0" ]; then
        TESTS=$((TESTS + 1)); FAIL=$((FAIL + 1))
        echo -e "  ${RED}✗${NC} B's followers = 0 (got $b_followers2)"
    else
        TESTS=$((TESTS + 1)); PASS=$((PASS + 1))
        echo -e "  ${GREEN}✓${NC} B's followers = 0"
    fi

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
