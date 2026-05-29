-- name: InsertObjectKey :exec
insert into media_service.object_keys (video_id, object_key) values ($1, $2);

-- name: GetObjectKeyForVideo :one
select object_key from media_service.object_keys where video_id = $1;

-- name: DeleteObjectKeyForVideo :exec
delete from media_service.object_keys where video_id = $1;

-- name: InsertUpload :exec
insert into media_service.uploads (id, object_key, expires_at) values ($1, $2, $3);

-- name: DeleteUpload :exec
delete from media_service.uploads where id = $1;

-- name: GetUploadIdForObject :one
select id from media_service.uploads where object_key = $1 and expires_at > now();

-- name: DeleteExpiredUploads :many
delete from media_service.uploads where expires_at <= now() returning object_key;

-- name: DeleteObjectKeysInList :many
delete from media_service.object_keys where object_key = any (@object_keys::varchar[]) returning video_id;


