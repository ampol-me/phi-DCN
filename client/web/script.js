// API endpoints
const API_BASE = '/api';
const ENDPOINTS = {
    CONFIG: `${API_BASE}/config`,
    STATUS: `${API_BASE}/status`,
    START: `${API_BASE}/start`,
    STOP: `${API_BASE}/stop`,
    CLIENTS: `${API_BASE}/clients`,
    MICS: `${API_BASE}/mics`
};

// DOM elements
const serverStatus = document.getElementById('serverStatus');
const startStopBtn = document.getElementById('startStopBtn');
const configForm = document.getElementById('configForm');
const clientsList = document.getElementById('clientsList');
const micsList = document.getElementById('micsList');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadConfig();
    checkServerStatus();
    setInterval(checkServerStatus, 5000);
    setInterval(updateClients, 5000);
    setInterval(updateMics, 5000);
});

// Event listeners
startStopBtn.addEventListener('click', toggleServer);
configForm.addEventListener('submit', saveConfig);

// Functions
async function loadConfig() {
    try {
        const response = await fetch(ENDPOINTS.CONFIG);
        const config = await response.json();
        
        document.getElementById('apiHost').value = config.APIHost || '';
        document.getElementById('apiPort').value = config.APIPort || '';
        document.getElementById('tcpPort').value = config.TCPServerPort || '';
    } catch (error) {
        console.error('Error loading config:', error);
    }
}

async function checkServerStatus() {
    try {
        const response = await fetch(ENDPOINTS.STATUS);
        const status = await response.json();
        
        serverStatus.textContent = `Server Status: ${status.serverStatus}`;
        if (status.serverStatus === 'running') {
            startStopBtn.textContent = 'Stop Server';
            startStopBtn.classList.add('stopped');
        } else {
            startStopBtn.textContent = 'Start Server';
            startStopBtn.classList.remove('stopped');
        }
    } catch (error) {
        console.error('Error checking status:', error);
        serverStatus.textContent = 'Server Status: Error';
    }
}

async function toggleServer() {
    const endpoint = startStopBtn.textContent === 'Start Server' ? ENDPOINTS.START : ENDPOINTS.STOP;
    
    try {
        const response = await fetch(endpoint);
        const result = await response.json();
        
        if (result.status === 'success') {
            checkServerStatus();
        }
    } catch (error) {
        console.error('Error toggling server:', error);
    }
}

async function saveConfig(event) {
    event.preventDefault();
    
    const config = {
        APIHost: document.getElementById('apiHost').value,
        APIPort: document.getElementById('apiPort').value,
        TCPServerPort: document.getElementById('tcpPort').value
    };
    
    try {
        const response = await fetch(ENDPOINTS.CONFIG, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(config)
        });
        
        const result = await response.json();
        if (result.status === 'success') {
            alert('Configuration saved successfully');
        }
    } catch (error) {
        console.error('Error saving config:', error);
        alert('Error saving configuration');
    }
}

async function updateClients() {
    try {
        const response = await fetch(ENDPOINTS.CLIENTS);
        const data = await response.json();
        
        clientsList.innerHTML = `
            <p>Total Clients: ${data.count}</p>
            <ul>
                ${data.clients.map(client => `<li>${client}</li>`).join('')}
            </ul>
        `;
    } catch (error) {
        console.error('Error updating clients:', error);
    }
}

async function updateMics() {
    try {
        const response = await fetch(ENDPOINTS.MICS);
        const data = await response.json();
        
        micsList.innerHTML = `
            <p>Total Mics: ${data.count}</p>
            <ul>
                ${data.mics.map(mic => `<li>${mic}</li>`).join('')}
            </ul>
        `;
    } catch (error) {
        console.error('Error updating mics:', error);
    }
} 