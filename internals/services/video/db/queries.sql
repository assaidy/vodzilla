-- name: InsertVideo :exec
insert into video_service.videos (id, owner_id, title, description, status) values ($1, $2, $3, $4, $5);

-- name: GetVideoById :one
select * from video_service.videos where id = $1 for update;

-- name: UpdateVideoStatus :exec
update video_service.videos
set status = $1
where id = $2;

-- name: GetVideosForUser :many
select * from video_service.videos where owner_id = $1 and status = $2;

-- name: CheckVideo :one
select exists (select 1 from video_service.videos where id = $1 for update);

-- name: CheckWatchLater :one
select exists (select 1 from video_service.watch_later where video_id = $1 and user_id = $2 for update);

-- name: InsertIntoWatchLater :exec
insert into video_service.watch_later (video_id, user_id) values ($1, $2);

-- name: DeleteFromWatchLater :execrows
delete from video_service.watch_later where video_id = $1 and user_id = $2;

-- name: GetVideosInWatchLater :many
select v.*
from video_service.watch_later wl
join video_service.videos v on v.id = wl.video_id and v.owner_id = wl.user_id
where wl.user_id = $1
order by wl.added_at desc;
