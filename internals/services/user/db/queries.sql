-- name: CheckEmail :one
select exists (select 1 from user_service.users where email = $1 for update);

-- name: CheckUsername :one
select exists (select 1 from user_service.users where username = $1 for update);

-- name: CheckRetiredUsername :one
select exists (select 1 from user_service.retired_usernames where username = $1 for update);

-- name: CheckRetiredUsernameExcludingUser :one
select exists (select 1 from user_service.retired_usernames where username = $1 and user_id != $2 for update);

-- name: InsertUser :exec
insert into user_service.users (id, email, password_hash, name, username, bio)
values ($1, $2, $3, $4, $5, $6);

-- name: GetUserByEmail :one
select * from user_service.users where email = $1 for update;

-- name: GetUserById :one
select * from user_service.users where id = $1 for update;

-- name: GetUserByUsername :one
select * from user_service.users where username = $1 for update;

-- name: GetUsernameById :one
select username from user_service.users where id = $1 for update;

-- name: InsertSession :exec
insert into user_service.sessions (id, user_id, session_token, csrf_token, expires_at)
values ($1, $2, $3, $4, $5);

-- name: DeleteSessionForUser :execrows
delete from user_service.sessions where id = sqlc.arg(session_id) and user_id = sqlc.arg(user_id);

-- name: InsertEmailVerificationToken :exec
insert into user_service.email_verification_tokens (id, user_id, token, expires_at)
values ($1, $2, $3, $4);

-- name: VerifyEmailByToken :execrows
update user_service.users
set is_email_verified = true
where id in (
  select user_id
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
where id = sqlc.arg(user_id);

-- name: UpdatePassword :exec
update user_service.users set password_hash = $1 where id = sqlc.arg(user_id);


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

-- name: InsertRetiredUsername :exec
insert into user_service.retired_usernames (username, user_id) values ($1, $2)
on conflict (username) do nothing;

-- name: DeleteUser :exec
delete from user_service.users where id = $1;

-- name: DeleteAllSessionsForUser :exec
delete from user_service.sessions where user_id = $1 and expires_at > now();

-- name: DeleteAllEmailVerificationTokensForUser :exec
delete from user_service.email_verification_tokens where user_id = $1 and expires_at > now();

-- name: SearchUsers :many
with q as (
    select websearch_to_tsquery('english', sqlc.arg(query)) as query
),
results as (
    select
        u.*,
        ts_rank(u.search_vector, q.query) as rank
    from user_service.users u
    cross join q
    where u.search_vector @@ q.query
    limit 1000 -- cap results to 1000
)
select *
from results
where sqlc.narg(last_rank)::real is null or sqlc.narg(last_user_id)::uuid is null
   or (rank, id) < (sqlc.narg(last_rank)::real, sqlc.narg(last_user_id)::uuid)
order by rank desc, id desc
limit $1;
