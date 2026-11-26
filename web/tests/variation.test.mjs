import assert from 'node:assert/strict';
import { test } from 'node:test';
import { synthesize } from '../src/utils/synth.ts';
import { pickVariant, seededRng, varyLayers } from '../src/utils/variation.ts';

const layers = [
  { wave: 'noise', duration: 0.1, from: 4000, to: 300, volume: 0.3, decay: 1.5 },
  { wave: 'sine', duration: 0.2, from: 180, to: 50, volume: 0.3, delay: 0.01 },
];

test('seededRng repeats for a seed and differs between seeds', () => {
  const a = seededRng(1);
  const b = seededRng(1);
  const first = [a(), a(), a()];
  assert.deepEqual(first, [b(), b(), b()]);
  const other = seededRng(2);
  assert.notDeepEqual(first, [other(), other(), other()]);
  assert.ok(first.every((n) => n >= 0 && n < 1));
});

test('varyLayers keeps the shape, stays within the spread, and does not mutate the recipe', () => {
  const copy = structuredClone(layers);
  const varied = varyLayers(layers, seededRng(7), 0.2);
  assert.deepEqual(layers, copy);
  assert.equal(varied.length, layers.length);
  for (const [i, layer] of varied.entries()) {
    assert.equal(layer.wave, layers[i].wave);
    assert.ok(layer.duration >= layers[i].duration * 0.8 && layer.duration <= layers[i].duration * 1.2);
    assert.ok(layer.volume >= layers[i].volume * 0.8 && layer.volume <= layers[i].volume * 1.2);
    assert.notEqual(layer.from, layers[i].from);
  }
  assert.equal(varied[0].delay, undefined);
  assert.ok(varied[1].delay > 0);
});

test('varied layers render to different, reproducible samples', () => {
  const render = (seed) => {
    const rng = seededRng(seed);
    return synthesize(varyLayers(layers, rng, 0.2), 8000, rng);
  };
  assert.deepEqual(render(3), render(3));
  assert.notDeepEqual(render(3), render(4));
});

test('pickVariant never repeats the last variant and reaches all others', () => {
  const seen = new Set();
  for (let i = 0; i < 500; i++) {
    const next = pickVariant(4, 2);
    assert.ok(next >= 0 && next < 4 && next !== 2);
    seen.add(next);
  }
  assert.deepEqual([...seen].sort(), [0, 1, 3]);
  assert.equal(pickVariant(1, 0), 0);
  assert.ok(pickVariant(3, undefined) < 3);
  assert.ok(pickVariant(3, 9) < 3);
  assert.equal(pickVariant(3, 0, () => 0.999999), 2);
});
