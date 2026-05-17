// Package quickprint menyediakan format/parse field source dari
// /system/script ber-comment "QuickPrintMikhmon" (analisis §7).
//
// Format source (#-separated, index 1-based di analisis):
//
//	#<name>#<server>#<usermode>#<length>#<prefix>#<charset>#<profile>#<timelimit>#<datalimit>#<comment>#<validity>#<price>_<sprice>#<lock>
package quickprint
