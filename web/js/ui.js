export function renderStatusBadge(status) {
  console.log('renderStatusBadge called with status:', status);
  const badges = {
    'active': '<span class="status-badge status-active"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>Active</span>',
    'expired': '<span class="status-badge status-expired"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>Expired</span>',
    'revoked': '<span class="status-badge status-inactive"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>Inactive</span>'
  };
  return badges[status] || `<span class="status-badge"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>${status}</span>`;
}
