-- name: InsertVideo :exec
insert into media_service.videos (id, object_key) values ($1, $2);

-- name: GetVideoById :one
select * from media_service.videos where id = $1;

-- name: GetObjectKeyForVideo :one
select object_key from media_service.videos where id = $1 for update;

-- name: DeleteVideoById :exec
delete from media_service.videos where id = $1;

-- name: InsertAvatar :exec
insert into media_service.avatars (user_id, object_key) values ($1, $2);

-- name: GetAvatarByUserId :one
select object_key from media_service.avatars where user_id = $1;

-- name: DeleteAvatarByUserId :exec
delete from media_service.avatars where user_id = $1;
