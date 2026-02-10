const ws = new WebSocket(`ws://${window.location.host}/ws`);
let currentMode = 'sim';
let currentBoostType = 'fix';
const ctx = document.getElementById('powerChart').getContext('2d');

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
        labels: labels,
        datasets: [
            {
                label: 'Modified Output',
                data: dataOut,
                borderColor: '#3b82f6',
                backgroundColor: (context) => {
                    const ctx = context.chart.ctx;
                    return createGradient(ctx, 'rgba(59, 130, 246, 0.4)', 'rgba(59, 130, 246, 0.0)');
                },
                borderWidth: 2,
                tension: 0.4,
                fill: true,
                pointRadius: 0,
                pointHoverRadius: 6
            },
            {
                label: 'Real Input',
                data: dataReal,
                borderColor: '#647d8f',
                borderWidth: 2,
                borderDash: [4, 4],
                tension: 0.4,
                pointRadius: 0
            }
        ]
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
                suggestedMax: 300
            }
        },
        plugins: {
            legend: { display: false },
            tooltip: {
                backgroundColor: '#1e293b',
                titleColor: '#f8fafc',
                bodyColor: '#cbd5e1',
                borderColor: '#334155',
                borderWidth: 1,
                padding: 10,
                displayColors: true,
                usePointStyle: true
            }
        }
    }
});

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    document.getElementById('val-real').innerText = data.realPower;
    document.getElementById('val-out').innerText = data.outputPower;
    document.getElementById('val-hr').innerText = data.outputHR;

    updateChart(data.realPower, data.outputPower);

    if (data.mode !== currentMode) updateModeUI(data.mode);
    if (data.boostType !== currentBoostType) updateBoostTypeUI(data.boostType);

    const sliderBoost = document.getElementById('slider-boost');
    if (document.activeElement !== sliderBoost) {
        sliderBoost.value = data.boostValue;
        document.getElementById('lbl-boost').innerText = data.boostValue;
    }

    updateConnectionStatus(data.connected);
};

function updateConnectionStatus(isConnected) {
    const dot = document.getElementById('conn-dot');
    const text = document.getElementById('conn-text');
    const btnDisc = document.getElementById('btn-disconnect');

    if (isConnected) {
        dot.classList.add('connected');
        text.innerText = "LINK ESTABLISHED";
        text.style.color = "var(--success)";
        btnDisc.classList.remove('hidden');
    } else {
        dot.classList.remove('connected');
        text.innerText = "SEARCHING...";
        text.style.color = "var(--text-muted)";
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
    ws.send(JSON.stringify({ mode: mode }));
    updateModeUI(mode);
}

function updateModeUI(mode) {
    document.querySelectorAll('.mode-option').forEach(el => el.classList.remove('active'));

    document.getElementById(`mode-${mode}`).classList.add('active');

    if (mode === 'sim') {
        document.getElementById('ctrl-sim').classList.remove('hidden');
        document.getElementById('ctrl-bridge').classList.add('hidden');

        chart.data.datasets[0].borderColor = '#3b82f6';
        chart.data.datasets[0].backgroundColor = (context) => {
            const ctx = context.chart.ctx;
            return createGradient(ctx, 'rgba(59, 130, 246, 0.4)', 'rgba(59, 130, 246, 0.0)');
        };
    } else {
        document.getElementById('ctrl-sim').classList.add('hidden');
        document.getElementById('ctrl-bridge').classList.remove('hidden');

        chart.data.datasets[0].borderColor = '#f59e0b';
        chart.data.datasets[0].backgroundColor = (context) => {
            const ctx = context.chart.ctx;
            return createGradient(ctx, 'rgba(245, 158, 11, 0.4)', 'rgba(245, 158, 11, 0.0)');
        };
    }
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

    if (type === 'fix') {
        btnFix.classList.add('active');
        btnPct.classList.remove('active');
        lblUnit.innerText = 'W';
        slider.max = 300;
    } else {
        btnFix.classList.remove('active');
        btnPct.classList.add('active');
        lblUnit.innerText = '%';
        slider.max = 100;
    }
}

function sendBoost(val) {
    document.getElementById('lbl-boost').innerText = val;
    ws.send(JSON.stringify({ boost: parseInt(val) }));
}

function sendSim(val) {
    document.getElementById('lbl-sim').innerText = val + " W";
    ws.send(JSON.stringify({ sim: parseInt(val) }));
}

function disconnectTrainer() {
    if (confirm("Terminate Bluetooth Link?")) {
        ws.send(JSON.stringify({ disconnect: true }));
    }
}