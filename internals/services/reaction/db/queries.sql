-- name: InsertView :execrows
insert into reaction_service.views (target_id, user_id, kind) values ($1, $2, $3)
on conflict (target_id, user_id, kind) do nothing;

-- name: GetViewsCount :one
select count(*) from reaction_service.views where target_id = $1 and kind = $2;

-- name: UpsertFeeling :exec
insert into reaction_service.feelings (target_id, user_id, target_kind, kind) values ($1, $2, $3, $4)
on conflict (target_id, user_id, target_kind) do update set
  kind = excluded.kind,
  added_at = now();

-- name: DeleteFeeling :execrows
delete from reaction_service.feelings where target_id = $1 and user_id = $2 and target_kind = $3;

-- name: GetFeelingCounts :one
select
    count(*) filter (where kind = 'like')    as likes,
    count(*) filter (where kind = 'dislike') as dislikes
from reaction_service.feelings
where target_id = $1 and target_kind = $2;

-- name: GetUserFeeling :one
select kind from reaction_service.feelings
where target_id = $1 and user_id = $2 and target_kind = $3;

-- name: CheckComment :one
select exists (select 1 from reaction_service.comments where id = $1 for update);

-- name: GetCommentById :one
select id, user_id, content, created_at from reaction_service.comments where id = $1;

-- name: InsertComment :exec
insert into reaction_service.comments (id, target_id, target_kind, user_id, content) values ($1, $2, $3, $4, $5);

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
left join reaction_service.comments r on c.id = r.target_id
where c.target_id = $1 and (
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

-- name: DeleteAllViewsForTarget :exec
delete from reaction_service.views where target_id = $1 and kind = $2;

-- name: DeleteAllFeelingsForTarget :exec
delete from reaction_service.feelings where target_id = $1 and target_kind = $2;

-- name: GetCommentsCount :one
select count(*) from reaction_service.comments where target_id = $1;

-- name: DeleteAllCommentsForTarget :exec
delete from reaction_service.comments where target_id = $1;
