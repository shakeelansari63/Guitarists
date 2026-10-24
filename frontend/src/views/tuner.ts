import {
  TunerService,
  errorMessage,
  onTunerState,
  onTunerUpdate,
} from "../lib/api";
import { byId, clamp, h } from "../lib/dom";
import { showToast } from "../lib/toast";
import type { GuitarString, TunerState, TunerUpdate } from "../lib/types";
import type { View } from "../lib/view";

export function createTunerView(): View {
  const element = document.createElement("section");
  element.className = "view tuner-view";
  element.innerHTML = `
    <div class="view-head">
      <div>
        <p class="eyebrow">Guitar tuner</p>
        <h1>Find your pitch</h1>
        <p class="lede">Play one string at a time and watch the needle settle. The detector works best with a clean, sustained note.</p>
      </div>
      <button class="btn btn-primary btn-lg" id="tuner-toggle" type="button">Start listening</button>
    </div>
    <div class="tuner-grid">
      <div class="panel tuner-display" id="tuner-display">
        <div class="tuner-note" id="tuner-note">—</div>
        <div class="tuner-status" id="tuner-status">Microphone idle</div>
        <div class="tuner-freq"><span id="tuner-hz">0.0</span><small>Hz</small></div>
        <div class="headstock-stage" id="tuner-headstock" data-status="idle">
          <svg class="guitar-head" viewBox="0 0 360 336" role="img" aria-label="Guitar headstock with six tuning pegs">
            <defs>
              <linearGradient id="headWood" x1="0" y1="0" x2="1" y2="1">
                <stop offset="0" stop-color="#3a2b1e"/>
                <stop offset="0.55" stop-color="#241b14"/>
                <stop offset="1" stop-color="#18120e"/>
              </linearGradient>
              <linearGradient id="neckWood" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0" stop-color="#2c2118"/>
                <stop offset="1" stop-color="#17100b"/>
              </linearGradient>
            </defs>
            <path class="head-body" d="M180 12C127 12 100 44 100 86v44c0 34 24 60 58 64h44c34-4 58-30 58-64V86c0-42-27-74-80-74Z"/>
            <path class="head-neck" d="M150 192h60l14 140h-88Z"/>
            <rect class="head-nut" x="146" y="192" width="68" height="7" rx="2"/>
            <line class="head-fret" x1="141" y1="238" x2="219" y2="238"/>
            <line class="head-fret" x1="138" y1="284" x2="222" y2="284"/>
            <circle class="head-inlay" cx="180" cy="261" r="2.5"/>
            <circle class="head-inlay" cx="180" cy="310" r="2.5"/>
            <g class="head-strings">
              <polyline class="string-line" data-string="4" points="103,64 173,195 172,332"/>
              <polyline class="string-line" data-string="5" points="103,110 163,195 156,332"/>
              <polyline class="string-line" data-string="6" points="103,156 153,195 140,332"/>
              <polyline class="string-line" data-string="3" points="257,64 187,195 188,332"/>
              <polyline class="string-line" data-string="2" points="257,110 197,195 204,332"/>
              <polyline class="string-line" data-string="1" points="257,156 207,195 220,332"/>
            </g>
            <g class="peg" data-string="4" transform="translate(103 64)">
              <title>String 4 · D3</title>
              <circle class="peg-knob" r="16"/>
              <circle class="peg-cap" r="6"/>
              <line class="peg-slot" x1="-4" y1="-4" x2="4" y2="4"/>
              <text class="peg-label" y="3.5">D</text>
            </g>
            <g class="peg" data-string="5" transform="translate(103 110)">
              <title>String 5 · A2</title>
              <circle class="peg-knob" r="16"/>
              <circle class="peg-cap" r="6"/>
              <line class="peg-slot" x1="-4" y1="-4" x2="4" y2="4"/>
              <text class="peg-label" y="3.5">A</text>
            </g>
            <g class="peg" data-string="6" transform="translate(103 156)">
              <title>String 6 · E2</title>
              <circle class="peg-knob" r="16"/>
              <circle class="peg-cap" r="6"/>
              <line class="peg-slot" x1="-4" y1="-4" x2="4" y2="4"/>
              <text class="peg-label" y="3.5">E</text>
            </g>
            <g class="peg" data-string="3" transform="translate(257 64)">
              <title>String 3 · G3</title>
              <circle class="peg-knob" r="16"/>
              <circle class="peg-cap" r="6"/>
              <line class="peg-slot" x1="-4" y1="-4" x2="4" y2="4"/>
              <text class="peg-label" y="3.5">G</text>
            </g>
            <g class="peg" data-string="2" transform="translate(257 110)">
              <title>String 2 · B3</title>
              <circle class="peg-knob" r="16"/>
              <circle class="peg-cap" r="6"/>
              <line class="peg-slot" x1="-4" y1="-4" x2="4" y2="4"/>
              <text class="peg-label" y="3.5">B</text>
            </g>
            <g class="peg" data-string="1" transform="translate(257 156)">
              <title>String 1 · E4</title>
              <circle class="peg-knob" r="16"/>
              <circle class="peg-cap" r="6"/>
              <line class="peg-slot" x1="-4" y1="-4" x2="4" y2="4"/>
              <text class="peg-label" y="3.5">E</text>
            </g>
          </svg>
          <div class="tuner-action" id="tuner-action" data-direction="idle">
            <span class="tuner-action-icon" id="tuner-action-icon">•</span>
            <span id="tuner-action-text">Play a string to tune</span>
          </div>
        </div>
        <div class="tuner-gauge">
          <div class="gauge-track" aria-hidden="true">
            <span class="gauge-zone"></span>
            <span class="gauge-tick gauge-tick-1"></span>
            <span class="gauge-tick gauge-tick-2"></span>
            <span class="gauge-tick gauge-tick-3"></span>
            <span class="gauge-center"></span>
            <span class="gauge-needle" id="tuner-needle"></span>
          </div>
          <div class="gauge-labels" aria-hidden="true"><span>Flat</span><span>0</span><span>Sharp</span></div>
          <div class="tuner-cents" id="tuner-cents">0.0 cents</div>
        </div>
        <div class="tuner-meta">
          <div><span class="label">Target</span><strong id="tuner-target">Auto detect</strong></div>
          <div><span class="label">Clarity</span><strong id="tuner-clarity">0%</strong></div>
        </div>
      </div>
      <div class="panel string-panel">
        <h2>Strings</h2>
        <p class="muted">Pick a string to lock the target, or use auto-detect to follow whatever you play.</p>
        <div class="string-list" id="tuner-strings"></div>
        <div class="hint">Standard tuning · A4 = 440 Hz · tolerance ±5 cents</div>
      </div>
    </div>
  `;

  const toggleButton = byId<HTMLButtonElement>(element, "tuner-toggle");
  const display = byId<HTMLDivElement>(element, "tuner-display");
  const noteEl = byId<HTMLDivElement>(element, "tuner-note");
  const statusEl = byId<HTMLDivElement>(element, "tuner-status");
  const hzEl = byId<HTMLSpanElement>(element, "tuner-hz");
  const needleEl = byId<HTMLSpanElement>(element, "tuner-needle");
  const centsEl = byId<HTMLDivElement>(element, "tuner-cents");
  const targetEl = byId<HTMLElement>(element, "tuner-target");
  const clarityEl = byId<HTMLElement>(element, "tuner-clarity");
  const stringsEl = byId<HTMLDivElement>(element, "tuner-strings");
  const headstockEl = byId<HTMLDivElement>(element, "tuner-headstock");
  const actionEl = byId<HTMLDivElement>(element, "tuner-action");
  const actionIconEl = byId<HTMLSpanElement>(element, "tuner-action-icon");
  const actionTextEl = byId<HTMLSpanElement>(element, "tuner-action-text");

  let strings: GuitarString[] = [];
  let running = false;
  let targetString = 0;

  interface FeedbackLock {
    stringNumber: number;
    note: string;
    targetHz: number;
    direction: string;
    lockedAt: number;
  }

  const feedbackHoldMs = 2000;
  let feedbackLock: FeedbackLock | null = null;

  function setActiveString(number: number): void {
    for (const peg of headstockEl.querySelectorAll<SVGGElement>(".peg")) {
      peg.classList.toggle("is-active", Number(peg.dataset.string) === number);
    }
    for (const line of headstockEl.querySelectorAll<SVGPolylineElement>(".string-line")) {
      line.classList.toggle("is-active", Number(line.dataset.string) === number);
    }
  }

  function renderTargetMarker(): void {
    for (const peg of headstockEl.querySelectorAll<SVGGElement>(".peg")) {
      peg.classList.toggle(
        "is-target",
        targetString !== 0 && Number(peg.dataset.string) === targetString,
      );
    }
  }

  function setAction(direction: string, text: string, icon: string): void {
    actionEl.dataset.direction = direction;
    actionIconEl.textContent = icon;
    actionTextEl.textContent = text;
  }

  function renderStrings(): void {
    stringsEl.replaceChildren();
    const options: Array<{ number: number; label: string; detail: string }> = [
      { number: 0, label: "Auto", detail: "Closest string" },
      ...strings.map((s) => ({
        number: s.number,
        label: s.note,
        detail: `${s.frequency.toFixed(2)} Hz · string ${s.number}`,
      })),
    ];

    for (const option of options) {
      const button = h("button", "string-button");
      button.type = "button";
      button.dataset.target = String(option.number);
      button.append(h("span", "string-label", option.label));
      button.append(h("span", "string-detail", option.detail));
      if (option.number === targetString) {
        button.classList.add("is-active");
      }
      button.addEventListener("click", () => {
        void selectTarget(option.number);
      });
      stringsEl.append(button);
    }
    renderTargetMarker();
  }

  function targetFor(number: number): GuitarString | undefined {
    return strings.find((s) => s.number === number);
  }

  function updateTargetLabel(): void {
    if (targetString === 0) {
      targetEl.textContent = "Auto detect";
      return;
    }
    const target = targetFor(targetString);
    targetEl.textContent = target
      ? `${target.note} · ${target.frequency.toFixed(2)} Hz`
      : `String ${targetString}`;
  }

  async function selectTarget(number: number): Promise<void> {
    targetString = number;
    feedbackLock = null;
    updateTargetLabel();
    renderStrings();
    renderTargetMarker();
    try {
      await TunerService.setTargetString(number);
    } catch (error) {
      showToast(errorMessage(error), "error");
    }
  }

  function applyState(state: TunerState): void {
    running = state.running;
    toggleButton.textContent = running ? "Stop listening" : "Start listening";
    toggleButton.classList.toggle("is-running", running);
    if (!running) {
      feedbackLock = null;
      statusEl.textContent = "Microphone idle";
      display.dataset.status = "idle";
      headstockEl.dataset.status = "idle";
      setActiveString(0);
      setAction("idle", "Play a string to tune", "•");
    }
  }

  function resetDisplay(message: string): void {
    feedbackLock = null;
    noteEl.textContent = "—";
    statusEl.textContent = message;
    hzEl.textContent = "0.0";
    centsEl.textContent = "0.0 cents";
    clarityEl.textContent = "0%";
    needleEl.style.left = "50%";
    display.dataset.status = "idle";
    headstockEl.dataset.status = "idle";
    setActiveString(0);
    setAction("idle", "Play a string to tune", "•");
  }

  function applyUpdate(update: TunerUpdate): void {
    const clarity = Math.round(clamp(update.clarity, 0, 1) * 100);
    clarityEl.textContent = `${clarity}%`;
    const now = performance.now();

    if (!update.hasSignal || !update.note) {
      if (feedbackLock && now - feedbackLock.lockedAt < feedbackHoldMs) {
        return;
      }
      feedbackLock = null;
      if (running) {
        resetDisplay("Listening…");
        statusEl.textContent = "Listening — play a string";
        setAction("idle", "Listening — play a string", "•");
      }
      updateTargetLabel();
      return;
    }

    const detectedStatus = update.inTune ? "in-tune" : update.cents < 0 ? "flat" : "sharp";
    if (!feedbackLock || now - feedbackLock.lockedAt >= feedbackHoldMs) {
      feedbackLock = {
        stringNumber: update.stringNumber,
        note: update.note,
        targetHz: update.targetHz,
        direction: detectedStatus,
        lockedAt: now,
      };
    }

    const lock = feedbackLock;
    const cents = clamp(
      1200 * Math.log2(update.frequency / Math.max(lock.targetHz, 1)),
      -50,
      50,
    );
    const pegName = lock.note.replace(/[0-9]+$/, "");

    noteEl.textContent = lock.note;
    hzEl.textContent = update.frequency.toFixed(1);
    needleEl.style.left = `${50 + (cents / 50) * 45}%`;
    centsEl.textContent = `${cents >= 0 ? "+" : ""}${cents.toFixed(1)} cents`;
    targetEl.textContent = `${lock.note} · ${lock.targetHz.toFixed(2)} Hz`;

    display.dataset.status = lock.direction;
    headstockEl.dataset.status = lock.direction;
    setActiveString(lock.stringNumber);
    renderTargetMarker();

    if (lock.direction === "in-tune") {
      statusEl.textContent = `In tune · ${lock.note} string`;
      setAction("in-tune", `In tune · ${pegName} string`, "✓");
    } else if (lock.direction === "flat") {
      statusEl.textContent = `Flat · tighten the ${pegName} peg by ${Math.abs(cents).toFixed(0)} cents`;
      setAction("flat", `Tighten the ${pegName} string`, "↻");
    } else {
      statusEl.textContent = `Sharp · loosen the ${pegName} peg by ${Math.abs(cents).toFixed(0)} cents`;
      setAction("sharp", `Loosen the ${pegName} string`, "↺");
    }
  }

  async function toggle(): Promise<void> {
    toggleButton.disabled = true;
    try {
      if (running) {
        await TunerService.stop();
      } else {
        await TunerService.start();
        resetDisplay("Listening…");
      }
    } catch (error) {
      showToast(errorMessage(error), "error");
      await refreshState();
    } finally {
      toggleButton.disabled = false;
    }
  }

  async function refreshState(): Promise<void> {
    try {
      applyState(await TunerService.state());
    } catch {
      applyState({ running: false, targetString });
    }
  }

  toggleButton.addEventListener("click", () => {
    void toggle();
  });

  onTunerUpdate(applyUpdate);
  onTunerState(applyState);

  return {
    element,
    activate() {
      void (async () => {
        try {
          if (strings.length === 0) {
            strings = await TunerService.strings();
            strings.sort((a, b) => b.number - a.number);
            renderStrings();
            updateTargetLabel();
          }
          const state = await TunerService.state();
          targetString = state.targetString;
          renderStrings();
          updateTargetLabel();
          applyState(state);
        } catch (error) {
          showToast(errorMessage(error), "error");
        }
      })();
    },
    deactivate() {
      if (running) {
        void TunerService.stop().catch(() => undefined);
      }
    },
  };
}
