export const ARTWORK_STORAGE_KEY = "artwork:v1";

// Runs before paint so a reader who switched artwork off never sees it flash.
export const ARTWORK_BOOTSTRAP_SCRIPT = `try{if(localStorage.getItem("${ARTWORK_STORAGE_KEY}")==="off")document.documentElement.dataset.artwork="off"}catch{}`;

export function applyArtwork(on: boolean) {
  if (on) delete document.documentElement.dataset.artwork;
  else document.documentElement.dataset.artwork = "off";
  try {
    if (on) localStorage.removeItem(ARTWORK_STORAGE_KEY);
    else localStorage.setItem(ARTWORK_STORAGE_KEY, "off");
  } catch {}
}

export function readArtwork() {
  try {
    return localStorage.getItem(ARTWORK_STORAGE_KEY) !== "off";
  } catch {
    return true;
  }
}
