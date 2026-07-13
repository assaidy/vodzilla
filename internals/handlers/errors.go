package handlers

import (
	"errors"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
)

type apiError struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Details     any    `json:"details,omitempty"`
	statusCode  int
}

func newApiError(kind, description string, statusCode int) apiError {
	return apiError{
		Kind:        kind,
		Description: description,
		statusCode:  statusCode,
	}
}

func (me apiError) details(d any) apiError {
	me.Details = d
	return me
}

func (me apiError) Error() string {
	return fmt.Sprintf("%s: %v", me.Kind, me.Details)
}

var (
	errInvalidRequest          = newApiError("InvalidRequest", "The request contains malformed or invalid data.", fiber.StatusBadRequest)
	errInvalidData             = newApiError("InvalidData", "The request data fails validation rules.", fiber.StatusBadRequest)
	errUserNotFound            = newApiError("UserNotFound", "User not found.", fiber.StatusNotFound)
	errVideoNotFound           = newApiError("VideoNotFound", "Video not found.", fiber.StatusNotFound)
	errTokenNotFound           = newApiError("TokenNotFound", "Token not found.", fiber.StatusNotFound)
	errNotFollowing            = newApiError("NotFollowing", "Not following.", fiber.StatusNotFound)
	errAvatarNotFound          = newApiError("AvatarNotFound", "Avatar not found.", fiber.StatusNotFound)
	errWatchlaterVideoNotFound = newApiError("WatchlaterVideoNotFound", "Video not found in watchlaters.", fiber.StatusNotFound)
	errPlaylistNotFound        = newApiError("PlaylistNotFound", "Playlist not found.", fiber.StatusNotFound)
	errUsernameConflict        = newApiError("UsernameConflict", "Username already exists.", fiber.StatusConflict)
	errEmailConflict           = newApiError("EmailConflict", "Email already exists.", fiber.StatusConflict)
	errInternalFailure         = newApiError("InternalFailure", "An unexpected internal error occurred while processing the request.", fiber.StatusInternalServerError)
	errUnauthorized            = newApiError("Unauthorized", "Authentication is required or the provided credentials are invalid.", fiber.StatusUnauthorized)
	errEmailNotVerified        = newApiError("EmailNotVerified", "Email address has not been verified.", fiber.StatusForbidden)
	errInvalidCursor           = newApiError("InvalidCursor", "The provided pagination cursor is malformed or invalid.", fiber.StatusBadRequest)
	errInvalidEndpoint         = newApiError("InvalidEndpoint", "The requested API endpoint does not exist or is malformed.", fiber.StatusNotFound)
	errMethodNotAllowed        = newApiError("MethodNotAllowed", "The requested HTTP method is not allowed for this endpoint.", fiber.StatusMethodNotAllowed)
	errUpgradeRequired         = newApiError("UpgradeRequired", "Websocket upgrade is required for this endpoint.", fiber.StatusUpgradeRequired)
	errSelfFollowNotAllowed    = newApiError("SelfFollowNotAllowed", "Users cannot follow themselves.", fiber.StatusForbidden)
	errAlreadyFollowing        = newApiError("AlreadyFollowing", "The authenticated user is already following the specified user.", fiber.StatusConflict)
	errWatchlaterConflict      = newApiError("WatchlaterConflict", "The video is already in watchlaters", fiber.StatusConflict)
	errPlaylistNameConflict    = newApiError("PlaylistNameConflict", "A playlist with that name already exists.", fiber.StatusConflict)
	errPlaylistVideoConflict   = newApiError("PlaylistVideoConflict", "The video is already in the playlist.", fiber.StatusConflict)
	errPlaylistVideoNotFound   = newApiError("PlaylistVideoNotFound", "Video not found in playlist.", fiber.StatusNotFound)
	errCommentNotFound         = newApiError("CommentNotFound", "Comment not found.", fiber.StatusNotFound)
	errFeelingNotFound         = newApiError("FeelingNotFound", "Feeling not found.", fiber.StatusNotFound)
	errThumbnailNotFound       = newApiError("ThumbnailNotFound", "Thumbnail not found.", fiber.StatusNotFound)
	errObjectNotFound          = newApiError("ObjectNotFound", "Object not found.", fiber.StatusNotFound)
	errNotificationNotFound    = newApiError("NotificationNotFound", "Notification not found.", fiber.StatusNotFound)
)

func extractValidationError(err error) error {
	if err == nil {
		return nil
	}

	if ve, ok := errors.AsType[validation.Errors](err); ok {
		newM := make(validation.Errors, len(ve))
		for k, v := range ve {
			b := []byte(k)
			if 'A' <= b[0] && b[0] <= 'Z' {
				b[0] += 'a' - 'A'
			}
			newM[string(b)] = v
		}

		return errInvalidData.details(newM)
	}

	return err
}

func (me *Handler) WithErrorResolver(c fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	var apiErr apiError

	if fe, ok := errors.AsType[*fiber.Error](err); ok {
		// Catch errors returned by fiber's router.
		switch fe.Code {
		case fiber.StatusNotFound:
			apiErr = errInvalidEndpoint
			err = errInvalidEndpoint
		case fiber.StatusMethodNotAllowed:
			apiErr = errMethodNotAllowed
			err = errMethodNotAllowed
		default:
			me.logger.Warn("unhandled fiber error", "error", err)
			apiErr = errInternalFailure
		}
	} else if ae, ok := errors.AsType[apiError](err); ok {
		apiErr = ae
	} else {
		// Catch all untyped errors; all internal errors are handled here.
		// Notice I don't return internal error details to the client.
		apiErr = errInternalFailure
	}

	if writeErr := c.Status(apiErr.statusCode).JSON(apiErr); writeErr != nil {
		// Log write error without passing it to logger middleware;
		// we only want [WithLogging] to log the handler error.
		me.logger.Error("failed to write error response", "error", writeErr)
	}

	// Return original error for logging.
	return err
}
