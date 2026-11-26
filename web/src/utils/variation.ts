// helpers that make repeated sound effects differ from each other
import type { SoundLayer } from './synth.ts';

type Rng = () => number;

// small seeded generator, so the rendered variants are the same on every boot
export function seededRng(seed: number): Rng {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

// a copy of the layers with pitch sweep, length, fade and volume each nudged by up to ±amount
// (0.1 = 10%), so every variant is a slightly different sound rather than the same one pitched
export function varyLayers(layers: SoundLayer[], rng: Rng, amount: number): SoundLayer[] {
  const jitter = () => 1 + (rng() * 2 - 1) * amount;
  return layers.map((layer) => {
    // one factor for the whole sweep keeps its shape, and a second tilts the end of it
    const pitch = jitter();
    return {
      ...layer,
      from: layer.from * pitch,
      to: layer.to * pitch * jitter(),
      duration: layer.duration * jitter(),
      volume: layer.volume * jitter(),
      ...(layer.decay === undefined ? {} : { decay: layer.decay * jitter() }),
      ...(layer.delay === undefined ? {} : { delay: layer.delay * jitter() }),
    };
  });
}

// a random variant index that is never the one played last, so back-to-back sounds always differ
export function pickVariant(count: number, last: number | undefined, rng: Rng = Math.random): number {
  if (count <= 1) return 0;
  if (last === undefined || last < 0 || last >= count) return Math.floor(rng() * count);
  const next = Math.floor(rng() * (count - 1));
  return next >= last ? next + 1 : next;
}
