-- name: InsertWatchHistory :exec
insert into history_service.watch_history (user_id, video_id) values ($1, $2);

-- name: GetWatchHistory :many
select id, user_id, video_id, watched_at
from history_service.watch_history
where user_id = $1 and (
    sqlc.narg(last_id)::bigint is null
    or id < sqlc.narg(last_id)::bigint
)
order by id desc
limit $2;

-- name: DeleteWatchHistoryEntry :execrows
delete from history_service.watch_history where id = $1 and user_id = $2;

-- name: DeleteAllWatchHistoryForUser :exec
delete from history_service.watch_history where user_id = $1;

-- name: DeleteAllWatchHistoryForVideo :exec
delete from history_service.watch_history where video_id = $1;
