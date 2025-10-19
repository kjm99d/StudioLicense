import { state } from './state.js';

let globalEscHandlerBound = false;

export function setupModalBehaviors() {
  // Attach overlay click handlers to all modals (once)
  document.querySelectorAll('.modal').forEach(modal => {
    if (!modal._overlayClickBound) {
      modal.addEventListener('click', (e) => { 
        if (e.target === modal) closeModal(modal); 
      });
      modal._overlayClickBound = true;
    }
  });
  
  // Global ESC handler (bind only once)
  if (!globalEscHandlerBound) {
    document.addEventListener('keydown', handleGlobalEscape);
    globalEscHandlerBound = true;
  }
}

function handleGlobalEscape(e) {
  if (e.key === 'Escape') {
    const opened = document.querySelector('.modal.active');
    if (opened) {
      e.preventDefault();
      e.stopPropagation();
      closeModal(opened);
    }
  }
}

export function openModal(modal) {
  if (!modal) return;
  
  // Dynamic z-index stacking
  state.topZIndex += 2;
  modal.style.zIndex = String(state.topZIndex);
  const contentEl = modal.querySelector('.modal-content');
  if (contentEl) contentEl.style.zIndex = String(state.topZIndex + 1);

  modal.classList.add('active');
  document.body.classList.add('modal-open');
  
  // Focus trap setup
  const focusable = modal.querySelectorAll('button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"]):not([disabled])');
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  
  if (first) {
    setTimeout(() => first.focus(), 100); // Defer focus to avoid race conditions
  }
  
  // Remove old trap if exists
  if (modal._trap) {
    modal.removeEventListener('keydown', modal._trap);
  }
  
  modal._trap = (e) => {
    if (e.key === 'Tab') {
      if (focusable.length === 1) {
        e.preventDefault();
        return;
      }
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }
  };
  modal.addEventListener('keydown', modal._trap);
}

export function closeModal(modal) {
  if (!modal) return;
  
  modal.classList.remove('active');
  
  // Clean up focus trap
  if (modal._trap) {
    modal.removeEventListener('keydown', modal._trap);
    delete modal._trap;
  }
  
  // Unlock body scroll after animation completes (if no other modals are open)
  setTimeout(() => {
    const anyModalOpen = document.querySelector('.modal.active');
    if (!anyModalOpen) {
      document.body.classList.remove('modal-open');
    }
  }, 200);
  
  // Return focus to trigger element if available
  if (modal._triggerElement && modal._triggerElement.focus) {
    modal._triggerElement.focus();
    delete modal._triggerElement;
  }
}

export function openDialog({ title = '알림', message = '', showCancel = false }) {
  const modal = document.getElementById('dialog-modal');
  if (!modal) return Promise.resolve(false);
  
  const titleEl = document.getElementById('dialog-title');
  const msgEl = document.getElementById('dialog-message');
  const btnCancel = document.getElementById('dialog-cancel');
  const btnOk = document.getElementById('dialog-ok');

  if (titleEl) titleEl.textContent = title;
  if (msgEl) msgEl.textContent = message;
  if (btnCancel) {
    btnCancel.style.display = showCancel ? 'inline-flex' : 'none';
    btnCancel.style.visibility = showCancel ? 'visible' : 'hidden';
  }

  // Ensure dialog is always on top
  state.topZIndex = Math.max(state.topZIndex, 15000) + 4;
  modal.style.zIndex = String(state.topZIndex);
  const contentEl = modal.querySelector('.modal-content');
  if (contentEl) contentEl.style.zIndex = String(state.topZIndex + 1);
  
  openModal(modal);

  return new Promise((resolve) => {
    const close = (result) => {
      closeModal(modal);
      if (btnOk) btnOk.onclick = null;
      if (btnCancel) btnCancel.onclick = null;
      const closeBtn = modal.querySelector('.modal-close');
      if (closeBtn) closeBtn.onclick = null;
      resolve(result);
    };
    
    if (btnOk) btnOk.onclick = () => close(true);
    if (btnCancel) btnCancel.onclick = () => close(false);
    
    const closeBtn = modal.querySelector('.modal-close');
    if (closeBtn) closeBtn.onclick = () => close(false);
  });
}

export async function showAlert(message, title = '알림') {
  await openDialog({ title, message, showCancel: false });
}

export async function showConfirm(message, title = '확인') {
  return await openDialog({ title, message, showCancel: true });
}
