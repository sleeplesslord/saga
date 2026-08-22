const directions = {
  signal: {
    title: "Signal Box",
    line: "A railway interlocking board for coordinating work.",
    risk: "the route metaphor must stay subordinate to task names."
  },
  terminal: {
    title: "Phosphor Terminal",
    line: "A searchable machine transcript native to agent workflows.",
    risk: "terminal atmosphere can reduce graph legibility and long-session comfort."
  },
  onebit: {
    title: "One-bit Desktop",
    line: "A compact set of overlapping tools for list, graph, and detail.",
    risk: "the pixel language must feel operational rather than nostalgic."
  },
  miura: {
    title: "Deployable Sheet",
    line: "A constraint field that unfolds task relationships in space.",
    risk: "diagonal geometry can make routine title scanning less immediate."
  }
};

const tabs = [...document.querySelectorAll(".direction-tab")];
const mockups = [...document.querySelectorAll(".mockup")];
const title = document.querySelector("#directionTitle");
const line = document.querySelector("#directionLine");
const risk = document.querySelector("#directionRisk");

function showDirection(id, updateHash = true) {
  if (!directions[id]) return;
  tabs.forEach(tab => {
    const selected = tab.dataset.direction === id;
    tab.classList.toggle("is-active", selected);
    tab.setAttribute("aria-pressed", selected ? "true" : "false");
  });
  mockups.forEach(mockup => mockup.classList.toggle("is-visible", mockup.dataset.mockup === id));
  title.textContent = directions[id].title;
  line.textContent = directions[id].line;
  risk.innerHTML = `<b>Watch for:</b> ${directions[id].risk}`;
  if (updateHash) history.replaceState(null, "", `#${id}`);
}

tabs.forEach(tab => tab.addEventListener("click", () => showDirection(tab.dataset.direction)));

document.addEventListener("keydown", event => {
  if (!['ArrowLeft', 'ArrowRight'].includes(event.key) || event.target.matches('input, button')) return;
  const current = tabs.findIndex(tab => tab.classList.contains("is-active"));
  const delta = event.key === 'ArrowRight' ? 1 : -1;
  const next = (current + delta + tabs.length) % tabs.length;
  showDirection(tabs[next].dataset.direction);
  tabs[next].focus();
});

document.querySelector("#copyLink").addEventListener("click", async event => {
  try {
    await navigator.clipboard.writeText(location.href);
    const original = event.currentTarget.textContent;
    event.currentTarget.textContent = "Copied";
    setTimeout(() => { event.currentTarget.textContent = original; }, 1400);
  } catch {
    event.currentTarget.textContent = "Copy unavailable";
  }
});

function updateClock() {
  const clock = document.querySelector("#signalTime");
  if (clock) clock.textContent = new Date().toLocaleTimeString([], {hour12: false});
}
updateClock();
setInterval(updateClock, 1000);

const initial = location.hash.slice(1);
showDirection(directions[initial] ? initial : "signal", false);
