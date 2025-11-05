"use client";

import React from "react";
import { api, API_BASE, getId, normalizeTag } from "@/lib/api";
import MultiTagSelector from "@/components/MultiTagSelector";

type Batch = { id: number; name: string; created_at: string };

type Tag = { id: number; name: string };

export default function AddListPage() {
  const [name, setName] = React.useState(defaultBatchName());
  const [createdBy, setCreatedBy] = React.useState("");
  const [file, setFile] = React.useState<File | null>(null);
  const [tags, setTags] = React.useState<Tag[]>([]);

  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [batch, setBatch] = React.useState<Batch | null>(null);
  const [jobId, setJobId] = React.useState<number | null>(null);
  const [taggingProgress, setTaggingProgress] = React.useState<string>("");
  const [done, setDone] = React.useState(false);

  async function onSubmit() {
    if (!file) { setError("Please choose a file"); return; }
    if (tags.length === 0) { setError("Please select at least one tag"); return; }

    setBusy(true); setError(null); setDone(false); setTaggingProgress("");
    try {
      // 1) Create batch
  const bRaw = await api<any>(`/api/batches`, { method: "POST", body: JSON.stringify({ name, created_by: createdBy || undefined }) });
  const bid = getId(bRaw);
  const b: Batch = { id: bid!, name: bRaw.name ?? bRaw.Name ?? name, created_at: bRaw.created_at ?? bRaw.CreatedAt ?? new Date().toISOString() };
  setBatch(b);

      // 2) Upload file to this batch
      const fd = new FormData();
      fd.append("file", file);
  const res = await fetch(`${API_BASE}/api/batches/${b.id}/upload`, { method: "POST", body: fd });
      if (!res.ok) {
        let extra = ""; try { extra = await res.text(); } catch {}
        throw new Error(`${res.status} ${res.statusText}${extra ? `: ${extra}` : ""}`);
      }
  const job = await res.json();
      setJobId(job.id);

      // 3) Start idempotent apply loop: apply tags to the batch every few seconds until no new additions
  await applyTagsUntilStable(b.id, tags.map(t => (t as any).id ?? (t as any).ID), setTaggingProgress);

      setDone(true);
    } catch (e:any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="container">
      <h1 className="text-2xl font-semibold">Add a List and Apply Tags</h1>
      <p className="text-gray-600 mt-1">Upload a list (CSV/TSV/TXT/XLSX/XLS; first column is the domain). We’ll extract labels and apply the tags you choose.</p>

      <div className="card">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="label">Tags</label>
            <MultiTagSelector value={tags as any} onChange={(t) => setTags(t as any)} />
            <p className="text-xs text-gray-500 mt-1">Select one or more tags. You can also type a new tag name and click Create.</p>
          </div>
          <div>
            <label className="label">Batch name</label>
            <input className="input w-full" value={name} onChange={(e) => setName(e.target.value)} />
            <p className="text-xs text-gray-500 mt-1">Used to group and export this import later.</p>
          </div>
          <div>
            <label className="label">Created by (optional)</label>
            <input className="input w-full" value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} placeholder="you@company.com" />
          </div>
          <div>
            <label className="label">Upload file</label>
            <input className="file:mr-3 file:py-2 file:px-3 file:rounded-md file:border file:border-gray-300 file:bg-white file:text-sm" type="file" onChange={(e) => setFile(e.target.files?.[0] || null)} />
          </div>
        </div>
        <div className="row mt-3">
          <button className="btn-primary" onClick={onSubmit} disabled={busy || !file || tags.length===0}>Upload & Apply</button>
        </div>
      </div>

      {error && <div className="card text-red-700">{error}</div>}

      {batch && (
        <div className="card">
          <h3 className="font-semibold">Upload started</h3>
          <div className="mt-1">Batch <strong>#{batch.id}</strong>: {batch.name}</div>
          {jobId && <div>Job ID: <strong>{jobId}</strong> <span className="text-gray-500">(importing…)</span></div>}
          <div className="mt-2">
            <strong>Applying tags:</strong>
            <pre className="whitespace-pre-wrap mt-1 text-sm bg-gray-50 border border-gray-200 rounded-md p-2">{taggingProgress || "waiting for progress…"}</pre>
          </div>
          {done && <div className="text-green-700 mt-2"><strong>Done.</strong> All selected tags have been applied to this batch.</div>}
        </div>
      )}
    </div>
  );
}

async function applyTagsUntilStable(batchId: number, tagIds: number[], log: (s: string) => void, opts?: { intervalMs?: number; idleCycles?: number; maxCycles?: number }) {
  const intervalMs = opts?.intervalMs ?? 3000;
  const idleCyclesTarget = opts?.idleCycles ?? 3; // consider done after 3 consecutive 0-add cycles
  const maxCycles = opts?.maxCycles ?? 120; // ~6 minutes
  let cycle = 0;
  let idleCycles = 0;
  let totalAdded = 0;
  log("");
  while (cycle < maxCycles) {
    cycle++;
    let cycleAdded = 0;
    for (const tagId of tagIds) {
      try {
        const res = await api<{ added: number }>(`/api/labels/tags/apply`, {
          method: "POST",
          body: JSON.stringify({ tagId, filter: { batch: batchId, mode: "any", limit: 0, offset: 0 } }),
        });
        cycleAdded += res.added || 0;
      } catch (e:any) {
        // if backend rejects temporarily, continue and retry
        log(`cycle ${cycle}: tag ${tagId} error: ${e.message}`);
      }
    }
    totalAdded += cycleAdded;
    if (cycleAdded === 0) idleCycles++; else idleCycles = 0;
    log(`cycle ${cycle}: +${cycleAdded}, total ${totalAdded}${idleCycles ? ` (idle ${idleCycles}/${idleCyclesTarget})` : ""}`);
    if (idleCycles >= idleCyclesTarget) break;
    await new Promise(r => setTimeout(r, intervalMs));
  }
}

function defaultBatchName() {
  const d = new Date();
  const pad = (n:number) => String(n).padStart(2, "0");
  return `import-${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}`;
}
