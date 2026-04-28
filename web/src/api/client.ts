export type EDIStandard = "auto" | "x12" | "edifact";
export type TranslateMode = "structural" | "annotated" | "semantic";
export type ValidationLevel = "syntax" | "schema" | "partner";

export interface APIIssue {
  severity?: string;
  code?: string;
  message?: string;
  hint?: string;
  [key: string]: unknown;
}

export interface APIResponse<T = unknown> {
  ok?: boolean;
  result?: T;
  warnings?: APIIssue[];
  errors?: APIIssue[];
  metadata?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface BaseRequest {
  input: string;
  standard: EDIStandard;
}

export interface TranslateRequest extends BaseRequest {
  mode: TranslateMode;
  schemaId?: string;
  options: {
    pretty: boolean;
    includeEnvelope: boolean;
    includeRawSegments: boolean;
  };
}

export interface ValidateRequest extends BaseRequest {
  level: ValidationLevel;
  schemaId?: string;
}

export async function postJSON<TResponse>(
  apiBase: string,
  path: string,
  body: unknown
): Promise<APIResponse<TResponse>> {
  const base = apiBase.replace(/\/+$/, "");
  const response = await fetch(`${base}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  });

  const text = await response.text();
  let parsed: APIResponse<TResponse>;
  try {
    parsed = text ? (JSON.parse(text) as APIResponse<TResponse>) : {};
  } catch {
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

  return parsed;
}
