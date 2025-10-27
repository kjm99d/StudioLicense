const DEFAULT_FIELD_TYPES = [
  { value: 'string', label: '문자열' },
  { value: 'number', label: '숫자' },
  { value: 'boolean', label: '불리언' },
  { value: 'json', label: 'JSON' }
];

const MODE_FORM = 'form';
const MODE_JSON = 'json';

export class PolicyEditor {
  constructor(rootEl, options = {}) {
    this.rootEl = rootEl;
    this.mode = rootEl.dataset.initialMode === MODE_JSON ? MODE_JSON : MODE_FORM;
    this.options = options;

    this.formContainer = rootEl.querySelector('.policy-editor-form');
    this.fieldListEl = rootEl.querySelector('.policy-field-list');
    this.jsonContainer = rootEl.querySelector('.policy-editor-json');
    this.textarea = this.jsonContainer?.querySelector('textarea');
    this.addFieldBtn = this.formContainer?.querySelector('.add-policy-field-btn');
    this.errorEl = document.createElement('div');
    this.errorEl.className = 'policy-editor-error';
    this.errorEl.style.cssText = 'color:#dc2626;margin-top:8px;font-size:12px;';
    this.formContainer?.appendChild(this.errorEl);

    this._bindModeButtons();
    this._bindAddField();
    this._syncVisibility();
    this.resetFields();
  }

  _bindModeButtons() {
    this.modeButtons = Array.from(
      this.rootEl.querySelectorAll('.editor-mode-btn')
    );

    this.modeButtons.forEach((btn) => {
      btn.addEventListener('click', () => {
        const targetMode = btn.dataset.mode === MODE_JSON ? MODE_JSON : MODE_FORM;
        this.setMode(targetMode, { fromToggle: true });
      });
    });
  }

  _bindAddField() {
    if (!this.addFieldBtn) return;
    this.addFieldBtn.addEventListener('click', () => this.addField());
  }

  _syncVisibility() {
    const isForm = this.mode === MODE_FORM;

    if (this.formContainer) {
      this.formContainer.hidden = !isForm;
    }
    if (this.jsonContainer) {
      this.jsonContainer.hidden = isForm;
    }

    this.modeButtons?.forEach((btn) => {
      if (btn.dataset.mode === this.mode) {
        btn.classList.add('active');
        btn.classList.remove('grey', 'lighten-2');
      } else {
        btn.classList.remove('active');
        btn.classList.add('grey', 'lighten-2');
      }
    });

    if (this.textarea) {
      if (isForm) {
        this.textarea.removeAttribute('required');
      } else {
        this.textarea.setAttribute('required', 'required');
      }
    }
  }

  setMode(targetMode, { fromToggle = false } = {}) {
    if (targetMode === this.mode) return;

    if (targetMode === MODE_JSON && fromToggle) {
      try {
        const jsonString = this.toJsonString();
        if (this.textarea) {
          this.textarea.value = jsonString;
        }
      } catch (err) {
        this._showError(err.message);
        return;
      }
    } else if (targetMode === MODE_FORM && fromToggle && this.textarea) {
      try {
        this.loadFromJson(this.textarea.value || '{}');
      } catch (err) {
        this._showError(err.message);
        return;
      }
    }

    this.mode = targetMode;
    this._clearError();
    this._syncVisibility();
  }

  resetFields() {
    if (!this.fieldListEl) return;
    this.fieldListEl.innerHTML = '';
    this.addField();
    if (this.textarea) {
      this.textarea.value = '';
    }
    this._clearError();
  }

  addField(field = { key: '', type: 'string', value: '' }) {
    if (!this.fieldListEl) return;

    const row = document.createElement('div');
    row.className = 'policy-field-row';
    row.style.cssText = 'display:flex;gap:8px;align-items:flex-start;margin-bottom:8px;';

    const keyInput = document.createElement('input');
    keyInput.type = 'text';
    keyInput.className = 'policy-field-key';
    keyInput.placeholder = '키';
    keyInput.value = field.key || '';
    keyInput.required = true;
    keyInput.style.cssText = 'flex:1;';

    const typeSelect = document.createElement('select');
    typeSelect.className = 'policy-field-type';
    typeSelect.style.cssText = 'width:110px;';

    DEFAULT_FIELD_TYPES.forEach((opt) => {
      const optionEl = document.createElement('option');
      optionEl.value = opt.value;
      optionEl.textContent = opt.label;
      typeSelect.appendChild(optionEl);
    });
    typeSelect.value = DEFAULT_FIELD_TYPES.some((t) => t.value === field.type)
      ? field.type
      : 'string';

    const valueInput = document.createElement('textarea');
    valueInput.className = 'policy-field-value';
    valueInput.rows = 1;
    valueInput.placeholder = '값';
    valueInput.style.cssText = 'flex:1.5;font-family:monospace;';
    valueInput.value = field.value !== undefined && field.value !== null ? String(field.value) : '';

    const removeBtn = document.createElement('button');
    removeBtn.type = 'button';
    removeBtn.className = 'btn btn-small red lighten-2';
    removeBtn.textContent = '삭제';

    removeBtn.addEventListener('click', () => {
      if (this.fieldListEl?.children.length <= 1) {
        keyInput.value = '';
        valueInput.value = '';
        typeSelect.value = 'string';
        return;
      }
      row.remove();
    });

    row.appendChild(keyInput);
    row.appendChild(typeSelect);
    row.appendChild(valueInput);
    row.appendChild(removeBtn);

    this.fieldListEl.appendChild(row);
  }

  loadFromJson(jsonString) {
    if (!jsonString || !jsonString.trim()) {
      this.resetFields();
      return;
    }

    let parsed;
    try {
      parsed = JSON.parse(jsonString);
    } catch (err) {
      throw new Error('유효한 JSON 형식이 아닙니다.');
    }

    if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
      throw new Error('폼 입력은 최상위 JSON 객체만 지원합니다.');
    }

    if (!this.fieldListEl) return;
    this.fieldListEl.innerHTML = '';

    Object.entries(parsed).forEach(([key, value]) => {
      const field = this._buildFieldFromValue(key, value);
      this.addField(field);
    });

    if (!this.fieldListEl.children.length) {
      this.addField();
    }

    if (this.textarea && this.mode === MODE_JSON) {
      this.textarea.value = JSON.stringify(parsed, null, 2);
    }

    this._clearError();
  }

  toJsonString() {
    if (!this.fieldListEl) return '{}';

    const rows = Array.from(this.fieldListEl.querySelectorAll('.policy-field-row'));
    const result = {};

    rows.forEach((row) => {
      const key = row.querySelector('.policy-field-key')?.value.trim();
      const type = row.querySelector('.policy-field-type')?.value || 'string';
      const valueEl = row.querySelector('.policy-field-value');
      const originalValue = valueEl ? valueEl.value : '';
      const trimmedValue = originalValue.trim();

      if (!key) {
        throw new Error('모든 필드에 키를 입력해주세요.');
      }
      if (Object.prototype.hasOwnProperty.call(result, key)) {
        throw new Error(`중복된 키 "${key}" 가 존재합니다.`);
      }

      result[key] = this._convertValue(type, trimmedValue, key, originalValue);
    });

    return JSON.stringify(result, null, 2);
  }

  getJsonString() {
    if (this.mode === MODE_JSON) {
      if (!this.textarea) return '{}';
      const value = this.textarea.value.trim();
      if (!value) {
        throw new Error('정책 데이터를 입력해주세요.');
      }
      try {
        const parsed = JSON.parse(value);
        return JSON.stringify(parsed, null, 2);
      } catch (err) {
        throw new Error('유효한 JSON 형식으로 입력해주세요.');
      }
    }
    return this.toJsonString();
  }

  getMode() {
    return this.mode;
  }

  clearError() {
    this._clearError();
  }

  displayError(message) {
    this._showError(message);
  }

  _convertValue(type, rawValue, keyLabel, originalValue = rawValue) {
    switch (type) {
      case 'number': {
        if (rawValue === '') {
          throw new Error(`"${keyLabel}" 값이 비어 있습니다.`);
        }
        const parsed = Number(rawValue);
        if (Number.isNaN(parsed)) {
          throw new Error(`"${keyLabel}" 값이 숫자가 아닙니다.`);
        }
        return parsed;
      }
      case 'boolean': {
        if (rawValue === '') {
          throw new Error(`"${keyLabel}" 값이 비어 있습니다.`);
        }
        if (rawValue.toLowerCase() === 'true') return true;
        if (rawValue.toLowerCase() === 'false') return false;
        throw new Error(`"${keyLabel}" 값은 true 또는 false 여야 합니다.`);
      }
      case 'json': {
        if (!rawValue) {
          throw new Error(`"${keyLabel}" 값이 비어 있습니다.`);
        }
        try {
          return JSON.parse(rawValue);
        } catch (err) {
          throw new Error(`"${keyLabel}" 필드의 JSON 값이 올바르지 않습니다.`);
        }
      }
      case 'string':
      default:
        return originalValue;
    }
  }

  _buildFieldFromValue(key, value) {
    if (typeof value === 'string') {
      return { key, type: 'string', value };
    }
    if (typeof value === 'number') {
      return { key, type: 'number', value: String(value) };
    }
    if (typeof value === 'boolean') {
      return { key, type: 'boolean', value: value ? 'true' : 'false' };
    }
    return {
      key,
      type: 'json',
      value: JSON.stringify(value, null, 2)
    };
  }

  _showError(message) {
    if (this.errorEl) {
      this.errorEl.textContent = message;
    }
  }

  _clearError() {
    if (this.errorEl) {
      this.errorEl.textContent = '';
    }
  }
}

export function initPolicyEditors() {
  const editors = [];
  document.querySelectorAll('.policy-editor').forEach((rootEl) => {
    const editor = new PolicyEditor(rootEl);
    editors.push(editor);
  });
  return editors;
}
