import { useMemo, useState } from "react";
import {
  APIIssue,
  APIResponse,
  EDIStandard,
  TranslateMode,
  postJSON
} from "./api/client";

const sampleX12 =
  "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~GS*PO*SENDER*RECEIVER*20260427*1200*1*X*004010~ST*850*0001~BEG*00*SA*PO-10001**20260427~N1*BY*Example Buyer*92*BUYER01~N1*ST*Example Ship To*92*SHIPTO01~N3*100 Warehouse Road~N4*Greenville*SC*29601*US~PO1*1*10*EA*3.50**VN*SKU-100~CTT*1~SE*9*0001~GE*1*1~IEA*1*000000001~";

const sampleEdifact =
  "UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1'UNH+1+ORDERS:D:96A:UN'BGM+220+PO-10001+9'DTM+137:20260427:102'NAD+BY+BUYER01::92++Example Buyer'NAD+DP+SHIPTO01::92++Example Ship To'LIN+1++SKU-100:VN'QTY+21:10:EA'UNS+S'UNT+9+1'UNZ+1+1'";

function defaultApiBase(): string {
  if (window.location.protocol === "http:" || window.location.protocol === "https:") {
    return window.location.origin;
  }
  return "http://127.0.0.1:8765";
}

function renderIssue(issue: APIIssue): string {
  const code = issue.code ? `${issue.code}: ` : "";
  return `${code}${issue.message || issue.hint || JSON.stringify(issue)}`;
}

function IssueList({ issues, kind }: { issues?: APIIssue[]; kind: "error" | "warning" }) {
  if (!issues || issues.length === 0) {
    return <li>None</li>;
  }

  return (
    <>
      {issues.map((issue, index) => (
        <li className={kind} key={`${kind}-${index}`}>
          {renderIssue(issue)}
        </li>
      ))}
    </>
  );
}

type UnknownRecord = Record<string, unknown>;

interface SegmentElement {
  index?: number;
  value?: string;
  components?: string[];
}

interface Segment {
  tag: string;
  position?: number;
  elements: SegmentElement[];
  raw?: string;
  offset?: number;
}

interface SegmentEntry {
  segment: Segment;
  context: string;
  index: number;
}

interface PathPart {
  kind: string;
  index?: number;
  type?: string;
}

function asRecord(value: unknown): UnknownRecord | undefined {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as UnknownRecord)
    : undefined;
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function display(value: unknown, fallback = "None"): string {
  return value === undefined || value === null || value === "" ? fallback : String(value);
}

function isSegment(value: unknown): value is Segment {
  const record = asRecord(value);
  return Boolean(record && typeof record.tag === "string" && Array.isArray(record.elements));
}

function segmentContext(path: PathPart[]): string {
  return path
    .map((part) => {
      if (part.kind === "interchange") return `Interchange ${part.index}`;
      if (part.kind === "group") return `Group ${part.index}`;
      if (part.kind === "transaction") return `TX ${display(part.type, String(part.index))}`;
      if (part.kind === "message") return `MSG ${display(part.type, String(part.index))}`;
      if (part.kind === "rawEnvelope") return "Envelope";
      return "";
    })
    .filter(Boolean)
    .join(" / ");
}

function collectSegments(value: unknown, path: PathPart[] = [], out: SegmentEntry[] = [], seen = new Set<object>()) {
  if (value === null || typeof value !== "object" || seen.has(value)) {
    return out;
  }
  seen.add(value);

  if (isSegment(value)) {
    out.push({ segment: value, context: segmentContext(path), index: out.length });
    return out;
  }

  if (Array.isArray(value)) {
    value.forEach((item) => collectSegments(item, path, out, seen));
    return out;
  }

  Object.entries(value as UnknownRecord).forEach(([key, next]) => {
    if (key === "elements") {
      return;
    }
    if (key === "interchanges") {
      asArray(next).forEach((item, index) =>
        collectSegments(item, [...path, { kind: "interchange", index: index + 1 }], out, seen)
      );
      return;
    }
    if (key === "groups") {
      asArray(next).forEach((item, index) =>
        collectSegments(item, [...path, { kind: "group", index: index + 1 }], out, seen)
      );
      return;
    }
    if (key === "transactions") {
      asArray(next).forEach((item, index) =>
        collectSegments(item, [...path, { kind: "transaction", index: index + 1, type: display(asRecord(item)?.type, "") }], out, seen)
      );
      return;
    }
    if (key === "messages") {
      asArray(next).forEach((item, index) =>
        collectSegments(item, [...path, { kind: "message", index: index + 1, type: display(asRecord(item)?.type, "") }], out, seen)
      );
      return;
    }
    if (key === "rawEnvelope") {
      collectSegments(next, [...path, { kind: "rawEnvelope" }], out, seen);
      return;
    }
    collectSegments(next, path, out, seen);
  });
  return out;
}

function firstInterchange(result: UnknownRecord): UnknownRecord {
  return (asRecord(asArray(result.interchanges)[0]) || {});
}

function allTransactions(result: UnknownRecord): UnknownRecord[] {
  return asArray(result.interchanges).flatMap((interchange) =>
    asArray(asRecord(interchange)?.groups).flatMap((group) =>
      asArray(asRecord(group)?.transactions).map((transaction) => asRecord(transaction) || {})
    )
  );
}

function allMessages(result: UnknownRecord): UnknownRecord[] {
  return asArray(result.interchanges).flatMap((interchange) =>
    asArray(asRecord(interchange)?.messages).map((message) => asRecord(message) || {})
  );
}

function topTags(segments: SegmentEntry[]): string {
  const counts = segments.reduce<Record<string, number>>((next, entry) => {
    next[entry.segment.tag] = (next[entry.segment.tag] || 0) + 1;
    return next;
  }, {});
  return Object.keys(counts)
    .sort((a, b) => counts[b] - counts[a] || a.localeCompare(b))
    .slice(0, 6)
    .map((tag) => `${tag} ${counts[tag]}`)
    .join(", ");
}

function SummaryList({ rows }: { rows: [string, unknown][] }) {
  return (
    <dl>
      {rows.map(([label, value]) => (
        <div className="summary-row" key={label}>
          <dt>{label}</dt>
          <dd>{display(value)}</dd>
        </div>
      ))}
    </dl>
  );
}

export function App() {
  const [input, setInput] = useState("");
  const [standard, setStandard] = useState<EDIStandard>("auto");
  const [mode, setMode] = useState<TranslateMode>("structural");
  const [schemaId, setSchemaId] = useState("x12-850-basic");
  const [apiBase, setApiBase] = useState(defaultApiBase);
  const [status, setStatus] = useState("Idle");
  const [statusKind, setStatusKind] = useState("");
  const [responseLabel, setResponseLabel] = useState("No response");
  const [response, setResponse] = useState<APIResponse>({});
  const [segmentFilter, setSegmentFilter] = useState("");
  const [selectedSegmentIndex, setSelectedSegmentIndex] = useState(-1);

  const inputStats = useMemo(() => {
    const segments = input ? input.split(/[~']/).filter(Boolean).length : 0;
    return `${input.length} chars, ${segments} segments`;
  }, [input]);

  const formattedResponse = useMemo(() => JSON.stringify(response, null, 2), [response]);
  const formattedMetadata = useMemo(
    () => JSON.stringify(response.metadata || {}, null, 2),
    [response.metadata]
  );
  const workbench = useMemo(() => {
    const result = asRecord(response.result) || {};
    const metadata = response.metadata || asRecord(result.metadata) || {};
    const interchange = firstInterchange(result);
    const transactions = allTransactions(result);
    const messages = allMessages(result);
    const segments = collectSegments(result);
    const types = [...transactions, ...messages].map((item) => item.type).filter(Boolean).join(", ");

    return {
      result,
      metadata,
      segments,
      envelopeRows: [
        ["Standard", result.standard || response.standard || metadata.standard],
        ["Version", result.version || interchange.version],
        ["Sender", interchange.senderId],
        ["Receiver", interchange.receiverId],
        ["Control", interchange.controlNumber]
      ] as [string, unknown][],
      transactionRows: [
        ["Transactions", metadata.transactions || transactions.length],
        ["Messages", metadata.messages || messages.length],
        ["Groups", metadata.groups || asArray(interchange.groups).length],
        ["Types", types],
        ["Mode", metadata.mode]
      ] as [string, unknown][],
      segmentRows: [
        ["Total", metadata.segments || segments.length],
        ["Listed", segments.length],
        ["Top tags", topTags(segments)],
        ["Warnings", response.warnings?.length || 0],
        ["Errors", response.errors?.length || 0]
      ] as [string, unknown][]
    };
  }, [response]);
  const filteredSegments = useMemo(() => {
    const filter = segmentFilter.trim().toUpperCase();
    return workbench.segments.filter((entry) => !filter || entry.segment.tag.toUpperCase().startsWith(filter));
  }, [segmentFilter, workbench.segments]);
  const selectedSegment = useMemo(
    () => filteredSegments.find((entry) => entry.index === selectedSegmentIndex) || filteredSegments[0],
    [filteredSegments, selectedSegmentIndex]
  );

  function setDone(next: APIResponse, label: string) {
    setResponse(next);
    setResponseLabel(label);
    setStatus(next.ok === false ? "Errors" : "Ready");
    setStatusKind(next.ok === false ? "error" : "ok");
    setSelectedSegmentIndex(-1);
  }

  async function submit(path: string, body: unknown, label: string) {
    if (!input.trim()) {
      setDone(
        {
          ok: false,
          errors: [
            {
              severity: "error",
              code: "EMPTY_INPUT",
              message: "EDI input is empty."
            }
          ],
          warnings: [],
          metadata: {}
        },
        "Local validation"
      );
      return;
    }

    setStatus("Working");
    setStatusKind("");
    try {
      const result = await postJSON(apiBase, path, body);
      setDone(result, label);
    } catch (error) {
      setDone(
        {
          ok: false,
          errors: [
            {
              severity: "error",
              code: "REQUEST_FAILED",
              message: error instanceof Error ? error.message : "Request failed"
            }
          ],
          warnings: [],
          metadata: {
            endpoint: `${apiBase}${path}`
          }
        },
        `${label} failed`
      );
    }
  }

  async function readFile(file: File) {
    setInput(await file.text());
    setStatus("File loaded");
    setStatusKind("ok");
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <div>
          <h1>EDIForge</h1>
          <p>Local EDI to JSON workbench</p>
        </div>
        <div className={`server-state ${statusKind}`}>{status}</div>
      </header>

      <main className="workspace">
        <section className="pane">
          <div className="pane-header">
            <div>
              <h2>EDI Input</h2>
              <span>{inputStats}</span>
            </div>
            <label className="file-button">
              <input
                type="file"
                accept=".edi,.txt,.x12,.edifact"
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  if (file) {
                    void readFile(file);
                  }
                }}
              />
              Load file
            </label>
          </div>

          <textarea
            spellCheck={false}
            placeholder="Paste EDI here"
            value={input}
            onChange={(event) => setInput(event.target.value)}
          />

          <div className="control-grid">
            <label>
              Standard
              <select
                value={standard}
                onChange={(event) => setStandard(event.target.value as EDIStandard)}
              >
                <option value="auto">Auto</option>
                <option value="x12">X12</option>
                <option value="edifact">EDIFACT</option>
              </select>
            </label>
            <label>
              Mode
              <select value={mode} onChange={(event) => setMode(event.target.value as TranslateMode)}>
                <option value="structural">Structural</option>
                <option value="annotated">Annotated</option>
                <option value="semantic">Semantic</option>
              </select>
            </label>
            <label>
              Schema ID
              <input value={schemaId} onChange={(event) => setSchemaId(event.target.value)} />
            </label>
            <label>
              API Base
              <input value={apiBase} onChange={(event) => setApiBase(event.target.value)} />
            </label>
          </div>

          <div className="button-row">
            <button
              type="button"
              onClick={() => submit("/api/v1/detect", { input, standard }, "Detect")}
            >
              Detect
            </button>
            <button
              type="button"
              className="primary"
              onClick={() =>
                submit(
                  "/api/v1/translate",
                  {
                    input,
                    standard,
                    mode,
                    schemaId: schemaId.trim() || undefined,
                    options: {
                      pretty: true,
                      includeEnvelope: true,
                      includeRawSegments: false
                    }
                  },
                  "Translate"
                )
              }
            >
              Translate
            </button>
            <button
              type="button"
              onClick={() =>
                submit(
                  "/api/v1/validate",
                  {
                    input,
                    standard,
                    level: schemaId.trim() ? "schema" : "syntax",
                    schemaId: schemaId.trim() || undefined
                  },
                  "Validate"
                )
              }
            >
              Validate
            </button>
          </div>

          <div className="button-row samples">
            <button
              type="button"
              onClick={() => {
                setInput(sampleX12);
                setStandard("x12");
                setSchemaId("x12-850-basic");
              }}
            >
              X12 sample
            </button>
            <button
              type="button"
              onClick={() => {
                setInput(sampleEdifact);
                setStandard("edifact");
                setSchemaId("edifact-orders-basic");
              }}
            >
              EDIFACT sample
            </button>
            <button
              type="button"
              onClick={() => {
                setInput("");
                setResponse({});
                setResponseLabel("No response");
                setStatus("Idle");
                setStatusKind("");
              }}
            >
              Clear
            </button>
          </div>
        </section>

        <section className="pane">
          <div className="pane-header">
            <div>
              <h2>JSON Output</h2>
              <span>{responseLabel}</span>
            </div>
            <div className="button-row compact">
              <button type="button" onClick={() => void navigator.clipboard.writeText(formattedResponse)}>
                Copy
              </button>
              <button
                type="button"
                onClick={() => {
                  const blob = new Blob([formattedResponse], { type: "application/json" });
                  const url = URL.createObjectURL(blob);
                  const link = document.createElement("a");
                  link.href = url;
                  link.download = "ediforge-result.json";
                  document.body.appendChild(link);
                  link.click();
                  link.remove();
                  URL.revokeObjectURL(url);
                }}
              >
                Download
              </button>
            </div>
          </div>
          <pre>{formattedResponse}</pre>
        </section>
      </main>

      <section className="workbench">
        <div className="summary-grid">
          <article className="summary-card">
            <h2>Envelope</h2>
            <SummaryList rows={workbench.envelopeRows} />
          </article>
          <article className="summary-card">
            <h2>Transactions</h2>
            <SummaryList rows={workbench.transactionRows} />
          </article>
          <article className="summary-card">
            <h2>Segments</h2>
            <SummaryList rows={workbench.segmentRows} />
          </article>
        </div>

        <div className="segment-workbench">
          <div className="segment-panel">
            <div className="segment-toolbar">
              <div>
                <h2>Segment List</h2>
                <span>
                  {filteredSegments.length} of {workbench.segments.length} segments
                </span>
              </div>
              <label>
                Filter tag
                <input
                  type="search"
                  placeholder="BEG"
                  value={segmentFilter}
                  onChange={(event) => setSegmentFilter(event.target.value)}
                />
              </label>
            </div>
            <ol className="segment-list">
              {filteredSegments.length === 0 ? (
                <li className="empty-segment">
                  {workbench.segments.length ? "No matching segments." : "No segment data in this response."}
                </li>
              ) : (
                filteredSegments.map((entry) => (
                  <li key={`${entry.segment.tag}-${entry.index}`}>
                    <button
                      type="button"
                      className={`segment-item ${entry.index === (selectedSegment?.index ?? -1) ? "selected" : ""}`}
                      onClick={() => setSelectedSegmentIndex(entry.index)}
                    >
                      <strong>{entry.segment.tag}</strong>
                      <span>
                        #{display(entry.segment.position, String(entry.index + 1))}
                        {entry.context ? ` - ${entry.context}` : ""}
                      </span>
                      <small>
                        {entry.segment.elements
                          .map((element) =>
                            element.components?.length ? element.components.join(":") : element.value || ""
                          )
                          .filter(Boolean)
                          .slice(0, 5)
                          .join(" | ")}
                      </small>
                    </button>
                  </li>
                ))
              )}
            </ol>
          </div>
          <div className="segment-detail">
            <h2>Segment Detail</h2>
            <pre>
              {selectedSegment
                ? JSON.stringify(
                    {
                      context: selectedSegment.context || "Document",
                      tag: selectedSegment.segment.tag,
                      position: selectedSegment.segment.position,
                      elements: selectedSegment.segment.elements,
                      raw: selectedSegment.segment.raw,
                      offset: selectedSegment.segment.offset
                    },
                    null,
                    2
                  )
                : "Select a segment to inspect details."}
            </pre>
          </div>
        </div>
      </section>

      <section className="result-strip">
        <div className="result-column">
          <h2>Errors</h2>
          <ul>
            <IssueList issues={response.errors} kind="error" />
          </ul>
        </div>
        <div className="result-column">
          <h2>Warnings</h2>
          <ul>
            <IssueList issues={response.warnings} kind="warning" />
          </ul>
        </div>
        <div className="result-column">
          <h2>Metadata</h2>
          <pre>{formattedMetadata}</pre>
        </div>
      </section>
    </div>
  );
}
