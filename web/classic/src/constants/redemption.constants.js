/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export const REDEMPTION_STATUS = {
  UNUSED: 1, // Unused
  DISABLED: 2, // Disabled
  USED: 3, // Used
};

// Redemption code status display mapping
export const REDEMPTION_STATUS_MAP = {
  [REDEMPTION_STATUS.UNUSED]: {
    color: 'green',
    text: '未使用',
  },
  [REDEMPTION_STATUS.DISABLED]: {
    color: 'red',
    text: '已禁用',
  },
  [REDEMPTION_STATUS.USED]: {
    color: 'grey',
    text: '已使用',
  },
};

// Action type constants
export const REDEMPTION_ACTIONS = {
  DELETE: 'delete',
  ENABLE: 'enable',
  DISABLE: 'disable',
};

// Redemption code currency — admin chooses which currency the credit amount
// is denominated in. Backend converts to internal quota using FX rates.
export const REDEMPTION_CURRENCIES = ['usd', 'cny', 'cad'];

export const REDEMPTION_CURRENCY_SYMBOLS = {
  usd: '$',
  cny: '¥',
  cad: 'C$',
};

export const REDEMPTION_CURRENCY_LABELS = {
  usd: 'USD',
  cny: 'CNY',
  cad: 'CAD',
};

export const REDEMPTION_CURRENCY_OPTION_LIST = REDEMPTION_CURRENCIES.map(
  (c) => ({
    value: c,
    label: `${REDEMPTION_CURRENCY_SYMBOLS[c]} ${REDEMPTION_CURRENCY_LABELS[c]}`,
  }),
);

export const isRedemptionCurrency = (v) =>
  REDEMPTION_CURRENCIES.includes((v || '').toLowerCase());

export const normalizeRedemptionCurrency = (v) => {
  const lower = (v || '').toLowerCase();
  return isRedemptionCurrency(lower) ? lower : 'usd';
};

export const REDEMPTION_VALIDATION = {
  AMOUNT_MIN: 0.000001,
  MAX_REDEEMS_MIN: 1,
  MAX_REDEEMS_MAX: 100000,
};

// Format the price for table display, with legacy quota fallback.
export const formatRedemptionPrice = (record) => {
  if (!record) return '';
  const currency = normalizeRedemptionCurrency(record.currency);
  const symbol = REDEMPTION_CURRENCY_SYMBOLS[currency] || '$';
  const label = REDEMPTION_CURRENCY_LABELS[currency] || 'USD';
  let amount = Number(record.amount || 0);
  if (!amount && record.quota) {
    // Legacy rows: derive USD amount from quota (500000 quota = $1).
    amount = Number(record.quota) / 500000;
  }
  return `${symbol}${amount.toFixed(2)} ${label}`;
};
