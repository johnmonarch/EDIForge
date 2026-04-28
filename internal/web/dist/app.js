(function () {
  "use strict";

  var sampleX12 = "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~GS*PO*SENDER*RECEIVER*20260427*1200*1*X*004010~ST*850*0001~BEG*00*SA*PO-10001**20260427~N1*BY*Example Buyer*92*BUYER01~N1*ST*Example Ship To*92*SHIPTO01~N3*100 Warehouse Road~N4*Greenville*SC*29601*US~PO1*1*10*EA*3.50**VN*SKU-100~CTT*1~SE*9*0001~GE*1*1~IEA*1*000000001~";
  var sampleEdifact = "UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1'UNH+1+ORDERS:D:96A:UN'BGM+220+PO-10001+9'DTM+137:20260427:102'NAD+BY+BUYER01::92++Example Buyer'NAD+DP+SHIPTO01::92++Example Ship To'LIN+1++SKU-100:VN'QTY+21:10:EA'UNS+S'UNT+9+1'UNZ+1+1'";

  var ediInput = document.getElementById("ediInput");
  var fileInput = document.getElementById("fileInput");
  var standardSelect = document.getElementById("standardSelect");
  var modeSelect = document.getElementById("modeSelect");
  var schemaIdInput = document.getElementById("schemaIdInput");
  var apiBaseInput = document.getElementById("apiBaseInput");
  var serverState = document.getElementById("serverState");
  var inputStats = document.getElementById("inputStats");
  var responseMeta = document.getElementById("responseMeta");
  var jsonOutput = document.getElementById("jsonOutput");
  var errorsList = document.getElementById("errorsList");
  var warningsList = document.getElementById("warningsList");
  var metadataOutput = document.getElementById("metadataOutput");
  var envelopeSummary = document.getElementById("envelopeSummary");
  var transactionSummary = document.getElementById("transactionSummary");
  var segmentSummary = document.getElementById("segmentSummary");
  var segmentFilter = document.getElementById("segmentFilter");
  var segmentCount = document.getElementById("segmentCount");
  var segmentList = document.getElementById("segmentList");
  var segmentDetail = document.getElementById("segmentDetail");
  var currentResponse = {};
  var currentSegments = [];
  var selectedSegmentIndex = -1;

  function defaultApiBase() {
    if (window.location.protocol === "http:" || window.location.protocol === "https:") {
      return window.location.origin;
    }
    return "http://127.0.0.1:8765";
  }

  apiBaseInput.value = defaultApiBase();

  function endpoint(path) {
    var base = apiBaseInput.value.trim().replace(/\/+$/, "");
    return base + path;
  }

  function setState(text, kind) {
    serverState.textContent = text;
    serverState.className = "server-state" + (kind ? " " + kind : "");
  }

  function updateInputStats() {
    var value = ediInput.value;
    var chars = value.length;
    var segments = value ? value.split(/[~']/).filter(Boolean).length : 0;
    inputStats.textContent = chars + " chars, " + segments + " segments";
  }

  function pretty(value) {
    return JSON.stringify(value, null, 2);
  }

  function text(value, fallback) {
    if (value === undefined || value === null || value === "") {
      return fallback === undefined ? "None" : fallback;
    }
    return String(value);
  }

  function addSummaryRow(target, label, value) {
    var dt = document.createElement("dt");
    var dd = document.createElement("dd");
    dt.textContent = label;
    dd.textContent = text(value);
    target.appendChild(dt);
    target.appendChild(dd);
  }

  function renderSummary(target, rows) {
    target.innerHTML = "";
    rows.forEach(function (row) {
      addSummaryRow(target, row[0], row[1]);
    });
  }

  function isObject(value) {
    return value !== null && typeof value === "object";
  }

  function isSegment(value) {
    return isObject(value) && typeof value.tag === "string" && Array.isArray(value.elements);
  }

  function segmentContext(path) {
    var labels = [];
    path.forEach(function (part) {
      if (part.kind === "interchange") {
        labels.push("Interchange " + part.index);
      } else if (part.kind === "group") {
        labels.push("Group " + part.index);
      } else if (part.kind === "transaction") {
        labels.push("TX " + text(part.type, part.index));
      } else if (part.kind === "message") {
        labels.push("MSG " + text(part.type, part.index));
      } else if (part.kind === "rawEnvelope") {
        labels.push("Envelope");
      }
    });
    return labels.join(" / ");
  }

  function collectSegments(value, path, out, seen) {
    if (!isObject(value) || seen.indexOf(value) !== -1) {
      return;
    }
    seen.push(value);

    if (isSegment(value)) {
      out.push({
        segment: value,
        context: segmentContext(path),
        index: out.length
      });
      return;
    }

    if (Array.isArray(value)) {
      value.forEach(function (item) {
        collectSegments(item, path, out, seen);
      });
      return;
    }

    Object.keys(value).forEach(function (key) {
      var next = value[key];
      var nextPath = path.slice();
      if (key === "interchanges" && Array.isArray(next)) {
        next.forEach(function (item, index) {
          collectSegments(item, nextPath.concat([{ kind: "interchange", index: index + 1 }]), out, seen);
        });
      } else if (key === "groups" && Array.isArray(next)) {
        next.forEach(function (item, index) {
          collectSegments(item, nextPath.concat([{ kind: "group", index: index + 1 }]), out, seen);
        });
      } else if (key === "transactions" && Array.isArray(next)) {
        next.forEach(function (item, index) {
          collectSegments(item, nextPath.concat([{ kind: "transaction", index: index + 1, type: item && item.type }]), out, seen);
        });
      } else if (key === "messages" && Array.isArray(next)) {
        next.forEach(function (item, index) {
          collectSegments(item, nextPath.concat([{ kind: "message", index: index + 1, type: item && item.type }]), out, seen);
        });
      } else if (key === "rawEnvelope" && Array.isArray(next)) {
        collectSegments(next, nextPath.concat([{ kind: "rawEnvelope" }]), out, seen);
      } else if (key !== "elements") {
        collectSegments(next, nextPath, out, seen);
      }
    });
  }

  function firstInterchange(result) {
    return result && Array.isArray(result.interchanges) && result.interchanges.length ? result.interchanges[0] : {};
  }

  function allTransactions(result) {
    var transactions = [];
    (result && result.interchanges || []).forEach(function (interchange) {
      (interchange.groups || []).forEach(function (group) {
        (group.transactions || []).forEach(function (transaction) {
          transactions.push(transaction);
        });
      });
    });
    return transactions;
  }

  function allMessages(result) {
    var messages = [];
    (result && result.interchanges || []).forEach(function (interchange) {
      (interchange.messages || []).forEach(function (message) {
        messages.push(message);
      });
    });
    return messages;
  }

  function topTags(segments) {
    var counts = {};
    segments.forEach(function (entry) {
      var tag = entry.segment.tag || "UNKNOWN";
      counts[tag] = (counts[tag] || 0) + 1;
    });
    return Object.keys(counts).sort(function (a, b) {
      return counts[b] - counts[a] || a.localeCompare(b);
    }).slice(0, 6).map(function (tag) {
      return tag + " " + counts[tag];
    }).join(", ");
  }

  function renderSegmentDetail(entry) {
    if (!entry) {
      segmentDetail.textContent = "Select a segment to inspect details.";
      return;
    }
    segmentDetail.textContent = pretty({
      context: entry.context || "Document",
      tag: entry.segment.tag,
      position: entry.segment.position,
      elements: entry.segment.elements,
      raw: entry.segment.raw,
      offset: entry.segment.offset
    });
  }

  function renderSegments() {
    var filter = segmentFilter.value.trim().toUpperCase();
    var filtered = currentSegments.filter(function (entry) {
      return !filter || (entry.segment.tag || "").toUpperCase().indexOf(filter) === 0;
    });

    segmentList.innerHTML = "";
    segmentCount.textContent = filtered.length + " of " + currentSegments.length + " segments";

    if (!filtered.length) {
      var empty = document.createElement("li");
      empty.className = "empty-segment";
      empty.textContent = currentSegments.length ? "No matching segments." : "No segment data in this response.";
      segmentList.appendChild(empty);
      renderSegmentDetail(null);
      return;
    }

    if (filtered.every(function (entry) { return entry.index !== selectedSegmentIndex; })) {
      selectedSegmentIndex = filtered[0].index;
    }

    filtered.forEach(function (entry) {
      var segment = entry.segment;
      var button = document.createElement("button");
      button.type = "button";
      button.className = "segment-item" + (entry.index === selectedSegmentIndex ? " selected" : "");

      var tag = document.createElement("strong");
      tag.textContent = segment.tag || "UNK";
      var meta = document.createElement("span");
      meta.textContent = "#" + text(segment.position, entry.index + 1) + (entry.context ? " - " + entry.context : "");
      var elements = document.createElement("small");
      elements.textContent = (segment.elements || []).map(function (element) {
        if (element && Array.isArray(element.components) && element.components.length) {
          return element.components.join(":");
        }
        return element && element.value !== undefined ? element.value : "";
      }).filter(Boolean).slice(0, 5).join(" | ");

      button.appendChild(tag);
      button.appendChild(meta);
      if (elements.textContent) {
        button.appendChild(elements);
      }
      button.addEventListener("click", function () {
        selectedSegmentIndex = entry.index;
        renderSegments();
        renderSegmentDetail(entry);
      });

      var li = document.createElement("li");
      li.appendChild(button);
      segmentList.appendChild(li);
    });

    renderSegmentDetail(filtered.filter(function (entry) {
      return entry.index === selectedSegmentIndex;
    })[0]);
  }

  function renderWorkbench(data) {
    data = data || {};
    var result = data.result || {};
    var metadata = data.metadata || result.metadata || {};
    var interchange = firstInterchange(result);
    var transactions = allTransactions(result);
    var messages = allMessages(result);
    var segments = [];
    collectSegments(result, [], segments, []);
    currentSegments = segments;
    selectedSegmentIndex = segments.length ? segments[0].index : -1;

    renderSummary(envelopeSummary, [
      ["Standard", result.standard || data.standard || metadata.standard],
      ["Version", result.version || interchange.version],
      ["Sender", interchange.senderId],
      ["Receiver", interchange.receiverId],
      ["Control", interchange.controlNumber]
    ]);

    renderSummary(transactionSummary, [
      ["Transactions", metadata.transactions || transactions.length],
      ["Messages", metadata.messages || messages.length],
      ["Groups", metadata.groups || (interchange.groups || []).length],
      ["Types", transactions.map(function (item) { return item.type; }).concat(messages.map(function (item) { return item.type; })).filter(Boolean).join(", ")],
      ["Mode", metadata.mode]
    ]);

    renderSummary(segmentSummary, [
      ["Total", metadata.segments || segments.length],
      ["Listed", segments.length],
      ["Top tags", topTags(segments)],
      ["Warnings", Array.isArray(data.warnings) ? data.warnings.length : 0],
      ["Errors", Array.isArray(data.errors) ? data.errors.length : 0]
    ]);

    renderSegments();
  }

  function renderList(target, items, kind) {
    target.innerHTML = "";
    if (!Array.isArray(items) || items.length === 0) {
      var empty = document.createElement("li");
      empty.textContent = "None";
      target.appendChild(empty);
      return;
    }
    items.forEach(function (item) {
      var li = document.createElement("li");
      li.className = kind;
      if (typeof item === "string") {
        li.textContent = item;
      } else {
        var code = item.code ? item.code + ": " : "";
        var message = item.message || item.hint || pretty(item);
        li.textContent = code + message;
      }
      target.appendChild(li);
    });
  }

  function renderResponse(data, label) {
    currentResponse = data || {};
    jsonOutput.textContent = pretty(currentResponse);
    responseMeta.textContent = label || "Response";
    renderList(errorsList, currentResponse.errors, "error");
    renderList(warningsList, currentResponse.warnings, "warning");
    metadataOutput.textContent = pretty(currentResponse.metadata || {});
    renderWorkbench(currentResponse);
  }

  function payload(extra) {
    var body = {
      input: ediInput.value,
      standard: standardSelect.value
    };
    Object.keys(extra || {}).forEach(function (key) {
      body[key] = extra[key];
    });
    return body;
  }

  function request(path, body, label) {
    if (!ediInput.value.trim()) {
      renderResponse({
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
      }, "Local validation");
      setState("Input needed", "error");
      return Promise.resolve();
    }

    setState("Working", "");
    return fetch(endpoint(path), {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(body)
    })
      .then(function (response) {
        return response.text().then(function (text) {
          var parsed;
          try {
            parsed = text ? JSON.parse(text) : {};
          } catch (error) {
            parsed = {
              ok: false,
              errors: [
                {
                  severity: "error",
                  code: "NON_JSON_RESPONSE",
                  message: text || response.statusText
                }
              ],
              metadata: {
                status: response.status
              }
            };
          }
          if (!response.ok && parsed.ok !== false) {
            parsed.ok = false;
          }
          renderResponse(parsed, label + " (" + response.status + ")");
          setState(parsed.ok === false ? "Errors" : "Ready", parsed.ok === false ? "error" : "ok");
        });
      })
      .catch(function (error) {
        renderResponse({
          ok: false,
          errors: [
            {
              severity: "error",
              code: "REQUEST_FAILED",
              message: error.message
            }
          ],
          warnings: [],
          metadata: {
            endpoint: endpoint(path)
          }
        }, label + " failed");
        setState("Offline", "error");
      });
  }

  document.getElementById("detectButton").addEventListener("click", function () {
    request("/api/v1/detect", payload({}), "Detect");
  });

  document.getElementById("translateButton").addEventListener("click", function () {
    request("/api/v1/translate", payload({
      mode: modeSelect.value,
      schemaId: schemaIdInput.value.trim() || undefined,
      options: {
        pretty: true,
        includeEnvelope: true,
        includeRawSegments: false
      }
    }), "Translate");
  });

  document.getElementById("validateButton").addEventListener("click", function () {
    request("/api/v1/validate", payload({
      level: schemaIdInput.value.trim() ? "schema" : "syntax",
      schemaId: schemaIdInput.value.trim() || undefined
    }), "Validate");
  });

  document.getElementById("clearButton").addEventListener("click", function () {
    ediInput.value = "";
    updateInputStats();
    renderResponse({}, "No response");
    setState("Idle", "");
  });

  document.querySelectorAll("[data-sample]").forEach(function (button) {
    button.addEventListener("click", function () {
      var sample = button.getAttribute("data-sample");
      ediInput.value = sample === "edifact" ? sampleEdifact : sampleX12;
      standardSelect.value = sample === "edifact" ? "edifact" : "x12";
      schemaIdInput.value = sample === "edifact" ? "edifact-orders-basic" : "x12-850-basic";
      updateInputStats();
      setState("Sample loaded", "ok");
    });
  });

  document.getElementById("copyButton").addEventListener("click", function () {
    var text = pretty(currentResponse);
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function () {
        setState("Copied", "ok");
      }).catch(function () {
        setState("Copy failed", "error");
      });
    } else {
      setState("Copy unavailable", "error");
    }
  });

  document.getElementById("downloadButton").addEventListener("click", function () {
    var blob = new Blob([pretty(currentResponse)], { type: "application/json" });
    var url = URL.createObjectURL(blob);
    var link = document.createElement("a");
    link.href = url;
    link.download = "ediforge-result.json";
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    setState("Downloaded", "ok");
  });

  fileInput.addEventListener("change", function () {
    var file = fileInput.files && fileInput.files[0];
    if (!file) {
      return;
    }
    file.text().then(function (text) {
      ediInput.value = text;
      updateInputStats();
      setState("File loaded", "ok");
    }).catch(function (error) {
      renderResponse({
        ok: false,
        errors: [
          {
            severity: "error",
            code: "FILE_READ_FAILED",
            message: error.message
          }
        ],
        warnings: [],
        metadata: {}
      }, "File read failed");
      setState("File error", "error");
    });
  });

  segmentFilter.addEventListener("input", renderSegments);
  ediInput.addEventListener("input", updateInputStats);
  updateInputStats();
  renderResponse({}, "No response");
}());
