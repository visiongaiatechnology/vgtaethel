// STATUS: DIAMANT VGT SUPREME

export class GeoPerformanceController {
  constructor(viewer, onBudgetChange) {
    this.viewer = viewer;
    this.onBudgetChange = typeof onBudgetChange === 'function' ? onBudgetChange : () => {};
    this.profile = null;
    this.frameSamples = [];
    this.lastFrameAt = 0;
    this.lastDecisionAt = 0;
    this.degradeVotes = 0;
    this.recoverVotes = 0;
    this.budgetLevel = 0;
    this.averageFPS = 0;
    this.removePostRender = null;
    this.onPostRender = this.onPostRender.bind(this);
  }

  start(profile) {
    this.stop();
    this.profile = profile;
    if (!this.viewer?.scene) return;
    this.removePostRender = this.viewer.scene.postRender.addEventListener(this.onPostRender);
  }

  updateProfile(profile) {
    this.profile = profile;
    this.resetSamples();
  }

  onPostRender() {
    const now = performance.now();
    if (this.lastFrameAt > 0) {
      const delta = now - this.lastFrameAt;
      if (delta > 0 && delta < 1000) this.frameSamples.push(1000 / delta);
      if (this.frameSamples.length > 120) this.frameSamples.splice(0, this.frameSamples.length - 120);
    }
    this.lastFrameAt = now;
    if (!this.profile?.dynamicQuality || now - this.lastDecisionAt < 3000 || this.frameSamples.length < 30) return;
    this.lastDecisionAt = now;
    const averageFPS = this.frameSamples.reduce((sum, fps) => sum + fps, 0) / this.frameSamples.length;
    this.averageFPS = averageFPS;
    const target = this.profile.targetFPS;
    if (averageFPS < 30) {
      this.degradeVotes += 1;
      this.recoverVotes = 0;
    } else if (averageFPS > Math.max(42, target * 0.78)) {
      this.recoverVotes += 1;
      this.degradeVotes = 0;
    } else {
      this.degradeVotes = 0;
      this.recoverVotes = 0;
    }
    if (this.degradeVotes >= 2 && this.budgetLevel < 3) {
      this.budgetLevel += 1;
      this.degradeVotes = 0;
      this.onBudgetChange(this.budgetLevel, averageFPS);
    } else if (this.recoverVotes >= 4 && this.budgetLevel > 0) {
      this.budgetLevel -= 1;
      this.recoverVotes = 0;
      this.onBudgetChange(this.budgetLevel, averageFPS);
    }
  }

  resetSamples() {
    this.frameSamples.length = 0;
    this.lastFrameAt = 0;
    this.degradeVotes = 0;
    this.recoverVotes = 0;
  }

  diagnostics() {
    return Object.freeze({ averageFPS: this.averageFPS, budgetLevel: this.budgetLevel, samples: this.frameSamples.length });
  }

  stop() {
    if (typeof this.removePostRender === 'function') this.removePostRender();
    this.removePostRender = null;
    this.resetSamples();
  }
}
