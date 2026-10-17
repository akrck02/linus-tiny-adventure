package game

import "testing"

func TestProgressAndOptionsComeBackAsWritten(t *testing.T) {
	progress := Progress{CurrentLevel: 2, MaxLevel: 3}
	if read := ReadProgress(progress.Marshal()); read != progress {
		t.Fatalf("the progress came back as %+v", read)
	}
	options := Options{Filter: FilterMidnight, Music: 0.2, Effects: 1}
	if read := ReadOptions(options.Marshal()); read != options {
		t.Fatalf("the options came back as %+v", read)
	}
}

func TestMissingOrBrokenFilesStartAnew(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("{"), []byte(`{"max_level": 0}`)} {
		if progress := ReadProgress(data); progress != NewProgress() {
			t.Fatalf("%q read as %+v", data, progress)
		}
	}
	if options := ReadOptions([]byte(`{"audio": {"music": 0.3}}`)); options.Music != 0.3 || options.Effects != 0.5 || options.Filter != DefaultFilter {
		t.Fatalf("a partial options file read as %+v", options)
	}
}
