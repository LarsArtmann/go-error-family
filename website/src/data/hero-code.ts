import { siteConfig } from "./config";

const importPath = siteConfig.github.replace("https://github.com/", "github.com/");

export const heroCode = `package main

import (
    "errors"
    "os"

    errorfamily "${importPath}"
)

func main() {
    err := errors.New("connection refused")

    // Classify any error into a behavioral family
    family := errorfamily.Classify(err)
    // → Transient (unknown errors are retryable by default)

    if errorfamily.IsRetryable(err) {
        retry() // backoff/jitter/idempotency are yours
    }

    os.Exit(errorfamily.ExitCode(err))
    // → 75 (EX_TEMPFAIL)
}`;
