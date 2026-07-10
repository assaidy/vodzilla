-- name: CheckFollow :one
select exists (select 1 from social_service.follows where follower_id = $1 and followed_id = $2 for update);

-- name: InsertFollow :exec
insert into social_service.follows (follower_id, followed_id) values ($1, $2);

-- name: DeleteFollow :execrows
delete from social_service.follows where follower_id = $1 and followed_id = $2;

-- name: GetFollowCounts :one
select
    count(*) filter (where followed_id = $1) as followers_count,
    count(*) filter (where follower_id = $1) as followeds_count
from social_service.follows
where followed_id = $1 or follower_id = $1;

-- name: GetFollowerIds :many
select follower_id
from social_service.follows
where followed_id = @user_id and (
    sqlc.narg(last_user_id)::uuid is null
    or follower_id < sqlc.narg(last_user_id)::uuid
)
order by follower_id desc
limit $1;

-- name: GetFollowedIds :many
select followed_id
from social_service.follows
where follower_id = @user_id and (
    sqlc.narg(last_user_id)::uuid is null
    or followed_id < sqlc.narg(last_user_id)::uuid
)
order by followed_id desc
limit $1;

-- name: GetAllFollowedIds :many
select followed_id
from social_service.follows
where follower_id = @user_id;

-- name: GetAllFollowerIds :many
select follower_id
from social_service.follows
where followed_id = @user_id;

-- name: DeleteFollowsForUser :exec
delete from social_service.follows where follower_id = $1 or followed_id = $1;
