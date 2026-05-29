-- name: InsertVideo :exec
insert into video_service.videos (id, owner_id, title, description, status) values ($1, $2, $3, $4, $5);

-- name: GetVideoById :one
select * from video_service.videos where id = $1 and status = $2 for update;

-- name: UpdateVideoStatus :exec
update video_service.videos
set status = $1
where id = $2;

-- name: GetAllVideosForUser :many
select * from video_service.videos where owner_id = $1 and status = $2;

-- name: CheckVideo :one
select exists (select 1 from video_service.videos where id = $1 and status = $2 for update);

-- name: CheckVideoInWatchlaters :one
select exists (select 1 from video_service.watchlaters where video_id = $1 and user_id = $2 for update);

-- name: InsertIntoWatchlaters :exec
insert into video_service.watchlaters (video_id, user_id) values ($1, $2);

-- name: DeleteFromWatchlaters :execrows
delete from video_service.watchlaters where video_id = $1 and user_id = $2;

-- name: GetAllVideosInWatchlatersForUser :many
select v.*
from video_service.watchlaters wl
join video_service.videos v on v.id = wl.video_id and v.owner_id = wl.user_id
where wl.user_id = $1
order by wl.added_at desc;

-- name: CheckPlaylistByNameForUser :one
select exists (select 1 from video_service.playlists where name = $1 and owner_id = $2 for update);

-- name: InsertPlaylist :exec
insert into video_service.playlists (id, name, owner_id) values ($1, $2, $3);

-- name: DeletePlaylist :execrows
delete from video_service.playlists where id = $1 and owner_id = $2;

-- name: CheckPlaylist :one
select exists (select 1 from video_service.playlists where id = $1 for update);

-- name: InsertIntoPlaylist :exec
insert into video_service.playlist_videos (playlist_id, video_id) values ($1, $2);

-- name: CheckPlaylistForUser :one
select exists (select 1 from video_service.playlists where id = $1 and owner_id = $2 for update);

-- name: DeleteVideoFromPlaylist :execrows
delete from video_service.playlist_videos where playlist_id = $1 and video_id = $2;

-- name: GetAllPlaylistsForUser :many
select 
  p.id,
  p.name,
  count(pv.video_id) as videos_count
from video_service.playlists p
left join video_service.playlist_videos pv on p.id = pv.playlist_id
where owner_id = $1
group by p.id
order by p.created_at desc;

-- name: GetAllVideosInPlaylist :many
select v.*
from video_service.playlist_videos pv
join video_service.videos v on v.id = pv.video_id
where pv.playlist_id = $1
order by pv.added_at desc;

-- name: CheckVideoInPlaylist :one
select exists (select 1 from video_service.playlist_videos where playlist_id = $1 and video_id = $2 for update);

-- name: GetPlaylistForUser :one
select
  p.id,
  p.name,
  count(pv.video_id) as videos_count
from video_service.playlists p
left join video_service.playlist_videos pv on p.id = pv.playlist_id
where p.id = $1 and p.owner_id = $2
group by p.id;

-- name: DeleteAllVideosForUser :many
delete from video_service.videos where owner_id = $1 returning id;

-- name: DeleteAllWatchlatersForUser :exec
delete from video_service.watchlaters where user_id= $1;

-- name: DeleteAllPlaylistsForUser :exec
delete from video_service.playlists where owner_id = $1;

-- name: DeleteVideoByIdForUser :execrows
delete from video_service.videos where id = $1 and owner_id = $2 and status = $3;

-- name: DeleteVideoById :exec
delete from video_service.videos where id = $1;
