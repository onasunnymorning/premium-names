export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8081";

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
    // Next.js fetch caching hint: default no-store for dynamic API
    cache: "no-store",
  });
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const j = await res.json();
      if (j?.error) msg += `: ${j.error}`;
    } catch {}
    throw new Error(msg);
  }
  // Try JSON, fallback to text
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) return (await res.json()) as T;
  // @ts-ignore
  return (await res.text()) as T;
}

export function toQuery(params: Record<string, string | number | undefined | null>) {
  const usp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === "") continue;
    usp.set(k, String(v));
  }
  const s = usp.toString();
  return s ? `?${s}` : "";
}

// --- Normalizers for backend structs that use PascalCase JSON field names ---
export type NTag = { id: number; name: string; group_name?: string | null };

export function normalizeTag(t: any): NTag {
  if (!t) return { id: 0, name: "" };
  return {
    id: t.id ?? t.ID ?? 0,
    name: t.name ?? t.Name ?? "",
    group_name: t.group_name ?? t.GroupName ?? null,
  };
}

export function getId(x: any): number | undefined {
  return x?.id ?? x?.ID ?? undefined;
}

// Labels returned by the API currently use PascalCase keys. Normalize them to
// idiomatic JS snake/camel so the UI can rely on consistent shapes.
export type NLabel = {
  id: number;
  label_ascii: string;
  label_unicode: string;
  created_at?: string;
  tags?: NTag[];
};

export function normalizeLabel(l: any): NLabel {
  if (!l) return { id: 0, label_ascii: "", label_unicode: "" };
  return {
    id: l.id ?? l.ID ?? 0,
    label_ascii: l.label_ascii ?? l.LabelASCII ?? l.ascii ?? "",
    label_unicode: l.label_unicode ?? l.LabelUnicode ?? l.unicode ?? "",
    created_at: l.created_at ?? l.CreatedAt ?? undefined,
    tags: Array.isArray(l.tags)
      ? (l.tags as any[]).map(normalizeTag)
      : Array.isArray(l.Tags)
      ? (l.Tags as any[]).map(normalizeTag)
      : [],
  };
}
