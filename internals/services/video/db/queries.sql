-- name: InsertVideo :exec
insert into video_service.videos (id, object_key, owner_id, title, description, status) values ($1, $2, $3, $4, $5, $6);

-- name: GetVideoByObjectKey :one
select * from video_service.videos where object_key = $1 for update;

-- name: GetVideoById :one
select * from video_service.videos where id = $1 for update;

-- name: UpdateVideoStatus :exec
update video_service.videos
set status = $1
where id = $2;
