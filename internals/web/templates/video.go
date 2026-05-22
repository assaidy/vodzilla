package templates

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/assaidy/hyper/v2"
	"github.com/oklog/ulid/v2"
)

type PostVideoFormParams struct {
	Title            string
	TitleErr         error
	Description      string
	DescriptionErr   error
	VideoErr         error
	CloseDialogModal bool
}

func PostVideoForm(params ...PostVideoFormParams) HyperNode {
	var p PostVideoFormParams
	if len(params) > 0 {
		p = params[0]
	}

	pendingVideoId := ulid.Make()

	return FORM(
		AttrId("POST_VIDEO_FORM"),
		AttrClass("space-y-4"),
		Attr("hx-post", "/videos"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .submit-button"),
		Attr("hx-disable", "find .submit-button"),
		Attr("hx-vals", fmt.Sprintf(`js:{
			contentType:    FORM_VIDEO.files[0].type,
			fileSize:       FORM_VIDEO.files[0].size,
			pendingVideoId: %q,
		}`, pendingVideoId)),
		Attr("hx-on::before:request", fmt.Sprintf(`
			window._pendingVideos[%q] = FORM_VIDEO.files[0];
		`, pendingVideoId)),
		Attr("hx-on::after:request", fmt.Sprintf(`
			if (event.detail.ctx.response.status >= 400) delete window._pendingVideos[%q];
		`, pendingVideoId)),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_TITLE"))(
				SPAN(AttrClass("label-text"))("Title"),
			),
			INPUT(
				AttrId("FORM_TITLE"),
				AttrClass(IfElse(p.TitleErr == nil, "input w-full", "input input-error w-full")),
				AttrType(TypeText),
				AttrName("title"),
				AttrValue(p.Title),
				AttrRequired(true),
			),
			If(p.TitleErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.TitleErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_DESCRIPTION"))(
				SPAN(AttrClass("label-text"))("Description"),
			),
			TEXTAREA(
				AttrId("FORM_DESCRIPTION"),
				AttrClass("block w-full textarea"+IfElseZero(p.DescriptionErr != nil, " textarea-error")),
				AttrName("description"),
			)(
				p.Description,
			),
			If(p.DescriptionErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.DescriptionErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_VIDEO"))(
				SPAN(AttrClass("label-text"))("Video"),
			),
			INPUT(
				// doesn't have a name, so htmx will not send it with the form
				AttrId("FORM_VIDEO"),
				AttrClass("file-input w-full "+IfElseZero(p.VideoErr != nil, "file-input-error")),
				AttrType(TypeFile),
				AttrAccept("video/*"),
				AttrRequired(true),
			),
			If(p.VideoErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.VideoErr),
				),
			),
		),

		DIV(AttrClass("pt-2"))(
			BUTTON(
				AttrClass("btn btn-primary w-full submit-button group"),
				AttrType(TypeSubmit),
			)(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Post"),
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block"))(),
			),
		),

		If(p.CloseDialogModal,
			SCRIPT()(RawText(`
				POST_VIDEO_MODAL.close();
				document.currentScript.remove();
			`)),
		),
	)
}

type VideoUploaderParams struct {
	PendingVideoId string
	VideoTitle     string
	VideoId        string
	UploadId       string
	PartSize       int64
	UploadUrls     []string
}

func VideoUploader(params VideoUploaderParams) HyperNode {
	encodedUrls, _ := json.Marshal(params.UploadUrls)

	return SCRIPT()(RawText(fmt.Sprintf(`
		(async () => {
			const script = document.currentScript;
			const pendingVideoId = %q;
			const videoTitle     = %q;
			const videoId        = %q;
			const uploadId       = %q;
			const partSize       = %d;
			const uploadUrls     = %s;

			try {
				await window._videoUploadManager.upload({
					pendingVideoId,
					videoTitle,
					videoId,
					uploadId,
					partSize,
					uploadUrls,
					completeUploadUrl: "/videos/complete_upload",
				});
			} catch (err) {
				console.error(err);
				window._videoUploadManager.removeUpload(pendingVideoId);
			} finally {
				script.remove();
			}
		})();
	`,
		params.PendingVideoId,
		params.VideoTitle,
		params.VideoId,
		params.UploadId,
		params.PartSize,
		string(encodedUrls),
	)))
}

func videoUploadersContainer() HyperNode {
	return DIV(AttrId("VIDEO_UPLOADERS_CONTAINER"))(
		SCRIPT()(RawText(`
			window._pendingVideos = {};

			window._videoUploadManager = {
				_uploads: {},
				addUpload(id, title, totalChunks) {
					this._uploads[id] = { title, totalChunks, completedChunks: 0 };
					this._updateIndicator();
				},
				markChunkComplete(id) {
					const u = this._uploads[id];
					if (!u) return;
					u.completedChunks++;
					if (u.completedChunks >= u.totalChunks) delete this._uploads[id];
					this._updateIndicator();
				},
				removeUpload(id) {
					delete this._uploads[id];
					this._updateIndicator();
				},
				_updateIndicator() {
					const count = Object.keys(this._uploads).length;
					UPLOAD_INDICATOR_COUNT.textContent = count;
					this._renderUploadList();
					UPLOAD_INDICATOR.classList.toggle('hidden', count === 0);
				},
				_renderUploadList() {
						const entries = Object.entries(this._uploads);
						if (entries.length === 0) {
								UPLOAD_LIST_DIALOG.close();
								return;
						}
						let html = '';
						for (const [, u] of entries) {
								html += '<div class="flex flex-col gap-1 py-2">'
										 +  '<div class="flex justify-between text-sm">'
										 +  '<span class="truncate">' + u.title + '</span>'
										 +  '<span class="shrink-0">' + u.completedChunks + '/' + u.totalChunks + '</span>'
										 +  '</div>'
										 +  '<progress class="progress progress-primary w-full" value="' + u.completedChunks + '" max="' + u.totalChunks + '"></progress>'
										 +  '</div>';
						}
						UPLOAD_LIST_BODY.innerHTML = html;
				},
			  async upload({ pendingVideoId, videoTitle, partSize, uploadUrls, videoId, uploadId, completeUploadUrl }) {
					this.addUpload(pendingVideoId, videoTitle, uploadUrls.length);

					const file = window._pendingVideos[pendingVideoId];
					if (!file) throw new Error("pending video not found");

					const completedParts = [];
					const uploads = uploadUrls.map(async (url, i) => {
					 	const start = i * partSize;
					 	const end = i === uploadUrls.length - 1 ? file.size : start + partSize;
					 	const blob = file.slice(start, end);

					 	const response = await fetch(url, { method: 'PUT', body: blob });
					 	if (!response.ok) throw new Error("upload failed");

					 	completedParts.push({
					 	 	etag: (response.headers.get('ETag') ?? '').replaceAll('"', ''),
					 	 	partNumber: i + 1,
					 	});

						this.markChunkComplete(pendingVideoId);
					});

					await Promise.all(uploads);

					await fetch(completeUploadUrl, {
					 	method: 'POST',
					 	headers: { 'Content-Type': 'application/json' },
					 	body: JSON.stringify({ videoId, uploadId, parts: completedParts }),
					});

					delete window._pendingVideos[pendingVideoId];
			 	}
			};
	`)),
	)
}

func videoUploadIndicator() HyperNode {
	return DIV(
		AttrId("UPLOAD_INDICATOR"),
		AttrClass("hidden fixed bottom-6 right-6 z-50"),
	)(
		DIV(AttrClass("indicator"))(
			BUTTON(
				AttrClass("btn btn-circle btn-primary btn-lg shadow-lg relative"),
				AttrOnClick("UPLOAD_LIST_DIALOG.showModal()"),
			)(
				RawText(`<svg class="w-5 h-5 animate-bounce" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-arrow-up-from-line-icon lucide-arrow-up-from-line"><path d="m18 9-6-6-6 6"/><path d="M12 3v14"/><path d="M5 21h14"/></svg>`),
			),
			SPAN(AttrId("UPLOAD_INDICATOR_COUNT"), AttrClass("indicator-item indicator-bottom indicator-center badge badge-secondary"))("0"),
		),
		DIALOG(AttrId("UPLOAD_LIST_DIALOG"), AttrClass("modal"))(
			DIV(AttrClass("modal-box"))(
				H3(AttrClass("text-lg font-bold"))("Uploading Videos"),
				DIV(AttrId("UPLOAD_LIST_BODY"), AttrClass("mt-4 space-y-2"))(),
			),
			FORM(AttrMethod(MethodDialog), AttrClass("modal-backdrop"))(
				BUTTON()("close"),
			),
		),
	)
}

type profileVideosParams struct {
	videoCards []VideoCardParams
}

func profileVideosContainer(params profileVideosParams) HyperNode {
	return DIV(AttrId("PROFILE_VIDEOS_CONTAINER"), AttrClass("mt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"))(
		Range(params.videoCards, func(p VideoCardParams) HyperNode {
			return VideoCard(p)
		}),
	)
}

type VideoCardParams struct {
	VideoId       string
	Title         string
	Timestamp     time.Time
	OwnerName     string
	OwnerUsername string
	ViewsCount    int
	ThumbnailUrl  string
	AvatarUrl     string
}

func VideoCard(params VideoCardParams) HyperNode {
	ownerProfilePageLink := fmt.Sprintf("/@%s", params.OwnerUsername)
	videoPageLink := fmt.Sprintf("/videos/%s", params.VideoId)
	visiteProfileAttrs := []Attribute{
		Attr("hx-get", fmt.Sprintf("%s/content", ownerProfilePageLink)),
		Attr("hx-push-url", ownerProfilePageLink),
		Attr("hx-target", "#APP_PAGE_CONTENT"),
		Attr("hx-swap", "innerHTML"),
		Attr("hx-trigger", "click consume"),
		Attr("hx-indicator", "#PAGE_CONTENT_CONTAINER"),
	}

	return DIV(
		AttrClass("card bg-base-100 transition-shadow duration-200 cursor-pointer"),
		Attr("hx-get", fmt.Sprintf("%s/content", videoPageLink)),
		Attr("hx-push-url", videoPageLink),
		Attr("hx-target", "#APP_PAGE_CONTENT"),
		Attr("hx-swap", "innerHTML"),
		Attr("hx-indicator", "#PAGE_CONTENT_CONTAINER"),
	)(
		FIGURE(AttrClass("relative aspect-video overflow-hidden group"))(
			DIV(AttrClass("w-full h-full transition-transform duration-200 group-hover:scale-105"))(
				If(params.ThumbnailUrl != "",
					IMG(
						AttrClass("w-full h-full object-cover"),
						AttrSrc(params.ThumbnailUrl),
					),
				).Else(
					videoCardThumbnailPlaceholder(),
				),
			),
			DIV(AttrClass("absolute inset-0 flex items-center justify-center bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity duration-200"))(
				RawText(`<svg class="w-12 h-12 text-white drop-shadow-lg" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>`),
			),
		),
		DIV(AttrClass("card-body flex flex-row gap-3 p-3"))(
			DIV()(
				DIV(append(visiteProfileAttrs, AttrClass("shrink-0 cursor-pointer"))...)(
					If(params.AvatarUrl != "",
						IMG(AttrClass("w-9 h-9 rounded-full"), AttrSrc(params.AvatarUrl)),
					).Else(
						videoCardAvatarPlaceholder(),
					),
				),
			),
			DIV(AttrClass("min-w-0 flex-1"))(
				H2(AttrClass("card-title text-base font-bold leading-tight line-clamp-2"))(params.Title),
				A(append(visiteProfileAttrs, AttrClass("link link-hover text-xs text-base-content/60"))...)(
					params.OwnerName,
				),
				DIV(AttrClass("text-xs text-base-content/60"))(
					normalizeViewsCount(params.ViewsCount), " views",
					" . ",
					normalizeTimestamp(params.Timestamp), " ago",
				),
			),
		),
	)
}

func normalizeTimestamp(t time.Time) any {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / (24 * 30))
		if months == 1 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", months)
	default:
		years := int(d.Hours() / (24 * 365))
		if years == 1 {
			return "1 year"
		}
		return fmt.Sprintf("%d years", years)
	}
}

func normalizeViewsCount(i int) any {
	switch {
	case i >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(i)/1_000_000_000)
	case i >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(i)/1_000_000)
	case i >= 1_000:
		return fmt.Sprintf("%.1fK", float64(i)/1_000)
	default:
		return fmt.Sprintf("%d", i)
	}
}

func videoCardAvatarPlaceholder() HyperNode {
	return DIV(AttrClass("avatar placeholder"))(
		DIV(AttrClass("bg-neutral text-neutral-content rounded-full w-10 h-10 flex items-center justify-center text-xs"))(
			RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`),
		),
	)
}

type WatchLaterButtonParams struct {
	VideoId  string
	IsActive bool
}

func WatchLaterButton(params WatchLaterButtonParams) HyperNode {
	return DIV(AttrId("WATCH_LATER_BUTTON"))(
		BUTTON(
			AttrClass("btn btn-soft btn-sm tooltip tooltip-top"),
			Attr("data-tip", IfElse(params.IsActive, "Remove from Watch Later", "Add to Watch Later")),
			IfElse(params.IsActive,
				Attr("hx-delete", fmt.Sprintf("/videos/%s/watch_later", params.VideoId)),
				Attr("hx-post", fmt.Sprintf("/videos/%s/watch_later", params.VideoId)),
			),
			Attr("hx-target", "#WATCH_LATER_BUTTON"),
			Attr("hx-swap", "outerHTML"),
		)(
			RawText(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="%s"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>`,
				IfElse(params.IsActive, "text-primary", ""),
			)),
		),
	)
}

type PlaylistCheckboxParams struct {
	VideoId    string
	PlaylistId string
	Name       string
	Checked    bool
}

func PlaylistCheckbox(params PlaylistCheckboxParams) HyperNode {
	return DIV(AttrClass("form-control"))(
		LABEL(AttrClass("label cursor-pointer flex justify-between"))(
			SPAN(AttrClass("text-base-content"))(params.Name),
			INPUT(
				AttrType(TypeCheckbox),
				AttrClass("checkbox checkbox-sm checkbox-primary"),
				IfElse(params.Checked,
					Attr("hx-delete", fmt.Sprintf("/videos/%s/playlists/%s", params.VideoId, params.PlaylistId)),
					Attr("hx-post", fmt.Sprintf("/videos/%s/playlists/%s", params.VideoId, params.PlaylistId)),
				),
				Attr("hx-on::after:request", "if (evt.detail.ctx.response.ok) this.checked = !this.checked"),
				Attr("hx-swap", "none"),
				AttrChecked(params.Checked),
			),
		),
	)
}

type AddToPlaylistModalParams struct {
	VideoId   string
	Playlists []PlaylistCheckboxParams
}

func AddToPlaylistButton(videoId string) HyperNode {
	return DIV(AttrClass("tooltip tooltip-top"), Attr("data-tip", "Add to Playlist"))(
		BUTTON(
			AttrClass("btn btn-soft btn-sm"),
			AttrOnClick("ADD_TO_PLAYLIST_MODAL.show()"),
		)(
			RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 5H3"/><path d="M10 12H3"/><path d="M10 19H3"/><path d="M15 12.003a1 1 0 0 1 1.517-.859l4.997 2.997a1 1 0 0 1 0 1.718l-4.997 2.997a1 1 0 0 1-1.517-.86z"/></svg>`),
		),
	)
}

func AddToPlaylistModal(params AddToPlaylistModalParams) HyperNode {
	return DIALOG(AttrId("ADD_TO_PLAYLIST_MODAL"), AttrClass("modal"))(
		DIV(AttrClass("modal-box"))(
			H3(AttrClass("text-lg font-bold mb-4"))("Add to Playlist"),
			DIV(AttrId("PLAYLIST_CHECKBOX_LIST"), AttrClass("space-y-2"))(
				If(len(params.Playlists) == 0,
					P(AttrClass("text-sm text-base-content/60"))("No playlists yet. Create one below."),
				).Else(
					Group(
						Range(params.Playlists, func(p PlaylistCheckboxParams) HyperNode {
							return PlaylistCheckbox(p)
						}),
					),
				),
			),
			DIV(AttrClass("divider my-4"))(),
			CreatePlaylistForm(CreatePlaylistFormParams{VideoId: params.VideoId}),
		),
		FORM(AttrMethod(MethodDialog), AttrClass("modal-backdrop"))(
			BUTTON()("close"),
		),
	)
}

type CreatePlaylistFormParams struct {
	VideoId string
	Name    string
	NameErr error
}

func CreatePlaylistForm(params ...CreatePlaylistFormParams) HyperNode {
	var p CreatePlaylistFormParams
	if len(params) > 0 {
		p = params[0]
	}

	return FORM(
		AttrId("CREATE_PLAYLIST_FORM"),
		AttrClass("flex gap-2"),
		Attr("hx-post", "/playlists"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-target", "this"),
		Attr("hx-vals", fmt.Sprintf(`js:{videoId: %q}`, p.VideoId)),
	)(
		INPUT(
			AttrId("FORM_PLAYLIST_NAME"),
			AttrType(TypeText),
			AttrName("name"),
			AttrValue(p.Name),
			AttrPlaceholder("New playlist name..."),
			AttrClass("input w-full"+IfElseZero(p.NameErr != nil, " input-error")),
			AttrRequired(true),
		),
		BUTTON(
			AttrClass("btn btn-primary"),
			AttrType(TypeSubmit),
		)(
			"Create",
		),
		If(p.NameErr != nil,
			P(AttrClass("text-xs text-error"))(p.NameErr),
		),
	)
}

func videoCardThumbnailPlaceholder() HyperNode {
	return DIV(AttrClass("w-full h-full flex items-center justify-center bg-base-200"))(
		RawText(`<svg class="w-10 h-10 text-base-content/30" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-clapperboard-icon lucide-clapperboard"><path d="m12.296 3.464 3.02 3.956"/><path d="M20.2 6 3 11l-.9-2.4c-.3-1.1.3-2.2 1.3-2.5l13.5-4c1.1-.3 2.2.3 2.5 1.3z"/><path d="M3 11h18v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="m6.18 5.276 3.1 3.899"/></svg>`),
	)
}
