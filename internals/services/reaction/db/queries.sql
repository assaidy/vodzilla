-- name: InsertView :execrows
insert into reaction_service.views (video_id, user_id) values ($1, $2)
on conflict (video_id, user_id) do nothing;

-- name: GetViewsCount :one
select count(*) from reaction_service.views where video_id = $1;

