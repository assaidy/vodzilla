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
