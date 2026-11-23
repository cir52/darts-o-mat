// Tells the player when the server can't be reached (it restarts, the network is gone),
// and brings the live part of the page up to date when the connection is back: what
// changed in between was pushed to nobody. htmx alone shows nothing when a request fails.
// It also keeps an open <details data-keep-open="name"> open when htmx swaps the part of the
// page that holds it (the game page swaps its board on every dart and every pause): a
// request from inside the details (choosing in a menu) closes it.
(function () {
  "use strict";

  function notice(show) {
    const el = document.getElementById("offline");
    if (el) el.hidden = !show;
  }

  // A request that didn't reach the server, or a proxy that has no server behind it.
  document.addEventListener("htmx:sendError", () => notice(true));
  document.addEventListener("htmx:responseError", (e) => {
    if ([502, 503, 504].includes(e.detail.xhr.status)) notice(true);
  });
  document.addEventListener("htmx:afterRequest", (e) => {
    if (e.detail.successful) notice(false);
  });

  // The live stream (data-refresh names the part it swaps): the first open is the page
  // loading, every later one a reconnect, after which that part is fetched again.
  document.addEventListener("htmx:sseError", () => notice(true));
  document.addEventListener("htmx:sseOpen", (e) => {
    notice(false);
    const el = e.target;
    if (!el.dataset.refresh) return;
    if (!el.dataset.opened) {
      el.dataset.opened = "1";
      return;
    }
    const target = el.dataset.refresh;
    htmx.ajax("GET", window.location.href, { target: target, select: target, swap: "outerHTML" });
  });

  // The names of the open details that keep open. A capturing listener, since toggle
  // doesn't bubble; the one for a details that is reopened below finds its name there.
  const open = new Set();
  document.addEventListener(
    "toggle",
    (e) => {
      const name = e.target.dataset && e.target.dataset.keepOpen;
      if (!name) return;
      if (e.target.open) open.add(name);
      else open.delete(name);
    },
    true,
  );
  // htmx:load fires for the new content of every swap, the live stream's too.
  document.addEventListener("htmx:load", (e) => {
    const root = e.target;
    if (!root.querySelectorAll) return;
    const found = root.matches("details[data-keep-open]") ? [root] : [];
    root.querySelectorAll("details[data-keep-open]").forEach((d) => found.push(d));
    for (const d of found) if (open.has(d.dataset.keepOpen)) d.open = true;
  });
  document.addEventListener("htmx:beforeRequest", (e) => {
    const d = e.target.closest && e.target.closest("details[data-keep-open]");
    if (d) open.delete(d.dataset.keepOpen);
  });
})();
