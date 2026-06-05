// Search bar filtration
document.getElementById('flow-search').addEventListener('input', function(e) {
    const term = e.target.value.toLowerCase();
    const links = document.querySelectorAll('.flow-link');
    
    links.forEach(link => {
        const text = link.querySelector('.nav-text').textContent.toLowerCase();
        if (text.includes(term)) {
            link.style.display = 'flex';
        } else {
            link.style.display = 'none';
        }
    });
});

// Interactive tab switching
function switchTab(btn, paneId) {
    const pane = document.getElementById(paneId);
    if (!pane) return;
    
    // Deactivate sibling buttons
    const nav = btn.parentElement;
    nav.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    
    // Deactivate sibling panes
    const panes = pane.parentElement;
    panes.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
    pane.classList.add('active');
}

// Clipboard copying
function copyToClipboard(btn) {
    const code = btn.parentElement.nextElementSibling.querySelector('code').textContent;
    navigator.clipboard.writeText(code).then(() => {
        const originalText = btn.textContent;
        btn.textContent = 'Copied!';
        btn.style.backgroundColor = 'var(--success-bg)';
        btn.style.color = '#34d399';
        
        setTimeout(() => {
            btn.textContent = originalText;
            btn.style.backgroundColor = '';
            btn.style.color = '';
        }, 1500);
    });
}

const copySnippet = copyToClipboard;

function copyText(btn) {
    const cmd = btn.previousElementSibling.textContent;
    navigator.clipboard.writeText(cmd).then(() => {
        const originalText = btn.textContent;
        btn.textContent = 'Copied!';
        setTimeout(() => {
            btn.textContent = originalText;
        }, 1500);
    });
}

// HTMX Lifecycle and Page State Synchronization
document.body.addEventListener('htmx:afterSwap', function(evt) {
    // 1. Re-render Mermaid sequence diagrams dynamically
    const swappedContent = evt.detail.target;
    const diagrams = swappedContent.querySelectorAll('.mermaid');
    if (diagrams.length > 0) {
        try {
            mermaid.run({
                nodes: diagrams
            });
        } catch (e) {
            console.error('Mermaid render error:', e);
        }
    }
    
    // 2. Sync active sidebar item highlight
    // Extract slug from URL path (e.g. /flows/buyer-auth -> buyer-auth)
    const path = window.location.pathname;
    let activeId = 'nav-dashboard';
    let currentCrumb = 'System Dashboard';
    
    if (path.startsWith('/flows/')) {
        const slug = path.split('/')[2];
        activeId = `nav-${slug}`;
        
        const activeLink = document.getElementById(activeId);
        if (activeLink) {
            currentCrumb = activeLink.querySelector('.nav-text').textContent;
        }
    }
    
    // Update active highlight classes
    document.querySelectorAll('.nav-link').forEach(link => {
        if (link.id === activeId) {
            link.classList.add('active');
        } else {
            link.classList.remove('active');
        }
    });
    
    // Update Breadcrumbs current text
    const breadcrumb = document.getElementById('breadcrumb-current');
    if (breadcrumb) {
        breadcrumb.textContent = currentCrumb;
    }
});

// Run Mermaid initial render on direct full page loads
document.addEventListener('DOMContentLoaded', function() {
    const diagrams = document.querySelectorAll('.mermaid');
    if (diagrams.length > 0) {
        try {
            mermaid.run({
                nodes: diagrams
            });
        } catch (e) {
            console.error('Mermaid init error:', e);
        }
    }
});
