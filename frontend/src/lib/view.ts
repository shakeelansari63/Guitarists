export interface View {
  element: HTMLElement;
  activate(): void;
  deactivate(): void;
}
