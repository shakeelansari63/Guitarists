import {
  MetronomeService,
  errorMessage,
  onMetronomeBeat,
  onMetronomeState,
} from "../lib/api";
import { byId, h } from "../lib/dom";
import { showToast } from "../lib/toast";
import type {
  BeatEvent,
  MetronomeConfig,
  MetronomeState,
} from "../lib/types";
import type { View } from "../lib/view";

const defaultConfig: MetronomeConfig = {
  bpm: 120,
  beatsPerMeasure: 4,
  subdivision: 1,
  accent: true,
};

export function createMetronomeView(): View {
  const element = document.createElement("section");
  element.className = "view metronome-view";
  element.innerHTML = `
    <div class="view-head">
      <div>
        <p class="eyebrow">Metronome</p>
        <h1>Keep time</h1>
        <p class="lede">Set the tempo, choose a subdivision, and let the accented downbeat mark each measure.</p>
      </div>
      <button class="btn btn-primary btn-lg" id="metro-toggle" type="button">Start</button>
    </div>
    <div class="metro-grid">
      <div class="panel metro-main">
        <div class="metro-display">
          <div class="metro-bpm"><span id="metro-bpm-display">120</span><small>BPM</small></div>
          <div class="metro-beats" id="metro-beats" aria-hidden="true"></div>
          <div class="metro-position" id="metro-position">Measure 1 · Beat 1</div>
        </div>
        <div class="metro-controls">
          <label class="field" for="metro-bpm">
            <span class="field-label">Tempo <output id="metro-bpm-output">120 BPM</output></span>
            <input id="metro-bpm" type="range" min="20" max="400" step="1" value="120" />
          </label>
          <div class="control-row">
            <label class="field" for="metro-number">
              <span class="field-label">Exact BPM</span>
              <input id="metro-number" type="number" min="20" max="400" step="1" value="120" />
            </label>
            <label class="field" for="metro-beats-per-measure">
              <span class="field-label">Beats per measure</span>
              <select id="metro-beats-per-measure"></select>
            </label>
            <label class="field" for="metro-subdivision">
              <span class="field-label">Subdivision</span>
              <select id="metro-subdivision">
                <option value="1">Quarter notes</option>
                <option value="2">Eighth notes</option>
                <option value="3">Triplets</option>
                <option value="4">Sixteenth notes</option>
              </select>
            </label>
            <label class="checkbox-field" for="metro-accent">
              <input id="metro-accent" type="checkbox" checked />
              <span>Accent downbeat</span>
            </label>
          </div>
        </div>
      </div>
      <div class="panel metro-side">
        <h2>Tap tempo</h2>
        <p class="muted">Tap the pad in time and the tempo follows your pulse.</p>
        <button id="metro-tap" class="tap-pad" type="button">Tap</button>
        <p class="tap-hint" id="metro-tap-hint">Tap at least twice.</p>
        <div class="hint">Click interval <strong id="metro-interval">500 ms</strong></div>
      </div>
    </div>
  `;

  const toggleButton = byId<HTMLButtonElement>(element, "metro-toggle");
  const bpmDisplay = byId<HTMLSpanElement>(element, "metro-bpm-display");
  const beatsEl = byId<HTMLDivElement>(element, "metro-beats");
  const positionEl = byId<HTMLDivElement>(element, "metro-position");
  const bpmRange = byId<HTMLInputElement>(element, "metro-bpm");
  const bpmNumber = byId<HTMLInputElement>(element, "metro-number");
  const bpmOutput = byId<HTMLOutputElement>(element, "metro-bpm-output");
  const beatsSelect = byId<HTMLSelectElement>(element, "metro-beats-per-measure");
  const subdivisionSelect = byId<HTMLSelectElement>(element, "metro-subdivision");
  const accentInput = byId<HTMLInputElement>(element, "metro-accent");
  const tapButton = byId<HTMLButtonElement>(element, "metro-tap");
  const tapHint = byId<HTMLParagraphElement>(element, "metro-tap-hint");
  const intervalLabel = byId<HTMLElement>(element, "metro-interval");

  let config: MetronomeConfig = { ...defaultConfig };
  let running = false;
  let beatTimers: number[] = [];

  for (let beats = 1; beats <= 12; beats++) {
    const option = document.createElement("option");
    option.value = String(beats);
    option.textContent = `${beats} / 4`;
    if (beats === 4) {
      option.selected = true;
    }
    beatsSelect.append(option);
  }

  function intervalMs(): number {
    return Math.round(60000 / (config.bpm * config.subdivision));
  }

  function updateInterval(): void {
    intervalLabel.textContent = `${intervalMs()} ms`;
    bpmOutput.textContent = `${config.bpm} BPM`;
    bpmDisplay.textContent = String(config.bpm);
    bpmRange.value = String(config.bpm);
    bpmNumber.value = String(config.bpm);
  }

  function renderBeats(): void {
    beatsEl.replaceChildren();
    for (let i = 1; i <= config.beatsPerMeasure; i++) {
      const beat = h("span", "beat-dot");
      beat.dataset.beat = String(i);
      if (i === 1) {
        beat.classList.add("is-downbeat");
      }
      beatsEl.append(beat);
    }
  }

  function applyState(state: MetronomeState): void {
    config = { ...state.config };
    running = state.running;
    toggleButton.textContent = running ? "Stop" : "Start";
    toggleButton.classList.toggle("is-running", running);
    bpmRange.value = String(config.bpm);
    bpmNumber.value = String(config.bpm);
    beatsSelect.value = String(config.beatsPerMeasure);
    subdivisionSelect.value = String(config.subdivision);
    accentInput.checked = config.accent;
    updateInterval();
    renderBeats();
    if (!running) {
      positionEl.textContent = "Ready";
    }
  }

  function clearBeatTimers(): void {
    for (const timer of beatTimers) {
      window.clearTimeout(timer);
    }
    beatTimers = [];
    for (const dot of beatsEl.querySelectorAll<HTMLElement>(".beat-dot")) {
      dot.classList.remove("is-hit", "is-hit-accent");
    }
  }

  function applyBeat(event: BeatEvent): void {
    if (event.beat > config.beatsPerMeasure) {
      return;
    }
    positionEl.textContent = `Measure ${event.measure} · Beat ${event.beat}`;
    if (event.subIndex !== 0) {
      return;
    }
    clearBeatTimers();
    const dot = beatsEl.querySelector<HTMLElement>(
      `.beat-dot[data-beat="${event.beat}"]`,
    );
    if (dot) {
      dot.classList.add(event.accent ? "is-hit-accent" : "is-hit");
      beatTimers.push(
        window.setTimeout(() => {
          dot.classList.remove("is-hit", "is-hit-accent");
        }, 110),
      );
    }
  }

  function readConfig(): MetronomeConfig {
    return {
      bpm: config.bpm,
      beatsPerMeasure: Number(beatsSelect.value),
      subdivision: Number(subdivisionSelect.value),
      accent: accentInput.checked,
    };
  }

  async function pushConfig(next: MetronomeConfig): Promise<void> {
    const previous = config;
    config = next;
    updateInterval();
    if (next.beatsPerMeasure !== previous.beatsPerMeasure) {
      renderBeats();
    }
    try {
      await MetronomeService.setConfig(next);
    } catch (error) {
      showToast(errorMessage(error), "error");
      config = previous;
      applyState({ config: previous, running });
    }
  }

  function commitBpm(value: number): void {
    const bpm = Math.min(400, Math.max(20, Math.round(value) || config.bpm));
    void pushConfig({ ...readConfig(), bpm });
  }

  bpmRange.addEventListener("input", () => {
    const value = Number(bpmRange.value);
    bpmNumber.value = String(value);
    bpmOutput.textContent = `${value} BPM`;
    bpmDisplay.textContent = String(value);
  });
  bpmRange.addEventListener("change", () => commitBpm(Number(bpmRange.value)));
  bpmNumber.addEventListener("change", () => commitBpm(Number(bpmNumber.value)));
  beatsSelect.addEventListener("change", () => void pushConfig(readConfig()));
  subdivisionSelect.addEventListener("change", () => void pushConfig(readConfig()));
  accentInput.addEventListener("change", () => void pushConfig(readConfig()));

  toggleButton.addEventListener("click", () => {
    void (async () => {
      toggleButton.disabled = true;
      try {
        if (running) {
          await MetronomeService.stop();
          clearBeatTimers();
        } else {
          await MetronomeService.start();
        }
      } catch (error) {
        showToast(errorMessage(error), "error");
        try {
          applyState(await MetronomeService.state());
        } catch {
          // Ignore state refresh failures; the toast already reported the problem.
        }
      } finally {
        toggleButton.disabled = false;
      }
    })();
  });

  tapButton.addEventListener("click", () => {
    void (async () => {
      try {
        const bpm = await MetronomeService.tap();
        config = { ...readConfig(), bpm };
        updateInterval();
        tapHint.textContent = `${bpm} BPM`;
      } catch {
        tapHint.textContent = "Tap again…";
      }
    })();
  });

  onMetronomeBeat(applyBeat);
  onMetronomeState(applyState);

  return {
    element,
    activate() {
      void (async () => {
        try {
          applyState(await MetronomeService.state());
        } catch (error) {
          showToast(errorMessage(error), "error");
        }
      })();
    },
    deactivate() {
      // The metronome keeps running so tempo can be used while editing tabs.
    },
  };
}
