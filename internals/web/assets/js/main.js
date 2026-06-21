function $(selector) {
  return document.querySelector(selector);
}

function $$(selector) {
  return document.querySelectorAll(selector);
}

function $cookie(name) {
  return (
    document.cookie
      .split("; ")
      .map((cookie) => cookie.split("="))
      .find(([key]) => key === name)
      ?.map(decodeURIComponent)[1] || ""
  );
}

document.addEventListener("DOMContentLoaded", () => {
  const CLIENT_ID = document.body.dataset.clientId;

  htmx.on("htmx:config:request", (event) => {
    event.detail.ctx.request.headers["X-CSRF-Token"] = $cookie("csrf_token");
    event.detail.ctx.request.headers["X-Client-ID"] = CLIENT_ID;
  });
});
