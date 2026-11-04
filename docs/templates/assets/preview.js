// Local-only presentation behavior. No requests, storage, authentication or jobs.
(() => {
  'use strict';
  const icons = {
    grid:'<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
    layers:'<path d="m12 3 9 5-9 5-9-5 9-5Z"/><path d="m3 12 9 5 9-5M3 16l9 5 9-5"/>',
    play:'<path d="m9 5 11 7-11 7V5Z"/>',
    shield:'<path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z"/><path d="m8 12 3 3 5-6"/>',
    file:'<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9l-6-6Z"/><path d="M14 3v6h6M8 13h8M8 17h5"/>',
    arrow:'<path d="M5 12h14m-5-5 5 5-5 5"/>',
    back:'<path d="M19 12H5m5-5-5 5 5 5"/>',
    plus:'<path d="M12 5v14M5 12h14"/>',
    search:'<circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 5 5"/>',
    globe:'<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c5 5 5 13 0 18-5-5-5-13 0-18Z"/>',
    lock:'<rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3"/>',
    check:'<path d="m5 12 4 4L19 6"/>',
    info:'<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7h.01"/>',
    clock:'<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    stop:'<rect x="5" y="5" width="14" height="14" rx="2"/>',
    download:'<path d="M12 3v12m-5-5 5 5 5-5M4 17v4h16v-4"/>',
    key:'<circle cx="8" cy="9" r="4"/><path d="m11 12 9 9m-3-3 3-3m-6 0 3-3"/>',
    users:'<circle cx="9" cy="8" r="3"/><path d="M3 21v-3a6 6 0 0 1 12 0v3M16 5a3 3 0 0 1 0 6m2 4a5 5 0 0 1 3 4v2"/>',
    chevron:'<path d="m9 5 7 7-7 7"/>',
    activity:'<path d="M2 12h5l3-8 4 16 3-8h5"/>',
    link:'<path d="m9 15 6-6M8 17l-1 1a4 4 0 0 1-6-6l5-5a4 4 0 0 1 6 0m0 0 1-1a4 4 0 0 1 6 6l-5 5a4 4 0 0 1-6 0" transform="translate(2 0)"/>',
    alert:'<path d="m12 3 10 18H2L12 3Z"/><path d="M12 9v5m0 3h.01"/>',
    external:'<path d="M14 3h7v7m0-7L10 14M10 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-5"/>',
    code:'<path d="m8 7-5 5 5 5m8-10 5 5-5 5m-3-14-2 18"/>'
  };
  const svg = (name) => `<svg class="icon" viewBox="0 0 24 24" aria-hidden="true">${icons[name] || icons.grid}</svg>`;
  document.querySelectorAll('[data-icon]').forEach(el => { el.innerHTML = svg(el.dataset.icon); });
  const page = document.body.dataset.page || '';
  const nav = document.body.dataset.nav || '';
  const shell = document.querySelector('[data-shell]');
  if (shell) {
    const links = [['applications','applications.html','layers','Applications'],['assessments','assessments.html','play','Assessments'],['findings','findings.html','shield','Findings'],['reports','reports.html','file','Reports']];
    shell.innerHTML = `<a class="skip-link" href="#main">Skip to content</a><aside class="sidebar"><a class="brand" href="index.html" aria-label="Tsunami preview home"><svg viewBox="0 0 32 32" aria-hidden="true"><path d="M3 11c5-7 9 7 14 0s9 0 12-3M3 17c5-7 9 7 14 0s9 0 12-3M3 23c5-7 9 7 14 0s9 0 12-3"/></svg>tsunami<span style="color:#67d2bc">.</span></a><div class="brand-sub">Assessment workspace</div><div class="workspace-card"><div class="avatar">DW</div><div><strong class="small" style="color:white">Demo workspace</strong><div class="small">Fictional customer</div></div></div><div class="nav-label">Workspace</div><nav class="nav" aria-label="Workspace navigation">${links.map(([id,href,icon,label]) => `<a href="${href}" class="${nav === id ? 'active' : ''}" ${nav === id ? 'aria-current="page"' : ''}>${svg(icon)}${label}</a>`).join('')}</nav><div class="sidebar-bottom"><span class="demo-dot"></span>Local design preview<a href="index.html">${svg('grid')}Explore the user journey</a><a href="../design.md">${svg('file')}Read the product design</a></div></aside>`;
  }
  const top = document.querySelector('[data-topbar]');
  if (top) {
    const crumb = top.dataset.topbar;
    top.innerHTML = `<header class="topbar"><div class="breadcrumbs"><span>Demo workspace</span><span>/</span><strong></strong></div><div class="top-actions"><a href="index.html">${svg('grid')}Flow map</a><span class="avatar" aria-label="Demo operator">DO</span></div></header><div class="prototype-bar">${svg('info')}<span><strong>Visual prototype.</strong> Fictional data · planned interface · no backend connection · actions are simulated.</span></div>`;
    top.querySelector('.breadcrumbs strong').textContent = crumb;
  }
  document.querySelectorAll('[data-toast]').forEach(button => button.addEventListener('click', () => toast(button.dataset.toast)));
  function toast(message) {
    document.querySelector('.toast')?.remove();
    const box = document.createElement('div'); box.className = 'toast'; box.setAttribute('role','status'); box.textContent = message; document.body.append(box); setTimeout(() => box.remove(),5000);
  }
  document.querySelectorAll('form').forEach(form => form.addEventListener('submit', event => event.preventDefault()));
  document.querySelectorAll('[data-tab]').forEach(button => button.addEventListener('click', () => {
    const group = button.closest('[data-tabs]'); if (!group) return;
    group.querySelectorAll('[data-tab]').forEach(tab => { const selected = tab === button; tab.classList.toggle('active',selected); tab.setAttribute('aria-selected',String(selected)); });
    group.querySelectorAll('[data-panel]').forEach(panel => { panel.hidden = panel.dataset.panel !== button.dataset.tab; });
  }));
  document.querySelectorAll('[role="tablist"]').forEach(list => list.addEventListener('keydown', event => {
    const tabs = [...list.querySelectorAll('[role="tab"]')];
    const current = tabs.indexOf(document.activeElement);
    if (current < 0 || !['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return;
    event.preventDefault();
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (current + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length;
    tabs[next].focus(); tabs[next].click();
  }));
  document.querySelectorAll('[data-filter]').forEach(input => input.addEventListener('input', () => {
    const term = input.value.trim().toLowerCase();
    document.querySelectorAll('[data-searchable]').forEach(row => { row.hidden = !row.textContent.toLowerCase().includes(term); });
  }));
  document.querySelectorAll('input[name="connectivity"]').forEach(input => input.addEventListener('change', () => {
    const privatePanel = document.querySelector('[data-private]'); if (privatePanel) privatePanel.hidden = input.value !== 'private';
  }));
  const approve = document.querySelector('#approve-scope'); const launch = document.querySelector('#preview-launch');
  if (approve && launch) {
    launch.disabled = !approve.checked;
    approve.addEventListener('change', () => { launch.disabled = !approve.checked; });
    launch.addEventListener('click', () => { if (approve.checked) window.location.href = 'assessment-run.html'; });
  }
  const stopDialog = document.querySelector('#stop-dialog');
  document.querySelectorAll('[data-stop]').forEach(button => button.addEventListener('click', () => stopDialog?.showModal()));
  document.querySelectorAll('[data-close-dialog]').forEach(button => button.addEventListener('click', () => stopDialog?.close()));
  document.querySelectorAll('[data-reset-form]').forEach(button => button.addEventListener('click', () => { document.querySelector('form')?.reset(); toast('Preview fields reset. Nothing was saved.'); }));
  document.querySelectorAll('[data-print]').forEach(button => button.addEventListener('click', () => window.print()));
  if (page === 'finding-detail') {
    const decision = document.querySelector('#review-decision'); const reviewButton = document.querySelector('#preview-review');
    if (decision && reviewButton) {
      decision.addEventListener('change', () => { reviewButton.textContent = decision.value === 'confirm' ? 'Preview reviewed report' : 'Preview saved disposition'; });
      reviewButton.addEventListener('click', () => {
        if (decision.value === 'confirm') window.location.href = 'report.html';
        else toast(`Simulated disposition: ${decision.options[decision.selectedIndex].text}. Evidence remains retained; no rated finding is created.`);
      });
    }
  }
})();
