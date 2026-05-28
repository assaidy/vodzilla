-- name: CheckEmail :one
select exists (select 1 from user_service.users where email = $1 and is_deleted = false for update);

-- name: CheckUsername :one
-- Don't check for is_deleted. Username can only be aquired once. 
-- We don't need unexpected profiles when navigating a url.
select exists (select 1 from user_service.users where username = $1 for update);

-- name: InsertUser :exec
insert into user_service.users (id, email, password_hash, name, username, bio)
values ($1, $2, $3, $4, $5, $6);

-- name: GetUserByEmail :one
select * from user_service.users where email = $1 and is_deleted = false for update;

-- name: GetUserById :one
select * from user_service.users where id = $1 and is_deleted = false for update;

-- name: GetUserByUsername :one
select * from user_service.users where username = $1 and is_deleted = false for update;

-- name: InsertSession :exec
insert into user_service.sessions (id, owner_id, session_token, csrf_token, expires_at)
values ($1, $2, $3, $4, $5);

-- name: DeleteSessionForUser :execrows
delete from user_service.sessions where id = @session_id and owner_id = @user_id;

-- name: InsertEmailVerificationToken :exec
insert into user_service.email_verification_tokens (id, owner_id, token, expires_at)
values ($1, $2, $3, $4);

-- name: VerifyEmailByToken :execrows
update user_service.users
set is_verified = true
where id in (
  select owner_id
  from user_service.email_verification_tokens evt
  where token = $1 and expires_at > now()
);

-- name: GetSessionById :one
select * from user_service.sessions where id = $1;

-- name: UpdateProfile :exec
update user_service.users
set
  name = $1,
  username = $2,
  bio = $3
where id = @user_id;

-- name: BatchDeleteExpiredEmailVerificationTokens :exec
do $$
declare
  rows_deleted int;
begin
  loop
    delete from user_service.email_verification_tokens
    where ctid in (
      select ctid
      from user_service.email_verification_tokens
      where expires_at <= now()
      limit 1000
    );
    get diagnostics rows_deleted = row_count;
    exit when rows_deleted = 0;
  end loop;
end
$$;

-- name: BatchDeleteExpiredSessions :exec
do $$
declare
  rows_deleted int;
begin
  loop
    delete from user_service.sessions
    where ctid in (
      select ctid
      from user_service.sessions
      where expires_at <= now()
      limit 1000
    );
    get diagnostics rows_deleted = row_count;
    exit when rows_deleted = 0;
  end loop;
end
$$;

-- name: SoftDeleteUserById :execrows
update user_service.users set is_deleted = true where id = $1;

-- name: DeleteAllSessionsForUser :exec
delete from user_service.sessions where owner_id = $1 and expires_at > now();

-- name: DeleteAllEmailVerificationTokensForUser :exec
delete from user_service.email_verification_tokens where owner_id = $1 and expires_at > now();
