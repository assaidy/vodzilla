-- name: InsertVideo :exec
insert into video_service.videos (id, object_key, owner_id, title, description) values ($1, $2, $3, $4, $5);
