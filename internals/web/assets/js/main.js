function $cookie(name) {
  return (
    document.cookie
      .split("; ")
      .map((cookie) => cookie.split("="))
      .find(([key]) => key === name)
      ?.map(decodeURIComponent)[1] || ""
  );
}

const _clientId = document.body.dataset.clientId;

htmx.on("htmx:config:request", (event) => {
  event.detail.ctx.request.headers["X-CSRF-Token"] = $cookie("csrf_token");
  event.detail.ctx.request.headers["X-Client-ID"] = _clientId;
});
