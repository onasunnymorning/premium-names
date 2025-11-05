"use client";

import React from "react";
import useSWR from "swr";
import { API_BASE, normalizeTag } from "@/lib/api";

type Tag = { id: number; name: string; group_name?: string | null };

export default function TagSelector({ value, onChange, placeholder }: {
  value?: Tag | null;
  onChange: (tag: Tag | null) => void;
  placeholder?: string;
}) {
  const [q, setQ] = React.useState("");
  const { data } = useSWR<Tag[]>(q ? `${API_BASE}/api/tags?prefix=${encodeURIComponent(q)}&limit=20` : null, async (url: string) => {
    const raw = await fetch(url).then(r => r.json());
    return (raw as any[]).map(normalizeTag);
  });

  return (
    <div>
      <input
        type="text"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder={placeholder || "Search tags..."}
        style={{ width: 260 }}
      />
      {q && (
        <div className="card" style={{ position: "absolute", zIndex: 10, width: 280, maxHeight: 240, overflowY: "auto" }}>
          {(data || []).map((t) => (
            <div key={t.id} className="row" style={{ justifyContent: "space-between" }}>
              <button onClick={() => { onChange(t); setQ(""); }} style={{ background: "none", border: 0, padding: 0 }}>
                {t.name} {t.group_name ? <small>({t.group_name})</small> : null}
              </button>
            </div>
          ))}
          {data && data.length === 0 && <div>No tags found</div>}
        </div>
      )}
      {value && (
        <div style={{ marginTop: 6 }}>
          Selected: <strong>{value.name}</strong>{" "}
          <button onClick={() => onChange(null)} style={{ marginLeft: 8 }}>Clear</button>
        </div>
      )}
    </div>
  );
}
