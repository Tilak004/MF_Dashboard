// Logout handler
document.getElementById('logoutBtn').addEventListener('click', async () => {
    await fetch('/api/logout', { method: 'POST' });
    window.location.href = '/login';
});

// Format currency
function formatCurrency(amount) {
    return '₹' + amount.toLocaleString('en-IN', { maximumFractionDigits: 0 });
}

// Format percentage
function formatPercent(value) {
    return value.toFixed(2) + '%';
}

// Show loading spinner
function showLoading(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.innerHTML = '<div class="loading"></div>';
    }
}

// Animate number counting
function animateValue(element, start, end, duration) {
    const range = end - start;
    const increment = range / (duration / 16);
    let current = start;

    const timer = setInterval(() => {
        current += increment;
        if ((increment > 0 && current >= end) || (increment < 0 && current <= end)) {
            element.textContent = formatCurrency(end);
            clearInterval(timer);
        } else {
            element.textContent = formatCurrency(Math.round(current));
        }
    }, 16);
}

// Show toast notification
function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: ${type === 'error' ? '#ef4444' : type === 'success' ? '#10b981' : '#6366f1'};
        color: white;
        padding: 1rem 2rem;
        border-radius: 8px;
        box-shadow: 0 10px 30px rgba(0,0,0,0.3);
        z-index: 10000;
        animation: slideIn 0.3s ease-out;
    `;
    toast.textContent = message;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.style.animation = 'fadeOut 0.3s ease-out';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// Load dashboard data
async function loadDashboard() {
    try {
        // Show loading state
        document.getElementById('totalAUM').textContent = 'Loading...';
        document.getElementById('totalClients').textContent = '...';
        document.getElementById('totalInvested').textContent = 'Loading...';
        document.getElementById('overallReturns').textContent = '...';

        const response = await fetch('/api/dashboard/summary');
        if (!response.ok) throw new Error('Failed to fetch dashboard data');

        const data = await response.json();

        // Update summary stats with animation
        setTimeout(() => {
            const aumElement = document.getElementById('totalAUM');
            const investedElement = document.getElementById('totalInvested');

            animateValue(aumElement, 0, data.total_aum || 0, 1000);
            animateValue(investedElement, 0, data.total_invested || 0, 1000);

            document.getElementById('totalClients').textContent = data.total_clients || 0;
            document.getElementById('overallReturns').textContent = formatPercent(data.overall_returns || 0);
        }, 100);

        // Update SIP alert count
        const missedSips = data.missed_sips || 0;
        const alertBadge = document.getElementById('sipAlertCount');
        alertBadge.textContent = missedSips;
        alertBadge.style.display = missedSips > 0 ? 'inline-block' : 'none';

        // Render asset allocation chart
        renderAllocationChart(data.asset_allocation || {});

        // Load SIP alerts
        loadSIPAlerts();

        // Render client table
        renderClientsTable(data.clients_with_aum || []);

        // Render top performers
        renderTopPerformers(data.top_performers || {});

    } catch (error) {
        console.error('Failed to load dashboard:', error);
        showToast('Failed to load dashboard data', 'error');
    }
}

// Render asset allocation pie chart
function renderAllocationChart(allocation) {
    const ctx = document.getElementById('allocationChart');

    const labels = Object.keys(allocation);
    const values = Object.values(allocation);

    // Destroy existing chart if it exists
    if (window.allocationChartInstance) {
        window.allocationChartInstance.destroy();
    }

    window.allocationChartInstance = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: labels.length > 0 ? labels : ['No Data'],
            datasets: [{
                data: values.length > 0 ? values : [1],
                backgroundColor: [
                    'rgba(99, 102, 241, 0.8)',   // Indigo
                    'rgba(16, 185, 129, 0.8)',   // Green
                    'rgba(245, 158, 11, 0.8)',   // Orange
                    'rgba(139, 92, 246, 0.8)',   // Purple
                    'rgba(239, 68, 68, 0.8)'     // Red
                ],
                borderColor: [
                    'rgba(99, 102, 241, 1)',
                    'rgba(16, 185, 129, 1)',
                    'rgba(245, 158, 11, 1)',
                    'rgba(139, 92, 246, 1)',
                    'rgba(239, 68, 68, 1)'
                ],
                borderWidth: 2
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        color: '#cbd5e1',
                        padding: 15,
                        font: {
                            size: 12,
                            weight: '500'
                        }
                    }
                },
                tooltip: {
                    backgroundColor: 'rgba(30, 41, 59, 0.95)',
                    titleColor: '#fff',
                    bodyColor: '#cbd5e1',
                    borderColor: 'rgba(99, 102, 241, 0.5)',
                    borderWidth: 1,
                    padding: 12,
                    displayColors: true,
                    callbacks: {
                        label: function(context) {
                            return context.label + ': ' + context.parsed.toFixed(2) + '%';
                        }
                    }
                }
            },
            animation: {
                animateRotate: true,
                animateScale: true,
                duration: 1000,
                easing: 'easeOutQuart'
            }
        }
    });
}

// Grouped SIP data store — indexed so client names with special chars are safe in onclick
let _sipGroups = []; // [{clientName, alerts:[]}]

// Format SIP date for notification message (e.g. "19 February 2026")
function formatSIPDateLong(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-IN', { day: 'numeric', month: 'long', year: 'numeric' });
}

// Copy single-SIP WhatsApp message
function copyNotifyMessage(dateStr) {
    const formattedDate = formatSIPDateLong(dateStr);
    const message = `Hello 😊\nYour SIP is coming up on *${formattedDate}*. A quick nudge to keep sufficient balance in your account so the investment goes through smoothly.\n\nConsistency is key to long-term wealth creation 👍`;
    _writeClipboard(message, 'Message copied!');
}

// Copy cumulative message — all SIPs for a client in one go
function copyCumulativeMessage(groupIndex) {
    const group = _sipGroups[groupIndex];
    if (!group) return;

    const lines = group.alerts.map(a => {
        const date = formatSIPDateLong(a.next_sip_date);
        const amt  = '₹' + a.amount.toLocaleString('en-IN');
        return `• *${a.fund_name}* — ${amt} on ${date}`;
    }).join('\n');

    const message = `Hello 😊\n\nYour upcoming SIPs:\n\n${lines}\n\nA quick nudge to keep sufficient balance in your account so the investments go through smoothly.\n\nConsistency is key to long-term wealth creation 👍`;
    _writeClipboard(message, 'Cumulative message copied!');
}

// Shared clipboard helper
function _writeClipboard(text, successMsg) {
    navigator.clipboard.writeText(text).then(() => {
        showToast(successMsg, 'success');
    }).catch(() => {
        const ta = document.createElement('textarea');
        ta.value = text;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
        showToast(successMsg, 'success');
    });
}

// Load SIP alerts
async function loadSIPAlerts() {
    try {
        const response = await fetch('/api/dashboard/sip-alerts');
        const alerts = await response.json();

        const container = document.getElementById('sipAlerts');

        if (alerts && alerts.length > 0) {
            // Group by client name, preserve insertion order
            const map = new Map();
            alerts.forEach(alert => {
                if (!map.has(alert.client_name)) map.set(alert.client_name, []);
                map.get(alert.client_name).push(alert);
            });

            // Store indexed so onclick can reference safely
            _sipGroups = Array.from(map.entries()).map(([clientName, clientAlerts]) => ({
                clientName, alerts: clientAlerts
            }));

            const btnBase = 'padding:0.2rem 0.55rem; border-radius:5px; cursor:pointer; font-size:0.7rem; white-space:nowrap;';

            container.innerHTML = _sipGroups.map((group, gi) => {
                const rows = group.alerts.map(alert => {
                    const dateShort = new Date(alert.next_sip_date).toLocaleDateString('en-IN');
                    return `
                        <div style="display:flex; align-items:center; justify-content:space-between; padding:0.35rem 0; border-bottom:1px solid rgba(255,255,255,0.04);">
                            <div style="flex:1; min-width:0;">
                                <div style="color:#cbd5e1; font-size:0.78rem; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">${alert.fund_name}</div>
                                <div style="margin-top:0.1rem;">
                                    <span style="color:#f59e0b; font-size:0.75rem;">₹${alert.amount.toLocaleString('en-IN')}</span>
                                    <span style="color:#94a3b8; font-size:0.75rem;"> · ${dateShort}</span>
                                    <span style="color:#10b981; font-size:0.75rem; font-weight:600;"> · in ${alert.days_until}d</span>
                                </div>
                            </div>
                            <button onclick="copyNotifyMessage('${alert.next_sip_date}')" title="Copy single SIP message"
                                style="${btnBase} margin-left:0.5rem; background:rgba(99,102,241,0.1); border:1px solid rgba(99,102,241,0.3); color:#818cf8;"
                                onmouseover="this.style.background='rgba(99,102,241,0.22)'" onmouseout="this.style.background='rgba(99,102,241,0.1)'">📋</button>
                        </div>`;
                }).join('');

                return `
                    <div class="alert-item" style="animation-delay:${gi * 0.1}s; padding:0.75rem 1rem; margin-bottom:0.5rem;">
                        <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:0.5rem;">
                            <strong style="font-size:0.88rem;">${group.clientName}</strong>
                            <button onclick="copyCumulativeMessage(${gi})" title="Copy all SIPs for this client in one message"
                                style="${btnBase} padding:0.25rem 0.7rem; background:rgba(16,185,129,0.1); border:1px solid rgba(16,185,129,0.32); color:#34d399;"
                                onmouseover="this.style.background='rgba(16,185,129,0.22)'" onmouseout="this.style.background='rgba(16,185,129,0.1)'">📋 Copy All</button>
                        </div>
                        ${rows}
                    </div>`;
            }).join('');
        } else {
            container.innerHTML = `
                <div style="text-align: center; padding: 2rem; color: #94a3b8;">
                    <div style="font-size: 3rem; margin-bottom: 0.5rem;">📅</div>
                    <p style="font-weight: 600;">No upcoming SIP installments</p>
                </div>
            `;
        }
    } catch (error) {
        console.error('Failed to load SIP alerts:', error);
        document.getElementById('sipAlerts').innerHTML = `
            <p style="color: #ef4444;">Failed to load SIP alerts</p>
        `;
    }
}

// Render clients table
function renderClientsTable(clients) {
    const tbody = document.querySelector('#clientsTable tbody');

    if (clients.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" style="text-align: center; padding: 2rem; color: #64748b;">
                    No clients found. Add clients to get started!
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = clients.map((client, index) => `
        <tr style="animation: fadeIn 0.5s ease-out ${index * 0.05}s both;">
            <td style="font-weight: 600;">${client.client_name}</td>
            <td style="font-weight: 600; color: #6366f1;">${formatCurrency(client.current_value)}</td>
            <td>${formatCurrency(client.total_invested)}</td>
            <td style="color: ${client.absolute_return >= 0 ? '#10b981' : '#ef4444'}; font-weight: 600;">
                ${formatCurrency(client.absolute_return)}
                ${client.absolute_return >= 0 ? '▲' : '▼'}
            </td>
            <td style="color: ${client.xirr >= 0 ? '#10b981' : '#ef4444'}; font-weight: 600;">
                ${formatPercent(client.xirr)}
                ${client.xirr >= 0 ? '▲' : '▼'}
            </td>
            <td>
                <a href="/client-portfolio?id=${client.client_id}" class="btn btn-sm btn-primary" style="text-decoration: none; padding: 0.5rem 1rem;">
                    View Portfolio
                </a>
            </td>
        </tr>
    `).join('');
}

// Render top performers
function renderTopPerformers(topPerformers) {
    const tbody = document.querySelector('#topClientsTable tbody');

    const topClients = topPerformers.top_clients || [];

    if (topClients.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="4" style="text-align: center; padding: 2rem; color: #64748b;">
                    No performance data available yet
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = topClients.map((client, index) => {
        const medals = ['🥇', '🥈', '🥉'];
        const medal = index < 3 ? medals[index] : '';

        // Color based on XIRR value
        const xirrColor = client.xirr >= 0 ? '#10b981' : '#ef4444';
        const xirrArrow = client.xirr >= 0 ? '▲' : '▼';

        return `
            <tr style="animation: fadeIn 0.5s ease-out ${index * 0.1}s both;">
                <td style="font-size: 1.5rem;">${medal} ${index + 1}</td>
                <td style="font-weight: 600;">${client.client_name}</td>
                <td style="color: ${xirrColor}; font-weight: 700; font-size: 1.1rem;">
                    ${formatPercent(client.xirr)} ${xirrArrow}
                </td>
                <td style="font-weight: 600; color: #6366f1;">
                    ${formatCurrency(client.current_value)}
                </td>
            </tr>
        `;
    }).join('');
}

// Refresh NAV
async function refreshNAV(event) {
    const btn = event.target;
    const originalText = btn.textContent;

    btn.disabled = true;
    btn.innerHTML = '<span class="loading"></span> Refreshing...';

    try {
        const response = await fetch('/api/dashboard/refresh-nav', { method: 'POST' });
        if (response.ok) {
            showToast('NAV refresh completed successfully!', 'success');
            await loadDashboard();
        } else {
            throw new Error('NAV refresh failed');
        }
    } catch (error) {
        showToast('NAV refresh failed: ' + error.message, 'error');
    } finally {
        btn.disabled = false;
        btn.textContent = originalText;
    }
}

// Add fadeOut animation to CSS dynamically
const style = document.createElement('style');
style.textContent = `
    @keyframes fadeOut {
        from { opacity: 1; transform: translateY(0); }
        to { opacity: 0; transform: translateY(-20px); }
    }
`;
document.head.appendChild(style);

// Load market overview
async function loadMarketData() {
    try {
        const response = await fetch('/api/market/overview');
        if (!response.ok) throw new Error('Market API error');
        const indices = await response.json();

        const grid = document.getElementById('marketGrid');
        const statusEl = document.getElementById('marketStatus');

        if (!indices || indices.length === 0) {
            grid.innerHTML = '<div style="color:#64748b; padding:1rem;">Market data unavailable</div>';
            return;
        }

        const anyOpen = indices.some(i => i.is_market_open);
        statusEl.textContent = anyOpen ? '🟢 Market Open' : '🔴 Market Closed';

        grid.innerHTML = indices.map(idx => {
            const isGold = idx.symbol === 'xauusd';
            const isForex = idx.symbol === 'usdinr';
            const up = idx.change >= 0;
            const color = up ? '#10b981' : '#ef4444';
            const arrow = up ? '▲' : '▼';

            let priceStr;
            if (isForex) {
                priceStr = '₹' + idx.price.toFixed(2);
            } else if (isGold) {
                priceStr = '$' + idx.price.toLocaleString('en-US', { maximumFractionDigits: 2 });
            } else {
                priceStr = idx.price.toLocaleString('en-IN', { maximumFractionDigits: 2 });
            }

            return `
                <div style="background:rgba(255,255,255,0.04); border:1px solid rgba(255,255,255,0.08); border-radius:10px; padding:1rem; text-align:center;">
                    <div style="color:#94a3b8; font-size:0.75rem; margin-bottom:0.3rem;">${idx.name}</div>
                    <div style="font-size:1.15rem; font-weight:700; color:#f8fafc;">${priceStr}</div>
                    <div style="color:${color}; font-size:0.85rem; font-weight:600; margin-top:0.25rem;">
                        ${arrow} ${Math.abs(idx.change).toFixed(2)} (${Math.abs(idx.change_percent).toFixed(2)}%)
                    </div>
                    <div style="color:#475569; font-size:0.7rem; margin-top:0.3rem;">${idx.last_updated}</div>
                </div>`;
        }).join('');
    } catch (err) {
        console.error('Market data failed:', err);
        document.getElementById('marketGrid').innerHTML =
            '<div style="color:#64748b; padding:1rem;">Market data unavailable</div>';
    }
}

// Load dashboard on page load
loadDashboard();
loadMarketData();
