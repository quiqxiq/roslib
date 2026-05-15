package capability

import "strings"

// mutationActions adalah action name yang selalu jadi ClassMutation,
// terlepas dari arg apa yang dimilikinya. Ini cara paling pasti untuk
// memisahkan command mengubah-state dari query/streaming.
var mutationActions = map[string]struct{}{
	"add":                {},
	"remove":             {},
	"set":                {},
	"enable":             {},
	"disable":            {},
	"comment":            {},
	"move":               {},
	"edit":               {},
	"reset":              {},
	"reset-counters":     {},
	"reset-mac-address":  {},
	"unset":              {},
}

// inherentlyStreamingExact adalah path-action kombinasi yang sudah pasti
// streaming. Diisi dari hasil eksplorasi JSON 7.20.8.
var inherentlyStreamingExact = map[string]struct{}{
	"/tool/ping":              {},
	"/tool/torch":             {},
	"/tool/flood-ping":        {},
	"/tool/traceroute":        {},
	"/tool/bandwidth-test":    {},
	"/tool/speed-test":        {},
	"/tool/ping-speed":        {},
	"/tool/ip-scan":           {},
	"/tool/mac-scan":          {},
	"/interface/monitor-traffic": {},
}

// inherentlyStreamingPrefixes adalah sub-tree command yang seluruh anggotanya
// streaming (kecuali action mutation yang sudah ke ClassMutation duluan).
// Mis. semua di /tool/sniffer/ kecuali set/save adalah streaming.
var inherentlyStreamingPrefixes = []string{
	"/tool/sniffer/",
}

// classify menentukan Class untuk satu Command berdasarkan:
//
//  1. action name → mutation actions selalu ClassMutation
//  2. word exact match di inherentlyStreamingExact
//  3. word match prefix subtree streaming
//  4. action == "monitor" → ClassStreaming (capture /interface/.../monitor)
//  5. action == "print" dengan arg follow/follow-only/interval →
//     ClassStreamablePrint
//  6. arg punya "interval" tapi bukan print → ClassStreaming (capture
//     command monitoring lain yang punya interval polling)
//  7. else → ClassOneShot
func classify(word, action string, args map[string]struct{}) Class {
	if _, ok := mutationActions[action]; ok {
		return ClassMutation
	}
	if _, ok := inherentlyStreamingExact[word]; ok {
		return ClassStreaming
	}
	for _, p := range inherentlyStreamingPrefixes {
		if strings.HasPrefix(word, p) {
			return ClassStreaming
		}
	}
	if action == "monitor" {
		return ClassStreaming
	}
	if action == "print" {
		_, hasFollow := args["follow"]
		_, hasFollowOnly := args["follow-only"]
		_, hasInterval := args["interval"]
		if hasFollow || hasFollowOnly || hasInterval {
			return ClassStreamablePrint
		}
		return ClassOneShot
	}
	if _, hasInterval := args["interval"]; hasInterval {
		return ClassStreaming
	}
	return ClassOneShot
}
