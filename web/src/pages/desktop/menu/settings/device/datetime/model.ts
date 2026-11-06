const dateTimeLocalPattern = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/;

function pad(value: string | undefined, length = 2) {
  return (value ?? '').padStart(length, '0');
}

export function formatLocalizedDateTime(formatter: Intl.DateTimeFormat, date: Date): string {
  const formatted = formatter.format(date);
  if (!formatter.resolvedOptions().locale.toLowerCase().startsWith('zh')) return formatted;

  return formatted.replace(/(\d+)\s*年\s*(\d+)\s*月\s*(\d+)\s*日/u, '$1 年 $2 月 $3 日');
}

export function formatDateTimeInTimeZone(date: Date, timezone: string): string | null {
  if (!Number.isFinite(date.getTime()) || !timezone) return null;

  try {
    const formatter = new Intl.DateTimeFormat('en-CA', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hourCycle: 'h23'
    });
    const parts = new Map(
      formatter
        .formatToParts(date)
        .filter((part) => part.type !== 'literal')
        .map((part) => [part.type, part.value])
    );

    const year = parts.get('year');
    const month = parts.get('month');
    const day = parts.get('day');
    const hour = parts.get('hour');
    const minute = parts.get('minute');
    const second = parts.get('second');
    if (!year || !month || !day || !hour || !minute || !second) return null;

    return `${pad(year, 4)}-${pad(month)}-${pad(day)}T${pad(hour)}:${pad(minute)}:${pad(second)}`;
  } catch {
    return null;
  }
}

export function normalizeDateTimeLocal(value: string): string | null {
  const match = dateTimeLocalPattern.exec(value);
  if (!match) return null;

  const [, year, month, day, hour, minute, second = '00'] = match;
  const fields = [year, month, day, hour, minute, second].map(Number);
  if (fields.some((field) => !Number.isInteger(field))) return null;

  const [numericYear, numericMonth, numericDay, numericHour, numericMinute, numericSecond] = fields;
  const candidate = new Date(
    Date.UTC(numericYear, numericMonth - 1, numericDay, numericHour, numericMinute, numericSecond)
  );
  if (
    candidate.getUTCFullYear() !== numericYear ||
    candidate.getUTCMonth() !== numericMonth - 1 ||
    candidate.getUTCDate() !== numericDay ||
    candidate.getUTCHours() !== numericHour ||
    candidate.getUTCMinutes() !== numericMinute ||
    candidate.getUTCSeconds() !== numericSecond
  ) {
    return null;
  }

  return `${pad(year, 4)}-${pad(month)}-${pad(day)}T${pad(hour)}:${pad(minute)}:${pad(second)}`;
}
