export default function Page() {
  return (
    <div className="container">
      <h1>Premium Names</h1>
      <p>
        This UI lets you browse labels, manage tags, and import batches. Use the
        navigation above to get started.
      </p>
      <div className="grid">
        <div className="card">
          <h3>Labels</h3>
          <p>Search labels and filter by tags or batch, export CSV, and bulk-apply a tag.</p>
        </div>
        <div className="card">
          <h3>Tags</h3>
          <p>Create, rename, delete tags; type-ahead search.</p>
        </div>
        <div className="card">
          <h3>Batches</h3>
          <p>Create a batch and upload a file (CSV/TSV/TXT/XLSX/XLS, first column is the label).</p>
        </div>
      </div>
    </div>
  );
}
