function handleClearClick() {
  const input = document.querySelector("#comment-input");
  if (input) input.value = "";
  const err = document.querySelector("#create-comment-form .text-error");
  if (err) err.remove();
}

function handleReadMoreClick(btn) {
  const p = btn.closest(".comment-text-container").querySelector("p");
  const fullText = btn.dataset.fullText;
  const COMMENT_CHAR_LIMIT = 100;

  if (btn.textContent === "Read more") {
    p.textContent = fullText;
    btn.textContent = "Show less";
  } else {
    p.textContent = fullText.length > COMMENT_CHAR_LIMIT ? fullText.slice(0, COMMENT_CHAR_LIMIT) + "..." : fullText;
    btn.textContent = "Read more";
  }
}

function handleRepliesToggle(toggle) {
  const commentId = toggle.dataset.commentId;
  const container = document.querySelector(`#replies-${commentId}`);
  if (!container) return;

  container.classList.toggle("hidden");
  toggle.textContent = container.classList.contains("hidden") ? toggle.dataset.viewText : "Hide replies";
}

function handleToggleReplyForm(btn) {
  const commentId = btn.dataset.commentId;
  const form = document.querySelector(`#reply-form-${commentId}`);
  if (!form) return;

  form.classList.toggle("hidden");
}

function handleCancelReply(btn) {
  const commentId = btn.dataset.commentId;
  const form = document.querySelector(`#reply-form-${commentId}`);
  if (!form) return;

  form.classList.add("hidden");
  const input = form.querySelector("textarea");
  if (input) input.value = "";
  const err = form.querySelector(".text-error");
  if (err) err.remove();
}

(() => {
  const section = document.querySelector("#comment-section");
  if (!section) return;

  section.addEventListener("click", (e) => {
    const clearBtn = e.target.closest("#comment-clear-btn");
    if (clearBtn) {
      handleClearClick();
      return;
    }

    const replyToggle = e.target.closest("[data-reply-toggle]");
    if (replyToggle) {
      handleToggleReplyForm(replyToggle);
      return;
    }

    const cancelReply = e.target.closest("[data-reply-cancel]");
    if (cancelReply) {
      handleCancelReply(cancelReply);
      return;
    }

    const btn = e.target.closest(".comment-read-more");
    if (btn) {
      handleReadMoreClick(btn);
      return;
    }

    const toggle = e.target.closest("[data-replies-toggle]");
    if (toggle) {
      handleRepliesToggle(toggle);
    }
  });
})();
