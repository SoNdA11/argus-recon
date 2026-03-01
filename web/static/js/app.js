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

// Global Font change for SecOps Theme
Chart.defaults.font.family = "'JetBrains Mono', monospace";
Chart.defaults.color = '#64748b';

const chart = new Chart(ctx, {
    type: 'line',
    data: {
        labels,
        datasets: [
            {
                label: 'Spoofed Egress',
                data: dataOut,
                borderColor: '#00e5ff', // Cyan
                backgroundColor: (context) => createGradient(context.chart.ctx, 'rgba(0, 229, 255, 0.2)', 'rgba(0,0,0,0)'),
                borderWidth: 2,
                tension: 0.3, // Smooth, precise curve
                fill: true,
                pointRadius: 0,
            },
            {
                label: 'Raw Ingress',
                data: dataReal,
                borderColor: '#475569', // Slate
                borderWidth: 2,
                borderDash: [5, 5],
                tension: 0.3,
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
                grid: { color: 'rgba(255, 255, 255, 0.05)', tickLength: 0 },
                border: { display: false },
                ticks: { padding: 10 },
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
                { label: 'Latency (ms)', data: [], backgroundColor: 'rgba(0, 229, 255, 0.7)' }, // Cyan
                { label: 'Jitter (ms)', data: [], backgroundColor: 'rgba(255, 42, 109, 0.7)' }, // Magenta/Red
            ],
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            plugins: { legend: { labels: { color: '#64748b' } } },
            scales: {
                x: { ticks: { color: '#64748b' }, grid: { color: 'rgba(255, 255, 255, 0.05)' } },
                y: { ticks: { color: '#64748b' }, grid: { color: 'rgba(255, 255, 255, 0.05)' }, beginAtZero: true },
            },
        },
    })
    : null;

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    localVirtualAddr = data.localVirtualAddr || '';

    document.getElementById('val-real').innerText = data.realPower;
    document.getElementById('val-out').innerText = data.outputPower;
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
        updateSlidersFill();
    }

    updateConnectionStatus(data.connected);
};

function updateTargets(devices, trainerAddress) {
    const box = document.getElementById('targets-list');

    if (!devices.length) {
        box.innerHTML = '<div class="target-meta">Listening on BLE interfaces...</div>';
        return;
    }

    if (box.children.length === 1 && box.children[0].classList.contains('target-meta')) {
        box.innerHTML = '';
    }

    if (!selectedTarget && trainerAddress) selectedTarget = trainerAddress;
    if (!selectedTarget && devices[0]) selectedTarget = devices[0].address;

    // 1. Cria um Set com os endereços atuais para limpar quem desconectou
    const currentAddresses = new Set(devices.map(d => d.address));
    Array.from(box.children).forEach(child => {
        const addr = child.getAttribute('data-address');
        if (addr && !currentAddresses.has(addr)) {
            child.remove();
        }
    });

    devices.forEach((d) => {
        const type = [d.hasCyclingPower ? 'PWR' : '', d.hasHeartRate ? 'HRM' : ''].filter(Boolean).join(' / ');
        const rssiText = d.address === localVirtualAddr ? 'EMULATED' : `${d.rssi} dBm`;

        let row = box.querySelector(`.target-row[data-address="${d.address}"]`);

        if (!row) {
            row = document.createElement('div');
            row.className = 'target-row';
            row.setAttribute('data-address', d.address);
            row.onclick = () => selectTarget(d.address);
            box.appendChild(row);
        }

        if (selectedTarget === d.address) {
            row.classList.add('active');
        } else {
            row.classList.remove('active');
        }

        row.innerHTML = `
            <div>
                <strong>${d.name || 'Unknown Device'}</strong>
                <div class="target-meta">${d.address}</div>
            </div>
            <div class="target-meta" style="text-align: right;">
                ${rssiText} <br> ${type || 'No Profile'}
            </div>
        `;
    });
}

function selectTarget(addr) {
    selectedTarget = addr;
    ws.send(JSON.stringify({ trainerAddress: addr }));
}

function updateIntegrity(report) {
    if (!report) return;

    document.getElementById('sig-class').innerText = (report.classification || 'UNKNOWN').toUpperCase();
    document.getElementById('sig-latency').innerText = `${report.signals?.latencyMeanMs ?? '--'} ms`;
    document.getElementById('sig-jitter').innerText = `${report.signals?.latencyJitterMs ?? '--'} ms`;
    document.getElementById('sig-hz').innerText = `${report.signals?.powerNotifyHz ?? '--'} Hz`;
    document.getElementById('sig-drift').innerText = `${report.signals?.powerCadenceDrift ?? '--'}`;
    document.getElementById('sig-stress').innerText = `${report.signals?.stressDropRate ?? '--'}`;

    document.getElementById('sig-oui').innerText = report.observedOui || '--';
    document.getElementById('sig-vendor').innerText = report.vendorGuess || '--';

    const mfgData = report.manufacturerData || '0x00 (No Adv Data)';
    document.getElementById('sig-mfg-data').innerText = mfgData;

    const hash = report.gattHash || 'Awaiting deep introspection...';
    document.getElementById('sig-gatt-hash').innerText = hash;

    // Coloração baseada na classificação
    const classEl = document.getElementById('sig-class');
    classEl.style.color = report.classification === 'genuine' ? 'var(--success)' :
        report.classification === 'suspect' ? 'var(--accent-attack)' : 'var(--danger)';

    document.getElementById('reason-list').innerHTML = (report.reasons || []).map((r) => {
        let icon = r.includes('[+]') ? "<i class='bx bx-check-shield' style='color:var(--success)'></i>" :
            r.includes('[-]') ? "<i class='bx bx-error-alt' style='color:var(--danger)'></i>" :
                "<i class='bx bx-info-circle' style='color:var(--primary)'></i>";
        return `<li>${icon} <span>${r.replace(/\[\+\]|\[-\]|\[!\]/g, '')}</span></li>`;
    }).join('');
}

function updateIntegrityChart(reports, devices) {
    if (!integrityChart) return;

    const chartLabels = [];
    const lat = [];
    const jit = [];

    devices.forEach((d) => {
        const r = reports[d.address];
        if (!r) return;
        chartLabels.push((d.name || d.address).slice(0, 10));
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
        text.style.color = 'var(--text-main)';
        btnDisc.classList.remove('hidden');
    } else {
        dot.classList.remove('connected');
        text.innerText = 'ACQUIRING TARGET...';
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

    chart.data.datasets[0].borderColor = mode === 'sim' ? '#00e5ff' : '#ff2a6d';
    chart.data.datasets[0].backgroundColor = mode === 'sim' ?
        (context) => createGradient(context.chart.ctx, 'rgba(0, 229, 255, 0.2)', 'rgba(0,0,0,0)') :
        (context) => createGradient(context.chart.ctx, 'rgba(255, 42, 109, 0.2)', 'rgba(0,0,0,0)');
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
    slider.max = type === 'fix' ? 1000 : 100;

    updateSlidersFill();
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
    if (confirm('Terminate Active Link?')) {
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

function updateSlidersFill() {
    document.querySelectorAll('.tech-slider').forEach(slider => {
        const min = parseFloat(slider.min || 0);
        const max = parseFloat(slider.max || 100);
        const val = parseFloat(slider.value || 0);

        let percent = ((val - min) / (max - min)) * 100;

        slider.style.setProperty('--progress', `${percent}%`);
    });
}

document.addEventListener('input', (e) => {
    if (e.target.classList.contains('tech-slider')) {
        updateSlidersFill();
    }
});


updateSlidersFill();

window.addEventListener('hashchange', applyRoute);
applyRoute();
bindNavRoutes();