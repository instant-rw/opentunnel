export function clearUserCodeFromUrl() {
  if (typeof window === "undefined") {
    return
  }

  const url = new URL(window.location.href)
  if (!url.searchParams.has("user_code")) {
    return
  }

  url.searchParams.delete("user_code")
  const next = `${url.pathname}${url.search}${url.hash}`
  window.history.replaceState({}, "", next || "/")
}
