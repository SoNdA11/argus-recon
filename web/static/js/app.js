const ws = new WebSocket(`ws://${window.location.host}/ws`);
let currentMode = 'sim';
let currentBoostType = 'fix';
let selectedTarget = '';
let localVirtualAddr = '';

const ctx = document.getElementById('powerChart').getContext('2d');
const integrityCtx = document.getElementById('integrityChart')?.getContext('2d');

function createGradient(ctx, colorStart, colorEnd) {
    const gradient = ctx.createLinearGradient(0, 0, 0, 300);
    gradient.addColorStop(0, colorStart);
    gradient.addColorStop(1, colorEnd);
    return gradient;
}

const initialDataCount = 60;
const labels = Array(initialDataCount).fill('');
const dataReal = Array(initialDataCount).fill(0);
const dataOut = Array(initialDataCount).fill(0);

const chart = new Chart(ctx, {
    type: 'line',
    data: {
        labels,
        datasets: [
            {
                label: 'Modified Output',
                data: dataOut,
                borderColor: '#3b82f6',
                backgroundColor: (context) => createGradient(context.chart.ctx, 'rgba(59, 130, 246, 0.4)', 'rgba(59, 130, 246, 0.0)'),
                borderWidth: 2,
                tension: 0.4,
                fill: true,
                pointRadius: 0,
            },
            {
                label: 'Real Input',
                data: dataReal,
                borderColor: '#647d8f',
                borderWidth: 2,
                borderDash: [4, 4],
                tension: 0.4,
                pointRadius: 0,
            },
        ],
    },
    options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        interaction: { intersect: false, mode: 'index' },
        scales: {
            x: { display: false },
            y: {
                grid: { color: '#334155', tickLength: 0 },
                border: { display: false },
                ticks: { color: '#94a3b8', font: { family: 'Inter', size: 10 }, padding: 10 },
                suggestedMin: 0,
                suggestedMax: 300,
            },
        },
        plugins: { legend: { display: false } },
    },
});

const integrityChart = integrityCtx
    ? new Chart(integrityCtx, {
        type: 'bar',
        data: {
            labels: [],
            datasets: [
                { label: 'Latency (ms)', data: [], backgroundColor: 'rgba(59,130,246,0.65)' },
                { label: 'Jitter (ms)', data: [], backgroundColor: 'rgba(16,185,129,0.65)' },
            ],
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            plugins: { legend: { labels: { color: '#cbd5e1' } } },
            scales: {
                x: { ticks: { color: '#94a3b8' }, grid: { color: '#334155' } },
                y: { ticks: { color: '#94a3b8' }, grid: { color: '#334155' }, beginAtZero: true },
            },
        },
    })
    : null;

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    localVirtualAddr = data.localVirtualAddr || '';

    document.getElementById('val-real').innerText = data.realPower;
    document.getElementById('val-out').innerText = data.outputPower;
    document.getElementById('val-score').innerText = data.integrity?.score ?? '--';
    document.getElementById('val-hr').innerText = data.outputHR ?? '--';

    updateChart(data.realPower, data.outputPower);
    updateIntegrity(data.integrity);
    updateTargets(data.discoveredDevices || [], data.trainerAddress || '');
    updateIntegrityChart(data.integrityReports || {}, data.discoveredDevices || []);

    if (data.mode !== currentMode) updateModeUI(data.mode);
    if (data.boostType !== currentBoostType) updateBoostTypeUI(data.boostType);

    const sliderBoost = document.getElementById('slider-boost');
    if (sliderBoost && document.activeElement !== sliderBoost) {
        sliderBoost.value = data.boostValue;
        document.getElementById('lbl-boost').innerText = data.boostValue;
    }

    updateConnectionStatus(data.connected);
};

function updateTargets(devices, trainerAddress) {
    const box = document.getElementById('targets-list');
    if (!devices.length) {
        box.innerHTML = '<div class="target-meta">No BLE targets found yet...</div>';
        return;
    }

    if (!selectedTarget && trainerAddress) selectedTarget = trainerAddress;
    if (!selectedTarget && devices[0]) selectedTarget = devices[0].address;

    box.innerHTML = devices
        .map((d) => {
            const active = selectedTarget === d.address ? 'active' : '';
            const type = [d.hasCyclingPower ? 'CyclingPower' : '', d.hasHeartRate ? 'HeartRate' : ''].filter(Boolean).join(', ');
            const rssiText = d.address === localVirtualAddr ? 'RSSI N/A' : `RSSI ${d.rssi}`;
            return `<div class="target-row ${active}" onclick="selectTarget('${d.address}')">
            <div><strong>${d.name || 'Unknown'}</strong><div class="target-meta">${d.address}</div></div>
            <div class="target-meta">${rssiText} • ${type || 'Unknown profile'}</div>
        </div>`;
        })
        .join('');
}

function selectTarget(addr) {
    selectedTarget = addr;
    ws.send(JSON.stringify({ trainerAddress: addr }));
}

function updateIntegrity(report) {
    if (!report) return;
    document.getElementById('sig-class').innerText = report.classification || '--';
    document.getElementById('sig-latency').innerText = `${report.signals?.latencyMeanMs ?? '--'} ms`;
    document.getElementById('sig-jitter').innerText = `${report.signals?.latencyJitterMs ?? '--'} ms`;
    document.getElementById('sig-hz').innerText = `${report.signals?.powerNotifyHz ?? '--'} Hz`;
    document.getElementById('sig-drift').innerText = `${report.signals?.powerCadenceDrift ?? '--'}`;
    document.getElementById('sig-stress').innerText = `${report.signals?.stressDropRate ?? '--'}`;
    document.getElementById('sig-oui').innerText = report.observedOui || '--';
    document.getElementById('sig-vendor').innerText = report.vendorGuess || '--';
    document.getElementById('reason-list').innerHTML = (report.reasons || []).map((r) => `<li>${r}</li>`).join('');
}

function updateIntegrityChart(reports, devices) {
    if (!integrityChart) return;

    const chartLabels = [];
    const lat = [];
    const jit = [];

    devices.forEach((d) => {
        const r = reports[d.address];
        if (!r) return;
        chartLabels.push((d.name || d.address).slice(0, 14));
        lat.push(r.signals?.latencyMeanMs || 0);
        jit.push(r.signals?.latencyJitterMs || 0);
    });

    integrityChart.data.labels = chartLabels;
    integrityChart.data.datasets[0].data = lat;
    integrityChart.data.datasets[1].data = jit;
    integrityChart.update();
}

function updateConnectionStatus(isConnected) {
    const dot = document.getElementById('conn-dot');
    const text = document.getElementById('conn-text');
    const btnDisc = document.getElementById('btn-disconnect');

    if (isConnected) {
        dot.classList.add('connected');
        text.innerText = 'LINK ESTABLISHED';
        text.style.color = 'var(--success)';
        btnDisc.classList.remove('hidden');
    } else {
        dot.classList.remove('connected');
        text.innerText = 'SEARCHING...';
        text.style.color = 'var(--text-muted)';
        btnDisc.classList.add('hidden');
    }
}

function updateChart(real, out) {
    chart.data.datasets[0].data.shift();
    chart.data.datasets[1].data.shift();
    chart.data.datasets[0].data.push(out);
    chart.data.datasets[1].data.push(real);
    chart.update();
}

function setMode(mode) {
    currentMode = mode;
    ws.send(JSON.stringify({ mode }));
    updateModeUI(mode);
}

function updateModeUI(mode) {
    document.querySelectorAll('.mode-option').forEach((el) => el.classList.remove('active'));
    document.getElementById(`mode-${mode}`)?.classList.add('active');
    document.getElementById('ctrl-sim')?.classList.toggle('hidden', mode !== 'sim');
    document.getElementById('ctrl-bridge')?.classList.toggle('hidden', mode === 'sim');
    chart.data.datasets[0].borderColor = mode === 'sim' ? '#3b82f6' : '#f59e0b';
    chart.update();
}

function setBoostType(type) {
    currentBoostType = type;
    ws.send(JSON.stringify({ boostType: type }));
    updateBoostTypeUI(type);
}

function updateBoostTypeUI(type) {
    const btnFix = document.getElementById('btn-fix');
    const btnPct = document.getElementById('btn-pct');
    const slider = document.getElementById('slider-boost');
    const lblUnit = document.getElementById('lbl-unit');

    btnFix?.classList.toggle('active', type === 'fix');
    btnPct?.classList.toggle('active', type !== 'fix');

    lblUnit.innerText = type === 'fix' ? 'W' : '%';
    slider.max = type === 'fix' ? 300 : 100;
}

function sendBoost(val) {
    document.getElementById('lbl-boost').innerText = val;
    ws.send(JSON.stringify({ boost: parseInt(val, 10) }));
}

function sendSim(val) {
    document.getElementById('lbl-sim').innerText = `${val} W`;
    ws.send(JSON.stringify({ sim: parseInt(val, 10) }));
}

function disconnectTrainer() {
    if (confirm('Terminate Bluetooth Link?')) {
        ws.send(JSON.stringify({ disconnect: true }));
    }
}

function applyRoute() {
    const route = (window.location.hash || '#dashboard').replace('#', '');
    const dashboard = document.getElementById('view-dashboard');
    const integrity = document.getElementById('view-integrity');
    const settings = document.getElementById('view-settings');

    const navDashboard = document.getElementById('nav-dashboard');
    const navIntegrity = document.getElementById('nav-integrity');
    const navSettings = document.getElementById('nav-settings');

    dashboard?.classList.add('hidden');
    integrity?.classList.add('hidden');
    settings?.classList.add('hidden');

    navDashboard?.classList.remove('active');
    navIntegrity?.classList.remove('active');
    navSettings?.classList.remove('active');

    if (route === 'integrity') {
        integrity?.classList.remove('hidden');
        navIntegrity?.classList.add('active');
        return;
    }

    if (route === 'settings') {
        settings?.classList.remove('hidden');
        navSettings?.classList.add('active');
        return;
    }

    dashboard?.classList.remove('hidden');
    navDashboard?.classList.add('active');
}

function bindNavRoutes() {
    document.querySelectorAll('.nav-links a[href^="#"]').forEach((a) => {
        a.addEventListener('click', (ev) => {
            ev.preventDefault();
            const href = a.getAttribute('href') || '#dashboard';
            if (window.location.hash !== href) {
                window.location.hash = href;
            } else {
                applyRoute();
            }
        });
    });
}

window.addEventListener('hashchange', applyRoute);
applyRoute();
bindNavRoutes();