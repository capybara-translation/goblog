// Progressive enhancement for the /health charts (see internal/healthchart).
// Each SVG data point already has a <title> tooltip for mouse hover; this
// script adds a tap-to-show equivalent for touch users (who have no hover)
// by listening for clicks on the transparent hit-target circles the server
// renders with a data-chart-label attribute.
(() => {
  "use strict";

  // Tracks the element whose tooltip is currently shown, so a second click
  // on the same point toggles it off instead of re-showing it.
  let activeTarget = null;

  const removeTip = () => {
    const existing = document.querySelector(".chart-tip");
    if (existing) existing.remove();
  };

  document.addEventListener("click", (ev) => {
    const hadTip = document.querySelector(".chart-tip") !== null;
    const previousTarget = activeTarget;
    removeTip();

    const target = ev.target.closest && ev.target.closest("[data-chart-label]");
    if (!target) {
      activeTarget = null;
      return;
    }

    if (hadTip && previousTarget === target) {
      // Clicking the same point again just hides the tip (already removed
      // above) rather than immediately re-showing it.
      activeTarget = null;
      return;
    }

    const label = target.getAttribute("data-chart-label");
    if (!label) {
      activeTarget = null;
      return;
    }

    const tip = document.createElement("div");
    tip.className = "chart-tip";
    tip.textContent = label;
    // Inline styles only: Tailwind's content scan covers templates, not this
    // JS file, so utility classes here would be purged from the built CSS.
    tip.style.cssText =
      "background:#3f3f46;color:#fafafa;padding:4px 8px;border-radius:4px;" +
      "font-size:12px;position:absolute;z-index:10;pointer-events:none;white-space:nowrap";
    document.body.appendChild(tip);

    // Measure after insertion so offsetWidth/Height reflect the rendered
    // size, then center the tip above the point.
    const rect = target.getBoundingClientRect();
    const top = rect.top + window.scrollY - tip.offsetHeight - 6;
    const left = rect.left + window.scrollX + rect.width / 2 - tip.offsetWidth / 2;
    tip.style.top = `${top}px`;
    tip.style.left = `${left}px`;

    activeTarget = target;
  });
})();
