// backtest_charts.js — native Canvas line charts for the backtest report.
// No third-party charting library: only the 2D canvas API (Requirement 17.1).
(function (global) {
  'use strict';

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

  function extent(values) {
    let lo = Infinity, hi = -Infinity;
    for (const v of values) {
      if (v < lo) lo = v;
      if (v > hi) hi = v;
    }
    if (!isFinite(lo) || !isFinite(hi)) { lo = 0; hi = 1; }
    if (lo === hi) { lo -= 1; hi += 1; }
    return [lo, hi];
  }

  // drawLine renders a single series with axes onto a canvas.
  // opts: { color, fill, yMin, yMax, label, markers: [{i, type}] }
  function drawLine(canvas, values, opts) {
    opts = opts || {};
    const { ctx, w, h } = dpiCanvas(canvas);
    const padL = 52, padR = 10, padT = 12, padB = 20;
    const plotW = w - padL - padR;
    const plotH = h - padT - padB;

    ctx.clearRect(0, 0, w, h);
    if (!values || values.length === 0) return;

    let [yMin, yMax] = (opts.yMin != null && opts.yMax != null)
      ? [opts.yMin, opts.yMax] : extent(values);
    const xOf = i => padL + (values.length === 1 ? 0 : (i / (values.length - 1)) * plotW);
    const yOf = v => padT + (1 - (v - yMin) / (yMax - yMin)) * plotH;

    // Grid + y labels.
    ctx.strokeStyle = 'rgba(128,128,128,0.2)';
    ctx.fillStyle = 'rgba(150,150,150,0.9)';
    ctx.font = '10px monospace';
    ctx.lineWidth = 1;
    for (let g = 0; g <= 4; g++) {
      const yv = yMin + (g / 4) * (yMax - yMin);
      const y = yOf(yv);
      ctx.beginPath(); ctx.moveTo(padL, y); ctx.lineTo(w - padR, y); ctx.stroke();
      ctx.fillText(yv.toFixed(2), 4, y + 3);
    }

    // Optional fill under curve.
    if (opts.fill) {
      ctx.beginPath();
      ctx.moveTo(xOf(0), yOf(values[0]));
      for (let i = 1; i < values.length; i++) ctx.lineTo(xOf(i), yOf(values[i]));
      ctx.lineTo(xOf(values.length - 1), yOf(yMin));
      ctx.lineTo(xOf(0), yOf(yMin));
      ctx.closePath();
      ctx.fillStyle = opts.fill;
      ctx.fill();
    }

    // Line.
    ctx.beginPath();
    ctx.moveTo(xOf(0), yOf(values[0]));
    for (let i = 1; i < values.length; i++) ctx.lineTo(xOf(i), yOf(values[i]));
    ctx.strokeStyle = opts.color || '#3b82f6';
    ctx.lineWidth = 1.5;
    ctx.stroke();

    // Buy/sell markers (Requirement 17.4).
    if (opts.markers) {
      for (const m of opts.markers) {
        if (m.i < 0 || m.i >= values.length) continue;
        const x = xOf(m.i), y = yOf(values[m.i]);
        ctx.beginPath();
        ctx.arc(x, y, 3, 0, Math.PI * 2);
        ctx.fillStyle = m.type === 'buy' ? '#16a34a' : '#dc2626';
        ctx.fill();
      }
    }

    if (opts.label) {
      ctx.fillStyle = 'rgba(150,150,150,0.9)';
      ctx.fillText(opts.label, padL + 4, padT + 10);
    }
  }

  global.BacktestCharts = {
    // equityChart draws the equity curve with buy/sell markers.
    equityChart(canvas, equity, markers) {
      drawLine(canvas, equity, { color: '#3b82f6', fill: 'rgba(59,130,246,0.08)', label: '资金曲线', markers });
    },
    // drawdownChart draws drawdown as negative percentages (0% to -100%).
    drawdownChart(canvas, drawdown) {
      const pct = drawdown.map(d => -d * 100);
      drawLine(canvas, pct, { color: '#dc2626', fill: 'rgba(220,38,38,0.10)', yMin: -100, yMax: 0, label: '回撤(%)' });
    },
  };
})(window);
