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
select
    count(*) filter (where kind = 'like')    as likes,
    count(*) filter (where kind = 'dislike') as dislikes
from reaction_service.reactions
where video_id = $1;

-- name: GetVideoReactionForUser :one
select
    kind = 'like'    as is_like,
    kind = 'dislike' as is_dislike
from reaction_service.reactions
where video_id = $1 and user_id = $2;

-- name: CheckComment :one
select exists (select 1 from reaction_service.comments where id = $1 for update);

-- name: InsertComment :exec
insert into reaction_service.comments (id, owner_id, video_id, content, parent_id) values ($1, $2, $3, $4, $5);

-- name: CheckCommentForUser :one
select exists (select 1 from reaction_service.comments where id = $1 and owner_id = $2 for update);

-- name: UpdateComment :exec
update reaction_service.comments 
set content = $1
where id = $2;

-- name: DeleteComment :execrows
delete from reaction_service.comments where id = $1 and owner_id = $2;

-- name: GetAllVideoComments :many
select 
  c.id,
  c.owner_id,
  c.content,
  c.created_at,
  count(r.id) as replies_count
from reaction_service.comments c
left join reaction_service.comments r on c.id = r.parent_id
where c.video_id = $1 and c.parent_id is null
group by c.id, c.owner_id, c.content, c.created_at
order by c.created_at desc;

-- name: GetAllCommentReplies :many
select 
  c.id,
  c.owner_id,
  c.content,
  c.created_at,
  count(r.id) as replies_count
from reaction_service.comments c
left join reaction_service.comments r on c.id = r.parent_id
where c.parent_id = @comment_id::varchar
group by c.id, c.owner_id, c.content, c.created_at
order by c.created_at asc;

-- name: DeleteAllViewsForUser :exec
delete from reaction_service.views where user_id = $1;

-- name: DeleteAllReactionsForUser :exec
delete from reaction_service.reactions where user_id = $1;

-- name: DeleteAllCommentsForUser :exec
delete from reaction_service.comments where owner_id = $1;

-- name: DeleteAllViewsForVideo :exec
delete from reaction_service.views where video_id = $1;

-- name: DeleteAllReactionsForVideo :exec
delete from reaction_service.reactions where video_id = $1;

-- name: DeleteAllCommentsForVideo :exec
delete from reaction_service.comments where video_id = $1;
