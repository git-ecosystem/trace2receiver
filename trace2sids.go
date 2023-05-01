package trace2receiver

import (
	"crypto/sha256"
	"strings"
)

// TODO `trace.TraceID` and `trace.SpanID` are defined as fixed-sized
// byte arrays but we are using the underlying [16]byte and [8]byte
// types rather than the defined types.
//
// I'm not sure if we can/should reference those classes from a receiver.
// They are more associated with a traceProvider and generating traces and
// spans.  Which is a whole different portion of the overall API from the
// `ptrace` APIs that we use in a receiver.  So for now, just hard-code the
// types here.

var zeroSpanID [8]byte

// Synthesize OTEL Trace and Span IDs using data in the Trace2 SID string.
//
// A Trace2 SID looks like a "<sid_0>/<sid_1>/.../<sid_n>" where a
// top-level Git command has SID "<sid_0>" and an immediate child
// process has SID "<sid_0>/<sid_1>" and so on.
//
// Since each of these commands will independently log telemetry to separate
// receiver workers threads (and child processes will finish before the
// parent process), we need a unique TraceID for the set to tie them together.
// And to synthesize Span IDs to capture the parent/child relationships
// encoded in the SID.
//
// These IDs cannot be constructed using a random number generator, so we
// use SHA256 on parts of the SID and (since each bit in a SHA result is
// uniformly distributed) extract substrings from the hashes in well-defined
// ways (so that other worker threads will compute the same values on the
// SIDs from other processes).
func extractIDsfromSID(rawSid string) (tid [16]byte, spid [8]byte, spidParent [8]byte) {
	sidArray := strings.Split(rawSid, "/")

	// Compute the hash on <sid_0> for the TraceID, since all child
	// processes will have <sid_0> in their SIDs.

	hash_0 := sha256.Sum256([]byte(sidArray[0]))
	copy(tid[:], hash_0[0:16])

	if len(sidArray) == 1 {
		// We are top-level command, so we have no parent Span and
		// we extract some bits from the SID for our SpanID.
		copy(spidParent[:], zeroSpanID[:])
		copy(spid[:], hash_0[16:24])
	} else {
		// We are a child (grandchild*) of a top-level command.
		// Compute hashes on the last 2 <sid_k> and extract SpanID
		// bits for this process and its parent process.
		n := len(sidArray) - 1

		hash_n1 := sha256.Sum256([]byte(sidArray[n-1]))
		copy(spidParent[:], hash_n1[16:24])

		hash_n := sha256.Sum256([]byte(sidArray[n]))
		copy(spid[:], hash_n[16:24])
	}

	return
}
