// MikroTik Controller - Main JavaScript

// ========== Utility Functions ==========

function showToast(message, isError) {
    var toast = document.getElementById('toast');
    var msg = document.getElementById('toast-message');
    msg.textContent = message;
    toast.className = 'toast ' + (isError ? 'toast-error' : 'toast-success');
    toast.classList.remove('hidden');
    setTimeout(function() { toast.classList.add('hidden'); }, 3000);
}

async function apiCall(url, method, body) {
    const opts = {
        method: method || 'GET',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
        }
    };
    if (body) {
        if (body instanceof FormData) {
            // Convert FormData to URL-encoded string
            const params = new URLSearchParams();
            for (const [key, value] of body.entries()) {
                params.append(key, value);
            }
            opts.body = params.toString();
        } else {
            opts.body = body;
        }
    }
    const resp = await fetch(url, opts);
    const data = await resp.json();
    if (!resp.ok) {
        throw new Error(data.error || 'Request failed');
    }
    return data;
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatRate(bytesPerSec) {
    return formatBytes(bytesPerSec) + '/s';
}

// ========== Router Management ==========

function showAddRouterModal() {
    document.getElementById('modalTitle').textContent = 'Add Router';
    document.getElementById('routerId').value = '';
    document.getElementById('routerName').value = '';
    document.getElementById('routerHost').value = '';
    document.getElementById('routerPort').value = '80';
    document.getElementById('routerUsername').value = '';
    document.getElementById('routerPassword').value = '';
    document.getElementById('routerPassword').required = true;
    document.getElementById('routerModal').classList.remove('hidden');
}

function editRouter(id, name, host, port, username) {
    document.getElementById('modalTitle').textContent = 'Edit Router';
    document.getElementById('routerId').value = id;
    document.getElementById('routerName').value = name;
    document.getElementById('routerHost').value = host;
    document.getElementById('routerPort').value = port;
    document.getElementById('routerUsername').value = username;
    document.getElementById('routerPassword').value = '';
    document.getElementById('routerPassword').required = false;
    document.getElementById('routerModal').classList.remove('hidden');
}

function closeModal() {
    document.getElementById('routerModal').classList.add('hidden');
}

async function submitRouter(e) {
    e.preventDefault();
    const id = document.getElementById('routerId').value;
    const formData = new FormData();
    formData.append('name', document.getElementById('routerName').value);
    formData.append('host', document.getElementById('routerHost').value);
    formData.append('port', document.getElementById('routerPort').value);
    formData.append('username', document.getElementById('routerUsername').value);
    formData.append('password', document.getElementById('routerPassword').value);

    try {
        if (id) {
            await apiCall('/api/routers/' + id, 'PUT', formData);
        } else {
            await apiCall('/api/routers', 'POST', formData);
        }
        showToast('Router saved successfully');
        closeModal();
        setTimeout(() => location.reload(), 500);
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deleteRouter(id) {
    if (!confirm('Are you sure you want to delete this router?')) return;
    try {
        await apiCall('/api/routers/' + id, 'DELETE');
        showToast('Router deleted');
        setTimeout(() => location.reload(), 500);
    } catch (err) {
        showToast(err.message, true);
    }
}

async function connectRouter(id) {
    try {
        await apiCall('/api/routers/' + id + '/connect', 'POST');
        showToast('Connected successfully');
        setTimeout(() => location.reload(), 1000);
    } catch (err) {
        showToast('Connection failed: ' + err.message, true);
    }
}

async function disconnectRouter(id) {
    try {
        await apiCall('/api/routers/' + id + '/disconnect', 'POST');
        showToast('Disconnected');
        setTimeout(() => location.reload(), 500);
    } catch (err) {
        showToast(err.message, true);
    }
}

// ========== Interface Management ==========

let currentRouterId = null;
let interfacesData = [];

async function loadRouterData(routerId) {
    if (!routerId) {
        document.getElementById('interfaceContent').classList.add('hidden');
        return;
    }
    currentRouterId = routerId;
    document.getElementById('interfaceContent').classList.remove('hidden');
    await Promise.all([loadInterfaces(), loadIPAddresses(), loadDNS()]);
}

async function loadInterfaces() {
    try {
        const data = await apiCall('/api/routers/' + currentRouterId + '/interfaces');
        interfacesData = data || [];
        const tbody = document.getElementById('interfacesTable');
        tbody.innerHTML = '';
        interfacesData.forEach(function(iface) {
            const statusClass = iface.running && !iface.disabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800';
            const statusText = iface.disabled ? 'Disabled' : (iface.running ? 'Running' : 'Not Running');
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + iface.name + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + iface.type + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (iface.mac || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (iface.mtu || '-') + '</td>' +
                '<td class="px-4 py-3"><span class="px-2 py-1 rounded-full text-xs ' + statusClass + '">' + statusText + '</span></td>' +
                '<td class="px-4 py-3 text-right"><button onclick="toggleIface(\'' + iface.name + '\')" class="text-blue-600 hover:text-blue-800 text-sm">' + (iface.disabled ? 'Enable' : 'Disable') + '</button></td>' +
                '</tr>';
        });
        // Update IP interface dropdown
        updateInterfaceDropdown();
    } catch (err) {
        showToast('Failed to load interfaces: ' + err.message, true);
    }
}

async function toggleIface(name) {
    try {
        await apiCall('/api/routers/' + currentRouterId + '/interfaces/' + name, 'PUT');
        showToast('Interface toggled');
        loadInterfaces();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function loadIPAddresses() {
    try {
        const data = await apiCall('/api/routers/' + currentRouterId + '/ip-addresses');
        const tbody = document.getElementById('ipAddressesTable');
        tbody.innerHTML = '';
        (data || []).forEach(function(addr) {
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + addr.address + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + addr.interface + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (addr.network || '-') + '</td>' +
                '<td class="px-4 py-3 text-right"><button onclick="deleteIP(\'' + addr.address + '\')" class="text-red-600 hover:text-red-800 text-sm">Delete</button></td>' +
                '</tr>';
        });
    } catch (err) {
        showToast('Failed to load IP addresses: ' + err.message, true);
    }
}

async function loadDNS() {
    try {
        const data = await apiCall('/api/routers/' + currentRouterId + '/dns');
        document.getElementById('dnsServers').value = (data.servers || []).join(',');
        document.getElementById('dnsAllowRemote').checked = data.allow_remote || false;
    } catch (err) {
        // DNS might not be available on all routers
    }
}

function showAddIPModal() {
    document.getElementById('ipModal').classList.remove('hidden');
}
function closeIPModal() {
    document.getElementById('ipModal').classList.add('hidden');
}

function updateInterfaceDropdown() {
    const sel = document.getElementById('ipInterface');
    sel.innerHTML = '';
    interfacesData.forEach(function(iface) {
        sel.innerHTML += '<option value="' + iface.name + '">' + iface.name + '</option>';
    });
}

async function addIPAddress(e) {
    e.preventDefault();
    const formData = new FormData();
    formData.append('address', document.getElementById('ipAddress').value);
    formData.append('interface', document.getElementById('ipInterface').value);
    try {
        await apiCall('/api/routers/' + currentRouterId + '/ip-addresses', 'POST', formData);
        showToast('IP address added');
        closeIPModal();
        loadIPAddresses();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deleteIP(address) {
    if (!confirm('Delete IP address ' + address + '?')) return;
    try {
        await apiCall('/api/routers/' + currentRouterId + '/ip-addresses/' + address, 'DELETE');
        showToast('IP address deleted');
        loadIPAddresses();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function updateDNS(e) {
    e.preventDefault();
    const formData = new FormData();
    formData.append('servers', document.getElementById('dnsServers').value);
    formData.append('allow_remote', document.getElementById('dnsAllowRemote').checked ? 'true' : 'false');
    try {
        await apiCall('/api/routers/' + currentRouterId + '/dns', 'PUT', formData);
        showToast('DNS updated');
    } catch (err) {
        showToast(err.message, true);
    }
}

// ========== PPPoE Management ==========

let pppoeRouterId = null;
let pppoeUsersData = [];

async function loadPPPoEData(routerId) {
    if (!routerId) {
        document.getElementById('pppoeContent').classList.add('hidden');
        return;
    }
    pppoeRouterId = routerId;
    document.getElementById('pppoeContent').classList.remove('hidden');
    loadPPPoEUsers();
}

async function loadPPPoEUsers() {
    try {
        const data = await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/users');
        pppoeUsersData = data || [];
        renderPPPoEUsers(pppoeUsersData);
    } catch (err) {
        showToast('Failed to load PPPoE users: ' + err.message, true);
    }
}

function renderPPPoEUsers(users) {
    const tbody = document.getElementById('pppoeUsersTable');
    tbody.innerHTML = '';
    users.forEach(function(user) {
        const statusClass = user.disabled ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800';
        const statusText = user.disabled ? 'Disabled' : 'Active';
        tbody.innerHTML += '<tr>' +
            '<td class="px-4 py-3 text-sm font-medium">' + user.name + '</td>' +
            '<td class="px-4 py-3 text-sm text-gray-500">' + user.profile + '</td>' +
            '<td class="px-4 py-3 text-sm text-gray-500">' + user.service + '</td>' +
            '<td class="px-4 py-3 text-sm text-gray-500">' + (user.comment || '-') + '</td>' +
            '<td class="px-4 py-3"><span class="px-2 py-1 rounded-full text-xs ' + statusClass + '">' + statusText + '</span></td>' +
            '<td class="px-4 py-3 text-right">' +
            '<button onclick="editPPPoEUser(\'' + user.name + '\', \'' + user.password + '\', \'' + user.profile + '\', \'' + user.service + '\', \'' + (user.comment || '') + '\', ' + user.disabled + ')" class="text-blue-600 hover:text-blue-800 text-sm mr-2">Edit</button>' +
            '<button onclick="deletePPPoEUser(\'' + user.name + '\')" class="text-red-600 hover:text-red-800 text-sm">Delete</button>' +
            '</td></tr>';
    });
}

function filterUsers() {
    const q = document.getElementById('searchUsers').value.toLowerCase();
    const filtered = pppoeUsersData.filter(function(u) {
        return u.name.toLowerCase().includes(q) || (u.comment || '').toLowerCase().includes(q);
    });
    renderPPPoEUsers(filtered);
}

function showAddUserModal() {
    document.getElementById('userModalTitle').textContent = 'Add PPPoE User';
    document.getElementById('editUserName').value = '';
    document.getElementById('pppoeUserName').value = '';
    document.getElementById('pppoeUserName').disabled = false;
    document.getElementById('pppoeUserPassword').value = '';
    document.getElementById('pppoeUserProfile').value = 'default';
    document.getElementById('pppoeUserService').value = 'pppoe';
    document.getElementById('pppoeUserComment').value = '';
    document.getElementById('pppoeUserDisabled').checked = false;
    document.getElementById('userModal').classList.remove('hidden');
}

function editPPPoEUser(name, password, profile, service, comment, disabled) {
    document.getElementById('userModalTitle').textContent = 'Edit PPPoE User';
    document.getElementById('editUserName').value = name;
    document.getElementById('pppoeUserName').value = name;
    document.getElementById('pppoeUserPassword').value = password;
    document.getElementById('pppoeUserProfile').value = profile;
    document.getElementById('pppoeUserService').value = service;
    document.getElementById('pppoeUserComment').value = comment;
    document.getElementById('pppoeUserDisabled').checked = disabled;
    document.getElementById('userModal').classList.remove('hidden');
}

function closeUserModal() {
    document.getElementById('userModal').classList.add('hidden');
}

async function submitPPPoEUser(e) {
    e.preventDefault();
    const oldName = document.getElementById('editUserName').value;
    const formData = new FormData();
    formData.append('name', document.getElementById('pppoeUserName').value);
    formData.append('password', document.getElementById('pppoeUserPassword').value);
    formData.append('profile', document.getElementById('pppoeUserProfile').value);
    formData.append('service', document.getElementById('pppoeUserService').value);
    formData.append('comment', document.getElementById('pppoeUserComment').value);
    formData.append('disabled', document.getElementById('pppoeUserDisabled').checked ? 'true' : 'false');

    try {
        if (oldName) {
            await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/users/' + oldName, 'PUT', formData);
        } else {
            await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/users', 'POST', formData);
        }
        showToast('PPPoE user saved');
        closeUserModal();
        loadPPPoEUsers();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deletePPPoEUser(name) {
    if (!confirm('Delete PPPoE user ' + name + '?')) return;
    try {
        await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/users/' + name, 'DELETE');
        showToast('User deleted');
        loadPPPoEUsers();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function importCSV(input) {
    if (!input.files[0]) return;
    const formData = new FormData();
    formData.append('file', input.files[0]);
    try {
        const result = await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/import', 'POST', formData);
        showToast('Imported ' + result.imported + ' users');
        loadPPPoEUsers();
    } catch (err) {
        showToast('Import failed: ' + err.message, true);
    }
}

function showTab(tab) {
    ['users', 'profiles', 'sessions'].forEach(function(t) {
        document.getElementById('panel-' + t).classList.toggle('hidden', t !== tab);
        document.getElementById('tab-' + t).classList.toggle('border-blue-500', t === tab);
        document.getElementById('tab-' + t).classList.toggle('text-blue-600', t === tab);
        document.getElementById('tab-' + t).classList.toggle('border-transparent', t !== tab);
    });
    if (tab === 'profiles') loadPPPoEProfiles();
    if (tab === 'sessions') loadPPPoESessions();
}

async function loadPPPoEProfiles() {
    try {
        const data = await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/profiles');
        const tbody = document.getElementById('pppoeProfilesTable');
        tbody.innerHTML = '';
        (data || []).forEach(function(p) {
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + p.name + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.local_address || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.remote_address || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.rate_limit || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.parent_profile || '-') + '</td>' +
                '</tr>';
        });
    } catch (err) {
        showToast('Failed to load profiles: ' + err.message, true);
    }
}

function showAddProfileModal() {
    document.getElementById('profileModal').classList.remove('hidden');
}
function closeProfileModal() {
    document.getElementById('profileModal').classList.add('hidden');
}

async function submitProfile(e) {
    e.preventDefault();
    const formData = new FormData();
    formData.append('name', document.getElementById('profileName').value);
    formData.append('rate_limit', document.getElementById('profileRateLimit').value);
    formData.append('local_address', document.getElementById('profileLocalAddr').value);
    formData.append('remote_address', document.getElementById('profileRemoteAddr').value);
    try {
        await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/profiles', 'POST', formData);
        showToast('Profile created');
        closeProfileModal();
        loadPPPoEProfiles();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function loadPPPoESessions() {
    try {
        const data = await apiCall('/api/routers/' + pppoeRouterId + '/pppoe/sessions');
        const tbody = document.getElementById('pppoeSessionsTable');
        tbody.innerHTML = '';
        (data || []).forEach(function(s) {
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + s.username + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + s.interface + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.ip_address || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.caller_id || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + s.state + '</td>' +
                '</tr>';
        });
    } catch (err) {
        showToast('Failed to load sessions: ' + err.message, true);
    }
}

// ========== Traffic Monitoring ==========

let trafficRouterId = null;
let trafficChart = null;
let autoRefresh = true;
let refreshInterval = null;

async function loadTrafficData(routerId) {
    if (!routerId) {
        document.getElementById('trafficContent').classList.add('hidden');
        if (refreshInterval) clearInterval(refreshInterval);
        return;
    }
    trafficRouterId = routerId;
    document.getElementById('trafficContent').classList.remove('hidden');
    initChart();
    await loadTraffic();
    loadClients();
    if (autoRefresh) startAutoRefresh();
}

function initChart() {
    const ctx = document.getElementById('trafficChart').getContext('2d');
    if (trafficChart) trafficChart.destroy();
    trafficChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'RX',
                    data: [],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59,130,246,0.1)',
                    fill: true,
                    tension: 0.3
                },
                {
                    label: 'TX',
                    data: [],
                    borderColor: '#10b981',
                    backgroundColor: 'rgba(16,185,129,0.1)',
                    fill: true,
                    tension: 0.3
                }
            ]
        },
        options: {
            responsive: true,
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        callback: function(value) { return formatRate(value); }
                    }
                }
            },
            plugins: {
                tooltip: {
                    callbacks: {
                        label: function(ctx) {
                            return ctx.dataset.label + ': ' + formatRate(ctx.parsed.y);
                        }
                    }
                }
            }
        }
    });
}

async function loadTraffic() {
    try {
        const data = await apiCall('/api/routers/' + trafficRouterId + '/traffic');
        const tbody = document.getElementById('trafficTable');
        tbody.innerHTML = '';
        let totalRx = 0, totalTx = 0;
        (data || []).forEach(function(t) {
            var rxRate = t.rx_rate || 0;
            var txRate = t.tx_rate || 0;
            var rxBytes = t.rx_bytes || 0;
            var txBytes = t.tx_bytes || 0;
            totalRx += rxRate;
            totalTx += txRate;
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + (t.interface || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-right text-blue-600">' + formatRate(rxRate) + '</td>' +
                '<td class="px-4 py-3 text-sm text-right text-green-600">' + formatRate(txRate) + '</td>' +
                '<td class="px-4 py-3 text-sm text-right text-gray-500">' + formatBytes(rxBytes) + '</td>' +
                '<td class="px-4 py-3 text-sm text-right text-gray-500">' + formatBytes(txBytes) + '</td>' +
                '</tr>';
        });

        // Update chart
        if (trafficChart) {
            const now = new Date().toLocaleTimeString();
            trafficChart.data.labels.push(now);
            trafficChart.data.datasets[0].data.push(totalRx);
            trafficChart.data.datasets[1].data.push(totalTx);
            if (trafficChart.data.labels.length > 30) {
                trafficChart.data.labels.shift();
                trafficChart.data.datasets[0].data.shift();
                trafficChart.data.datasets[1].data.shift();
            }
            trafficChart.update('none');
        }
    } catch (err) {
        showToast('Failed to load traffic: ' + err.message, true);
    }
}

async function loadClients() {
    if (!trafficRouterId) return;
    try {
        const data = await apiCall('/api/routers/' + trafficRouterId + '/traffic/clients');
        const tbody = document.getElementById('clientsTable');
        tbody.innerHTML = '';
        (data || []).forEach(function(c) {
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + (c.ip_address || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (c.mac || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (c.interface || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (c.host || '-') + '</td>' +
                '</tr>';
        });
    } catch (err) {
        // Clients may not be available on all routers
    }
}

function toggleAutoRefresh() {
    autoRefresh = !autoRefresh;
    const btn = document.getElementById('refreshToggle');
    if (autoRefresh) {
        btn.textContent = 'ON';
        btn.classList.remove('bg-red-600');
        btn.classList.add('bg-green-600');
        startAutoRefresh();
    } else {
        btn.textContent = 'OFF';
        btn.classList.remove('bg-green-600');
        btn.classList.add('bg-red-600');
        if (refreshInterval) clearInterval(refreshInterval);
    }
}

function startAutoRefresh() {
    if (refreshInterval) clearInterval(refreshInterval);
    refreshInterval = setInterval(function() {
        if (trafficRouterId && autoRefresh) {
            loadTraffic();
        }
    }, 5000);
}

// ========== Hotspot Management ==========

let hsRouterId = null;
let hsUsersData = [];

async function loadHotspotData(routerId) {
    if (!routerId) {
        document.getElementById('hotspotContent').classList.add('hidden');
        return;
    }
    hsRouterId = parseInt(routerId);
    document.getElementById('hotspotContent').classList.remove('hidden');
    loadHsUsers();
}

function showHsTab(tab) {
    ['users', 'profiles', 'active', 'servers'].forEach(function(t) {
        var panel = document.getElementById('hs-panel-' + t);
        var tabBtn = document.getElementById('hs-tab-' + t);
        if (panel) panel.classList.toggle('hidden', t !== tab);
        if (tabBtn) {
            tabBtn.classList.toggle('border-blue-500', t === tab);
            tabBtn.classList.toggle('text-blue-600', t === tab);
            tabBtn.classList.toggle('border-transparent', t !== tab);
        }
    });
    if (tab === 'profiles') loadHsProfiles();
    if (tab === 'active') loadHsActive();
    if (tab === 'servers') loadHsServers();
}

async function loadHsUsers() {
    try {
        var data = await apiCall('/api/routers/' + hsRouterId + '/hotspot/users');
        hsUsersData = data || [];
        renderHsUsers(hsUsersData);
    } catch (err) {
        showToast('Failed to load hotspot users: ' + err.message, true);
    }
}

function renderHsUsers(users) {
    var tbody = document.getElementById('hsUsersTable');
    tbody.innerHTML = '';
    users.forEach(function(u) {
        var statusClass = u.disabled ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800';
        var statusText = u.disabled ? 'Disabled' : 'Active';
        tbody.innerHTML += '<tr>' +
            '<td class="px-4 py-3 text-sm font-medium">' + (u.name || '-') + '</td>' +
            '<td class="px-4 py-3 text-sm text-gray-500">' + (u.profile || '-') + '</td>' +
            '<td class="px-4 py-3 text-sm text-gray-500">' + (u.comment || '-') + '</td>' +
            '<td class="px-4 py-3 text-sm text-gray-500">' + (u.limit_uptime || '-') + '</td>' +
            '<td class="px-4 py-3 text-sm text-gray-500">' + (u.limit_bytes ? formatBytes(parseInt(u.limit_bytes) || 0) : '-') + '</td>' +
            '<td class="px-4 py-3"><span class="px-2 py-1 rounded-full text-xs ' + statusClass + '">' + statusText + '</span></td>' +
            '<td class="px-4 py-3 text-right">' +
            '<button onclick="editHsUser(\'' + escStr(u.name) + '\', \'' + escStr(u.password) + '\', \'' + escStr(u.profile) + '\', \'' + escStr(u.comment) + '\', \'' + escStr(u.limit_uptime) + '\', \'' + escStr(u.limit_bytes) + '\', ' + u.disabled + ')" class="text-blue-600 hover:text-blue-800 text-sm mr-2">Edit</button>' +
            '<button onclick="deleteHsUser(\'' + escStr(u.name) + '\')" class="text-red-600 hover:text-red-800 text-sm">Delete</button>' +
            '</td></tr>';
    });
}

function escStr(s) {
    if (!s) return '';
    return s.replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '\\"');
}

function filterHsUsers() {
    var q = document.getElementById('hsSearchUsers').value.toLowerCase();
    var filtered = hsUsersData.filter(function(u) {
        return (u.name || '').toLowerCase().includes(q) || (u.comment || '').toLowerCase().includes(q);
    });
    renderHsUsers(filtered);
}

function showAddHsUserModal() {
    document.getElementById('hsUserModalTitle').textContent = 'Add Hotspot User';
    document.getElementById('hsEditUserName').value = '';
    document.getElementById('hsUserName').value = '';
    document.getElementById('hsUserName').disabled = false;
    document.getElementById('hsUserPassword').value = '';
    document.getElementById('hsUserProfile').value = 'default';
    document.getElementById('hsUserComment').value = '';
    document.getElementById('hsUserLimitUptime').value = '';
    document.getElementById('hsUserLimitBytes').value = '';
    document.getElementById('hsUserDisabled').checked = false;
    document.getElementById('hsUserModal').classList.remove('hidden');
}

function editHsUser(name, password, profile, comment, limitUptime, limitBytes, disabled) {
    document.getElementById('hsUserModalTitle').textContent = 'Edit Hotspot User';
    document.getElementById('hsEditUserName').value = name;
    document.getElementById('hsUserName').value = name;
    document.getElementById('hsUserName').disabled = true;
    document.getElementById('hsUserPassword').value = password;
    document.getElementById('hsUserProfile').value = profile || 'default';
    document.getElementById('hsUserComment').value = comment || '';
    document.getElementById('hsUserLimitUptime').value = limitUptime || '';
    document.getElementById('hsUserLimitBytes').value = limitBytes || '';
    document.getElementById('hsUserDisabled').checked = disabled;
    document.getElementById('hsUserModal').classList.remove('hidden');
}

function closeHsUserModal() {
    document.getElementById('hsUserModal').classList.add('hidden');
}

async function submitHsUser(e) {
    e.preventDefault();
    var oldName = document.getElementById('hsEditUserName').value;
    var formData = new FormData();
    formData.append('name', document.getElementById('hsUserName').value);
    formData.append('password', document.getElementById('hsUserPassword').value);
    formData.append('profile', document.getElementById('hsUserProfile').value);
    formData.append('comment', document.getElementById('hsUserComment').value);
    formData.append('limit_uptime', document.getElementById('hsUserLimitUptime').value);
    formData.append('limit_bytes', document.getElementById('hsUserLimitBytes').value);
    formData.append('disabled', document.getElementById('hsUserDisabled').checked ? 'true' : 'false');

    try {
        if (oldName) {
            await apiCall('/api/routers/' + hsRouterId + '/hotspot/users/' + encodeURIComponent(oldName), 'PUT', formData);
        } else {
            await apiCall('/api/routers/' + hsRouterId + '/hotspot/users', 'POST', formData);
        }
        showToast('Hotspot user saved');
        closeHsUserModal();
        loadHsUsers();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function deleteHsUser(name) {
    if (!confirm('Delete hotspot user ' + name + '?')) return;
    try {
        await apiCall('/api/routers/' + hsRouterId + '/hotspot/users/' + encodeURIComponent(name), 'DELETE');
        showToast('Hotspot user deleted');
        loadHsUsers();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function importHsCSV(input) {
    if (!input.files[0]) return;
    var formData = new FormData();
    formData.append('file', input.files[0]);
    try {
        var result = await apiCall('/api/routers/' + hsRouterId + '/hotspot/import', 'POST', formData);
        showToast('Imported ' + result.imported + ' users');
        loadHsUsers();
    } catch (err) {
        showToast('Import failed: ' + err.message, true);
    }
}

async function loadHsProfiles() {
    try {
        var data = await apiCall('/api/routers/' + hsRouterId + '/hotspot/profiles');
        var tbody = document.getElementById('hsProfilesTable');
        tbody.innerHTML = '';
        (data || []).forEach(function(p) {
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + (p.name || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.rate_limit || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.session_timeout || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.idle_timeout || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.shared_users || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (p.login_by || '-') + '</td>' +
                '</tr>';
        });
    } catch (err) {
        showToast('Failed to load profiles: ' + err.message, true);
    }
}

function showAddHsProfileModal() {
    document.getElementById('hsProfileName').value = '';
    document.getElementById('hsProfileRateLimit').value = '';
    document.getElementById('hsProfileSessionTimeout').value = '';
    document.getElementById('hsProfileIdleTimeout').value = '';
    document.getElementById('hsProfileSharedUsers').value = '1';
    document.getElementById('hsProfileLoginBy').value = 'http-pap';
    document.getElementById('hsProfileModal').classList.remove('hidden');
}

function closeHsProfileModal() {
    document.getElementById('hsProfileModal').classList.add('hidden');
}

async function submitHsProfile(e) {
    e.preventDefault();
    var formData = new FormData();
    formData.append('name', document.getElementById('hsProfileName').value);
    formData.append('rate_limit', document.getElementById('hsProfileRateLimit').value);
    formData.append('session_timeout', document.getElementById('hsProfileSessionTimeout').value);
    formData.append('idle_timeout', document.getElementById('hsProfileIdleTimeout').value);
    formData.append('shared_users', document.getElementById('hsProfileSharedUsers').value);
    formData.append('login_by', document.getElementById('hsProfileLoginBy').value);
    try {
        await apiCall('/api/routers/' + hsRouterId + '/hotspot/profiles', 'POST', formData);
        showToast('Profile created');
        closeHsProfileModal();
        loadHsProfiles();
    } catch (err) {
        showToast(err.message, true);
    }
}

async function loadHsActive() {
    try {
        var data = await apiCall('/api/routers/' + hsRouterId + '/hotspot/active');
        var tbody = document.getElementById('hsActiveTable');
        tbody.innerHTML = '';
        (data || []).forEach(function(s) {
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + (s.username || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.ip || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.mac || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.uptime || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + formatBytes(s.bytes_in || 0) + ' / ' + formatBytes(s.bytes_out || 0) + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.idle_time || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.server || '-') + '</td>' +
                '</tr>';
        });
    } catch (err) {
        showToast('Failed to load active sessions: ' + err.message, true);
    }
}

async function loadHsServers() {
    try {
        var data = await apiCall('/api/routers/' + hsRouterId + '/hotspot/servers');
        var tbody = document.getElementById('hsServersTable');
        tbody.innerHTML = '';
        (data || []).forEach(function(s) {
            var statusClass = s.disabled ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800';
            var statusText = s.disabled ? 'Disabled' : 'Running';
            tbody.innerHTML += '<tr>' +
                '<td class="px-4 py-3 text-sm font-medium">' + (s.name || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.interface || '-') + '</td>' +
                '<td class="px-4 py-3 text-sm text-gray-500">' + (s.address_pool || '-') + '</td>' +
                '<td class="px-4 py-3"><span class="px-2 py-1 rounded-full text-xs ' + statusClass + '">' + statusText + '</span></td>' +
                '</tr>';
        });
    } catch (err) {
        showToast('Failed to load servers: ' + err.message, true);
    }
}

// ========== Auto-load on page ready ==========
document.addEventListener('DOMContentLoaded', function() {
    // Auto-select router if query param present
    var sel = document.getElementById('routerSelect');
    if (sel && sel.value) {
        sel.dispatchEvent(new Event('change'));
    }
    var hsSel = document.getElementById('hsRouterSelect');
    if (hsSel && hsSel.value) {
        hsSel.dispatchEvent(new Event('change'));
    }
});
