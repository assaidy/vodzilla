package templates

import (
	"fmt"
	"time"

	. "github.com/assaidy/hyper/v2"
	"github.com/assaidy/lucide"
	"github.com/google/uuid"
)

// TODO: the indentation for replies is too big. use the menu
// component from daisy ui; it comes with indentation styles.
func CommentSection(videoId uuid.UUID) HyperNode {
	return DIV(AttrId("COMMENT_SECTION"), AttrClass("mt-8"))(
		H2(AttrClass("text-xl font-bold mb-4"))("Comments"),
		CreateCommentForm(CreateCommentFormParams{VideoId: videoId}),
		DIV(
			AttrId("COMMENTS_LIST"),
			AttrClass("space-y-4 mt-6"),
			Attr("hx-get", fmt.Sprintf("/videos/%s/comments", videoId)),
			Attr("hx-trigger", "load"),
			Attr("hx-swap", "innerHTML"),
		)(),
	)
}

type CreateCommentFormParams struct {
	VideoId    uuid.UUID
	ParentId   uuid.UUID
	Content    string
	ContentErr error
}

// TODO: split comment form into two forms: create comment, create reply
// TODO: the loading of replies is not correct.
// there should be a button show/hide replies whose text changes according to the replies status (showed/hidden).
// the replies list is lazy loaded when first show. that way we can hide/show comments any time and also it fixes
// that the new reply is inserted event if replies is hidden, and that create two duplicate replies if we opened(loaded)
// replies after a new reply inserted. also the number of replies (in the button) will be updated dynamically when new replies inserted.
// TODO: make dropdown menu not to appear in the bottom
func CreateCommentForm(params CreateCommentFormParams) HyperNode {
	formId := "CREATE_COMMENT_FORM"
	textareaId := "CREATE_COMMENT_TEXTAREA"
	placeholder := "Add a comment..."
	submitText := "Comment"

	if params.ParentId != uuid.Nil {
		formId = fmt.Sprintf("REPLY_FORM_INNER_%s", params.ParentId)
		textareaId = fmt.Sprintf("REPLY_TEXTAREA_%s", params.ParentId)
		placeholder = "Write a reply..."
		submitText = "Reply"
	}

	return Group(
		FORM(
			AttrId(formId),
			AttrClass(Classes("flex gap-2 items-start", IfElseZero(params.ParentId != uuid.Nil, "hidden"))),
			Attr("hx-post", fmt.Sprintf("/videos/%s/comments", params.VideoId)),
			Attr("hx-swap", "outerHTML"),
			Attr("hx-indicator", "find .submit-btn"),
		)(
			If(params.ParentId == uuid.Nil,
				commentOwnerAvatarPlaceholder(),
			),
			DIV(AttrClass("flex-1 flex gap-2"))(
				TEXTAREA(
					AttrId(textareaId),
					AttrClass("textarea textarea-bordered w-full resize-none"),
					AttrName("content"),
					AttrPlaceholder(placeholder),
					AttrRequired(true),
					AttrMaxLength("500"),
				)(params.Content),
				If(params.ContentErr != nil,
					LABEL(AttrClass("label"))(
						SPAN(AttrClass("label-text-alt text-error"))(params.ContentErr),
					),
				),
				If(params.ParentId != uuid.Nil,
					INPUT(AttrType(TypeHidden), AttrName("parent_id"), AttrValue(params.ParentId.String())),
				),
				BUTTON(
					AttrClass("btn btn-primary btn-sm submit-btn"),
					AttrType(TypeSubmit),
				)(submitText),
			),
		),
	)
}

type CommentParams struct {
	VideoId       uuid.UUID
	CommentId     uuid.UUID
	OwnerName     string
	OwnerUsername string
	Content       string
	CreatedAt     time.Time
	RepliesCount  int
	IsOwner       bool
}

func Comment(params CommentParams) HyperNode {
	commentId := params.CommentId
	videoId := params.VideoId

	ownerProfilePageLink := fmt.Sprintf("/@%s", params.OwnerUsername)
	visitProfileAttrs := []Attribute{
		Attr("hx-get", fmt.Sprintf("%s/content", ownerProfilePageLink)),
		Attr("hx-push-url", ownerProfilePageLink),
		Attr("hx-target", "#APP_PAGE_CONTENT"),
		Attr("hx-swap", "innerHTML"),
		Attr("hx-trigger", "click consume"),
		Attr("hx-indicator", "#PAGE_CONTENT_CONTAINER"),
	}

	return DIV(
		AttrId(fmt.Sprintf("COMMENT_%s", commentId)),
		AttrClass("flex gap-2"),
	)(
		DIV(append(visitProfileAttrs, AttrClass("shrink-0 cursor-pointer"))...)(
			commentOwnerAvatarPlaceholder(),
		),

		DIV(AttrClass("flex-1 min-w-0"))(
			DIV(AttrClass("flex items-center gap-2 text-sm"))(
				A(append(visitProfileAttrs, AttrClass("link link-hover font-semibold"))...)(
					params.OwnerName,
				),
				SPAN(AttrClass("text-base-content/40 text-xs"))(normalizeTimestamp(params.CreatedAt)),
			),

			DIV(AttrId(fmt.Sprintf("COMMENT_BODY_%s", commentId)))(
				DIV(AttrId(fmt.Sprintf("COMMENT_CONTENT_%s", commentId)))(
					P(AttrClass("text-sm"))(params.Content),
				),
				If(params.IsOwner,
					EditCommentForm(EditCommentFormParams{
						VideoId:   videoId,
						CommentId: commentId,
						Content:   params.Content,
						Hide:      true,
					}),
				),
			),

			DIV(AttrClass("flex items-center gap-1 mt-1"))(
				BUTTON(
					AttrClass("btn btn-ghost btn-xs"),
					AttrOnClick(fmt.Sprintf("REPLY_FORM_%s.classList.toggle('hidden')", commentId)),
				)("Reply"),

				If(params.RepliesCount > 0,
					BUTTON(
						AttrClass("btn btn-soft btn-xs"),
						Attr("hx-get", fmt.Sprintf("/videos/%s/comments/%s/replies", videoId, commentId)),
						Attr("hx-target", fmt.Sprintf("#REPLIES_%s", commentId)),
						Attr("hx-swap", "append"),
						Attr("hx-on::after:request", "this.remove()"),
					)(
						fmt.Sprintf("Show replies (%d)", params.RepliesCount),
					),
				),

				If(params.IsOwner,
					DIV(AttrClass("dropdown dropdown-end"))(
						BUTTON(
							AttrClass("btn btn-ghost btn-xs btn-circle"),
							Attr("tabindex", "0"),
						)(
							RawText(lucide.EllipsisVertical()),
						),
						UL(
							AttrClass("dropdown-content menu p-2 shadow bg-base-100 rounded-box z-[1]"),
							Attr("tabindex", "0"),
						)(
							LI()(
								A(
									AttrOnClick(fmt.Sprintf(`
										COMMENT_CONTENT_%[1]s.classList.add('hidden');
										EDIT_FORM_%[1]s.classList.remove('hidden');
									`,
										commentId,
									)),
								)(
									"Edit",
								),
							),
							LI()(
								A(
									Attr("hx-delete", fmt.Sprintf("/videos/%s/comments/%s", videoId, commentId)),
									Attr("hx-target", fmt.Sprintf("#COMMENT_%s", commentId)),
									Attr("hx-swap", "delete"),
									Attr("hx-confirm", "Are you sure?"),
								)(
									"Delete",
								),
							),
						),
					),
				),
			),

			DIV(
				AttrId(fmt.Sprintf("REPLY_FORM_%s", commentId)),
				AttrClass("hidden mt-2"),
			)(
				CreateCommentForm(CreateCommentFormParams{VideoId: videoId, ParentId: commentId}),
			),

			DIV(
				AttrId(fmt.Sprintf("REPLIES_%s", commentId)),
				AttrClass("ml-8 border-l-2 border-base-300 pl-4 mt-2 space-y-3"),
			)(),
		),
	)
}

type EditCommentFormParams struct {
	VideoId    uuid.UUID
	CommentId  uuid.UUID
	Content    string
	ContentErr error
	Hide       bool
}

func EditCommentForm(params EditCommentFormParams) HyperNode {
	return FORM(
		AttrId(fmt.Sprintf("EDIT_FORM_%s", params.CommentId)),
		AttrClass(IfElseZero(params.Hide, "hidden")),
		Attr("hx-put", fmt.Sprintf("/videos/%s/comments/%s", params.VideoId, params.CommentId)),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .submit-btn"),
	)(
		TEXTAREA(
			AttrClass("textarea textarea-bordered w-full resize-none"),
			AttrName("content"),
			AttrRequired(true),
			AttrMaxLength("500"),
		)(
			params.Content,
		),
		If(params.ContentErr != nil,
			LABEL(AttrClass("label"))(
				SPAN(AttrClass("label-text-alt text-error"))(params.ContentErr),
			),
		),
		DIV(AttrClass("flex gap-2 mt-1"))(
			BUTTON(AttrClass("btn btn-primary btn-sm submit-btn"), AttrType(TypeSubmit))("Save"),
			BUTTON(
				AttrClass("btn btn-ghost btn-sm"),
				AttrType(TypeButton),
				AttrOnClick(fmt.Sprintf(`
					COMMENT_CONTENT_%[1]s.classList.remove('hidden');
					EDIT_FORM_%[1]s.classList.add('hidden');
				`, params.CommentId)),
			)(
				"Cancel",
			),
		),
	)
}

func commentOwnerAvatarPlaceholder() HyperNode {
	return DIV(AttrClass("avatar placeholder shrink-0"))(
		DIV(AttrClass("bg-neutral text-neutral-content rounded-full w-8 h-8 flex items-center justify-center text-xs"))(
			RawText(lucide.User()),
		),
	)
}
