export interface GuitarString {
  number: number;
  note: string;
  frequency: number;
}

export interface TunerState {
  running: boolean;
  targetString: number;
}

export interface TunerUpdate {
  hasSignal: boolean;
  frequency: number;
  clarity: number;
  rms: number;
  stringNumber: number;
  note: string;
  targetHz: number;
  cents: number;
  inTune: boolean;
  direction: "flat" | "sharp" | "in-tune" | string;
  autoDetect: boolean;
}

export type SongType = "tab" | "chord" | "mixed";
export type SongLineKind = "tab" | "chord";

export interface TabBeat {
  string: number;
  fret: number;
  slideTo?: number;
}

export interface TabMeasure {
  beats: TabBeat[];
}

export interface TabLine {
  type: SongLineKind;
  lyrics: string;
  chord: string;
  pause: number;
  measures: TabMeasure[];
}

export interface Tab {
  version: number;
  title: string;
  artist: string;
  tuning: string;
  tempo: number;
  type: SongType;
  strumming: string;
  lines: TabLine[];
}

export interface TabPlaybackState {
  playing: boolean;
  line: number;
}

export interface TabSummary {
  name: string;
  title: string;
  type: SongType;
  size: number;
  modified: string;
}

export interface MetronomeConfig {
  bpm: number;
  beatsPerMeasure: number;
  subdivision: number;
  accent: boolean;
}

export interface MetronomeState {
  config: MetronomeConfig;
  running: boolean;
}

export interface BeatEvent {
  index: number;
  measure: number;
  beat: number;
  subIndex: number;
  accent: boolean;
  bpm: number;
  subdivision: number;
  timestamp: number;
}
