// portfolio.js — portfolio backtest view: config form, request flow, rendering.
// Native DOM only, no framework (Requirements 7.5, 7.6, 8.4).
(function (global) {
  'use strict';

  const REQUEST_TIMEOUT_MS = 30000;

  function authHeaders() {
    const h = { 'Content-Type': 'application/json' };
    const token = localStorage.getItem('token');
    if (token) h['Authorization'] = 'Bearer ' + token;
    return h;
  }
  function el(id) { return document.getElementById(id); }
  function esc(v) {
    return String(v == null ? '' : v)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  async function postJSON(url, body) {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS);
    try {
      const resp = await fetch(url, { method: 'POST', headers: authHeaders(), body: JSON.stringify(body), signal: ctrl.signal });
      const data = await resp.json().catch(() => ({}));
      return { ok: resp.ok, data };
    } finally { clearTimeout(timer); }
  }

  function setStatus(msg, isError) {
    const s = el('pfStatus');
    if (!s) return;
    s.textContent = msg || '';
    s.className = isError ? 'bt-status bt-error' : 'bt-status';
  }

  function fmtPct(v) { return (v * 100).toFixed(2) + '%'; }

  function parseCodes() {
    const raw = (el('pfCodes') || {}).value || '';
    return raw.split(/[\s,]+/).map(s => s.trim()).filter(Boolean);
  }

  // runPortfolio is the submit handler (Requirements 7.10).
  async function runPortfolio() {
    const codes = parseCodes();
    if (codes.length < 2 || codes.length > 50) {
      setStatus('组合标的数量需在 2 到 50 之间', true);
      return;
    }
    const body = {
      codes,
      scheme: parseInt((el('pfScheme') || {}).value || '0', 10),
      perSymbol: false,
    };
    const periodBars = parseInt((el('pfRebalance') || {}).value || '0', 10);
    if (periodBars > 0) body.rebalance = { periodic: true, periodBars };
    const cash = parseFloat((el('pfCash') || {}).value);
    if (cash > 0) body.config = { initialCash: cash };
    const reserve = parseFloat((el('pfReserve') || {}).value);
    if (reserve > 0) body.risk = { cashReserve: reserve };

    const btn = el('pfRun');
    if (btn) btn.disabled = true;
    setStatus('组合回测中...', false);
    try {
      const { ok, data } = await postJSON('/api/portfolio/backtest', body);
      if (!ok) { setStatus((data && data.message) || '组合回测失败', true); return; }
      setStatus('', false);
      renderPortfolio(data.data);
    } catch (e) {
      if (e.name === 'AbortError') setStatus('组合回测请求超时(30秒)', true);
      else setStatus('网络错误: ' + e.message, true);
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  // renderPortfolio renders curves + tables (Requirements 7.5, 7.6).
  function renderPortfolio(res) {
    if (!res) return;
    const navCanvas = el('pfNavChart');
    const wCanvas = el('pfWeightChart');
    if (navCanvas && global.PortfolioCharts) global.PortfolioCharts.navChart(navCanvas, res.netValue || []);
    if (wCanvas && global.PortfolioCharts) global.PortfolioCharts.weightStackChart(wCanvas, res.symbols || [], res.weights || {}, res.cashWeight || []);
    renderMetrics(res.metrics, res.rebalanceCount, res.rebalanceCost);
    renderAttribution(res.attribution || []);
  }

  function renderMetrics(m, rebCount, rebCost) {
    const box = el('pfMetrics');
    if (!box || !m) return;
    const rows = [
      ['总收益率', fmtPct(m.totalReturn)],
      ['年化收益率', fmtPct(m.annualReturn)],
      ['最大回撤', fmtPct(m.maxDrawdown)],
      ['夏普比率', m.sharpe.toFixed(2)],
      ['索提诺比率', m.sortino.toFixed(2)],
      ['卡玛比率', m.calmar.toFixed(2)],
      ['换手率', fmtPct(m.turnover)],
      ['再平衡次数', String(rebCount || 0)],
      ['再平衡成本', (rebCost || 0).toFixed(2)],
    ];
    box.innerHTML = rows.map(([k, v]) => '<div class="bt-metric"><span>' + esc(k) + '</span><b>' + esc(v) + '</b></div>').join('');
  }

  function renderAttribution(attr) {
    const box = el('pfAttribution');
    if (!box) return;
    let html = '<table class="bt-table"><thead><tr><th>标的</th><th>收益贡献</th><th>最终权重</th></tr></thead><tbody>';
    attr.forEach(a => {
      html += '<tr><td>' + esc(a.symbol) + '</td><td class="' + (a.contribution >= 0 ? 'up' : 'down') + '">' +
        esc(fmtPct(a.contribution)) + '</td><td>' + esc(fmtPct(a.finalWeight)) + '</td></tr>';
    });
    html += '</tbody></table>';
    box.innerHTML = html;
  }

  function init() {
    const btn = el('pfRun');
    if (btn) btn.addEventListener('click', runPortfolio);
  }

  global.Portfolio = { init, runPortfolio, renderPortfolio };
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
})(window);
