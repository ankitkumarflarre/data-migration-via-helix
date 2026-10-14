// The dropdown values come from the same file the Go server validates against.
import data from '../../../../internal/billing/codes.json' with { type: 'json' };

export const codes = data;
export type Codes = typeof data;
