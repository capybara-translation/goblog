// Progressive enhancement for article reactions. Counts are server-rendered;
// this script overlays per-visitor reacted state and handles toggling. It
// supports multiple reaction blocks on one page (e.g. the home/tag listings):
// reacted state is fetched in a single batch request, while each block handles
// its own click toggles.
(() => {
  "use strict";

  const blocks = Array.from(document.querySelectorAll(".reaction-block[data-post-slug]"));
  if (!blocks.length) return;

  const HEADERS = { "X-Requested-With": "XMLHttpRequest" };

  const applyToBlock = (root, reactions) => {
    if (!reactions) return;
    reactions.forEach((r) => {
      const btn = root.querySelector(`.reaction-btn[data-reaction-id="${r.id}"]`);
      if (!btn) return;
      const countEl = btn.querySelector(".reaction-count");
      if (countEl) countEl.textContent = r.count;
      btn.setAttribute("aria-pressed", r.reacted ? "true" : "false");
      btn.classList.toggle("reaction-active", !!r.reacted);
    });
  };

  // Index blocks by slug and fetch all reacted state in one request.
  const bySlug = new Map();
  blocks.forEach((root) => {
    const slug = root.getAttribute("data-post-slug");
    if (slug) bySlug.set(slug, root);
  });
  const qs = Array.from(bySlug.keys()).map(encodeURIComponent).join(",");
  if (qs) {
    fetch(`/api/v1/reactions?slugs=${qs}`, { headers: HEADERS, credentials: "same-origin" })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (!data || !data.reactions) return;
        bySlug.forEach((root, slug) => {
          if (data.reactions[slug]) applyToBlock(root, data.reactions[slug]);
        });
      })
      .catch(() => {});
  }

  // Per-block click handling (toggle add/remove) with a generation guard so a
  // slow response can't roll back a newer one within the same block.
  blocks.forEach((root) => {
    const slug = root.getAttribute("data-post-slug");
    if (!slug) return;
    const base = `/api/v1/posts/${encodeURIComponent(slug)}/reactions`;
    let seq = 0;

    root.addEventListener("click", async (ev) => {
      const btn = ev.target.closest(".reaction-btn");
      if (!btn || btn.disabled || !root.contains(btn)) return;

      const id = btn.getAttribute("data-reaction-id");
      const method = btn.getAttribute("aria-pressed") === "true" ? "DELETE" : "POST";
      const mySeq = ++seq;

      btn.disabled = true;
      try {
        const res = await fetch(`${base}/${encodeURIComponent(id)}`, {
          method,
          headers: HEADERS,
          credentials: "same-origin",
        });
        if (res.ok) {
          const data = await res.json();
          if (mySeq === seq) applyToBlock(root, data.reactions);
        }
      } catch {
        // Non-critical; leave current state.
      } finally {
        btn.disabled = false;
      }
    });
  });
})();
