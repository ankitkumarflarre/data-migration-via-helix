// Workspace state shared by the billing screens: the system date record and
// a one-shot message handed from one screen to the next (Duck Creek shows the
// result of an action as a banner on the page it returns to).
import { billing, type SystemView } from './api';

export const ws = $state<{ system: SystemView | null; flash: { kind: 'info' | 'warn'; text: string; link?: { label: string; href: string; after?: string } } | null; tick: number }>({ system: null, flash: null, tick: 0 });

export async function loadSystem() { ws.system = await billing.system(); return ws.system; }

export function flash(text: string, kind: 'info' | 'warn' = 'info', link?: { label: string; href: string; after?: string }) { ws.flash = { kind, text, link }; }

// flashLinked shows a message with one word in it (an account number) as a link.
export function flashLinked(text: string, label: string, href: string) {
  const at = text.lastIndexOf(label);
  if (at < 0) { flash(text); return; }
  flash(text.slice(0, at), 'info', { label, href, after: text.slice(at + label.length) });
}

export function takeFlash() { const f = ws.flash; ws.flash = null; return f; }

// changed tells open screens to reload after the system date or settings moved.
export function changed() { ws.tick++; }

export const today = () => ws.system?.date ?? new Date().toISOString().slice(0, 10);
