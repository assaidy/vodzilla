package templates

import (
	"encoding/json"
	"fmt"

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
			window._pendingVideos[%q] = FORM_VIDEO.files[0].file;
		`, pendingVideoId)),
		Attr("hx-on::after:request", fmt.Sprintf(`
			if (!event.detail.ctx.response.ok) delete window._pendingVideos[%q];
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
				spinner(),
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
	VideoId        string
	UploadId       string
	UploadUrls     []string
	VideoTitle     string
}

func VideoUploader(params VideoUploaderParams) HyperNode {
	encodedUrls, _ := json.Marshal(params.UploadUrls)

	return SCRIPT()(RawText(fmt.Sprintf(`
		(async () => {
			const pendingVideoId = %q;
			const videoId = %q;
			const videoTitle = %q;
			const uploadId = %q;
			const uploadUrls = %s;
			const completeUploadUrl = %q;

			window._videoUploadManager.addUpload(pendingVideoId, videoTitle, uploadUrls.length);

			try {
				const file = window._pendingVideos[pendingVideoId];
				if (!file) {
					throw new Error("pending video not found");
				}

				const chunkSize = Math.ceil(file.size / uploadUrls.length);
				const completedParts = [];

				const uploads = uploadUrls.map(async (url, i) => {
					const start = i * chunkSize;
					const end = i === uploadUrls.length - 1
						? file.size
						: start + chunkSize;
					const blob = file.slice(start, end);

					const response = await fetch(url, {
						method: 'PUT',
						body: blob,
					});

					if (!response.ok) {
						throw new Error("upload failed");
					}

					completedParts.push({
						etag: (response.headers.get('ETag') ?? '').replaceAll('"', ''),
						partNumber: i + 1,
					});

					window._videoUploadManager.markChunkComplete(pendingVideoId);
				});

				await Promise.all(uploads);

				await fetch(
					completeUploadUrl,
					{
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							uploadId,
							parts: completedParts,
						}),
					},
				);

				delete window._pendingVideos[pendingVideoId];
			} catch (err) {
				console.error(err);
				window._videoUploadManager.removeUpload(pendingVideoId);
			} finally {
				document.currentScript.remove();
			}
		})();
	`,
		params.PendingVideoId,
		params.VideoId,
		params.VideoTitle,
		params.UploadId,
		string(encodedUrls),
		fmt.Sprintf("/videos/%s/complete_upload", params.VideoId),
	)))
}

func videoUploadersContainer() HyperNode {
	return DIV(AttrId("VIDEO_UPLOADERS_CONTAINER"))(
		SCRIPT()(RawText(`
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
					UPLOAD_INDICATOR_COUNT?.textContent = count;
					UPLOAD_INDICATOR?.classList.toggle('hidden', count === 0);
				}
			};
	`)),
	)
}

func uploadIndicator() HyperNode {
	return DIV(
		AttrId("UPLOAD_INDICATOR"),
		AttrClass("fixed bottom-6 right-6 z-50"),
	)(
		DIV(AttrClass("indicator"))(
			BUTTON(AttrClass("btn btn-circle btn-primary btn-lg shadow-lg relative"))(
				RawText(`<svg class="w-5 h-5 animate-bounce" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-arrow-up-from-line-icon lucide-arrow-up-from-line"><path d="m18 9-6-6-6 6"/><path d="M12 3v14"/><path d="M5 21h14"/></svg>`),
			),
			SPAN(AttrId("UPLOAD_INDICATOR_COUNT"), AttrClass("indicator-item indicator-bottom indicator-center badge badge-secondary"))("0"),
		),
	)
}
