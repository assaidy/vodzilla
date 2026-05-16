-- name: InsertObjectKey :exec
insert into media_service.object_keys (video_id, object_key) values ($1, $2);

-- name: GetObjectKeyForVideo :one
select object_key from media_service.object_keys where video_id = $1;
