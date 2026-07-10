-- name: InsertView :execrows
insert into reaction_service.views (video_id, user_id) values ($1, $2)
on conflict (video_id, user_id) do nothing;

-- name: GetViewsCount :one
select count(*) from reaction_service.views where video_id = $1;

-- name: CheckVideoViewer :one
select exists (select 1 from reaction_service.views where video_id = $1 and user_id = $2 for update);

-- name: UpsertFeeling :exec
insert into reaction_service.feelings (for_id, user_id, kind) values ($1, $2, $3)
on conflict (for_id, user_id) do update set
  kind = excluded.kind,
  added_at = now();

-- name: DeleteFeeling :execrows
delete from reaction_service.feelings where for_id = $1 and user_id = $2;

-- name: GetFeelingCounts :one
select
    count(*) filter (where kind = 'like')    as likes,
    count(*) filter (where kind = 'dislike') as dislikes
from reaction_service.feelings
where for_id = $1;

-- name: GetUserFeeling :one
select kind from reaction_service.feelings
where for_id = $1 and user_id = $2;

-- name: CheckComment :one
select exists (select 1 from reaction_service.comments where id = $1 for update);

-- name: GetCommentOwner :one
select user_id from reaction_service.comments where id = $1;

-- name: InsertComment :exec
insert into reaction_service.comments (id, for_id, user_id, content) values ($1, $2, $3, $4);

-- name: CheckCommentForUser :one
select exists (select 1 from reaction_service.comments where id = $1 and user_id = $2 for update);

-- name: UpdateComment :exec
update reaction_service.comments 
set content = $1
where id = $2;

-- name: DeleteComment :exec
delete from reaction_service.comments where id = $1 and user_id = $2;

-- name: GetComments :many
select 
  c.id,
  c.user_id,
  c.content,
  c.created_at,
  count(r.id) as replies_count
from reaction_service.comments c
left join reaction_service.comments r on c.id = r.for_id
where c.for_id = $1 is null and (
  sqlc.narg(last_comment_id)::uuid is null
  or c.id < sqlc.narg(last_comment_id)::uuid
)
group by c.id, c.user_id, c.content, c.created_at
order by c.id desc
limit $2;

-- name: DeleteAllViewsForUser :exec
delete from reaction_service.views where user_id = $1;

-- name: DeleteAllFeelingsForUser :exec
delete from reaction_service.feelings where user_id = $1;

-- name: DeleteAllCommentsForUser :exec
delete from reaction_service.comments where user_id = $1;

-- name: DeleteAllViewsForVideo :exec
delete from reaction_service.views where video_id = $1;

-- name: DeleteAllFeelingsByForId :exec
delete from reaction_service.feelings where for_id = $1;

-- name: GetCommentsCount :one
select count(*) from reaction_service.comments where for_id = $1;

-- name: DeleteAllCommentsFor :exec
delete from reaction_service.comments where for_id = $1;
