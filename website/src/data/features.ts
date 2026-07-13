import type { Feature } from "./types";

export const features: Feature[] = [
  {
    icon: "shield",
    title: "Behavioral Classification",
    desc: "Five families (Rejection, Conflict, Transient, Corruption, Infrastructure) map to retry decisions, exit codes, HTTP status, and user-facing tone.",
  },
  {
    icon: "lightning",
    title: "Universal Classify",
    desc: "Classify ANY error via a 6-step chain: multi-error, interface, sentinel, classifier, default. Zero allocations on the hot path.",
  },
  {
    icon: "refresh",
    title: "Multi-Error Support",
    desc: "errors.Join with Classify picks the worst family by severity. Deterministic regardless of argument order. Fail-closed.",
  },
  {
    icon: "lock",
    title: "HTTP Middleware",
    desc: "HTTPHandler maps errors to safe JSON responses with correct status codes. Never leaks internal error messages.",
  },
  {
    icon: "database",
    title: "Diagnostic Rules",
    desc: "Auto-discover root causes: filesystem, network, git, PostgreSQL. Structured Fix commands, not prose to parse.",
  },
  {
    icon: "folder",
    title: "Zero Dependencies",
    desc: "Root module has zero third-party deps. Only stdlib. Optional submodules for oops bridge and diagnostics.",
  },
];
