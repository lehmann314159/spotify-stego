package camelot

var keyToCode = map[string]string{
	// Minor keys (A)
	"Am":  "1A",
	"Em":  "2A",
	"Bm":  "3A",
	"F#m": "4A",
	"Gbm": "4A",
	"C#m": "5A",
	"Dbm": "5A",
	"G#m": "6A",
	"Abm": "6A",
	"Ebm": "7A",
	"D#m": "7A",
	"Bbm": "8A",
	"A#m": "8A",
	"Fm":  "9A",
	"Cm":  "10A",
	"Gm":  "11A",
	"Dm":  "12A",

	// Major keys (B)
	"C":  "1B",
	"G":  "2B",
	"D":  "3B",
	"A":  "4B",
	"E":  "5B",
	"B":  "6B",
	"F#": "7B",
	"Gb": "7B",
	"C#": "8B",
	"Db": "8B",
	"Ab": "9B",
	"G#": "9B",
	"Eb": "10B",
	"D#": "10B",
	"Bb": "11B",
	"A#": "11B",
	"F":  "12B",
}

// KeyToCode maps a musical key string to its Camelot wheel code.
// Returns ("", false) if the key is not recognized.
func KeyToCode(keyOf string) (string, bool) {
	code, ok := keyToCode[keyOf]
	return code, ok
}
