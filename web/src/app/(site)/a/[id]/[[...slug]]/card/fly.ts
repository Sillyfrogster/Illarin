const FLIGHT_MS = 900;
const LIFT_MS = 380;

function reduced(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/** lift raises the card off the page and sets it back down, the moment a download or send starts. */
export function lift(card: HTMLElement | null) {
  if (!card || reduced()) return;
  card.animate(
    [
      { transform: "none" },
      { transform: "translateY(-14px) scale(1.015)", offset: 0.45 },
      { transform: "none" },
    ],
    { duration: LIFT_MS * 2, easing: "cubic-bezier(0.22, 1, 0.36, 1)" },
  );
}

/** flyInto sends a copy of the card's art from the card into a target, which takes it with a small pulse. */
export function flyInto(
  card: HTMLElement | null,
  target: HTMLElement | null,
  picture: string | undefined,
) {
  if (!card || !target || reduced()) return;
  const art = card.querySelector<HTMLElement>("[data-card-art]") ?? card;
  const from = art.getBoundingClientRect();
  const to = target.getBoundingClientRect();
  const ghost = document.createElement("div");
  ghost.setAttribute("aria-hidden", "true");
  Object.assign(ghost.style, {
    position: "fixed",
    left: `${from.left}px`,
    top: `${from.top}px`,
    width: `${from.width}px`,
    height: `${from.height}px`,
    borderRadius: "var(--radius-art)",
    background: picture
      ? `center / cover no-repeat url("${picture}")`
      : "var(--v-fill)",
    boxShadow: "0 18px 40px -12px rgb(0 0 0 / 0.55)",
    zIndex: "60",
    pointerEvents: "none",
    transformOrigin: "top left",
  });
  document.body.append(ghost);
  const scale = Math.min(to.height / from.height, to.width / from.width);
  const dx = to.left + to.width / 2 - (from.left + (from.width * scale) / 2);
  const dy = to.top + to.height / 2 - (from.top + (from.height * scale) / 2);
  const flight = ghost.animate(
    [
      { transform: "translate(0, 0) scale(1) rotate(0deg)", opacity: 1 },
      {
        transform: `translate(${dx * 0.25}px, -40px) scale(0.55) rotate(-4deg)`,
        opacity: 1,
        offset: 0.35,
      },
      {
        transform: `translate(${dx * 0.85}px, ${dy * 0.85 - 20}px) scale(${scale * 2.2}) rotate(2deg)`,
        opacity: 0.9,
        offset: 0.75,
      },
      {
        transform: `translate(${dx}px, ${dy}px) scale(${scale}) rotate(0deg)`,
        opacity: 0,
      },
    ],
    { duration: FLIGHT_MS, easing: "cubic-bezier(0.5, 0, 0.2, 1)" },
  );
  flight.onfinish = () => {
    ghost.remove();
    target.animate(
      [
        { transform: "scale(1)" },
        { transform: "scale(1.05)" },
        { transform: "scale(1)" },
      ],
      { duration: 280, easing: "ease-out" },
    );
  };
  flight.oncancel = () => ghost.remove();
}
