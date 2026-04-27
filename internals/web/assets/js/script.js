function getCookie(name) {
  return (
    document.cookie
      .split("; ")
      .map((cookie) => cookie.split("="))
      .find(([key]) => key === name)
      ?.map(decodeURIComponent)[1] || null
  );
}
