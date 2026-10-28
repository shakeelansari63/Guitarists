import {
  TabService,
  TunerService,
  errorMessage,
  onTabPlayback,
} from "../lib/api";
import { byId, clamp, formatBytes, formatDate, h } from "../lib/dom";
import { showToast } from "../lib/toast";
import type {
  GuitarString,
  SongLineKind,
  SongType,
  Tab,
  TabBeat,
  TabLine,
  TabMeasure,
  TabSummary,
} from "../lib/types";
import type { View } from "../lib/view";

const STANDARD_TUNING = "E2 A2 D3 G3 B3 E4";

interface Cursor {
  line: number;
  measure: number;
  string: number;
}

function emptyMeasure(): TabMeasure {
  return { beats: [] };
}

function emptyLine(kind: SongLineKind = "tab", measures = 8): TabLine {
  if (kind === "chord") {
    return {
      type: "chord",
      lyrics: "",
      chord: "",
      pause: 0,
      measures: [],
    };
  }
  return {
    type: "tab",
    lyrics: "",
    chord: "",
    pause: 0,
    measures: Array.from({ length: measures }, emptyMeasure),
  };
}

function songTypeLabel(type: SongType): string {
  switch (type) {
    case "chord":
      return "Chords only";
    case "mixed":
      return "Tab + chords";
    default:
      return "Tab only";
  }
}

function lineKindLabel(kind: SongLineKind): string {
  return kind === "chord" ? "Chord" : "Tab";
}

function noteName(note: string): string {
  return note.replace(/\d+$/, "") || note;
}

function newSongDocument(type: SongType, title: string): Tab {
  return {
    version: 3,
    title,
    artist: "",
    tuning: STANDARD_TUNING,
    tempo: 120,
    type,
    strumming: "",
    lines: [emptyLine(type === "chord" ? "chord" : "tab")],
  };
}

export function createSongsView(): View {
  const element = document.createElement("section");
  element.className = "view tabs-view";
  element.innerHTML = `
    <div class="view-head">
      <div>
        <p class="eyebrow">Song editor</p>
        <h1>Write it down</h1>
        <p class="lede">Save songs as tablature, chords, or both. Chord songs keep lyrics with a chord line and an optional strumming pattern up top.</p>
      </div>
      <div class="head-actions">
        <button class="btn" id="tab-new" type="button">New song</button>
        <button class="btn btn-primary" id="tab-save" type="button">Save song</button>
      </div>
    </div>
    <div class="tab-layout">
      <div class="panel tab-editor-panel">
        <div class="tab-meta">
          <label class="field" for="tab-title">
            <span class="field-label">Title</span>
            <input id="tab-title" type="text" placeholder="Untitled" maxlength="200" />
          </label>
          <label class="field" for="tab-artist">
            <span class="field-label">Artist</span>
            <input id="tab-artist" type="text" placeholder="Unknown" maxlength="200" />
          </label>
          <label class="field field-narrow" for="tab-tempo">
            <span class="field-label">Tempo</span>
            <input id="tab-tempo" type="number" min="0" max="400" step="1" value="120" />
          </label>
          <label class="field field-name" for="tab-name">
            <span class="field-label">File name <em id="tab-dirty">saved</em></span>
            <input id="tab-name" type="text" placeholder="Song name" maxlength="100" />
          </label>
        </div>
        <div class="song-mode-row">
          <span class="song-type-badge" id="song-type" data-type="tab">Tab only</span>
          <label class="field field-strumming" id="song-strumming-field" for="song-strumming" hidden>
            <span class="field-label">Strumming pattern</span>
            <input id="song-strumming" type="text" placeholder="e.g. D DU UDU" maxlength="200" />
          </label>
        </div>
        <div class="tab-toolbar">
          <span class="tuning-badge" id="tab-tuning">E2 A2 D3 G3 B3 E4</span>
          <div class="toolbar-actions">
            <button class="btn btn-small btn-play" id="tab-play" type="button">Play</button>
            <button class="btn btn-small" id="tab-add-line" type="button">Add tab line</button>
            <button class="btn btn-small" id="tab-add-chord-line" type="button">Add chord line</button>
            <button class="btn btn-small" id="tab-clone-line" type="button">Clone line</button>
            <button class="btn btn-small" id="tab-remove-line" type="button">Remove line</button>
            <button class="btn btn-small" id="tab-add-measure" type="button">Add position</button>
            <button class="btn btn-small" id="tab-remove-measure" type="button">Remove position</button>
          </div>
        </div>
        <div class="tab-scroll">
          <div class="tab-lines" id="tab-lines" tabindex="0" role="grid" aria-label="Song editor"></div>
        </div>
        <p class="hint" id="song-hint">Click a string position on a tab line, then type a fret from 0 to 24. Arrow keys move along the staff. Backspace or R clears a note. Add chord lines to write lyrics with a chord above them. Press Ctrl+S (⌘S on macOS) to save.</p>
      </div>
      <aside class="panel tab-library">
        <div class="library-head">
          <h2>Songs</h2>
          <button class="btn btn-small" id="tab-refresh" type="button">Refresh</button>
        </div>
        <p class="hint tab-dir" id="tab-dir"></p>
        <div class="tab-list" id="tab-list"></div>
      </aside>
    </div>
    <dialog class="song-dialog" id="tab-new-dialog">
      <form class="song-dialog-form" id="tab-new-form">
        <div>
          <p class="eyebrow">New song</p>
          <h2>Start a song</h2>
        </div>
        <label class="field" for="new-song-title">
          <span class="field-label">Title</span>
          <input id="new-song-title" type="text" placeholder="Untitled" maxlength="200" />
        </label>
        <label class="field" for="new-song-name">
          <span class="field-label">File name</span>
          <input id="new-song-name" type="text" placeholder="Song name" maxlength="100" />
        </label>
        <fieldset class="song-type-options">
          <legend>Song type</legend>
          <label class="song-type-option">
            <input type="radio" name="song-type" value="tab" checked />
            <span>
              <strong>Tab only</strong>
              <small>Six-string tablature with lyrics for each line.</small>
            </span>
          </label>
          <label class="song-type-option">
            <input type="radio" name="song-type" value="chord" />
            <span>
              <strong>Chords only</strong>
              <small>Lyrics with a chord line and a strumming pattern.</small>
            </span>
          </label>
          <label class="song-type-option">
            <input type="radio" name="song-type" value="mixed" />
            <span>
              <strong>Tab + chords</strong>
              <small>Mix tablature lines and chord lines in one song.</small>
            </span>
          </label>
        </fieldset>
        <div class="song-dialog-actions">
          <button class="btn" id="new-song-cancel" type="button">Cancel</button>
          <button class="btn btn-primary" type="submit">Create song</button>
        </div>
      </form>
    </dialog>
  `;

  const linesEl = byId<HTMLDivElement>(element, "tab-lines");
  const titleInput = byId<HTMLInputElement>(element, "tab-title");
  const artistInput = byId<HTMLInputElement>(element, "tab-artist");
  const tempoInput = byId<HTMLInputElement>(element, "tab-tempo");
  const nameInput = byId<HTMLInputElement>(element, "tab-name");
  const tuningEl = byId<HTMLElement>(element, "tab-tuning");
  const dirtyEl = byId<HTMLElement>(element, "tab-dirty");
  const typeEl = byId<HTMLElement>(element, "song-type");
  const strummingField = byId<HTMLElement>(element, "song-strumming-field");
  const strummingInput = byId<HTMLInputElement>(element, "song-strumming");
  const listEl = byId<HTMLDivElement>(element, "tab-list");
  const dirEl = byId<HTMLParagraphElement>(element, "tab-dir");
  const removeLineButton = byId<HTMLButtonElement>(element, "tab-remove-line");
  const addLineButton = byId<HTMLButtonElement>(element, "tab-add-line");
  const addChordLineButton = byId<HTMLButtonElement>(element, "tab-add-chord-line");
  const addMeasureButton = byId<HTMLButtonElement>(element, "tab-add-measure");
  const removeMeasureButton = byId<HTMLButtonElement>(element, "tab-remove-measure");
  const playButton = byId<HTMLButtonElement>(element, "tab-play");
  const newDialog = byId<HTMLDialogElement>(element, "tab-new-dialog");
  const newForm = byId<HTMLFormElement>(element, "tab-new-form");
  const newTitleInput = byId<HTMLInputElement>(element, "new-song-title");
  const newNameInput = byId<HTMLInputElement>(element, "new-song-name");

  let tab: Tab = newSongDocument("tab", "");
  let strings: GuitarString[] = [];
  let cursor: Cursor = { line: 0, measure: 0, string: 1 };
  let currentName = "";
  let dirty = false;
  let initialized = false;
  let active = false;
  let playing = false;
  let playingLine = -1;
  let fretBuffer = "";
  let slideBuffer = "";
  let slideMode = false;
  let fretTimer: number | undefined;

  interface PlaybackStep {
    line: number;
    measure: number;
    strings: number[];
    start: number;
    duration: number;
  }

  const playbackCells = new Map<string, HTMLElement>();
  let playbackSteps: PlaybackStep[] = [];
  let playbackStepIndex = 0;
  let playbackStartedAt = 0;
  let playbackFrame: number | undefined;
  let highlightedKeys: string[] = [];

  function noteFor(stringNumber: number): string {
    return strings.find((s) => s.number === stringNumber)?.note ?? `S${stringNumber}`;
  }

  function activeLine(): TabLine {
    return tab.lines[cursor.line] ?? tab.lines[0];
  }

  function hasTabNotes(): boolean {
    return tab.lines.some(
      (line) => line.type === "tab" && line.measures.some((measure) => measure.beats.length > 0),
    );
  }

  function markDirty(value = true): void {
    dirty = value;
    dirtyEl.textContent = value ? "unsaved" : "saved";
    dirtyEl.classList.toggle("is-dirty", value);
  }

  function updateToolbarState(): void {
    const line = activeLine();
    const tabLineActive = line?.type === "tab";
    addLineButton.hidden = tab.type === "chord";
    addChordLineButton.hidden = tab.type === "tab";
    addMeasureButton.hidden = tab.type === "chord" || !tabLineActive;
    removeMeasureButton.hidden = tab.type === "chord" || !tabLineActive;
    addMeasureButton.disabled = !tabLineActive;
    removeMeasureButton.disabled = !tabLineActive;
    removeLineButton.disabled = tab.lines.length <= 1;
    tuningEl.hidden = tab.type === "chord";
    playButton.hidden = tab.type === "chord";
    playButton.disabled = !hasTabNotes();
  }

  function syncForm(): void {
    titleInput.value = tab.title;
    artistInput.value = tab.artist;
    tempoInput.value = String(tab.tempo);
    nameInput.value = currentName;
    tuningEl.textContent = tab.tuning || "Standard";
    typeEl.textContent = songTypeLabel(tab.type);
    typeEl.dataset.type = tab.type;
    strummingInput.value = tab.strumming;
    strummingField.hidden = tab.type === "tab";
    markDirty(dirty);
    updateToolbarState();
  }

  function beatAt(measure: TabMeasure | undefined, stringNumber: number): TabBeat | undefined {
    return measure?.beats.find((beat) => beat.string === stringNumber);
  }

  function ensureMeasure(line: TabLine, index: number): TabMeasure {
    while (line.measures.length <= index) {
      line.measures.push(emptyMeasure());
    }
    return line.measures[index];
  }

  function renderRuler(line: TabLine, lineIndex: number): HTMLElement[] {
    const nodes: HTMLElement[] = [h("div", "tab-ruler-label")];
    for (let measureIndex = 0; measureIndex < line.measures.length; measureIndex++) {
      const cell = h("div", "tab-ruler-cell", String(measureIndex + 1));
      if (lineIndex === cursor.line && measureIndex === cursor.measure) {
        cell.classList.add("is-active");
      }
      nodes.push(cell);
    }
    return nodes;
  }

  function createLyricsInput(
    line: TabLine,
    lineIndex: number,
    className = "tab-lyrics",
  ): HTMLInputElement {
    const lyrics = document.createElement("input");
    lyrics.type = "text";
    lyrics.className = className;
    lyrics.value = line.lyrics;
    lyrics.placeholder = "Lyrics for this line…";
    lyrics.maxLength = 2000;
    lyrics.dataset.line = String(lineIndex);
    lyrics.setAttribute("aria-label", `Lyrics for line ${lineIndex + 1}`);
    lyrics.addEventListener("input", () => {
      line.lyrics = lyrics.value;
      markDirty();
    });
    lyrics.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        lyrics.blur();
        if (line.type === "tab") {
          cursor.line = lineIndex;
          clampCursor();
          clearFretBuffer();
          renderLines();
          linesEl.focus();
        } else {
          focusNextLine(lineIndex);
        }
      } else if (event.key === "Tab") {
        event.preventDefault();
        if (event.shiftKey) {
          if (lineIndex > 0) {
            focusLyrics(lineIndex - 1);
          }
        } else if (line.type === "tab") {
          cursor.line = lineIndex;
          clampCursor();
          clearFretBuffer();
          renderLines();
          linesEl.focus();
        } else {
          focusNextLine(lineIndex);
        }
      }
    });
    return lyrics;
  }

  function selectLine(lineIndex: number): void {
    if (cursor.line === lineIndex) {
      return;
    }
    cursor.line = lineIndex;
    cursor.measure = 0;
    cursor.string = 1;
    for (const node of linesEl.querySelectorAll<HTMLElement>(".tab-line")) {
      node.classList.toggle("is-active", Number(node.dataset.line) === lineIndex);
    }
    updateToolbarState();
  }

  function renderLine(line: TabLine, lineIndex: number): HTMLElement {
    const lineEl = h("section", "tab-line");
    lineEl.dataset.line = String(lineIndex);
    lineEl.setAttribute("role", "rowgroup");
    lineEl.setAttribute("aria-label", `Line ${lineIndex + 1}`);
    lineEl.classList.toggle("is-chord", line.type === "chord");
    if (lineIndex === cursor.line) {
      lineEl.classList.add("is-active");
    }
    lineEl.addEventListener("click", () => selectLine(lineIndex));
    lineEl.addEventListener("focusin", () => selectLine(lineIndex));

    const head = h("div", "tab-line-head");
    head.append(h("span", "line-label", `Line ${lineIndex + 1}`));
    head.append(h("span", `line-kind line-kind-${line.type}`, lineKindLabel(line.type)));

    if (line.type === "chord") {
      lineEl.append(head);

      const body = h("div", "chord-line-body");
      const chordField = h("label", "field chord-field");
      chordField.append(h("span", "field-label", "Chord"));
      const chordInput = document.createElement("input");
      chordInput.type = "text";
      chordInput.className = "chord-input";
      chordInput.value = line.chord;
      chordInput.placeholder = "e.g. G, Am, Bsus2 or a short progression";
      chordInput.maxLength = 200;
      chordInput.dataset.line = String(lineIndex);
      chordInput.setAttribute("aria-label", `Chord for line ${lineIndex + 1}`);
      chordInput.addEventListener("input", () => {
        line.chord = chordInput.value;
        markDirty();
      });
      chordInput.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || (event.key === "Tab" && !event.shiftKey)) {
          event.preventDefault();
          focusLyrics(lineIndex);
        } else if (event.key === "Tab" && event.shiftKey && lineIndex > 0) {
          event.preventDefault();
          focusLyrics(lineIndex - 1);
        }
      });
      chordField.append(chordInput);

      const lyricsField = h("label", "field chord-lyrics-field");
      lyricsField.append(h("span", "field-label", "Lyrics"));
      lyricsField.append(createLyricsInput(line, lineIndex, "tab-lyrics chord-lyrics"));
      body.append(chordField, lyricsField);
      lineEl.append(body);
      return lineEl;
    }

    const linePlay = h("button", "line-play");
    linePlay.type = "button";
    linePlay.dataset.line = String(lineIndex);
    const lineIsPlaying = playing && playingLine === lineIndex;
    linePlay.textContent = lineIsPlaying ? "Stop" : "Play";
    linePlay.classList.toggle("is-playing", lineIsPlaying);
    linePlay.disabled = !lineIsPlaying && line.measures.every((measure) => measure.beats.length === 0);
    linePlay.setAttribute("aria-label", `${lineIsPlaying ? "Stop" : "Play"} line ${lineIndex + 1}`);
    linePlay.addEventListener("click", (event) => {
      event.stopPropagation();
      if (playing && playingLine === lineIndex) {
        void stopPlayback();
      } else {
        void playLine(lineIndex);
      }
    });
    head.append(linePlay);
    head.append(createLyricsInput(line, lineIndex));

    const pauseLabel = h("label", "line-pause");
    pauseLabel.append(h("span", "line-pause-label", "Pause"));
    const pauseSelect = document.createElement("select");
    pauseSelect.setAttribute("aria-label", `Pause after line ${lineIndex + 1}`);
    for (const beats of [0, 1, 2, 4]) {
      const option = document.createElement("option");
      option.value = String(beats);
      option.textContent = beats === 0 ? "none" : `${beats} beat${beats === 1 ? "" : "s"}`;
      pauseSelect.append(option);
    }
    pauseSelect.value = String(line.pause);
    pauseSelect.addEventListener("change", () => {
      line.pause = Number(pauseSelect.value) || 0;
      markDirty();
    });
    pauseLabel.append(pauseSelect);
    head.append(pauseLabel);
    head.append(h("span", "line-count", `${line.measures.length} pos`));
    lineEl.append(head);

    const staff = h("div", "tab-staff");
    staff.style.gridTemplateColumns = `34px repeat(${line.measures.length}, minmax(34px, 1fr))`;
    for (const node of renderRuler(line, lineIndex)) {
      staff.append(node);
    }

    for (let stringNumber = 1; stringNumber <= 6; stringNumber++) {
      const label = h("div", "tab-string-label");
      const fullNote = noteFor(stringNumber);
      label.title = `String ${stringNumber} (${fullNote})`;
      const shortNote = noteName(fullNote);
      label.append(h("span", "string-note", stringNumber === 1 ? shortNote.toLowerCase() : shortNote));
      staff.append(label);

      for (let measureIndex = 0; measureIndex < line.measures.length; measureIndex++) {
        const measure = line.measures[measureIndex];
        const beat = beatAt(measure, stringNumber);
        const fret = beat?.fret;
        const slideDraft =
          slideMode &&
          lineIndex === cursor.line &&
          measureIndex === cursor.measure &&
          stringNumber === cursor.string &&
          fret !== undefined;
        const cell = h("button", "tab-note");
        cell.type = "button";
        cell.tabIndex = -1;
        cell.dataset.line = String(lineIndex);
        cell.dataset.measure = String(measureIndex);
        cell.dataset.string = String(stringNumber);
        playbackCells.set(`${lineIndex}:${measureIndex}:${stringNumber}`, cell);
        cell.setAttribute("role", "gridcell");
        const slideText = slideDraft ? slideBuffer : beat?.slideTo !== undefined ? String(beat.slideTo) : "";
        cell.setAttribute(
          "aria-label",
          `Line ${lineIndex + 1}, position ${measureIndex + 1}, string ${stringNumber}${fret === undefined ? ", empty" : `, fret ${fret}${slideText ? ` sliding to ${slideText}` : ""}`}`,
        );

        if (lineIndex === cursor.line && measureIndex === cursor.measure && stringNumber === cursor.string) {
          cell.classList.add("is-cursor");
          cell.setAttribute("aria-selected", "true");
        }
        if (fret === undefined) {
          cell.classList.add("is-empty");
        } else {
          const label = slideDraft ? `${fret}/${slideBuffer}` : slideText ? `${fret}/${slideText}` : String(fret);
          cell.append(h("span", "fret-value", label));
        }
        staff.append(cell);
      }
    }

    lineEl.append(staff);
    return lineEl;
  }

  function renderLines(): void {
    playbackCells.clear();
    linesEl.replaceChildren(...tab.lines.map(renderLine));
    updateToolbarState();
  }

  function clampCursor(): void {
    cursor.line = clamp(cursor.line, 0, Math.max(tab.lines.length - 1, 0));
    const line = activeLine();
    cursor.measure = clamp(cursor.measure, 0, Math.max(line.measures.length - 1, 0));
    cursor.string = clamp(cursor.string, 1, 6);
  }

  function clearFretBuffer(): void {
    window.clearTimeout(fretTimer);
    fretBuffer = "";
    slideBuffer = "";
    slideMode = false;
  }

  function adjacentTabLine(from: number, direction: number): number {
    let target = from + direction;
    while (target >= 0 && target < tab.lines.length && tab.lines[target].type !== "tab") {
      target += direction;
    }
    if (target < 0 || target >= tab.lines.length) {
      return from;
    }
    return target;
  }

  function moveCursor(measureDelta: number, stringDelta: number): void {
    if (activeLine().type !== "tab") {
      return;
    }
    clearFretBuffer();
    if (measureDelta !== 0) {
      let target = cursor.measure + measureDelta;
      if (target < 0) {
        const previous = adjacentTabLine(cursor.line, -1);
        if (previous !== cursor.line) {
          cursor.line = previous;
          target = activeLine().measures.length - 1;
        } else {
          target = 0;
        }
      } else if (target >= activeLine().measures.length) {
        const next = adjacentTabLine(cursor.line, 1);
        if (next !== cursor.line) {
          cursor.line = next;
          target = 0;
        } else {
          target = activeLine().measures.length - 1;
        }
      }
      cursor.measure = clamp(target, 0, Math.max(activeLine().measures.length - 1, 0));
    }
    cursor.string = clamp(cursor.string + stringDelta, 1, 6);
    clampCursor();
    renderLines();
    linesEl.focus();
  }

  function setFret(fret: number): void {
    if (fret < 0 || fret > 24) {
      return;
    }
    const line = activeLine();
    if (line.type !== "tab") {
      return;
    }
    const measure = ensureMeasure(line, cursor.measure);
    measure.beats = measure.beats.filter((beat) => beat.string !== cursor.string);
    measure.beats.push({ string: cursor.string, fret });
    measure.beats.sort((a, b) => a.string - b.string);
    markDirty();
    renderLines();
  }

  function clearCell(): void {
    clearFretBuffer();
    const line = activeLine();
    if (line.type !== "tab") {
      return;
    }
    const measure = line.measures[cursor.measure];
    if (!measure) {
      return;
    }
    const next = measure.beats.filter((beat) => beat.string !== cursor.string);
    if (next.length !== measure.beats.length) {
      measure.beats = next;
      markDirty();
      renderLines();
    }
  }

  function typeDigit(digit: string): void {
    const candidate = fretBuffer + digit;
    let value = Number(candidate);
    if (candidate.length > 1 && value > 24) {
      fretBuffer = digit;
      value = Number(digit);
    } else {
      fretBuffer = candidate;
    }
    setFret(value);
    window.clearTimeout(fretTimer);
    fretTimer = window.setTimeout(() => {
      fretBuffer = "";
    }, 800);
  }

  function startSlide(): void {
    const line = activeLine();
    if (line.type !== "tab") {
      return;
    }
    const beat = beatAt(line.measures[cursor.measure], cursor.string);
    if (!beat) {
      return;
    }
    if (beat.slideTo !== undefined) {
      beat.slideTo = undefined;
      markDirty();
      renderLines();
      return;
    }
    slideMode = true;
    slideBuffer = "";
    window.clearTimeout(fretTimer);
    fretTimer = window.setTimeout(() => {
      slideMode = false;
      slideBuffer = "";
      renderLines();
    }, 800);
    renderLines();
  }

  function typeSlideDigit(digit: string): void {
    const line = activeLine();
    if (line.type !== "tab") {
      return;
    }
    const beat = beatAt(line.measures[cursor.measure], cursor.string);
    if (!beat) {
      clearFretBuffer();
      return;
    }
    const candidate = slideBuffer + digit;
    let value = Number(candidate);
    if (candidate.length > 1 && value > 24) {
      slideBuffer = digit;
      value = Number(digit);
    } else {
      slideBuffer = candidate;
    }
    beat.slideTo = value;
    markDirty();
    renderLines();
    window.clearTimeout(fretTimer);
    fretTimer = window.setTimeout(() => {
      slideMode = false;
      slideBuffer = "";
      renderLines();
    }, 800);
  }

  function commitPending(): void {
    clearFretBuffer();
    renderLines();
  }

  linesEl.addEventListener("click", (event) => {
    const target = event.target as HTMLElement;
    const cell = target.closest<HTMLElement>(".tab-note");
    if (!cell) {
      return;
    }
    cursor = {
      line: Number(cell.dataset.line ?? 0),
      measure: Number(cell.dataset.measure ?? 0),
      string: Number(cell.dataset.string ?? 1),
    };
    clearFretBuffer();
    clampCursor();
    renderLines();
    linesEl.focus();
  });

  linesEl.addEventListener("keydown", (event) => {
    if (event.target instanceof HTMLInputElement) {
      return;
    }
    if (event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }
    if (activeLine().type !== "tab") {
      return;
    }
    if (event.key === "/") {
      event.preventDefault();
      startSlide();
      return;
    }
    if (/^[0-9]$/.test(event.key)) {
      event.preventDefault();
      if (slideMode) {
        typeSlideDigit(event.key);
      } else {
        typeDigit(event.key);
      }
      return;
    }
    if (event.key === "Tab") {
      event.preventDefault();
      if (event.shiftKey) {
        focusLyrics(cursor.line);
      } else if (cursor.line + 1 < tab.lines.length) {
        focusLyrics(cursor.line + 1);
      } else {
        addLine("tab");
        focusLyrics(cursor.line);
      }
      return;
    }
    switch (event.key) {
      case "ArrowLeft":
        event.preventDefault();
        moveCursor(-1, 0);
        break;
      case "ArrowRight":
        event.preventDefault();
        moveCursor(1, 0);
        break;
      case "ArrowUp":
        event.preventDefault();
        moveCursor(0, -1);
        break;
      case "ArrowDown":
        event.preventDefault();
        moveCursor(0, 1);
        break;
      case "Home":
        event.preventDefault();
        clearFretBuffer();
        cursor.measure = 0;
        renderLines();
        break;
      case "End":
        event.preventDefault();
        clearFretBuffer();
        cursor.measure = activeLine().measures.length - 1;
        renderLines();
        break;
      case "Backspace":
      case "Delete":
        event.preventDefault();
        clearCell();
        break;
      case "r":
      case "R":
        event.preventDefault();
        clearCell();
        break;
      case "Enter":
        event.preventDefault();
        commitPending();
        break;
      case "Escape":
        clearFretBuffer();
        renderLines();
        break;
      default:
        break;
    }
  });

  function normalizeLine(line: Partial<TabLine> | undefined, kind: SongLineKind): TabLine {
    const measures =
      kind === "tab" && Array.isArray(line?.measures)
        ? line.measures.map((measure) => ({
            ...measure,
            beats: Array.isArray(measure.beats) ? measure.beats : [],
          }))
        : [];
    return {
      type: kind,
      lyrics: typeof line?.lyrics === "string" ? line.lyrics : "",
      chord: kind === "chord" && typeof line?.chord === "string" ? line.chord : "",
      pause: typeof line?.pause === "number" ? line.pause : 0,
      measures: kind === "tab" ? (measures.length ? measures : [emptyMeasure()]) : [],
    };
  }

  function normalizeTab(tabData: Tab): Tab {
    const raw = tabData as Partial<Tab> & {
      measures?: TabMeasure[];
      lines?: Array<Partial<TabLine>>;
    };
    const rawType: SongType =
      raw.type === "chord" || raw.type === "mixed" || raw.type === "tab" ? raw.type : "tab";
    const kindFor = (line: Partial<TabLine> | undefined): SongLineKind => {
      if (rawType === "tab") {
        return "tab";
      }
      if (rawType === "chord") {
        return "chord";
      }
      if (line?.type === "chord" || line?.type === "tab") {
        return line.type;
      }
      if (typeof line?.chord === "string" && line.chord !== "" && !line.measures?.length) {
        return "chord";
      }
      return "tab";
    };
    const rawLines: Array<Partial<TabLine>> =
      Array.isArray(raw.lines) && raw.lines.length > 0
        ? raw.lines
        : [{ measures: raw.measures }];
    const lines = rawLines.map((line) => normalizeLine(line, kindFor(line)));
    return {
      version: 3,
      title: typeof raw.title === "string" ? raw.title : "",
      artist: typeof raw.artist === "string" ? raw.artist : "",
      tuning: typeof raw.tuning === "string" && raw.tuning ? raw.tuning : STANDARD_TUNING,
      tempo: typeof raw.tempo === "number" ? raw.tempo : 120,
      type: rawType,
      strumming: typeof raw.strumming === "string" ? raw.strumming : "",
      lines: lines.length ? lines : [emptyLine(rawType === "chord" ? "chord" : "tab")],
    };
  }

  function applyLoaded(tabData: Tab, name: string): void {
    if (playing) {
      void stopPlayback();
    }
    tab = normalizeTab(tabData);
    currentName = name;
    cursor = { line: 0, measure: 0, string: 1 };
    clearFretBuffer();
    dirty = false;
    syncForm();
    renderLines();
  }

  function updateMeta(): void {
    tab.title = titleInput.value;
    tab.artist = artistInput.value;
    tab.tempo = clamp(Number(tempoInput.value) || 0, 0, 400);
    markDirty();
  }

  titleInput.addEventListener("input", updateMeta);
  artistInput.addEventListener("input", updateMeta);
  tempoInput.addEventListener("input", updateMeta);
  nameInput.addEventListener("input", () => {
    currentName = nameInput.value;
    markDirty();
  });
  strummingInput.addEventListener("input", () => {
    tab.strumming = strummingInput.value;
    markDirty();
  });

  function addLine(kind?: SongLineKind): void {
    const requested = kind ?? (tab.type === "chord" ? "chord" : "tab");
    const measureCount = tab.lines.find((line) => line.type === "tab")?.measures.length ?? 8;
    const insertAt = cursor.line + 1;
    tab.lines.splice(insertAt, 0, emptyLine(requested, measureCount));
    cursor = { line: insertAt, measure: 0, string: 1 };
    markDirty();
    renderLines();
    if (requested === "chord") {
      focusChord(insertAt);
    } else {
      linesEl.focus();
    }
  }

  function focusLyrics(lineIndex: number): void {
    cursor.line = clamp(lineIndex, 0, tab.lines.length - 1);
    clampCursor();
    clearFretBuffer();
    renderLines();
    const input = linesEl.querySelector<HTMLInputElement>(
      `.tab-lyrics[data-line="${cursor.line}"]`,
    );
    input?.focus();
  }

  function focusChord(lineIndex: number): void {
    cursor.line = clamp(lineIndex, 0, tab.lines.length - 1);
    clampCursor();
    clearFretBuffer();
    renderLines();
    const input = linesEl.querySelector<HTMLInputElement>(
      `.chord-input[data-line="${cursor.line}"]`,
    );
    input?.focus();
  }

  function focusNextLine(lineIndex: number): void {
    if (lineIndex + 1 < tab.lines.length) {
      focusLyrics(lineIndex + 1);
      return;
    }
    const kind = tab.lines[lineIndex]?.type === "chord" ? "chord" : "tab";
    addLine(kind);
    focusLyrics(cursor.line);
  }

  addLineButton.addEventListener("click", () => {
    addLine("tab");
  });

  addChordLineButton.addEventListener("click", () => {
    addLine("chord");
  });

  byId<HTMLButtonElement>(element, "tab-clone-line").addEventListener("click", () => {
    const source = activeLine();
    const copy: TabLine =
      source.type === "chord"
        ? {
            type: "chord",
            lyrics: source.lyrics,
            chord: source.chord,
            pause: source.pause,
            measures: [],
          }
        : {
            type: "tab",
            lyrics: source.lyrics,
            chord: "",
            pause: source.pause,
            measures: source.measures.map((measure) => ({
              beats: measure.beats.map((beat) => ({ ...beat })),
            })),
          };
    const insertAt = cursor.line + 1;
    tab.lines.splice(insertAt, 0, copy);
    cursor = { line: insertAt, measure: 0, string: 1 };
    markDirty();
    renderLines();
    if (copy.type === "chord") {
      focusChord(insertAt);
    } else {
      linesEl.focus();
    }
  });

  removeLineButton.addEventListener("click", () => {
    if (tab.lines.length <= 1) {
      return;
    }
    tab.lines.splice(cursor.line, 1);
    cursor.line = Math.min(cursor.line, tab.lines.length - 1);
    clampCursor();
    markDirty();
    renderLines();
    linesEl.focus();
  });

  addMeasureButton.addEventListener("click", () => {
    const line = activeLine();
    if (line.type !== "tab") {
      return;
    }
    line.measures.push(emptyMeasure());
    cursor.measure = line.measures.length - 1;
    markDirty();
    renderLines();
    linesEl.focus();
  });

  removeMeasureButton.addEventListener("click", () => {
    const line = activeLine();
    if (line.type !== "tab") {
      return;
    }
    if (line.measures.length <= 1) {
      line.measures[0] = emptyMeasure();
    } else {
      line.measures.splice(cursor.measure, 1);
      cursor.measure = Math.min(cursor.measure, line.measures.length - 1);
    }
    markDirty();
    renderLines();
    linesEl.focus();
  });

  function buildPlaybackTimeline(line: number): PlaybackStep[] {
    const beatMs = 60000 / (tab.tempo > 0 ? tab.tempo : 120);
    const lineIndices = line < 0 ? tab.lines.map((_, index) => index) : [line];
    const steps: PlaybackStep[] = [];
    let start = 0;

    lineIndices.forEach((lineIndex, order) => {
      const source = tab.lines[lineIndex];
      if (!source || source.type !== "tab") {
        return;
      }
      source.measures.forEach((measure, measureIndex) => {
        steps.push({
          line: lineIndex,
          measure: measureIndex,
          strings: measure.beats.map((beat) => beat.string),
          start,
          duration: beatMs,
        });
        start += beatMs;
      });
      if (line < 0 && order < lineIndices.length - 1 && source.pause > 0) {
        start += source.pause * beatMs;
      }
    });
    return steps;
  }

  function applyPlaybackHighlight(step: PlaybackStep | undefined): void {
    for (const key of highlightedKeys) {
      playbackCells.get(key)?.classList.remove("is-playing");
    }
    highlightedKeys = step
      ? step.strings.map((stringNumber) => `${step.line}:${step.measure}:${stringNumber}`)
      : [];
    for (const key of highlightedKeys) {
      playbackCells.get(key)?.classList.add("is-playing");
    }
  }

  function stopPlaybackTicker(): void {
    if (playbackFrame !== undefined) {
      window.cancelAnimationFrame(playbackFrame);
      playbackFrame = undefined;
    }
    applyPlaybackHighlight(undefined);
    playbackSteps = [];
    playbackStepIndex = 0;
  }

  function startPlaybackTicker(line: number): void {
    stopPlaybackTicker();
    playbackSteps = buildPlaybackTimeline(line);
    playbackStepIndex = 0;
    playbackStartedAt = performance.now();

    const tick = () => {
      const elapsed = performance.now() - playbackStartedAt;
      while (
        playbackStepIndex < playbackSteps.length &&
        elapsed >= playbackSteps[playbackStepIndex].start + playbackSteps[playbackStepIndex].duration
      ) {
        playbackStepIndex += 1;
      }
      if (playbackStepIndex >= playbackSteps.length) {
        applyPlaybackHighlight(undefined);
        playbackFrame = undefined;
        return;
      }
      const current = playbackSteps[playbackStepIndex];
      applyPlaybackHighlight(elapsed >= current.start ? current : undefined);
      playbackFrame = window.requestAnimationFrame(tick);
    };

    playbackFrame = window.requestAnimationFrame(tick);
  }

  function applyPlaybackState(value: boolean, line: number): void {
    const wasPlaying = playing;
    const previousLine = playingLine;
    playing = value;
    playingLine = value ? line : -1;
    if (value) {
      if (!wasPlaying || previousLine !== line) {
        startPlaybackTicker(line);
      }
    } else {
      stopPlaybackTicker();
    }
    updatePlaybackButtons();
  }

  function updatePlaybackButtons(): void {
    playButton.textContent = playing ? "Stop" : "Play";
    playButton.classList.toggle("is-playing", playing);
    playButton.setAttribute("aria-label", playing ? "Stop song playback" : "Play whole song");
    for (const button of linesEl.querySelectorAll<HTMLButtonElement>(".line-play")) {
      const line = Number(button.dataset.line);
      const current = playing && playingLine === line;
      button.textContent = current ? "Stop" : "Play";
      button.classList.toggle("is-playing", current);
      button.setAttribute("aria-label", `${current ? "Stop" : "Play"} line ${line + 1}`);
    }
  }

  async function playTab(): Promise<void> {
    if (!hasTabNotes()) {
      showToast("Add some tab notes before playing.", "error");
      return;
    }
    try {
      await TabService.play(tab);
      applyPlaybackState(true, -1);
    } catch (error) {
      applyPlaybackState(false, -1);
      showToast(errorMessage(error), "error");
    }
  }

  async function playLine(line: number): Promise<void> {
    if (tab.lines[line]?.type !== "tab") {
      showToast("Chord lines have no tab notes to play.", "error");
      return;
    }
    try {
      await TabService.playLine(tab, line);
      applyPlaybackState(true, line);
    } catch (error) {
      applyPlaybackState(false, -1);
      showToast(errorMessage(error), "error");
    }
  }

  async function stopPlayback(): Promise<void> {
    try {
      await TabService.stop();
    } catch (error) {
      showToast(errorMessage(error), "error");
    } finally {
      applyPlaybackState(false, -1);
    }
  }

  async function refreshList(): Promise<void> {
    try {
      const summaries = await TabService.list();
      renderList(summaries);
    } catch (error) {
      showToast(errorMessage(error), "error");
    }
  }

  function renderList(summaries: TabSummary[]): void {
    listEl.replaceChildren();
    if (summaries.length === 0) {
      listEl.append(h("p", "empty-state", "No saved songs yet."));
      return;
    }
    for (const summary of summaries) {
      const row = h("div", "tab-list-row");
      if (summary.name === currentName) {
        row.classList.add("is-current");
      }
      const summaryType: SongType =
        summary.type === "chord" || summary.type === "mixed" ? summary.type : "tab";
      const loadButton = h("button", "tab-list-load");
      loadButton.type = "button";
      loadButton.append(h("span", "item-title", summary.title || summary.name.replace(/\.xml$/i, "")));
      loadButton.append(
        h(
          "span",
          "item-meta",
          `${songTypeLabel(summaryType)} · ${summary.name} · ${formatBytes(summary.size)} · ${formatDate(summary.modified)}`,
        ),
      );
      loadButton.addEventListener("click", () => {
        void loadTab(summary.name);
      });

      const deleteButton = h("button", "tab-list-delete", "Delete");
      deleteButton.type = "button";
      deleteButton.addEventListener("click", () => {
        void deleteTab(summary.name);
      });

      row.append(loadButton, deleteButton);
      listEl.append(row);
    }
  }

  async function loadTab(name: string): Promise<void> {
    try {
      const loaded = await TabService.load(name);
      applyLoaded(loaded, name);
      await refreshList();
      showToast(`Loaded ${name}`, "success");
    } catch (error) {
      showToast(errorMessage(error), "error");
    }
  }

  async function deleteTab(name: string): Promise<void> {
    if (!window.confirm(`Delete ${name}? This cannot be undone.`)) {
      return;
    }
    try {
      await TabService.remove(name);
      if (name === currentName) {
        currentName = "";
        syncForm();
      }
      await refreshList();
      showToast(`Deleted ${name}`, "success");
    } catch (error) {
      showToast(errorMessage(error), "error");
    }
  }

  function openNewSongDialog(): void {
    newTitleInput.value = "";
    newNameInput.value = "";
    const defaultType = newForm.querySelector<HTMLInputElement>(
      'input[name="song-type"][value="tab"]',
    );
    if (defaultType) {
      defaultType.checked = true;
    }
    if (!newDialog.open) {
      newDialog.showModal();
    }
    newTitleInput.focus();
  }

  function createNewSong(): void {
    const selected = newForm.querySelector<HTMLInputElement>(
      'input[name="song-type"]:checked',
    );
    const rawType = selected?.value ?? "tab";
    const type: SongType = rawType === "chord" || rawType === "mixed" ? rawType : "tab";
    const title = newTitleInput.value.trim();
    const name = newNameInput.value.trim();
    applyLoaded(newSongDocument(type, title), name);
    markDirty(true);
    newDialog.close();
    if (type === "chord") {
      focusChord(0);
    } else {
      linesEl.focus();
    }
  }

  newForm.addEventListener("submit", (event) => {
    event.preventDefault();
    createNewSong();
  });

  byId<HTMLButtonElement>(element, "new-song-cancel").addEventListener("click", () => {
    newDialog.close();
  });

  async function saveTab(): Promise<void> {
    const fallback = tab.title.trim() || "Untitled";
    const name = nameInput.value.trim() || fallback;
    try {
      const summary = await TabService.save(name, tab);
      currentName = summary.name;
      nameInput.value = summary.name;
      markDirty(false);
      await refreshList();
      showToast(`Saved ${summary.name}`, "success");
    } catch (error) {
      showToast(errorMessage(error), "error");
    }
  }

  byId<HTMLButtonElement>(element, "tab-new").addEventListener("click", () => {
    openNewSongDialog();
  });
  byId<HTMLButtonElement>(element, "tab-play").addEventListener("click", () => {
    if (playing) {
      void stopPlayback();
    } else {
      void playTab();
    }
  });
  byId<HTMLButtonElement>(element, "tab-save").addEventListener("click", () => {
    void saveTab();
  });
  byId<HTMLButtonElement>(element, "tab-refresh").addEventListener("click", () => {
    void refreshList();
  });

  onTabPlayback((state) => {
    applyPlaybackState(state.playing, state.line);
  });

  document.addEventListener("keydown", (event) => {
    if (!active || newDialog.open || !(event.ctrlKey || event.metaKey) || event.altKey) {
      return;
    }
    if (event.key.toLowerCase() !== "s") {
      return;
    }
    event.preventDefault();
    void saveTab();
  });

  renderLines();

  return {
    element,
    activate() {
      active = true;
      void (async () => {
        try {
          if (strings.length === 0) {
            strings = await TunerService.strings();
          }
        } catch {
          strings = [];
        }
        try {
          const dir = await TabService.directory();
          dirEl.textContent = dir;
        } catch {
          dirEl.textContent = "";
        }
        try {
          const state = await TabService.playbackState();
          applyPlaybackState(state.playing, state.line);
        } catch {
          applyPlaybackState(false, -1);
        }
        if (!initialized) {
          initialized = true;
          try {
            applyLoaded(await TabService.newTab(), "");
          } catch (error) {
            showToast(errorMessage(error), "error");
            renderLines();
          }
        } else {
          renderLines();
        }
        await refreshList();
      })();
    },
    deactivate() {
      active = false;
      clearFretBuffer();
      if (newDialog.open) {
        newDialog.close();
      }
    },
  };
}
