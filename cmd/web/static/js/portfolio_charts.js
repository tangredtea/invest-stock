// portfolio_charts.js — native Canvas charts for the portfolio backtest report.
// No third-party library: only the 2D canvas API (Requirements 7.5, 7.6, 8.4).
(function (global) {
  'use strict';

  const PALETTE = ['#3b82f6', '#16a34a', '#dc2626', '#d97706', '#7c3aed',
    '#0891b2', '#db2777', '#65a30d', '#9333ea', '#0d9488'];

  function dpiCanvas(canvas) {
    const ratio = global.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();
    const w = rect.width || canvas.width || 600;
    const h = rect.height || canvas.height || 200;
    canvas.width = w * ratio;
    canvas.height = h * ratio;
    const ctx = canvas.getContext('2d');
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    return { ctx, w, h };
  }

  // navChart draws the portfolio net-value (or equity) curve.
  function navChart(canvas, values) {
    const { ctx, w, h } = dpiCanvas(canvas);
    const padL = 56, padR = 10, padT = 12, padB = 20;
    const plotW = w - padL - padR, plotH = h - padT - padB;
    ctx.clearRect(0, 0, w, h);
    if (!values || !values.length) return;
    let lo = Infinity, hi = -Infinity;
    for (const v of values) { if (v < lo) lo = v; if (v > hi) hi = v; }
    if (lo === hi) { lo -= 1; hi += 1; }
    const xOf = i => padL + (values.length === 1 ? 0 : (i / (values.length - 1)) * plotW);
    const yOf = v => padT + (1 - (v - lo) / (hi - lo)) * plotH;
    ctx.strokeStyle = 'rgba(128,128,128,0.2)'; ctx.fillStyle = 'rgba(150,150,150,0.9)'; ctx.font = '10px monospace';
    for (let g = 0; g <= 4; g++) {
      const yv = lo + (g / 4) * (hi - lo), y = yOf(yv);
      ctx.beginPath(); ctx.moveTo(padL, y); ctx.lineTo(w - padR, y); ctx.stroke();
      ctx.fillText(yv.toFixed(3), 4, y + 3);
    }
    ctx.beginPath(); ctx.moveTo(xOf(0), yOf(values[0]));
    for (let i = 1; i < values.length; i++) ctx.lineTo(xOf(i), yOf(values[i]));
    ctx.strokeStyle = '#3b82f6'; ctx.lineWidth = 1.5; ctx.stroke();
    ctx.fillStyle = 'rgba(150,150,150,0.9)'; ctx.fillText('组合净值', padL + 4, padT + 10);
  }

  // weightStackChart draws stacked-area per-symbol weights over time.
  function weightStackChart(canvas, symbols, weights, cashWeight) {
    const { ctx, w, h } = dpiCanvas(canvas);
    const padL = 40, padR = 80, padT = 12, padB = 20;
    const plotW = w - padL - padR, plotH = h - padT - padB;
    ctx.clearRect(0, 0, w, h);
    if (!symbols || !symbols.length) return;
    const n = (weights[symbols[0]] || []).length;
    if (!n) return;
    const xOf = i => padL + (n === 1 ? 0 : (i / (n - 1)) * plotW);
    const yOf = v => padT + (1 - v) * plotH; // v in [0,1]
    // Draw stacked areas bottom-up.
    let cum = new Array(n).fill(0);
    symbols.forEach((sym, si) => {
      const wser = weights[sym] || [];
      ctx.beginPath();
      ctx.moveTo(xOf(0), yOf(cum[0]));
      for (let i = 0; i < n; i++) ctx.lineTo(xOf(i), yOf(cum[i]));
      for (let i = n - 1; i >= 0; i--) ctx.lineTo(xOf(i), yOf(cum[i] + (wser[i] || 0)));
      ctx.closePath();
      ctx.fillStyle = PALETTE[si % PALETTE.length];
      ctx.globalAlpha = 0.75; ctx.fill(); ctx.globalAlpha = 1;
      for (let i = 0; i < n; i++) cum[i] += (wser[i] || 0);
      // legend
      ctx.fillStyle = PALETTE[si % PALETTE.length];
      ctx.fillRect(w - padR + 6, padT + si * 14, 8, 8);
      ctx.fillStyle = 'rgba(150,150,150,0.9)'; ctx.font = '10px monospace';
      ctx.fillText(sym, w - padR + 18, padT + si * 14 + 8);
    });
  }

  global.PortfolioCharts = { navChart, weightStackChart };
})(window);
