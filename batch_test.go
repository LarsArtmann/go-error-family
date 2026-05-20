package errorfamily

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
)

// --- Outcome tests ---

func TestOutcomeIsOK(t *testing.T) {
	tests := []struct {
		name string
		o    Outcome[string]
		want bool
	}{
		{"success", Outcome[string]{Value: "hello", Err: nil}, true},
		{"failure", Outcome[string]{Value: "", Err: NewRejection("test", "fail")}, false},
		{"zero value success", Outcome[string]{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.IsOK(); got != tt.want {
				t.Errorf("Outcome.IsOK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOutcome(t *testing.T) {
	o := NewOutcome("value", nil)
	if !o.IsOK() || o.Value != "value" {
		t.Errorf("NewOutcome(\"value\", nil) = %+v, want OK with value \"value\"", o)
	}

	o2 := NewOutcome("val", NewTransient("t", "test"))
	if o2.IsOK() {
		t.Error("NewOutcome with error should not be OK")
	}
}

// --- ErrorBatch tests ---

func TestErrorBatchAdd(t *testing.T) {
	b := NewErrorBatch()
	if b.Len() != 0 {
		t.Errorf("new batch Len() = %d, want 0", b.Len())
	}
	if b.HasFailures() {
		t.Error("new batch should not have failures")
	}

	b.Add(NewTransient("t1", "timeout"))
	b.Add(NewRejection("r1", "bad input"))
	b.Add(nil) // should be ignored

	if b.Len() != 2 {
		t.Errorf("Len() = %d, want 2", b.Len())
	}
	if !b.HasFailures() {
		t.Error("should have failures")
	}
}

func TestErrorBatchAddBatch(t *testing.T) {
	b1 := NewErrorBatch()
	b1.Add(NewTransient("t1", "timeout"))

	b2 := NewErrorBatch()
	b2.Add(NewRejection("r1", "bad"))
	b2.AddBatch(b1)

	if b2.Len() != 2 {
		t.Errorf("Len() = %d, want 2", b2.Len())
	}
}

func TestErrorBatchErrors(t *testing.T) {
	b := NewErrorBatch()
	err1 := NewTransient("t1", "timeout")
	err2 := NewRejection("r1", "bad")
	b.Add(err1)
	b.Add(err2)

	errs := b.Errors()
	if len(errs) != 2 {
		t.Fatalf("Errors() len = %d, want 2", len(errs))
	}
	// Verify it's a copy
	errs[0] = nil
	if b.Len() != 2 {
		t.Error("modifying Errors() slice should not affect batch")
	}
}

func TestErrorBatchFamilies(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewTransient("t1", "timeout"))
	b.Add(NewTransient("t2", "timeout2"))
	b.Add(NewRejection("r1", "bad"))

	families := b.Families()
	if families[Transient] != 2 {
		t.Errorf("Transient count = %d, want 2", families[Transient])
	}
	if families[Rejection] != 1 {
		t.Errorf("Rejection count = %d, want 1", families[Rejection])
	}
}

func TestErrorBatchDominantFamily(t *testing.T) {
	tests := []struct {
		name   string
		errors []error
		want   Family
	}{
		{"empty", nil, Rejection},
		{"all transient", []error{NewTransient("t", ""), NewTransient("t", "")}, Transient},
		{"mixed transient rejection", []error{NewTransient("t", ""), NewRejection("r", "")}, Rejection},
		{"mixed with corruption", []error{NewTransient("t", ""), NewRejection("r", ""), NewCorruption("c", "")}, Corruption},
		{"infrastructure beats conflict", []error{NewConflict("c", ""), NewInfrastructure("i", "")}, Infrastructure},
		{"corruption is worst", []error{NewTransient("t", ""), NewRejection("r", ""), NewConflict("c", ""), NewCorruption("co", ""), NewInfrastructure("i", "")}, Corruption},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewErrorBatch()
			for _, e := range tt.errors {
				b.Add(e)
			}
			got := b.DominantFamily()
			if tt.errors == nil {
				return // empty batch returns Rejection but we don't add errors
			}
			if got != tt.want {
				t.Errorf("DominantFamily() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorBatchRetryable(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewTransient("t1", "timeout"))
	b.Add(NewRejection("r1", "bad"))
	b.Add(NewTransient("t2", "timeout2"))

	if !b.HasRetryable() {
		t.Error("should have retryable errors")
	}

	retryable := b.Retryable()
	if len(retryable) != 2 {
		t.Errorf("Retryable() len = %d, want 2", len(retryable))
	}
}

func TestErrorBatchNoRetryable(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewRejection("r1", "bad"))
	b.Add(NewConflict("c1", "dup"))

	if b.HasRetryable() {
		t.Error("should not have retryable errors")
	}
	if len(b.Retryable()) != 0 {
		t.Error("Retryable() should be empty")
	}
}

func TestErrorBatchErr(t *testing.T) {
	b := NewErrorBatch()
	if b.Err() != nil {
		t.Error("empty batch Err() should be nil")
	}

	b.Add(NewTransient("t1", "timeout"))
	b.Add(NewRejection("r1", "bad"))

	err := b.Err()
	if err == nil {
		t.Fatal("Err() should not be nil")
	}

	// Verify it implements the interfaces
	var classified Classified
	if !errors.As(err, &classified) {
		t.Error("batch error should implement Classified")
	}
	if classified.ErrorFamily() != Rejection {
		t.Errorf("ErrorFamily() = %v, want %v (Rejection dominates Transient)", classified.ErrorFamily(), Rejection)
	}

	var coded Coded
	if !errors.As(err, &coded) {
		t.Error("batch error should implement Coded")
	}
	if coded.ErrorCode() != "batch" {
		t.Errorf("ErrorCode() = %q, want %q", coded.ErrorCode(), "batch")
	}

	var contextual Contextual
	if !errors.As(err, &contextual) {
		t.Error("batch error should implement Contextual")
	}
	ctx := contextual.ErrorContext()
	if ctx["total"] != "2" || ctx["failed"] != "2" {
		t.Errorf("ErrorContext() = %v, want total=2, failed=2", ctx)
	}
}

func TestErrorBatchErrAllTransient(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewTransient("t1", "timeout"))
	b.Add(NewTransient("t2", "timeout2"))

	err := b.Err()
	if !IsRetryable(err) {
		t.Error("all-transient batch should be retryable")
	}
	if Classify(err) != Transient {
		t.Errorf("Classify() = %v, want Transient", Classify(err))
	}
}

func TestErrorBatchExitCode(t *testing.T) {
	tests := []struct {
		name string
		errs []error
		want int
	}{
		{"empty", nil, 0},
		{"all transient", []error{NewTransient("t", "")}, 75},
		{"rejection", []error{NewRejection("r", "")}, 1},
		{"corruption", []error{NewCorruption("c", "")}, 65},
		{"infrastructure", []error{NewInfrastructure("i", "")}, 69},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewErrorBatch()
			for _, e := range tt.errs {
				b.Add(e)
			}
			if got := b.ExitCode(); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestErrorBatchErrorString(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewTransient("t1", "timeout"))
	b.Add(NewRejection("r1", "bad"))

	err := b.Err()
	got := err.Error()
	if !strings.Contains(got, "[rejection:batch]") {
		t.Errorf("Error() = %q, should contain [rejection:batch]", got)
	}
	if !strings.Contains(got, "2 items failed") {
		t.Errorf("Error() = %q, should contain '2 items failed'", got)
	}
}

// --- BatchResult tests ---

func TestBatchResultBasic(t *testing.T) {
	br := NewBatchResult[string]()
	if br.Len() != 0 {
		t.Errorf("new BatchResult Len() = %d, want 0", br.Len())
	}

	br.Add("item1", nil)
	br.Add("item2", NewTransient("t1", "timeout"))
	br.Add("item3", nil)

	if br.Len() != 3 {
		t.Errorf("Len() = %d, want 3", br.Len())
	}
	if !br.HasFailures() {
		t.Error("should have failures")
	}
	if !br.IsPartial() {
		t.Error("should be partial (2 success, 1 failure)")
	}
	if br.AllSucceeded() {
		t.Error("should not be all succeeded")
	}
	if br.AllFailed() {
		t.Error("should not be all failed")
	}
}

func TestBatchResultAllSucceeded(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", nil)
	br.Add("b", nil)

	if !br.AllSucceeded() {
		t.Error("should be all succeeded")
	}
	if br.HasFailures() {
		t.Error("should not have failures")
	}
	if br.IsPartial() {
		t.Error("should not be partial")
	}
	if br.AllFailed() {
		t.Error("should not be all failed")
	}
}

func TestBatchResultAllFailed(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", NewRejection("r", "bad"))
	br.Add("b", NewTransient("t", "timeout"))

	if br.AllSucceeded() {
		t.Error("should not be all succeeded")
	}
	if !br.AllFailed() {
		t.Error("should be all failed")
	}
	if br.IsPartial() {
		t.Error("should not be partial")
	}
}

func TestBatchResultEmpty(t *testing.T) {
	br := NewBatchResult[string]()
	if br.AllSucceeded() {
		t.Error("empty should not be all succeeded")
	}
	if br.AllFailed() {
		t.Error("empty should not be all failed")
	}
	if br.IsPartial() {
		t.Error("empty should not be partial")
	}
	if br.HasFailures() {
		t.Error("empty should not have failures")
	}
	if br.Err() != nil {
		t.Error("empty Err() should be nil")
	}
	if br.ExitCode() != 0 {
		t.Error("empty ExitCode() should be 0")
	}
}

func TestBatchResultSuccesses(t *testing.T) {
	br := NewBatchResult[int]()
	br.Add(1, nil)
	br.Add(2, NewTransient("t", "fail"))
	br.Add(3, nil)

	successes := br.Successes()
	if len(successes) != 2 {
		t.Fatalf("Successes() len = %d, want 2", len(successes))
	}
	if successes[0] != 1 || successes[1] != 3 {
		t.Errorf("Successes() = %v, want [1, 3]", successes)
	}
}

func TestBatchResultFailures(t *testing.T) {
	br := NewBatchResult[int]()
	br.Add(1, nil)
	br.Add(2, NewTransient("t", "fail"))
	br.Add(3, NewRejection("r", "bad"))

	failures := br.Failures()
	if len(failures) != 2 {
		t.Fatalf("Failures() len = %d, want 2", len(failures))
	}
	if failures[0].Value != 2 {
		t.Errorf("Failures()[0].Value = %d, want 2", failures[0].Value)
	}
	if failures[1].Value != 3 {
		t.Errorf("Failures()[1].Value = %d, want 3", failures[1].Value)
	}
}

func TestBatchResultAddOutcome(t *testing.T) {
	br := NewBatchResult[string]()
	br.AddOutcome(NewOutcome("a", nil))
	br.AddOutcome(NewOutcome("b", NewRejection("r", "bad")))

	if br.Len() != 2 {
		t.Errorf("Len() = %d, want 2", br.Len())
	}
	if !br.IsPartial() {
		t.Error("should be partial")
	}
}

func TestBatchResultAddResult(t *testing.T) {
	br1 := NewBatchResult[string]()
	br1.Add("a", nil)
	br1.Add("b", NewTransient("t", "fail"))

	br2 := NewBatchResult[string]()
	br2.Add("c", nil)
	br2.AddResult(br1)

	if br2.Len() != 3 {
		t.Errorf("Len() = %d, want 3", br2.Len())
	}
	successes := br2.Successes()
	if len(successes) != 2 {
		t.Errorf("Successes() len = %d, want 2", len(successes))
	}
}

func TestBatchResultFamilies(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", NewTransient("t1", ""))
	br.Add("b", NewTransient("t2", ""))
	br.Add("c", NewRejection("r1", ""))
	br.Add("d", nil)

	families := br.Families()
	if families[Transient] != 2 {
		t.Errorf("Transient count = %d, want 2", families[Transient])
	}
	if families[Rejection] != 1 {
		t.Errorf("Rejection count = %d, want 1", families[Rejection])
	}
}

func TestBatchResultDominantFamily(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", NewTransient("t", ""))
	br.Add("b", NewCorruption("c", ""))
	br.Add("c", nil)

	if br.DominantFamily() != Corruption {
		t.Errorf("DominantFamily() = %v, want Corruption", br.DominantFamily())
	}
}

func TestBatchResultRetryable(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", NewTransient("t1", ""))
	br.Add("b", NewRejection("r1", ""))
	br.Add("c", NewTransient("t2", ""))
	br.Add("d", nil)

	if !br.HasRetryable() {
		t.Error("should have retryable failures")
	}

	retryable := br.RetryableFailures()
	if len(retryable) != 2 {
		t.Errorf("RetryableFailures() len = %d, want 2", len(retryable))
	}
}

func TestBatchResultNoRetryable(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", NewRejection("r1", ""))
	br.Add("b", NewConflict("c1", ""))

	if br.HasRetryable() {
		t.Error("should not have retryable failures")
	}
}

func TestBatchResultErr(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", nil)
	br.Add("b", nil)
	br.Add("c", NewTransient("t", "timeout"))
	br.Add("d", NewRejection("r", "bad"))

	err := br.Err()
	if err == nil {
		t.Fatal("Err() should not be nil")
	}

	// Should implement all interfaces
	if _, ok := err.(Classified); !ok {
		t.Error("should implement Classified")
	}
	if _, ok := err.(Coded); !ok {
		t.Error("should implement Coded")
	}
	if _, ok := err.(Contextual); !ok {
		t.Error("should implement Contextual")
	}
	if _, ok := err.(Retryable); !ok {
		t.Error("should implement Retryable")
	}

	// Family should be Rejection (dominates Transient)
	if Classify(err) != Rejection {
		t.Errorf("Classify() = %v, want Rejection", Classify(err))
	}

	// Context should have succeeded + failed + total
	ctx := err.(Contextual).ErrorContext()
	if ctx["total"] != "4" {
		t.Errorf("total = %q, want \"4\"", ctx["total"])
	}
	if ctx["succeeded"] != "2" {
		t.Errorf("succeeded = %q, want \"2\"", ctx["succeeded"])
	}
	if ctx["failed"] != "2" {
		t.Errorf("failed = %q, want \"2\"", ctx["failed"])
	}
}

func TestBatchResultErrNil(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", nil)
	br.Add("b", nil)

	if br.Err() != nil {
		t.Error("all-succeeded batch Err() should be nil")
	}
}

func TestBatchResultErrAllTransient(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", NewTransient("t1", ""))
	br.Add("b", NewTransient("t2", ""))

	err := br.Err()
	if !IsRetryable(err) {
		t.Error("all-transient batch should be retryable")
	}
}

func TestBatchResultErrorString(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", nil)
	br.Add("b", NewRejection("r", "bad"))
	br.Add("c", NewTransient("t", "timeout"))

	err := br.Err()
	got := err.Error()
	if !strings.Contains(got, "[rejection:batch]") {
		t.Errorf("Error() = %q, should contain [rejection:batch]", got)
	}
	if !strings.Contains(got, "2 of 3 items failed") {
		t.Errorf("Error() = %q, should contain '2 of 3 items failed'", got)
	}
}

func TestBatchResultErrorStringNoSuccesses(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", NewRejection("r", "bad"))
	br.Add("b", NewTransient("t", "timeout"))

	err := br.Err()
	got := err.Error()
	if !strings.Contains(got, "2 items failed") {
		t.Errorf("Error() = %q, should contain '2 items failed'", got)
	}
	if strings.Contains(got, "of") {
		t.Errorf("Error() = %q, should NOT contain 'of' for all-failed batch", got)
	}
}

func TestBatchResultFormatVerbose(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", nil)
	br.Add("b", NewRejection("r1", "bad input"))
	br.Add("c", NewTransient("t1", "timeout"))

	err := br.Err()
	got := fmt.Sprintf("%+v", err)
	if !strings.Contains(got, "[rejection:batch] 2 of 3 items failed") {
		t.Errorf("%%+v = %q, should contain summary line", got)
	}
	if !strings.Contains(got, "[1]") || !strings.Contains(got, "[2]") {
		t.Errorf("%%+v = %q, should contain numbered failures", got)
	}
	if !strings.Contains(got, "bad input") {
		t.Errorf("%%+v = %q, should contain error details", got)
	}
}

func TestBatchResultExitCode(t *testing.T) {
	tests := []struct {
		name string
		add  func(*BatchResult[string])
		want int
	}{
		{
			"all succeeded",
			func(br *BatchResult[string]) {
				br.Add("a", nil)
				br.Add("b", nil)
			},
			0,
		},
		{
			"partial transient",
			func(br *BatchResult[string]) {
				br.Add("a", nil)
				br.Add("b", NewTransient("t", ""))
			},
			75,
		},
		{
			"all rejection",
			func(br *BatchResult[string]) {
				br.Add("a", NewRejection("r", ""))
			},
			1,
		},
		{
			"corruption dominates",
			func(br *BatchResult[string]) {
				br.Add("a", NewTransient("t", ""))
				br.Add("b", NewCorruption("c", ""))
				br.Add("c", nil)
			},
			65,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := NewBatchResult[string]()
			tt.add(br)
			if got := br.ExitCode(); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

// --- Integration: HandleError with batch errors ---

func TestHandleErrorWithBatchError(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewTransient("t1", "timeout"))
	b.Add(NewRejection("r1", "bad"))

	err := b.Err()
	// HandleError should work because batchError implements Classified + Coded
	exitCode := HandleErrorWithConfig(err, HandleConfig{Output: io.Discard})
	if exitCode != 1 { // Rejection exit code
		t.Errorf("HandleError exit code = %d, want 1", exitCode)
	}
}

func TestHandleErrorDetailedWithBatchError(t *testing.T) {
	br := NewBatchResult[string]()
	br.Add("a", nil)
	br.Add("b", NewRejection("r1", "bad"))

	result := HandleErrorDetailed(br.Err())
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

// --- Integration: Classify with batch errors ---

func TestClassifyBatchError(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewTransient("t", ""))
	if Classify(b.Err()) != Transient {
		t.Error("all-transient batch should classify as Transient")
	}

	b2 := NewErrorBatch()
	b2.Add(NewCorruption("c", ""))
	b2.Add(NewTransient("t", ""))
	if Classify(b2.Err()) != Corruption {
		t.Error("corruption+transient batch should classify as Corruption")
	}
}

// --- Thread safety ---

func TestErrorBatchConcurrentAdd(t *testing.T) {
	b := NewErrorBatch()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b.Add(NewTransient("t", fmt.Sprintf("error %d", i)))
		}(i)
	}
	wg.Wait()
	if b.Len() != 100 {
		t.Errorf("Len() = %d, want 100", b.Len())
	}
}

func TestBatchResultConcurrentAdd(t *testing.T) {
	br := NewBatchResult[int]()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				br.Add(i, nil)
			} else {
				br.Add(i, NewTransient("t", fmt.Sprintf("error %d", i)))
			}
		}(i)
	}
	wg.Wait()
	if br.Len() != 100 {
		t.Errorf("Len() = %d, want 100", br.Len())
	}
	successes := br.Successes()
	if len(successes) != 50 {
		t.Errorf("Successes() len = %d, want 50", len(successes))
	}
	failures := br.Failures()
	if len(failures) != 50 {
		t.Errorf("Failures() len = %d, want 50", len(failures))
	}
}

// --- Edge cases ---

func TestErrorBatchFormat(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewRejection("r1", "bad input"))

	err := b.Err()

	// %s should use Error()
	got := fmt.Sprintf("%s", err)
	if !strings.Contains(got, "1 items failed") {
		t.Errorf("%%s = %q", got)
	}

	// %v should use Error()
	got = fmt.Sprintf("%v", err)
	if !strings.Contains(got, "[rejection:batch]") {
		t.Errorf("%%v = %q", got)
	}

	// %d (unknown verb) should fall back to Error()
	got = fmt.Sprintf("%d", err)
	if !strings.Contains(got, "[rejection:batch]") {
		t.Errorf("%%d = %q, should fall back to Error()", got)
	}
}

func TestErrorBatchFormatVerboseAllFailed(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewRejection("r1", "bad input"))

	got := fmt.Sprintf("%+v", b.Err())
	if !strings.Contains(got, "1 items failed") {
		t.Errorf("%%+v = %q, should contain '1 items failed' (no 'of' for all-failed)", got)
	}
	if strings.Contains(got, " of ") {
		t.Errorf("%%+v = %q, should NOT contain 'of' for all-failed batch", got)
	}
}

func TestBatchResultWithStructValue(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}

	br := NewBatchResult[Item]()
	br.Add(Item{ID: 1, Name: "first"}, nil)
	br.Add(Item{ID: 2, Name: "second"}, NewRejection("r", "duplicate"))

	successes := br.Successes()
	if len(successes) != 1 || successes[0].ID != 1 {
		t.Errorf("Successes() = %v", successes)
	}
}

func TestDominantFamilyEmpty(t *testing.T) {
	if dominantFamily(nil) != Transient {
		t.Error("empty slice should return Transient (default)")
	}
	if dominantFamily([]error{}) != Transient {
		t.Error("empty slice should return Transient (default)")
	}
}

func TestBatchErrorCode(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewRejection("r1", "bad"))

	err := b.Err().(*batchError)
	if err.ErrorCode() != "batch" {
		t.Errorf("ErrorCode() = %q, want \"batch\"", err.ErrorCode())
	}
	if err.ErrorFamily() != Rejection {
		t.Errorf("ErrorFamily() = %v, want Rejection", err.ErrorFamily())
	}
}

func TestBatchErrorRetryable(t *testing.T) {
	b := NewErrorBatch()
	b.Add(NewTransient("t", ""))
	err := b.Err()
	if !err.(Retryable).IsRetryable() {
		t.Error("all-transient batch should be retryable")
	}

	b2 := NewErrorBatch()
	b2.Add(NewRejection("r", ""))
	err2 := b2.Err()
	if err2.(Retryable).IsRetryable() {
		t.Error("rejection batch should not be retryable")
	}
}
