/** Removes the loading screen app.html shows until the app first renders; SvelteKit mounts next to it. */
export function endBootScreen(document: Document = globalThis.document): void {
  document.getElementById('boot')?.remove();
  delete document.documentElement.dataset.bootSidebar;
}
