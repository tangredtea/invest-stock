/**
 * Dashboard - 量化分析终端 JavaScript
 */
// ── State ──
let chartInstances = {};
let ws = null;
let currentCode = '';
let analysisData = null;

// ── Clock ──
setInterval(() => {
  const el = document.getElementById('clockEl');
  if (el) el.textContent = new Date().toLocaleTimeString('zh-CN', {hour12:false});
}, 1000);

// ── Enter key ──
document.addEventListener('DOMContentLoaded', () => {
  const inp = document.getElementById('codeInput');
  if (inp) inp.addEventListener('keydown', e => { if (e.key === 'Enter') analyze(); });
});

// ── Helpers ──
const fmt = (v, d=2) => v == null ? '--' : (v*100).toFixed(d) + '%';
const fmtP = (v, d=3) => v == null ? '--' : v.toFixed(d);
const fmtN = (v) => v == null ? '--' : v.toLocaleString('zh-CN', {maximumFractionDigits:0});
const esc = (v) => String(v == null ? '' : v)
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;');

const trendMap = {1:'上升趋势', '-1':'下降趋势', 0:'震荡'};
const trendClass = {1:'up', '-1':'down', 0:'flat'};
const dcaMap = {2:'强买入', 1:'买入', 0:'持有', '-1':'卖出', '-2':'强卖出'};
const dcaClass = v => v > 0 ? 'up' : v < 0 ? 'down' : 'flat';
const t0Map = {1:'正T(先买后卖)', '-1':'反T(先卖后买)', 0:'观望不做'};
function parseDate(d) { return d ? d.slice(0, 10) : ''; }

const chartOpts = {
  layout: {background:{color:'#0a0e17'}, textColor:'#556677', fontSize:10, fontFamily:'JetBrains Mono'},
  grid: {vertLines:{color:'#111827'}, horzLines:{color:'#111827'}},
  crosshair: {mode:0, vertLine:{color:'#2a3f6a',style:3}, horzLine:{color:'#2a3f6a',style:3}},
  timeScale: {borderColor:'#1e2d4a', timeVisible:false},
  rightPriceScale: {borderColor:'#1e2d4a'},
};

// ── Auth fetch helper ──
function authFetch(url, opts={}) {
  const token = localStorage.getItem('token');
  if (!opts.headers) opts.headers = {};
  if (token) opts.headers['Authorization'] = 'Bearer ' + token;
  return fetch(url, opts);
}

// ── Analyze ──
async function analyze() {
  const code = document.getElementById('codeInput').value.trim();
  if (!/^\d{6}$/.test(code)) return;
  currentCode = code;
  const btn = document.getElementById('analyzeBtn');
  btn.disabled = true; btn.textContent = '...';
  document.getElementById('statusText').textContent = '加载中';
  document.getElementById('statusDot').className = 'status-dot';
  try {
    const resp = await authFetch('/api/analyze?code=' + code);
    if (!resp.ok) throw new Error(await resp.text());
    analysisData = await resp.json();
    renderAll(analysisData);
    document.getElementById('statusDot').className = 'status-dot live';
    document.getElementById('statusText').textContent = code;
  } catch(e) {
    Toast.error('分析失败: ' + e.message);
    document.getElementById('statusText').textContent = '错误';
  } finally {
    btn.disabled = false; btn.textContent = '分析';
  }
}

function renderAll(d) {
  renderKline(d); renderRSI(d); renderMACD(d);
  renderSignal(d); renderBacktest(d);
}

// ── K-line Chart ──
function renderKline(d) {
  const el = document.getElementById('klineChart');
  el.innerHTML = '';
  const chart = LightweightCharts.createChart(el, {...chartOpts, autoSize:true});
  chartInstances.kline = chart;
  const dates = d.klines.map(k => parseDate(k.date));

  const candle = chart.addSeries(LightweightCharts.CandlestickSeries, {
    upColor:'#00d4aa', downColor:'#ff6b6b',
    borderUpColor:'#00d4aa', borderDownColor:'#ff6b6b',
    wickUpColor:'#00d4aa', wickDownColor:'#ff6b6b',
  });
  candle.setData(d.klines.map((k,i) => ({
    time:dates[i], open:k.open, high:k.high, low:k.low, close:k.close
  })));

  const ma5 = chart.addSeries(LightweightCharts.LineSeries, {
    color:'#ffd32a', lineWidth:1, priceLineVisible:false, lastValueVisible:false
  });
  ma5.setData(d.indicators.ma5.map((v,i) => v ? {time:dates[i], value:v} : null).filter(Boolean));

  const ma20 = chart.addSeries(LightweightCharts.LineSeries, {
    color:'#4facfe', lineWidth:1, priceLineVisible:false, lastValueVisible:false
  });
  ma20.setData(d.indicators.ma20.map((v,i) => v ? {time:dates[i], value:v} : null).filter(Boolean));

  const vol = chart.addSeries(LightweightCharts.HistogramSeries, {
    priceFormat:{type:'volume'}, priceScaleId:'vol'
  });
  chart.priceScale('vol').applyOptions({scaleMargins:{top:0.85, bottom:0}});
  vol.setData(d.klines.map((k,i) => ({
    time:dates[i], value:k.volume,
    color: k.close >= k.open ? 'rgba(0,212,170,.3)' : 'rgba(255,107,107,.3)',
  })));
  chart.timeScale().fitContent();
}

// ── RSI Chart ──
function renderRSI(d) {
  const el = document.getElementById('rsiChart');
  el.innerHTML = '';
  const chart = LightweightCharts.createChart(el, {
    ...chartOpts, autoSize:true,
    rightPriceScale:{...chartOpts.rightPriceScale, scaleMargins:{top:0.1,bottom:0.1}},
  });
  chartInstances.rsi = chart;
  const dates = d.klines.map(k => parseDate(k.date));

  const rsi = chart.addSeries(LightweightCharts.LineSeries, {
    color:'#ff9f43', lineWidth:1.5, priceLineVisible:false
  });
  rsi.setData(d.indicators.rsi14.map((v,i) => v ? {time:dates[i], value:v} : null).filter(Boolean));

  const ob = chart.addSeries(LightweightCharts.LineSeries, {
    color:'rgba(255,107,107,.3)', lineWidth:1, lineStyle:2,
    priceLineVisible:false, lastValueVisible:false
  });
  const os_ = chart.addSeries(LightweightCharts.LineSeries, {
    color:'rgba(0,212,170,.3)', lineWidth:1, lineStyle:2,
    priceLineVisible:false, lastValueVisible:false
  });
  ob.setData(dates.map(t => ({time:t, value:70})));
  os_.setData(dates.map(t => ({time:t, value:30})));
  chart.timeScale().fitContent();
}

// ── MACD Chart ──
function renderMACD(d) {
  const el = document.getElementById('macdChart');
  el.innerHTML = '';
  const chart = LightweightCharts.createChart(el, {...chartOpts, autoSize:true});
  chartInstances.macd = chart;
  const dates = d.klines.map(k => parseDate(k.date));

  const hist = chart.addSeries(LightweightCharts.HistogramSeries, {
    priceLineVisible:false, lastValueVisible:false
  });
  hist.setData(d.indicators.macdHist.map((v,i) => ({
    time:dates[i], value:v||0,
    color: v >= 0 ? 'rgba(0,212,170,.6)' : 'rgba(255,107,107,.6)',
  })));

  const ml = chart.addSeries(LightweightCharts.LineSeries, {
    color:'#4facfe', lineWidth:1, priceLineVisible:false, lastValueVisible:false
  });
  ml.setData(d.indicators.macdLine.map((v,i) =>
    v!=null ? {time:dates[i], value:v} : null).filter(Boolean));

  const sl = chart.addSeries(LightweightCharts.LineSeries, {
    color:'#ff9f43', lineWidth:1, priceLineVisible:false, lastValueVisible:false
  });
  sl.setData(d.indicators.macdSignal.map((v,i) =>
    v!=null ? {time:dates[i], value:v} : null).filter(Boolean));
  chart.timeScale().fitContent();
}

// ── Signal Panel ──
function renderSignal(d) {
  const s = d.signal, trend = String(s.trend), dca = String(s.dcaSignal);
  document.getElementById('signalPanel').innerHTML = `
    <div class="signal-grid">
      <div class="signal-card">
        <div class="label">趋势方向</div>
        <div class="value ${trendClass[trend]}">${trendMap[trend]||'--'}</div>
      </div>
      <div class="signal-card">
        <div class="label">定投信号</div>
        <div class="value ${dcaClass(s.dcaSignal)}">${dcaMap[dca]||'--'}</div>
      </div>
      <div class="signal-card">
        <div class="label">T+0方向</div>
        <div class="value ${s.t0Dir>0?'up':s.t0Dir<0?'down':'flat'}">${t0Map[String(s.t0Dir)]||'--'}</div>
      </div>
      <div class="signal-card">
        <div class="label">T+0股数</div>
        <div class="value">${s.t0Shares||'--'}</div>
      </div>
      <div class="signal-card">
        <div class="label">买入参考</div>
        <div class="value up">${fmtP(s.t0BuyPrice)}</div>
      </div>
      <div class="signal-card">
        <div class="label">卖出参考</div>
        <div class="value down">${fmtP(s.t0SellPrice)}</div>
      </div>
      <div class="signal-card full">
        <div class="label">指标依据</div>
        <ul class="reasons">${(s.t0Reasons||[]).map(r=>'<li>'+esc(r)+'</li>').join('')}</ul>
        ${s.reason ? '<div style="margin-top:4px;font-size:11px;color:var(--t-text2)">'+esc(s.reason)+'</div>' : ''}
      </div>
    </div>`;
}

// ── Backtest Table ──
function renderBacktest(d) {
  if (!d.backtest || !d.backtest.length) return;
  const wrap = document.getElementById('backtestWrap');
  const monitorCode = String(d.code || '').replace(/[^0-9A-Za-z_.-]/g, '');
  wrap.innerHTML = `<table class="bt-table">
    <thead><tr>
      <th>策略</th><th>总收益</th><th>年化</th><th>最大回撤</th><th>夏普</th>
      <th>持仓</th><th>均价</th><th>终值</th><th>投入</th><th></th>
    </tr></thead>
    <tbody>${d.backtest.map((b,i) => `<tr>
      <td style="color:var(--t-text)">${esc(b.name)}</td>
      <td class="${b.totalReturn>=0?'up':'down'}">${fmt(b.totalReturn)}</td>
      <td class="${b.annualReturn>=0?'up':'down'}">${fmt(b.annualReturn)}</td>
      <td class="down">${fmt(b.maxDrawdown)}</td>
      <td>${b.sharpeRatio.toFixed(2)}</td>
      <td>${fmtN(b.finalShares)}</td>
      <td>${fmtP(b.avgCost,4)}</td>
      <td>${fmtN(b.finalValue)}</td>
      <td>${fmtN(b.totalCost)}</td>
      <td><button class="btn-sm" onclick="startMonitor('${monitorCode}',${i},this)">监控</button></td>
    </tr>`).join('')}</tbody>
  </table>`;
}

// ── Monitor Log Helper ──
function addLog(cls, msg) {
  const log = document.getElementById('monitorLog');
  const time = new Date().toLocaleTimeString('zh-CN', {hour12:false});
  log.insertAdjacentHTML('beforeend',
    `<div class="log-entry ${esc(cls)}"><span class="log-time">${esc(time)}</span><span class="log-msg">${esc(msg)}</span></div>`);
  log.scrollTop = log.scrollHeight;
}

// ── WebSocket Monitor ──
function startMonitor(code, stratIdx, btn) {
  // Toggle off
  if (ws && btn.classList.contains('active')) {
    ws.close(); ws = null;
    btn.classList.remove('active'); btn.textContent = '监控';
    document.getElementById('monitorPanel').classList.remove('show');
    document.getElementById('monitorTag').textContent = '离线';
    document.getElementById('wsDot').className = 'status-dot';
    return;
  }
  // Close previous
  if (ws) {
    ws.close();
    document.querySelectorAll('.btn-sm.active').forEach(b => {
      b.classList.remove('active'); b.textContent = '监控';
    });
  }

  const panel = document.getElementById('monitorPanel');
  panel.classList.add('show');
  document.getElementById('monitorLog').innerHTML = '';
  document.getElementById('monitorPrice').textContent = '--';
  document.getElementById('monitorChg').textContent = '';
  document.getElementById('wsText').textContent = '连接中';
  document.getElementById('wsDot').className = 'status-dot';

  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = localStorage.getItem('token') || '';
  // Pass token via Sec-WebSocket-Protocol subprotocol, not the URL, so it does
  // not leak into proxy/server access logs.
  ws = new WebSocket(
    `${proto}//${location.host}/ws/monitor?code=${code}`,
    ['bearer', token]
  );

  ws.onopen = () => {
    btn.classList.add('active'); btn.textContent = '停止';
    document.getElementById('monitorTag').textContent = '在线';
    document.getElementById('wsText').textContent = '已连接';
    document.getElementById('wsDot').className = 'status-dot live';
    addLog('log-info', `已连接 ${code} 实时监控，每30秒刷新`);
  };

  ws.onmessage = (e) => {
    const msg = JSON.parse(e.data);
    if (msg.error) { addLog('log-warn', '错误: ' + msg.error); return; }
    if (msg.type === 'tick') {
      const priceEl = document.getElementById('monitorPrice');
      const chgEl = document.getElementById('monitorChg');
      const prev = parseFloat(priceEl.dataset.price || 0);
      priceEl.dataset.price = msg.price;
      priceEl.textContent = msg.price.toFixed(3);
      priceEl.className = 'monitor-price ' +
        (msg.price > prev ? 'up' : msg.price < prev ? 'down' : '');
      if (msg.quote && msg.quote.preClose > 0) {
        const chg = (msg.price - msg.quote.preClose) / msg.quote.preClose * 100;
        chgEl.textContent = (chg >= 0 ? '+' : '') + chg.toFixed(2) + '%';
        chgEl.className = 'monitor-chg ' + (chg >= 0 ? 'up' : 'down');
      }
      const sig = msg.signal;
      let sigText = '';
      if (sig && sig.t0BuyPrice > 0)
        sigText = ` | 买位:${sig.t0BuyPrice.toFixed(3)} 卖位:${sig.t0SellPrice.toFixed(3)}`;
      addLog('log-tick', `价格 ${msg.price.toFixed(3)}${sigText}`);
      if (sig && analysisData) {
        analysisData.signal = sig;
        renderSignal(analysisData);
      }
    }
    if (msg.type === 'alert') {
      const cls = msg.action === 'buy' ? 'log-buy' : 'log-sell';
      addLog(cls, `⚡ ${msg.message} @ ${msg.price.toFixed(3)}`);
    }
  };

  ws.onclose = () => {
    btn.classList.remove('active'); btn.textContent = '监控';
    document.getElementById('monitorTag').textContent = '离线';
    document.getElementById('wsText').textContent = '已断开';
    document.getElementById('wsDot').className = 'status-dot';
    addLog('log-warn', '连接已断开');
  };
}
