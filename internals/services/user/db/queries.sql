-- name: CheckEmail :one
select exists (select 1 from user_service.users where email = $1 for update);

-- name: CheckUsername :one
select exists (select 1 from user_service.users where username = $1 for update);

-- name: InsertUser :exec
insert into user_service.users (id, email, password_hash, name, username, bio)
values ($1, $2, $3, $4, $5, $6);

-- name: GetUserByEmail :one
select * from user_service.users where email = $1 for update;

-- name: GetUserByID :one
select * from user_service.users where id = $1 for update;

-- name: GetUserByUsername :one
select * from user_service.users where username = $1 for update;

-- name: InsertSession :exec
insert into user_service.sessions (id, owner_id, session_token, csrf_token, expires_at)
values ($1, $2, $3, $4, $5);

-- name: DeleteSessionForUser :execrows
delete from user_service.sessions where id = @session_id and owner_id = @user_id;

-- name: DeleteUser :execrows
delete from user_service.users where id = $1;

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

-- name: GetSessionByID :one
select * from user_service.sessions where id = $1;

-- name: UpdateProfile :exec
update user_service.users
set
  name = $1,
  username = $2,
  bio = $3
where id = @user_id;
