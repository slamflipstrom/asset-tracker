const moneyFormatter = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  maximumFractionDigits: 2
});

const quantityFormatter = new Intl.NumberFormat('en-US', {
  maximumFractionDigits: 8
});

export function formatMoney(value: number | null): string {
  if (value === null) {
    return '--';
  }

  return moneyFormatter.format(value);
}

export function formatQuantity(value: number): string {
  return quantityFormatter.format(value);
}

export function toDateInputValue(isoTimestamp: string): string {
  const date = new Date(isoTimestamp);
  if (Number.isNaN(date.getTime())) {
    return new Date().toISOString().slice(0, 10);
  }

  return date.toISOString().slice(0, 10);
}
