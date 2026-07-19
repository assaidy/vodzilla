-- name: GetVideoOwner :one
select user_id from video_service.videos where id = $1;

-- name: InsertVideo :exec
insert into video_service.videos (id, user_id, title, description) values ($1, $2, $3, $4);

-- name: GetVideoById :one
select * from video_service.videos where id = $1 for update;

-- name: GetVideosForUser :many
select * from video_service.videos
where user_id = sqlc.arg(user_id) and (
    sqlc.narg(last_video_id)::uuid is null
    or id < sqlc.narg(last_video_id)::uuid
)
order by id desc
limit $1;

-- name: GetVideosForMultipleUsers :many
select * from video_service.videos
where user_id = any (sqlc.arg(user_ids)::uuid[]) and (
    sqlc.narg(last_video_id)::uuid is null
    or id < sqlc.narg(last_video_id)::uuid
)
order by id desc
limit $1;

-- name: GetVideosCountForUser :one
select count(*) from video_service.videos where user_id = $1;

-- name: CheckVideo :one
select exists (select 1 from video_service.videos where id = $1 for update);

-- name: CheckVideoInWatchlaters :one
select exists (select 1 from video_service.watchlaters where video_id = $1 and user_id = $2 for update);

-- name: InsertIntoWatchlaters :exec
insert into video_service.watchlaters (video_id, user_id) values ($1, $2);

-- name: DeleteFromWatchlaters :execrows
delete from video_service.watchlaters where video_id = $1 and user_id = $2;

-- name: GetVideosInWatchlaters :many
select v.*, wl.id as watchlater_id
from video_service.watchlaters wl
join video_service.videos v on v.id = wl.video_id and v.user_id = wl.user_id
where wl.user_id = $1 and (
    sqlc.narg(last_watchlater_id)::bigint is null
    or wl.id < sqlc.narg(last_watchlater_id)::bigint
)
order by wl.id desc
limit $2;

-- name: InsertPlaylist :exec
insert into video_service.playlists (id, name, user_id, description, is_public) values ($1, $2, $3, $4, $5);

-- name: DeletePlaylist :execrows
delete from video_service.playlists where id = $1 and user_id = $2;

-- name: UpdatePlaylist :execrows
update video_service.playlists set name = $1, description = $2, is_public = $3 where id = $4 and user_id = $5;

-- name: CheckPlaylistForUser :one
select exists (select 1 from video_service.playlists where id = $1 and user_id = $2 for update);

-- name: InsertIntoPlaylist :exec
insert into video_service.playlist_videos (playlist_id, video_id)
values ($1, $2);

-- name: DeleteVideoFromPlaylist :execrows
delete from video_service.playlist_videos where playlist_id = $1 and video_id = $2;

-- name: GetPlaylistsForUser :many
select 
  p.id,
  p.name,
  p.user_id,
  p.description,
  p.is_public,
  count(pv.video_id) as videos_count
from video_service.playlists p
left join video_service.playlist_videos pv on p.id = pv.playlist_id
where user_id = sqlc.arg(user_id) and (
  p.is_public = true or sqlc.arg(include_privates)::boolean = true
) and (
  sqlc.narg(last_playlist_id)::uuid is null
  or p.id < sqlc.narg(last_playlist_id)::uuid
)
group by p.id
order by p.id desc
limit $1;

-- name: GetVideosInPlaylist :many
select v.*, pv.id as playlist_video_id
from video_service.playlist_videos pv
join video_service.videos v on v.id = pv.video_id
where pv.playlist_id = $1 and (
  sqlc.narg(last_id)::bigint is null
  or pv.id < sqlc.narg(last_id)::bigint
)
order by pv.id desc
limit $2;

-- name: CheckVideoInPlaylist :one
select exists (
  select 1 from video_service.playlist_videos
  where playlist_id = $1 and video_id = $2 for update
);

-- name: GetPlaylist :one
select
  p.id,
  p.user_id,
  p.name,
  p.description,
  p.is_public,
  count(pv.video_id) as videos_count
from video_service.playlists p
left join video_service.playlist_videos pv on p.id = pv.playlist_id
where p.id = $1
group by p.id;

-- name: DeleteAllVideosForUser :many
delete from video_service.videos where user_id = $1 returning id;

-- name: DeleteAllWatchlatersForUser :exec
delete from video_service.watchlaters where user_id= $1;

-- name: DeleteAllPlaylistsForUser :exec
delete from video_service.playlists where user_id = $1;

-- name: DeleteVideoByIdForUser :execrows
delete from video_service.videos where id = $1 and user_id = $2;

-- name: GetPlaylistsWithVideoStatusForUser :many
select 
  p.id,
  p.user_id,
  p.name,
  p.description,
  p.is_public,
  count(pv.video_id) as videos_count,
  exists (
    select 1 from video_service.playlist_videos pv2
    where pv2.playlist_id = p.id and pv2.video_id = sqlc.arg(video_id)
  ) as has_video
from video_service.playlists p
left join video_service.playlist_videos pv on p.id = pv.playlist_id
where p.user_id = sqlc.arg(user_id) and (
  p.is_public = true or sqlc.arg(include_privates)::boolean = true
) and (
  sqlc.narg(last_playlist_id)::uuid is null
  or p.id < sqlc.narg(last_playlist_id)::uuid
)
group by p.id
order by p.id desc
limit $1;

-- name: DeleteVideoById :exec
delete from video_service.videos where id = $1;

-- name: InsertPendingVideo :exec
insert into video_service.pending_videos (id, user_id, title, description) values ($1, $2, $3, $4);

-- name: GetPendingVideoById :one
select * from video_service.pending_videos where id = $1 for update;

-- name: DeletePendingVideoById :execrows
delete from video_service.pending_videos where id = $1;

-- name: DeleteExpiredPendingVideos :exec
delete from video_service.pending_videos where created_at < now() - '24 hours'::interval;

-- name: DeleteAllPendingVideosForUser :execrows
delete from video_service.pending_videos where user_id = $1;
