// Progressive enhancement for article reactions. Counts are server-rendered;
// this script layers on the visitor's reacted state and handles toggling.
(function () {
  "use strict";

  var root = document.getElementById("post-reactions");
  if (!root) return;

  var slug = root.getAttribute("data-post-slug");
  if (!slug) return;

  var base = "/api/v1/posts/" + encodeURIComponent(slug) + "/reactions";
  var HEADERS = { "X-Requested-With": "XMLHttpRequest" };

  function apply(reactions) {
    if (!reactions) return;
    reactions.forEach(function (r) {
      var btn = root.querySelector('.reaction-btn[data-reaction-id="' + r.id + '"]');
      if (!btn) return;
      var countEl = btn.querySelector(".reaction-count");
      if (countEl) countEl.textContent = r.count;
      btn.setAttribute("aria-pressed", r.reacted ? "true" : "false");
      btn.classList.toggle("reaction-active", !!r.reacted);
    });
  }

  // On load: fetch reacted state (and refreshed counts). This also issues the
  // visitor cookie server-side so the first click is attributed correctly.
  fetch(base, { headers: HEADERS, credentials: "same-origin" })
    .then(function (res) { return res.ok ? res.json() : null; })
    .then(function (data) { if (data) apply(data.reactions); })
    .catch(function () { /* reactions are non-critical; ignore */ });

  root.addEventListener("click", function (ev) {
    var btn = ev.target.closest(".reaction-btn");
    if (!btn || btn.disabled) return;

    var id = btn.getAttribute("data-reaction-id");
    var pressed = btn.getAttribute("aria-pressed") === "true";
    var method = pressed ? "DELETE" : "POST";

    btn.disabled = true;
    fetch(base + "/" + encodeURIComponent(id), {
      method: method,
      headers: HEADERS,
      credentials: "same-origin"
    })
      .then(function (res) { return res.ok ? res.json() : null; })
      .then(function (data) {
        if (data) apply(data.reactions);
        btn.disabled = false;
      })
      .catch(function () {
        btn.disabled = false; // re-enable even if the request failed
      });
  });
})();
