-- name: InsertVideo :exec
insert into media_service.videos (id, object_key, status) values ($1, $2, $3);

-- name: GetObjectKeyForVideo :one
select object_key from media_service.videos where id = $1 for update;

-- name: DeleteVideoById :exec
delete from media_service.videos where id = $1;

-- name: InsertUpload :exec
insert into media_service.uploads (id, video_id, expires_at) values ($1, $2, $3);

-- name: MarkUploadAsCompleted :exec
update media_service.uploads set completed_at = now() where video_id = $1;

-- name: UpdateVideoStatus :exec
update media_service.videos set status = $1 where id = $2;

-- name: GetUploadForVideo :one
select * from media_service.uploads where video_id = $1 for update;
