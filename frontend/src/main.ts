import { Screens, Window } from "@wailsio/runtime";
import "./styles/main.css";
import { SystemService } from "./lib/api";
import type { View } from "./lib/view";
import { createMetronomeView } from "./views/metronome";
import { createTabsView } from "./views/tabs";
import { createTunerView } from "./views/tuner";

const WINDOW_MIN_WIDTH = 900;
const WINDOW_MIN_HEIGHT = 620;
const WINDOW_MAX_WIDTH = 1600;
const WINDOW_MAX_HEIGHT = 1000;
const WINDOW_SCREEN_MARGIN = 96;

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

async function fitWindowToScreen(): Promise<void> {
  let availableWidth = window.screen.availWidth;
  let availableHeight = window.screen.availHeight;

  try {
    const screen = await Screens.GetPrimary();
    availableWidth = screen.WorkArea.Width;
    availableHeight = screen.WorkArea.Height;
  } catch {
  }

  const width = Math.round(
    clamp(
      availableWidth - WINDOW_SCREEN_MARGIN,
      WINDOW_MIN_WIDTH,
      WINDOW_MAX_WIDTH,
    ),
  );
  const height = Math.round(
    clamp(
      availableHeight - WINDOW_SCREEN_MARGIN,
      WINDOW_MIN_HEIGHT,
      WINDOW_MAX_HEIGHT,
    ),
  );

  try {
    await Window.SetSize(width, height);
    await Window.Center();
  } catch {
  }

  try {
    await Window.Show();
  } catch {
  }
}

void fitWindowToScreen();

type ViewName = "tuner" | "metronome" | "tabs";

const viewFactories: Record<ViewName, () => View> = {
  tuner: createTunerView,
  metronome: createMetronomeView,
  tabs: createTabsView,
};

const app = document.querySelector<HTMLDivElement>("#app");
if (!app) {
  throw new Error("Application root not found");
}

app.innerHTML = `
  <div class="app-shell">
    <header class="app-header">
      <div class="brand">
        <div class="brand-mark" aria-hidden="true">G</div>
        <div class="brand-copy">
          <strong>Guitarists</strong>
          <span>Practice tools for the serious player</span>
        </div>
      </div>
      <nav class="main-nav" aria-label="Main">
        <button class="nav-button" type="button" data-view="tuner">Tuner</button>
        <button class="nav-button" type="button" data-view="metronome">Metronome</button>
        <button class="nav-button" type="button" data-view="tabs">Tab editor</button>
      </nav>
      <div class="version" id="app-version"></div>
    </header>
    <main class="view-host" id="view-host"></main>
  </div>
`;

const host = document.querySelector<HTMLElement>("#view-host");
if (!host) {
  throw new Error("View host not found");
}

const views = {} as Record<ViewName, View>;
for (const [name, factory] of Object.entries(viewFactories) as Array<
  [ViewName, () => View]
>) {
  const view = factory();
  view.element.hidden = true;
  host.append(view.element);
  views[name] = view;
}

const navButtons = Array.from(
  document.querySelectorAll<HTMLButtonElement>(".nav-button"),
);

let current: ViewName | null = null;

function isViewName(value: string): value is ViewName {
  return value === "tuner" || value === "metronome" || value === "tabs";
}

function showView(name: ViewName): void {
  if (current === name) {
    return;
  }
  if (current) {
    views[current].deactivate();
    views[current].element.hidden = true;
  }
  current = name;
  views[name].element.hidden = false;
  views[name].activate();
  for (const button of navButtons) {
    const active = button.dataset.view === name;
    button.classList.toggle("is-active", active);
    button.setAttribute("aria-current", active ? "page" : "false");
  }
  if (location.hash !== `#${name}`) {
    history.replaceState(null, "", `#${name}`);
  }
}

for (const button of navButtons) {
  button.addEventListener("click", () => {
    const name = button.dataset.view;
    if (name && isViewName(name)) {
      showView(name);
    }
  });
}

window.addEventListener("hashchange", () => {
  const name = location.hash.replace(/^#/, "");
  if (isViewName(name)) {
    showView(name);
  }
});

void SystemService.version()
  .then((version) => {
    const element = document.querySelector<HTMLElement>("#app-version");
    if (element) {
      element.textContent = `v${version}`;
    }
  })
  .catch(() => undefined);

const initial = location.hash.replace(/^#/, "");
showView(isViewName(initial) ? initial : "tuner");
