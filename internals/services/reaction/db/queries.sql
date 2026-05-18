-- name: InsertView :execrows
insert into reaction_service.views (video_id, user_id) values ($1, $2)
on conflict (video_id, user_id) do nothing;

-- name: GetViewsCount :one
select count(*) from reaction_service.views where video_id = $1;

-- name: InsertReaction :exec
insert into reaction_service.reactions (video_id, user_id, kind) values ($1, $2, $3)
on conflict (video_id, user_id) do update set
  kind = excluded.kind,
  added_at = now();

-- name: DeleteReaction :exec
delete from reaction_service.reactions where video_id = $1 and user_id = $2 and kind = $3;

-- name: GetVideoReactions :one
SELECT
    count(*) filter (where kind = 'like')    as likes,
    count(*) filter (where kind = 'dislike') as dislikes
FROM reaction_service.reactions
where video_id = $1;

-- name: GetVideoReactionForUser :one
SELECT
    kind = 'like'    as is_like,
    kind = 'dislike' as is_dislike
FROM reaction_service.reactions
where video_id = $1 and user_id = $2;
