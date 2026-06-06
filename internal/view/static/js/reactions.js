// Progressive enhancement for article reactions. Counts are server-rendered;
// this script layers on the visitor's reacted state and handles toggling.
//
// Baseline: modern browsers. The script relies on fetch/Promise/Element.closest
// (all ES2015+ Web APIs) and is served as-is (not transpiled), so it is written
// in ES2015+ syntax to match. It is wrapped in an IIFE to avoid leaking globals.
(() => {
  "use strict";

  const root = document.getElementById("post-reactions");
  if (!root) return;

  const slug = root.getAttribute("data-post-slug");
  if (!slug) return;

  const base = `/api/v1/posts/${encodeURIComponent(slug)}/reactions`;
  const HEADERS = { "X-Requested-With": "XMLHttpRequest" };

  const apply = (reactions) => {
    if (!reactions) return;
    for (const r of reactions) {
      const btn = root.querySelector(`.reaction-btn[data-reaction-id="${r.id}"]`);
      if (!btn) continue;
      const countEl = btn.querySelector(".reaction-count");
      if (countEl) countEl.textContent = r.count;
      btn.setAttribute("aria-pressed", r.reacted ? "true" : "false");
      btn.classList.toggle("reaction-active", !!r.reacted);
    }
  };

  let seq = 0;
  const send = async (url, method) => {
    const mySeq = ++seq;
    const res = await fetch(url, { method, headers: HEADERS, credentials: "same-origin" });
    if (!res.ok) return;
    const data = await res.json();
    if (mySeq !== seq) return; // superseded by a newer request; don't roll back
    apply(data.reactions);
  };

  // On load: fetch reacted state (and refreshed counts). This also issues the
  // visitor cookie server-side so the first click is attributed correctly.
  // Reactions are non-critical, so failures are swallowed (SSR counts remain).
  send(base, "GET").catch(() => {});

  root.addEventListener("click", async (ev) => {
    const btn = ev.target.closest(".reaction-btn");
    if (!btn || btn.disabled) return;

    const id = btn.getAttribute("data-reaction-id");
    const method = btn.getAttribute("aria-pressed") === "true" ? "DELETE" : "POST";

    btn.disabled = true;
    try {
      await send(`${base}/${encodeURIComponent(id)}`, method);
    } catch {
      // Request failed; leave the SSR/previous state in place.
    } finally {
      btn.disabled = false;
    }
  });
})();
