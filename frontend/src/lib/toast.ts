type ToastKind = "info" | "success" | "error";

let container: HTMLElement | null = null;

function getContainer(): HTMLElement {
  if (!container || !container.isConnected) {
    container = document.createElement("div");
    container.className = "toast-stack";
    container.setAttribute("role", "status");
    container.setAttribute("aria-live", "polite");
    document.body.append(container);
  }
  return container;
}

export function showToast(message: string, kind: ToastKind = "info"): void {
  const toast = document.createElement("div");
  toast.className = `toast toast-${kind}`;
  toast.textContent = message;
  getContainer().append(toast);

  window.setTimeout(() => {
    toast.classList.add("toast-leaving");
    window.setTimeout(() => toast.remove(), 220);
  }, 3200);
}
