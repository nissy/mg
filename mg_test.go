package mg

import (
	"testing"
)

func TestDo(t *testing.T) {
	ms, err := OpenCfg("mg.sample.toml")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"postgres-sample", "mysql-sample"} {
		if err := ms[v].Do(UpDo); err != nil {
			t.Fatal(err)
		}
		if err := ms[v].Do(DownDo); err != nil {
			t.Fatal(err)
		}
		if err := ms[v].Do(StatusDo); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFilenameToVersion(t *testing.T) {
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
