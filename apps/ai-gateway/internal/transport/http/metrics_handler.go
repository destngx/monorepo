package httptransport

import (
	"apps/ai-gateway/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
)

type MetricsHandler struct {
	collector *service.MetricsCollector
}

func NewMetricsHandler(collector *service.MetricsCollector) *MetricsHandler {
	return &MetricsHandler{collector: collector}
}

// ServeHTTP handles the /metrics endpoint.
// @Summary Get gateway metrics
// @Description Returns an aggregated summary of gateway usage, including requests, errors, tokens, and latency by provider/model.
// @Tags monitoring
// @Produce json
// @Success 200 {object} domain.MetricsSummary
// @Router /metrics [get]
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	summary := h.collector.Summary()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

type MetricsResetHandler struct {
	collector *service.MetricsCollector
}

func NewMetricsResetHandler(collector *service.MetricsCollector) *MetricsResetHandler {
	return &MetricsResetHandler{collector: collector}
}

// ServeHTTP handles the /metrics/reset endpoint.
// @Summary Reset gateway metrics
// @Description Clears all in-memory metrics counters and buffers.
// @Tags monitoring
// @Produce json
// @Success 200 {object} map[string]string
// @Router /metrics/reset [post]
func (h *MetricsResetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.collector.Reset()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// ServeHTTP handles the /metrics/dashboard endpoint.
// @Summary Metrics dashboard
// @Description Returns the self-contained HTML/JS monitoring dashboard.
// @Tags monitoring
// @Produce html
// @Success 200 {string} string "HTML Dashboard"
// @Router /metrics/dashboard [get]
func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}

const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI Gateway | Analytics Dashboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
    <style>
        :root {
            --bg-dark: #0f172a;
            --bg-card: #1e293b;
            --primary: #38bdf8;
            --secondary: #818cf8;
            --accent: #f472b6;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --success: #4ade80;
            --error: #f87171;
            --border: #334155;
            --chart-grid: rgba(255, 255, 255, 0.05);
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Inter', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-main);
            line-height: 1.5;
            padding: 2rem;
        }

        .container { max-width: 1400px; margin: 0 auto; }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--border);
        }

        h1 { font-weight: 700; font-size: 1.5rem; letter-spacing: -0.025em; }
        .logo { color: var(--primary); }
        .uptime { font-size: 0.875rem; color: var(--text-muted); }

        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }

        .card {
            background-color: var(--bg-card);
            padding: 1.5rem;
            border-radius: 1rem;
            border: 1px solid var(--border);
            transition: all 0.2s ease;
        }
        .card:hover { border-color: var(--primary); }

        .card-label { font-size: 0.75rem; font-weight: 600; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.5rem; }
        .card-value { font-size: 2rem; font-weight: 700; font-family: 'JetBrains Mono', monospace; }
        .card-subtext { font-size: 0.875rem; margin-top: 0.25rem; }

        .legend-row {
            font-size: 0.75rem; color: var(--text-muted); margin-bottom: 2rem; display: flex; gap: 1rem; align-items: center; justify-content: center;
            background: rgba(30, 41, 59, 0.4); padding: 0.75rem; border-radius: 0.5rem;
        }

        .charts-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2.5rem;
        }

        .chart-container {
            background-color: var(--bg-card);
            padding: 1rem;
            border-radius: 1rem;
            border: 1px solid var(--border);
            position: relative;
            height: 280px;
            display: flex;
            flex-direction: column;
        }
        .chart-title { font-size: 0.875rem; font-weight: 600; margin-bottom: 1.5rem; color: var(--text-muted); display: flex; align-items: center; gap: 0.5rem; }
        .chart-title span { color: var(--primary); }

        .full-width { grid-column: 1 / -1; }

        .tabs { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
        .tab { padding: 0.5rem 1rem; background: var(--bg-dark); border: 1px solid var(--border); color: var(--text-muted); border-radius: 0.5rem; cursor: pointer; font-size: 0.75rem; transition: all 0.2s; }
        .tab.active { background: var(--primary); color: var(--bg-dark); border-color: var(--primary); font-weight: 600; }

        .tables-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 2rem;
            margin-bottom: 2.5rem;
        }
        @media (max-width: 1024px) { .tables-grid { grid-template-columns: 1fr; } .charts-grid { grid-template-columns: 1fr; } }

        .table-container { background-color: var(--bg-card); border-radius: 1rem; border: 1px solid var(--border); overflow: hidden; }
        table { width: 100%; border-collapse: collapse; text-align: left; }
        th { background-color: rgba(15, 23, 42, 0.5); padding: 1rem; font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase; font-weight: 600; }
        td { padding: 1rem; border-top: 1px solid var(--border); font-size: 0.875rem; }

        .mono { font-family: 'JetBrains Mono', monospace; }
        .text-success { color: var(--success); }
        .text-error { color: var(--error); }
        .text-primary { color: var(--primary); }
        .tag { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 0.375rem; background-color: rgba(56, 189, 248, 0.1); color: var(--primary); font-size: 0.75rem; font-weight: 500; }

        .btn-reset { background: none; border: 1px solid var(--error); color: var(--error); padding: 0.5rem 1rem; border-radius: 0.5rem; font-size: 0.875rem; cursor: pointer; transition: all 0.2s; }
        .btn-reset:hover { background-color: rgba(248, 113, 113, 0.1); }

        .live-indicator { width: 8px; height: 8px; background-color: var(--success); border-radius: 50%; display: inline-block; margin-right: 0.5rem; animation: pulse 2s infinite; }
        @keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.5; } 100% { opacity: 1; } }

        .error-msg { font-size: 0.75rem; color: var(--error); max-width: 200px; display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

        canvas { flex: 1; width: 100% !important; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1><span class="logo">AI</span> Gateway Analytics</h1>
            <div style="display: flex; align-items: center; gap: 1.5rem;">
                <div class="uptime"><span class="live-indicator"></span>Live / <span id="uptime-val">0</span>s uptime</div>
                <button class="btn-reset" onclick="resetMetrics()">Reset Data</button>
            </div>
        </header>

        <section class="summary-grid">
            <div class="card">
                <div class="card-label">Total Requests</div>
                <div class="card-value" id="total-requests">0</div>
                <div class="card-subtext" id="requests-rate">Lifetime overview</div>
            </div>
            <div class="card">
                <div class="card-label">Error Rate</div>
                <div class="card-value" id="error-rate">0%</div>
                <div class="card-subtext" id="total-errors">0 errors</div>
            </div>
            <div class="card">
                <div class="card-label">Total Tokens</div>
                <div class="card-value" id="total-tokens">0</div>
                <div class="card-subtext">Avg <span id="avg-tokens" class="text-primary">0</span> / req</div>
            </div>
            <div class="card">
                <div class="card-label">Avg Latency</div>
                <div class="card-value"><span id="avg-latency">0</span><span style="font-size: 1rem; color: var(--text-muted)">ms</span></div>
                <div class="card-subtext">Recent window</div>
            </div>
        </section>

        <section class="charts-grid">
            <!-- Full width Historical Chart -->
            <div class="chart-container full-width" style="height: 400px;">
                <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1rem;">
                    <div class="chart-title" style="margin-bottom: 0;"><span>#</span> Token Consumption History</div>
                    <div class="tabs">
                        <div class="tab" onclick="setHistoryGrain('daily')" id="tab-daily">Daily</div>
                        <div class="tab" onclick="setHistoryGrain('weekly')" id="tab-weekly">Weekly</div>
                        <div class="tab active" onclick="setHistoryGrain('monthly')" id="tab-monthly">Monthly</div>
                    </div>
                </div>
                <canvas id="historyChart"></canvas>
            </div>

            <div class="chart-container">
                <div class="chart-title"><span>#</span> Total Requests by Route</div>
                <canvas id="routeChart"></canvas>
            </div>
            <div class="chart-container">
                <div class="chart-title"><span>#</span> Tokens Consumed by Provider (Log)</div>
                <canvas id="providerChart"></canvas>
            </div>
            <div class="chart-container">
                <div class="chart-title"><span>#</span> Average Latency by Model (ms)</div>
                <canvas id="modelChart"></canvas>
            </div>
            <div class="chart-container full-width" style="height: 320px;">
                <div class="chart-title"><span>#</span> Recent Requests Timeline (Latency vs Tokens)</div>
                <canvas id="timelineChart"></canvas>
            </div>
        </section>

        <div class="tables-grid">
            <div class="table-wrapper">
                <div class="chart-title"><span>#</span> Activity by Provider</div>
                <div class="table-container">
                    <table>
                        <thead>
                            <tr><th>Provider</th><th>Reqs</th><th>Errors</th><th>Avg Latency</th><th>Tokens</th></tr>
                        </thead>
                        <tbody id="provider-body"></tbody>
                    </table>
                </div>
            </div>

            <div class="table-wrapper">
                <div class="chart-title"><span>#</span> Activity by Model</div>
                <div class="table-container">
                    <table>
                        <thead>
                            <tr><th>Model</th><th>Reqs</th><th>Errors</th><th>Avg Latency</th><th>Tokens</th></tr>
                        </thead>
                        <tbody id="model-body"></tbody>
                    </table>
                </div>
            </div>
        </div>

        <div class="chart-title"><span>#</span> Live Request Feed</div>
        <div class="table-container">
            <table>
                <thead>
                    <tr><th>Time</th><th>Route</th><th>Provider</th><th>Model</th><th>Status</th><th>Latency</th><th>Tokens (P/C/T)</th></tr>
                </thead>
                <tbody id="feed-body"></tbody>
            </table>
        </div>
    </div>

    <script>
        let charts = {};
        let lastTotalRequests = -1;
        let historyGrain = 'monthly';
        let latestData = null;

        const chartColors = {
            primary: '#38bdf8',
            secondary: '#818cf8',
            accent: '#f472b6',
            success: '#4ade80',
            error: '#f87171',
            muted: '#94a3b8'
        };

        const defaultOptions = {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: '#1e293b',
                    titleColor: '#f8fafc',
                    bodyColor: '#94a3b8',
                    borderColor: '#334155',
                    borderWidth: 1
                }
            },
            scales: {
                x: { grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#94a3b8' } },
                y: { grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#94a3b8' } }
            }
        };

        function initCharts() {
            Object.values(charts).forEach(c => c && c.destroy());

            // History Chart (Line)
            charts.history = new Chart(document.getElementById('historyChart'), {
                type: 'line',
                data: { labels: [], datasets: [{ label: 'Tokens', borderColor: chartColors.primary, backgroundColor: 'rgba(56, 189, 248, 0.1)', fill: true, tension: 0.3, pointRadius: 4, data: [] }] },
                options: { ...defaultOptions, scales: { x: { ...defaultOptions.scales.x }, y: { ...defaultOptions.scales.y, beginAtZero: true } } }
            });

            // Route Chart (Horizontal)
            charts.route = new Chart(document.getElementById('routeChart'), {
                type: 'bar',
                data: { labels: [], datasets: [{ backgroundColor: chartColors.primary, borderRadius: 4, data: [] }] },
                options: { ...defaultOptions, indexAxis: 'y', scales: { x: { ...defaultOptions.scales.x, title: { display: true, text: 'Requests' } } } }
            });

            // Provider Chart (Horizontal + Logarithmic)
            charts.provider = new Chart(document.getElementById('providerChart'), {
                type: 'bar',
                data: { labels: [], datasets: [{ backgroundColor: chartColors.secondary, borderRadius: 4, data: [] }] },
                options: { ...defaultOptions, indexAxis: 'y', scales: { x: { ...defaultOptions.scales.x, type: 'logarithmic' } } }
            });

            // Model Chart (Vertical)
            charts.model = new Chart(document.getElementById('modelChart'), {
                type: 'bar',
                data: { labels: [], datasets: [{ backgroundColor: chartColors.accent, borderRadius: 6, data: [] }] },
                options: { ...defaultOptions, scales: { y: { ...defaultOptions.scales.y, title: { display: true, text: 'Latency (ms)' } } } }
            });

            // Timeline Chart (X=Order, Y=Latency)
            charts.timeline = new Chart(document.getElementById('timelineChart'), {
                type: 'bubble',
                data: { datasets: [{ backgroundColor: 'rgba(56, 189, 248, 0.4)', borderColor: chartColors.primary, borderWidth: 1, data: [] }] },
                options: { ...defaultOptions, scales: { x: { type: 'linear', display: false }, y: { ...defaultOptions.scales.y, title: { display: true, text: 'Latency (ms)' } } } }
            });
        }

        async function fetchMetrics() {
            try {
                const res = await fetch('/metrics');
                const data = await res.json();
                latestData = data;
                
                if (data.total_requests !== lastTotalRequests) {
                    updateUI(data);
                    updateCharts(data);
                    lastTotalRequests = data.total_requests;
                } else {
                    document.getElementById('uptime-val').textContent = data.uptime_secs;
                }
            } catch (err) { console.error('Metrics fetch error', err); }
        }

        function setHistoryGrain(grain) {
            historyGrain = grain;
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.getElementById('tab-' + grain).classList.add('active');
            if (latestData) updateHistoryChart(latestData);
        }

        function updateUI(data) {
            document.getElementById('uptime-val').textContent = data.uptime_secs;
            document.getElementById('total-requests').textContent = data.total_requests.toLocaleString();
            
            const errRate = data.total_requests > 0 ? (data.total_errors / data.total_requests * 100).toFixed(1) : '0.0';
            const errEl = document.getElementById('error-rate');
            errEl.textContent = errRate + '%';
            errEl.className = 'card-value ' + (errRate > 5 ? 'text-error' : (errRate > 0 ? 'text-warning' : 'text-success'));
            document.getElementById('total-errors').textContent = data.total_errors + ' errors';
            
            document.getElementById('total-tokens').textContent = formatTokens(data.total_tokens);
            document.getElementById('avg-tokens').textContent = data.total_requests > 0 ? Math.round(data.total_tokens / data.total_requests) : 0;
            document.getElementById('avg-latency').textContent = Math.round(data.avg_duration_ms);

            fillTable('provider-body', data.by_provider);
            fillTable('model-body', data.by_model);
            fillFeed(data.recent_requests);
        }

        function updateCharts(data) {
            updateHistoryChart(data);

            const routeLabels = Object.keys(data.by_route || {});
            charts.route.data.labels = routeLabels;
            charts.route.data.datasets[0].data = routeLabels.map(k => data.by_route[k].count);
            charts.route.update();

            const provLabels = Object.keys(data.by_provider || {});
            charts.provider.data.labels = provLabels;
            charts.provider.data.datasets[0].data = provLabels.map(k => data.by_provider[k].total_tokens);
            charts.provider.update();

            const modelItems = Object.keys(data.by_model || {}).map(k => ({ label: k, val: data.by_model[k].avg_duration_ms })).sort((a,b) => b.val - a.val);
            charts.model.data.labels = modelItems.map(i => i.label);
            charts.model.data.datasets[0].data = modelItems.map(i => i.val);
            charts.model.update();

            // Update Timeline Chart (X=Order, Y=Latency)
            const timelineData = (data.recent_requests || []).map((r, i, arr) => ({
                x: arr.length - i,
                y: r.duration_ms,
                r: Math.max(4, Math.min(30, r.total_tokens / 100)),
                meta: { ...r, time: r.timestamp.split('T')[1].split('+')[0] }
            })).reverse();
            charts.timeline.data.datasets[0].data = timelineData;
            charts.timeline.update();
        }

        function updateHistoryChart(data) {
            if (!data.time_series) return;
            const points = data.time_series[historyGrain] || [];
            charts.history.data.labels = points.map(p => p.label);
            charts.history.data.datasets[0].data = points.map(p => p.tokens);
            charts.history.update();
        }

        function fillTable(id, stats) {
            const body = document.getElementById(id);
            body.innerHTML = '';
            const keys = Object.keys(stats || {}).sort((a, b) => stats[b].count - stats[a].count);
            if (keys.length === 0) {
                body.innerHTML = '<tr><td colspan="5" style="text-align:center; color: var(--text-muted); padding: 1rem;">No data</td></tr>';
                return;
            }
            keys.forEach(k => {
                const s = stats[k];
                const tr = document.createElement('tr');
                tr.innerHTML = '<td><span class="tag">' + k + '</span></td>' +
                    '<td class="mono">' + s.count + '</td>' +
                    '<td class="mono ' + (s.errors > 0 ? 'text-error' : '') + '">' + s.errors + '</td>' +
                    '<td class="mono">' + Math.round(s.avg_duration_ms) + 'ms</td>' +
                    '<td class="mono">' + formatTokens(s.total_tokens) + '</td>';
                body.appendChild(tr);
            });
        }

        function fillFeed(recent) {
            const body = document.getElementById('feed-body');
            body.innerHTML = '';
            if (!recent || recent.length === 0) {
                body.innerHTML = '<tr><td colspan="7" style="text-align:center; color: var(--text-muted); padding: 1rem;">No history</td></tr>';
                return;
            }
            recent.forEach(r => {
                const tr = document.createElement('tr');
                const timeStr = r.timestamp.split('T')[1].split('+')[0];
                const statusClass = r.status_code >= 400 ? 'text-error' : 'text-success';
                tr.innerHTML = '<td class="mono" style="font-size: 0.75rem">' + timeStr + '</td>' +
                    '<td style="font-size: 0.75rem">' + r.route + '</td>' +
                    '<td><span class="tag" style="background: rgba(129, 140, 248, 0.1); color: var(--secondary);">' + r.provider + '</span></td>' +
                    '<td class="mono" style="font-size: 0.75rem">' + r.model + '</td>' +
                    '<td class="mono ' + statusClass + '">' + r.status_code + '</td>' +
                    '<td class="mono">' + r.duration_ms + 'ms</td>' +
                    '<td class="mono">' + r.prompt_tokens + ' / ' + r.completion_tokens + ' / <span class="text-primary">' + r.total_tokens + '</span></td>';
                body.appendChild(tr);
            });
        }

        function formatTokens(n) {
            if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
            if (n >= 1000) return (n / 1000).toFixed(1) + 'k';
            return n;
        }

        async function resetMetrics() {
            if (!confirm('Clear all historical metrics?')) return;
            try {
                await fetch('/metrics/reset', { method: 'POST' });
                fetchMetrics();
            } catch (err) { alert('Reset failed'); }
        }

        initCharts();
        fetchMetrics();
        setInterval(fetchMetrics, 5000);
    </script>
</body>
</html>
`
