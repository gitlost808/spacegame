import Phaser from 'phaser';
import { getRenderQuality, setRenderQuality } from '../utils/display';

const label = () => `GRAPHICS: ${getRenderQuality() === 'saver' ? 'BATTERY SAVER' : 'SHARP'} (G)`;

// switches the drawing buffer between native resolution and a capped density for dense displays;
// the choice is remembered on this device
export class QualityToggle extends Phaser.GameObjects.Text {
  constructor(scene: Phaser.Scene, x: number, y: number) {
    super(scene, x, y, label(), { fontSize: '16px', color: '#2aa198' });
    scene.add.existing(this);
    this.setOrigin(0.5).setPadding(10).setInteractive({ useHandCursor: true });
    const toggle = () => {
      setRenderQuality(scene.game, getRenderQuality() === 'saver' ? 'native' : 'saver');
      this.setText(label());
    };
    this.on('pointerdown', toggle);
    const onKey = (event: KeyboardEvent) => {
      if (!event.repeat) toggle();
    };
    scene.input.keyboard.on('keydown-G', onKey);
    this.once(Phaser.GameObjects.Events.DESTROY, () => scene.input.keyboard.off('keydown-G', onKey));
  }
}
