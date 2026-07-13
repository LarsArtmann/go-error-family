import type { StepCard, ComparisonItem, UseCase, ComparisonMatrix } from "./types";

export const steps: StepCard[] = [
  {
    step: "1",
    stepColor: "accent",
    title: "Classify",
    desc: "Any error becomes a Family: interface, sentinel, classifier, or default.",
    code: "family := errorfamily.Classify(err)",
  },
  {
    step: "2",
    stepColor: "accent",
    title: "Decide",
    desc: "Retry? Exit code? HTTP status? All derived from the Family.",
    code: "errorfamily.IsRetryable(err) // true",
  },
  {
    step: "3",
    stepColor: "amber",
    title: "Handle",
    desc: "CLI boundary: structured What/Why/Fix/WayOut messages on stderr.",
    code: "os.Exit(errorfamily.HandleError(err))",
  },
  {
    step: "4",
    stepColor: "amber",
    title: "Diagnose",
    desc: "Auto-discover root cause: filesystem, network, git, PostgreSQL.",
    code: "runner.Run(ctx, err) → []DiagnosticResult",
  },
];

export const comparisons: ComparisonItem[] = [
  {
    variant: "fmt.Errorf",
    accent: false,
    pros: ["Zero dependencies", "stdlib, always available"],
    cons: [
      "No retry decision",
      "No exit code mapping",
      "No HTTP status",
      "No user-facing messages",
    ],
  },
  {
    variant: "go-error-family",
    accent: true,
    pros: [
      "Family = retry + exit code + HTTP status + tone",
      "Universal Classify for any error",
      "Multi-error worst-severity selection",
      "HTTP middleware with safe JSON responses",
      "Diagnostic rules for root-cause analysis",
      "Zero third-party dependencies (root module)",
    ],
    cons: [],
  },
  {
    variant: "DIY",
    accent: false,
    pros: ["No external deps"],
    cons: [
      "String matching for retry decisions",
      "Hardcoded exit codes everywhere",
      "Manual HTTP status mapping",
      "No diagnostic context",
    ],
  },
];

export const comparisonMatrix: ComparisonMatrix = {
  columns: ["fmt.Errorf", "DIY", "go-error-family"],
  rows: [
    { feature: "Behavioral classification", values: ["no", "manual", "yes"] },
    { feature: "Retry decisions", values: ["no", "string match", "yes"] },
    { feature: "Exit code mapping", values: ["no", "manual", "yes"] },
    { feature: "HTTP status mapping", values: ["no", "manual", "yes"] },
    { feature: "Multi-error severity", values: ["no", "no", "yes"] },
    { feature: "User-facing messages", values: ["no", "manual", "yes"] },
    { feature: "Diagnostic rules", values: ["no", "no", "yes"] },
    { feature: "Dependencies", values: ["0", "0", "0"] },
  ],
};

export const useCases: UseCase[] = [
  {
    title: "CLI Tools",
    desc: "Correct exit codes and user-facing messages from any error",
    icon: "cog",
  },
  {
    title: "HTTP APIs",
    desc: "Status codes and safe JSON responses without internal leakage",
    icon: "chart",
  },
  {
    title: "Libraries & SDKs",
    desc: "Classify errors at the domain boundary, let consumers decide",
    icon: "bolt",
  },
];
