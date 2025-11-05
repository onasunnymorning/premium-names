"use client";

import React from "react";
import useSWR from "swr";
import { API_BASE, normalizeTag } from "@/lib/api";

type Tag = { id: number; name: string; group_name?: string | null };

export default function MultiTagSelector({ value, onChange, placeholder }: {
  value: Tag[];
  onChange: (tags: Tag[]) => void;
  placeholder?: string;
}) {
  const [q, setQ] = React.useState("");
  const [err, setErr] = React.useState<string | null>(null);
  const fetcher = React.useCallback(async (url: string) => {
    const raw = await fetch(url).then(r => r.json());
    return (raw as any[]).map(normalizeTag);
  }, []);
  const { data } = useSWR<Tag[]>(q ? `${API_BASE}/api/tags?prefix=${encodeURIComponent(q)}&limit=20` : null, fetcher);

  const createTag = React.useCallback(async (name: string) => {
    const cleaned = name.trim();
    if (!cleaned) { setErr("Tag name cannot be empty"); return; }
    setErr(null);
    // Already selected?
    const exists = value.some(t => t.name.toLowerCase() === cleaned.toLowerCase());
    if (exists) { setQ(""); return; }
    const res = await fetch(`${API_BASE}/api/tags`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: cleaned }),
    });
    if (!res.ok) {
      // Try to resolve conflicts by selecting the existing server tag
      let body = ""; try { body = await res.text(); } catch {}
      if (res.status === 400 && /conflict|unique|exists/i.test(body)) {
        try {
          const raw = await fetch(`${API_BASE}/api/tags?prefix=${encodeURIComponent(cleaned)}&limit=1`).then(r => r.json());
          const found: Tag | undefined = (raw || []).map(normalizeTag).find((t: Tag) => t.name.toLowerCase() === cleaned.toLowerCase());
          if (found) {
            React.startTransition(() => {
              onChange([...value, found]);
              setQ("");
            });
            return;
          }
        } catch {}
      }
      setErr(body || `${res.status} ${res.statusText}`);
      return;
    }
    const t = normalizeTag(await res.json());
    React.startTransition(() => {
      onChange([...value, t]);
      setQ("");
    });
  }, [value, onChange]);

  const selectedIds = new Set(value.map(t => t.id));

  return (
    <div>
      <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
        {value.map(t => (
          <span key={t.id} className="card" style={{ padding: "4px 8px" }}>
            {t.name}
            <button onClick={() => {
              React.startTransition(() => {
                onChange(value.filter(x => x.id !== t.id));
              });
            }} style={{ marginLeft: 6 }}>×</button>
          </span>
        ))}
      </div>
      <div style={{ position: "relative", marginTop: 6 }}>
        <input
          type="text"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={placeholder || "Search or create tag"}
          style={{ width: 320 }}
        />
        {q && (
          <div className="card" style={{ position: "absolute", zIndex: 10, width: 340, maxHeight: 260, overflowY: "auto" }}>
            {(data || []).filter(t => !selectedIds.has(t.id)).map((t) => (
              <div key={t.id} className="row" style={{ justifyContent: "space-between" }}>
                <button onClick={() => {
                  React.startTransition(() => {
                    onChange([...value, t]);
                    setQ("");
                  });
                }} style={{ background: "none", border: 0, padding: 0 }}>
                  {t.name} {t.group_name ? <small>({t.group_name})</small> : null}
                </button>
              </div>
            ))}
            <div className="row" style={{ marginTop: 6 }}>
              <button onClick={() => createTag(q)}>Create &quot;{q}&quot;</button>
            </div>
          </div>
        )}
        {err && <div style={{ color: "crimson", marginTop: 6 }}>{err}</div>}
      </div>
    </div>
  );
}
