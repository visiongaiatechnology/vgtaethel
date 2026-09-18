// STATUS: DIAMANT VGT SUPREME
// Architecture adapted from God's Eye View renderGovernor (MIT); see THIRD_PARTY_NOTICES.md.

export class RenderGovernor {
  constructor(scene) {
    this.scene = scene;
    this.holds = new Set();
    this.destroyed = false;
    this.lastReason = 'initialization';
    this.applyMode();
  }

  hold(owner) {
    if (this.destroyed || !owner) return () => {};
    this.holds.add(owner);
    this.applyMode();
    this.request(`hold:${String(owner)}`);
    return () => this.release(owner);
  }

  release(owner) {
    if (this.destroyed) return;
    this.holds.delete(owner);
    this.applyMode();
    this.request(`release:${String(owner)}`);
  }

  request(reason = 'state-change') {
    if (this.destroyed || !this.scene) return;
    this.lastReason = reason;
    this.scene.requestRender();
  }

  applyMode() {
    if (!this.scene) return;
    const active = this.holds.size > 0;
    this.scene.requestRenderMode = !active;
    this.scene.maximumRenderTimeChange = active ? 0 : Number.POSITIVE_INFINITY;
  }

  diagnostics() {
    return Object.freeze({ active: this.holds.size > 0, holds: this.holds.size, lastReason: this.lastReason });
  }

  destroy() {
    this.destroyed = true;
    this.holds.clear();
    this.scene = null;
  }
}

