-- name: CheckEmail :one
select exists (select 1 from auth.users where email = $1 for update);

-- name: InsertUser :exec
insert into auth.users (id, email, password_hash)
values ($1, $2, $3);

-- name: GetUserByEmail :one
select * from auth.users where email = $1 for update;

-- name: InsertSession :exec
insert into auth.sessions (id, owner_id, session_token, csrf_token, expires_at)
values ($1, $2, $3, $4, $5);

-- name: DeleteSessionForUser :execrows
delete from auth.sessions where id = @session_id and owner_id = @user_id;

-- name: DeleteUser :execrows
delete from auth.users where id = $1;

-- name: InsertEmailVerificationToken :exec
insert into auth.email_verification_tokens (id, owner_id, token, expires_at)
values ($1, $2, $3, $4);

-- name: VerifyEmailByToken :execrows
update auth.users
set is_verified = true
where id in (
  select owner_id
  from auth.email_verification_tokens evt
  where token = $1 and expires_at > now()
);

-- name: GetSessionByID :one
select * from auth.sessions where id = $1;
