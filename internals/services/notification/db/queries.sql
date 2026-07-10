-- name: InsertNotification :exec
insert into notification_service.notifications (id, user_id, kind, payload) values ($1, $2, $3, $4);

-- name: MarkNotificationAsRead :execrows
update notification_service.notifications
set is_read = true
where id = $1 and user_id = $2;

-- name: GetNotifications :many
select id, user_id, kind, payload, created_at, is_read
from notification_service.notifications
where user_id = @user_id and (
  sqlc.narg(last_notification_id)::uuid is null
  or id < sqlc.narg(last_notification_id)::uuid
)
order by id desc
limit $1;

-- name: GetUnreadNotificationsCount :one
select count(*) from notification_service.notifications where user_id = $1 and is_read = false;

-- name: DeleteAllNotificationsForUser :exec
delete from notification_service.notifications where user_id = $1;
