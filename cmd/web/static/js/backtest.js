// backtest.js — backtest report view: config form, request flow, result render.
// Native DOM only, no framework (Requirement 17.1).
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

  // postJSON sends a POST with a 30s timeout (Requirement 18.7).
  async function postJSON(url, body) {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS);
    try {
      const resp = await fetch(url, {
        method: 'POST', headers: authHeaders(),
        body: JSON.stringify(body), signal: ctrl.signal,
      });
      const data = await resp.json().catch(() => ({}));
      return { ok: resp.ok, status: resp.status, data };
    } finally {
      clearTimeout(timer);
    }
  }

  // loadStrategies fetches the registry to render strategy options + params.
  async function loadStrategies() {
    try {
      const resp = await fetch('/api/strategies', { headers: authHeaders() });
      const j = await resp.json();
      return (j && j.data) || [];
    } catch (e) {
      return [];
    }
  }

  // renderParamForm builds inputs for a strategy's ParamSpec list, prefilled
  // with defaults (Requirements 18.2, 18.3).
  function renderParamForm(container, specs) {
    container.innerHTML = '';
    (specs || []).forEach(spec => {
      const wrap = document.createElement('div');
      wrap.className = 'bt-param';
      const label = document.createElement('label');
      label.textContent = spec.desc || spec.name;
      const input = document.createElement('input');
      input.dataset.param = spec.name;
      input.dataset.ptype = spec.type;
      const def = spec.default || {};
      if (spec.type === 2) { // enum
        input.value = def.str || '';
        input.placeholder = (spec.enum || []).join('/');
      } else if (spec.type === 0) { // int
        input.type = 'number'; input.value = def.int != null ? def.int : 0;
      } else if (spec.type === 1) { // float
        input.type = 'number'; input.step = 'any'; input.value = def.float != null ? def.float : 0;
      } else { // bool
        input.value = def.bool ? 'true' : 'false';
      }
      wrap.appendChild(label);
      wrap.appendChild(input);
      container.appendChild(wrap);
    });
  }

  // collectParams reads param inputs into a ParamValue map for the API.
  function collectParams(container) {
    const out = {};
    container.querySelectorAll('input[data-param]').forEach(input => {
      const name = input.dataset.param;
      const type = parseInt(input.dataset.ptype, 10);
      if (type === 0) out[name] = { type: 0, int: parseInt(input.value, 10) || 0 };
      else if (type === 1) out[name] = { type: 1, float: parseFloat(input.value) || 0 };
      else if (type === 2) out[name] = { type: 2, str: input.value.trim() };
      else out[name] = { type: 3, bool: input.value === 'true' };
    });
    return out;
  }

  // validateForm performs client-side validation (Requirement 18.4).
  function validateForm(code, initialCash) {
    if (!code || code.length < 1 || code.length > 32) return '标的代码长度需为 1-32 字符';
    if (!(initialCash >= 0.01 && initialCash <= 999999999.99)) return '初始资金需在 0.01 至 999999999.99 之间';
    return '';
  }

  function setStatus(msg, isError) {
    const s = el('btStatus');
    if (!s) return;
    s.textContent = msg || '';
    s.className = isError ? 'bt-status bt-error' : 'bt-status';
  }

  // renderResult draws charts + trade table for a single Result (Requirement 17).
  function renderResult(res) {
    const eqCanvas = el('btEquityChart');
    const ddCanvas = el('btDrawdownChart');
    const markers = (res.trades || []).map(t => {
      const i = (res.dates || []).findIndex(d => d === t.date);
      return { i, type: t.action };
    }).filter(m => m.i >= 0);

    if (eqCanvas) global.BacktestCharts.equityChart(eqCanvas, res.equity || [], markers);
    if (ddCanvas) global.BacktestCharts.drawdownChart(ddCanvas, res.drawdown || []);

    renderMetrics(res.metrics);
    renderTrades(res.trades || []);
  }

  function fmtPct(v) { return (v * 100).toFixed(2) + '%'; }

  function renderMetrics(m) {
    const box = el('btMetrics');
    if (!box || !m) return;
    box.innerHTML = '';
    const rows = [
      ['总收益率', fmtPct(m.totalReturn)],
      ['年化收益率', fmtPct(m.annualReturn)],
      ['最大回撤', fmtPct(m.maxDrawdown)],
      ['年化波动率', fmtPct(m.annualVolatility)],
      ['夏普比率', m.sharpe.toFixed(2)],
      ['索提诺比率', m.sortino.toFixed(2)],
      ['卡玛比率', m.calmar.toFixed(2)],
      ['交易次数', String(m.totalTrades)],
      ['胜率', fmtPct(m.winRate)],
      ['盈亏比', m.profitLossRatio.toFixed(2)],
    ];
    rows.forEach(([k, v]) => {
      const d = document.createElement('div');
      d.className = 'bt-metric';
      d.innerHTML = '<span>' + k + '</span><b>' + v + '</b>';
      box.appendChild(d);
    });
  }

  // renderTrades builds the trade detail table; empty trades show a note but
  // charts still render (Requirements 17.5, 17.8).
  function renderTrades(trades) {
    const box = el('btTrades');
    if (!box) return;
    if (!trades.length) {
      box.innerHTML = '<div class="empty" style="height:60px"><div class="icon">◇</div>本次回测无成交</div>';
      return;
    }
    let html = '<table class="bt-table"><thead><tr>' +
      '<th>日期</th><th>方向</th><th>价格</th><th>数量</th><th>成本</th></tr></thead><tbody>';
    trades.forEach(t => {
      const date = (t.date || '').slice(0, 10);
      const dir = t.action === 'buy' ? '买入' : '卖出';
      html += '<tr><td>' + date + '</td><td>' + dir + '</td><td>' +
        t.price.toFixed(3) + '</td><td>' + t.qty + '</td><td>' +
        t.costTotal.toFixed(2) + '</td></tr>';
    });
    html += '</tbody></table>';
    box.innerHTML = html;
  }

  // renderCompare builds a side-by-side metrics table for multiple strategies
  // (Requirement 17.6).
  function renderCompare(results) {
    const box = el('btCompare');
    if (!box) return;
    let html = '<table class="bt-table"><thead><tr>' +
      '<th>策略</th><th>总收益</th><th>年化</th><th>回撤</th><th>夏普</th><th>交易</th></tr></thead><tbody>';
    results.forEach(r => {
      const m = r.metrics;
      html += '<tr><td>' + r.strategyName + '</td><td>' + fmtPct(m.totalReturn) +
        '</td><td>' + fmtPct(m.annualReturn) + '</td><td>' + fmtPct(m.maxDrawdown) +
        '</td><td>' + m.sharpe.toFixed(2) + '</td><td>' + m.totalTrades + '</td></tr>';
    });
    html += '</tbody></table>';
    box.innerHTML = html;
  }

  // runBacktest is the form submit handler (Requirements 18.2, 18.5, 18.6).
  async function runBacktest() {
    const code = (el('btCode') || {}).value || '';
    const initialCash = parseFloat((el('btCash') || {}).value) || 0;
    const verr = validateForm(code.trim(), initialCash);
    if (verr) { setStatus(verr, true); return; }

    const strategy = (el('btStrategy') || {}).value;
    const params = collectParams(el('btParams'));
    const config = { initialCash };
    const sd = (el('btStart') || {}).value;
    const ed = (el('btEnd') || {}).value;
    if (sd) config.startDate = sd;
    if (ed) config.endDate = ed;

    const btn = el('btRun');
    if (btn) btn.disabled = true; // prevent double submit (Requirement 18.6)
    setStatus('回测中...', false);

    try {
      const { ok, data } = await postJSON('/api/backtest', { code: code.trim(), strategy, params, config });
      if (!ok) {
        setStatus((data && data.message) || '回测失败', true); // Requirement 18.5
        return;
      }
      setStatus('', false);
      renderResult(data.data);
    } catch (e) {
      if (e.name === 'AbortError') setStatus('回测请求超时(30秒)', true);
      else setStatus('网络错误: ' + e.message, true);
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  // runCompareAll runs all registered strategies (default params) and renders a
  // side-by-side comparison (Requirements 15.5, 17.6).
  async function runCompareAll() {
    const code = (el('btCode') || {}).value || '';
    const initialCash = parseFloat((el('btCash') || {}).value) || 0;
    const verr = validateForm(code.trim(), initialCash);
    if (verr) { setStatus(verr, true); return; }

    const strategies = await loadStrategies();
    if (!strategies.length) { setStatus('无可用策略', true); return; }
    const config = { initialCash };
    const sd = (el('btStart') || {}).value;
    const ed = (el('btEnd') || {}).value;
    if (sd) config.startDate = sd;
    if (ed) config.endDate = ed;
    const body = {
      code: code.trim(),
      config,
      strategies: strategies.map(s => ({ strategy: s.name })),
    };

    const btn = el('btCompare9');
    if (btn) btn.disabled = true;
    setStatus('对比回测中...', false);
    try {
      const { ok, data } = await postJSON('/api/backtest/compare', body);
      if (!ok) {
        setStatus((data && data.message) || '对比回测失败', true);
        return;
      }
      setStatus('', false);
      renderCompare(data.data || []);
    } catch (e) {
      if (e.name === 'AbortError') setStatus('回测请求超时(30秒)', true);
      else setStatus('网络错误: ' + e.message, true);
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  async function initBacktestView() {
    const sel = el('btStrategy');
    const paramsBox = el('btParams');
    if (!sel) return;
    const strategies = await loadStrategies();
    sel.innerHTML = '';
    strategies.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.name; opt.textContent = s.name;
      sel.appendChild(opt);
    });
    const specByName = {};
    strategies.forEach(s => { specByName[s.name] = s.params; });
    const refresh = () => renderParamForm(paramsBox, specByName[sel.value]);
    sel.addEventListener('change', refresh);
    refresh();

    const btn = el('btRun');
    if (btn) btn.addEventListener('click', runBacktest);
    const cbtn = el('btCompare9');
    if (cbtn) cbtn.addEventListener('click', runCompareAll);
  }

  global.Backtest = { init: initBacktestView, runBacktest, runCompareAll, renderResult, renderCompare };
  if (document.readyState !== 'loading') initBacktestView();
  else document.addEventListener('DOMContentLoaded', initBacktestView);
})(window);
