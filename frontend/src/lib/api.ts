import { Call, Events } from "@wailsio/runtime";
import type {
  BeatEvent,
  GuitarString,
  MetronomeConfig,
  MetronomeState,
  Tab,
  TabPlaybackState,
  TabSummary,
  TunerState,
  TunerUpdate,
} from "./types";

const servicePackage = "github.com/freak/guitarists/internal/services";

function call<T>(service: string, method: string, ...args: unknown[]): Promise<T> {
  return Call.ByName(`${servicePackage}.${service}.${method}`, ...args) as Promise<T>;
}

export const TunerService = {
  strings: () => call<GuitarString[]>("TunerService", "Strings"),
  state: () => call<TunerState>("TunerService", "State"),
  start: () => call<void>("TunerService", "Start"),
  stop: () => call<void>("TunerService", "Stop"),
  setTargetString: (number: number) =>
    call<void>("TunerService", "SetTargetString", number),
};

export const MetronomeService = {
  state: () => call<MetronomeState>("MetronomeService", "State"),
  setConfig: (config: MetronomeConfig) =>
    call<void>("MetronomeService", "SetConfig", config),
  start: () => call<void>("MetronomeService", "Start"),
  stop: () => call<void>("MetronomeService", "Stop"),
  tap: () => call<number>("MetronomeService", "Tap"),
};

export const TabService = {
  directory: () => call<string>("TabService", "Directory"),
  newTab: () => call<Tab>("TabService", "New"),
  list: () => call<TabSummary[]>("TabService", "List"),
  load: (name: string) => call<Tab>("TabService", "Load", name),
  save: (name: string, tab: Tab) => call<TabSummary>("TabService", "Save", name, tab),
  remove: (name: string) => call<void>("TabService", "Delete", name),
  play: (tab: Tab) => call<void>("TabService", "Play", tab),
  playLine: (tab: Tab, line: number) => call<void>("TabService", "PlayLine", tab, line),
  stop: () => call<void>("TabService", "Stop"),
  playbackState: () => call<TabPlaybackState>("TabService", "PlaybackState"),
};

export const SystemService = {
  version: () => call<string>("SystemService", "Version"),
  tabsDirectory: () => call<string>("SystemService", "TabsDirectory"),
};

export function onTunerUpdate(callback: (update: TunerUpdate) => void): () => void {
  return Events.On("tuner:update", (event) => callback(event.data as TunerUpdate));
}

export function onTunerState(callback: (state: TunerState) => void): () => void {
  return Events.On("tuner:state", (event) => callback(event.data as TunerState));
}

export function onMetronomeBeat(callback: (beat: BeatEvent) => void): () => void {
  return Events.On("metronome:beat", (event) => callback(event.data as BeatEvent));
}

export function onMetronomeState(callback: (state: MetronomeState) => void): () => void {
  return Events.On("metronome:state", (event) => callback(event.data as MetronomeState));
}

export function onTabPlayback(callback: (state: TabPlaybackState) => void): () => void {
  return Events.On("tabs:playback", (event) => callback(event.data as TabPlaybackState));
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }
  if (typeof error === "string" && error.trim() !== "") {
    return error;
  }
  if (error && typeof error === "object" && "message" in error) {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string" && message.trim() !== "") {
      return message;
    }
  }
  return "Something went wrong. Please try again.";
}
