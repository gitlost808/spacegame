import type Phaser from 'phaser';

// 'native' renders at the full device pixel ratio. 'saver' caps the drawing buffer's density: at
// DPR 3 that is 4x fewer pixels to fill each frame, at the cost of slightly softer edges and text.
export type RenderQuality = 'native' | 'saver';
const qualityKey = 'galactic-skirmish:renderQuality';
const saverMaxDensity = 1.5;
let quality: RenderQuality = loadRenderQuality();

function loadRenderQuality(): RenderQuality {
  try {
    return localStorage.getItem(qualityKey) === 'saver' ? 'saver' : 'native';
  } catch {
    return 'native';
  }
}

export function getRenderQuality(): RenderQuality {
  return quality;
}

export function setRenderQuality(game: Phaser.Game, value: RenderQuality): void {
  quality = value;
  try {
    localStorage.setItem(qualityKey, value);
  } catch {
    // ignore, the setting just won't survive a reload
  }
  resizeToDisplay(game);
}

// Phaser 4 has no renderer resolution setting. Size its drawing buffer in device
// pixels and use Scale.NONE's zoom to keep the canvas at the parent's CSS size.
export function resizeToDisplay(game: Phaser.Game): void {
  const parent = game.canvas.parentElement;
  if (!parent) return;
  const { width, height } = parent.getBoundingClientRect();
  if (width <= 0 || height <= 0) return;
  const deviceDensity = window.devicePixelRatio || 1;
  const density = quality === 'saver' ? Math.min(deviceDensity, saverMaxDensity) : deviceDensity;
  const pixelWidth = Math.round(width * density);
  const pixelHeight = Math.round(height * density);
  if (game.scale.width === pixelWidth && game.scale.height === pixelHeight && game.scale.zoom === 1 / density) return;

  game.scale.resize(pixelWidth, pixelHeight);
  // Set zoom last so Phaser also resets CSS dimensions when returning to 1x.
  game.scale.setZoom(1 / density);
}

export function followDisplaySize(game: Phaser.Game): void {
  const resize = () => resizeToDisplay(game);
  const observer = new ResizeObserver(resize);
  observer.observe(game.canvas.parentElement!);
  window.addEventListener('resize', resize);

  // Moving between monitors can change density without changing the CSS size.
  let densityQuery: MediaQueryList;
  const watchDensity = () => {
    densityQuery?.removeEventListener('change', onDensityChange);
    densityQuery = window.matchMedia(`(resolution: ${window.devicePixelRatio || 1}dppx)`);
    densityQuery.addEventListener('change', onDensityChange);
  };
  const onDensityChange = () => {
    resize();
    watchDensity();
  };
  watchDensity();
  resize();
  game.events.once('destroy', () => {
    observer.disconnect();
    window.removeEventListener('resize', resize);
    densityQuery.removeEventListener('change', onDensityChange);
  });
}
