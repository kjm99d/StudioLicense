const DEFAULT_FIELD_TYPES = [
  { value: 'string', label: 'String' },
  { value: 'number', label: 'Number' },
  { value: 'boolean', label: 'Boolean' }
];

const MODE_FORM = 'form';
const MODE_JSON = 'json';

let fieldIdCounter = 0;
function nextFieldId(prefix) {
  fieldIdCounter += 1;
  return `${prefix}_${fieldIdCounter}`;
}

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

    const keyId = nextFieldId('policy_key');
    const typeId = nextFieldId('policy_type');
    const valueId = nextFieldId('policy_value');

    const keyInput = document.createElement('input');
    keyInput.type = 'text';
    keyInput.id = keyId;
    keyInput.className = 'policy-field-input policy-field-key';
    keyInput.placeholder = '예: feature';
    keyInput.value = field.key || '';
    keyInput.required = true;

    const keyLabel = document.createElement('label');
    keyLabel.className = 'policy-field-label';
    keyLabel.setAttribute('for', keyId);
    keyLabel.textContent = '키';

    const keyWrapper = document.createElement('div');
    keyWrapper.className = 'policy-field policy-field-key-wrapper';
    keyWrapper.append(keyLabel, keyInput);

    const typeSelect = document.createElement('select');
    typeSelect.id = typeId;
    typeSelect.className = 'policy-field-input policy-field-type browser-default';
    typeSelect.setAttribute('aria-label', '값 타입 선택');
    typeSelect.title = '값 타입 선택';

    DEFAULT_FIELD_TYPES.forEach((opt) => {
      const optionEl = document.createElement('option');
      optionEl.value = opt.value;
      optionEl.textContent = opt.label;
      typeSelect.appendChild(optionEl);
    });
    typeSelect.value = DEFAULT_FIELD_TYPES.some((t) => t.value === field.type)
      ? field.type
      : 'string';

    const typeLabel = document.createElement('label');
    typeLabel.className = 'policy-field-label';
    typeLabel.setAttribute('for', typeId);
    typeLabel.textContent = '타입';

    const typeWrapper = document.createElement('div');
    typeWrapper.className = 'policy-field policy-field-type-wrapper';
    typeWrapper.append(typeLabel, typeSelect);

    const valueInput = document.createElement('textarea');
    valueInput.id = valueId;
    valueInput.className = 'policy-field-input policy-field-value';
    valueInput.rows = 2;
    valueInput.value = field.value !== undefined && field.value !== null ? String(field.value) : '';

    const valueLabel = document.createElement('label');
    valueLabel.className = 'policy-field-label';
    valueLabel.setAttribute('for', valueId);
    valueLabel.textContent = '값';

    const valueWrapper = document.createElement('div');
    valueWrapper.className = 'policy-field policy-field-value-wrapper';
    valueWrapper.append(valueLabel, valueInput);

    const removeBtn = document.createElement('button');
    removeBtn.type = 'button';
    removeBtn.className = 'btn btn-small red lighten-2 policy-field-remove';
    removeBtn.textContent = '삭제';
    removeBtn.title = '이 필드를 제거합니다';

    removeBtn.addEventListener('click', () => {
      if (this.fieldListEl?.children.length <= 1) {
        keyInput.value = '';
        valueInput.value = '';
        typeSelect.value = 'string';
        updateValuePlaceholder();
        return;
      }
      row.remove();
    });

    const actionsWrapper = document.createElement('div');
    actionsWrapper.className = 'policy-field-actions';
    actionsWrapper.appendChild(removeBtn);

    const updateValuePlaceholder = () => {
      switch (typeSelect.value) {
        case 'number':
          valueInput.placeholder = '숫자 (예: 10)';
          valueInput.rows = 1;
          break;
        case 'boolean':
          valueInput.placeholder = 'true 또는 false';
          valueInput.rows = 1;
          break;
        default:
          valueInput.placeholder = '텍스트 값 (예: enabled)';
          valueInput.rows = 2;
      }
    };

    typeSelect.addEventListener('change', () => {
      updateValuePlaceholder();
      if (typeSelect.value === 'boolean') {
        const lower = valueInput.value.trim().toLowerCase();
        if (lower !== 'true' && lower !== 'false') {
          valueInput.value = '';
        }
      }
    });

    updateValuePlaceholder();

    row.append(keyWrapper, typeWrapper, valueWrapper, actionsWrapper);
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

    try {
      Object.entries(parsed).forEach(([key, value]) => {
        const field = this._buildFieldFromValue(key, value);
        this.addField(field);
      });
    } catch (err) {
      this.fieldListEl.innerHTML = '';
      if (err instanceof Error) {
        throw err;
      }
      throw new Error('폼 모드는 문자열/숫자/불리언 값만 지원합니다.');
    }

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
      case 'string':
      default:
        return originalValue;
    }
  }

  _buildFieldFromValue(key, value) {
    if (value === null || typeof value === 'object') {
      throw new Error('폼 모드는 문자열/숫자/불리언 값만 지원합니다.');
    }
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
