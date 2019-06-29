package mg

import "testing"

func TestFileNameToVersion(t *testing.T) {
	for version, filename := range map[uint64]string{
		1:   "0000001.sql",
		200: "200_aaa.sql",
		111: "111aaa.sql",
	} {
		if v, err := filenameToVersion(filename); err != nil || version != v {
			t.Fatalf("FileName=%s, Version=%d, WantVersion=%d", filename, version, v)
		}
	}
}
