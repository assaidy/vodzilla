-- name: CheckFollow :one
select exists (select 1 from social_service.follows where follower_id = $1 and followed_id = $2 for update);

-- name: InsertFollow :exec
insert into social_service.follows (follower_id, followed_id) values ($1, $2);

-- name: DeleteFollow :execrows
delete from social_service.follows where follower_id = $1 and followed_id = $2;

-- name: GetFollowersCount :one
select count(*) as followers_count
from social_service.follows
where followed_id = $1;

-- name: DeleteFollowsForUser :exec
delete from social_service.follows where follower_id = $1 or followed_id = $1;
