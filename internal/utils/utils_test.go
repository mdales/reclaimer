package utils

import (
	"os"
	"path"
	"testing"
)

func TestMakeOutputDirRelative(t *testing.T) {
	tempdir := t.TempDir()
	err := os.Chdir(tempdir)
	if nil != err {
		panic(err)
	}

	name, err := MakeOutputPath("test.foo", "")
	if nil != err {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := path.Join(tempdir, "test.foo")
	if expected != name {
		t.Errorf("Expected %s, got %s", expected, name)
	}
}

func TestEmptySourceNameFails(t *testing.T) {
	testcases := []string{
		"",
		"blah.txt",
	}
	for _, testcase := range testcases {
		name, err := MakeOutputPath("", testcase)
		if nil == err {
			t.Errorf("Expected failure, but got a name: %s", name)
		}
	}
}

func TestMakeOutputDirSimple(t *testing.T) {
	tempdir := t.TempDir()
	err := os.Chdir(tempdir)
	if nil != err {
		panic(err)
	}

	name, err := MakeOutputPath("test.foo", "blam.txt")
	if nil != err {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := path.Join(tempdir, "blam.txt")
	if expected != name {
		t.Errorf("Expected %s, got %s", expected, name)
	}
}

func TestMakeOutputDirAbsolute(t *testing.T) {
	tempdir1 := t.TempDir()
	err := os.Chdir(tempdir1)
	if nil != err {
		panic(err)
	}

	tempdir2 := t.TempDir()
	if tempdir1 == tempdir2 {
		t.Fatalf("Expected each temp dir to be unique: %s and %s", tempdir1, tempdir2)
	}

	target := path.Join(tempdir2, "blam.txt")
	name, err := MakeOutputPath("test.foo", target)
	if nil != err {
		t.Errorf("Expected no error, got %v", err)
	}
	if target != name {
		t.Errorf("Expected %s, got %s", target, name)
	}
}

func TestMakeOutputDirDoesMakeDir(t *testing.T) {
	tempdir1 := t.TempDir()
	err := os.Chdir(tempdir1)
	if nil != err {
		panic(err)
	}

	tempdir2 := t.TempDir()
	if tempdir1 == tempdir2 {
		t.Fatalf("Expected each temp dir to be unique: %s and %s", tempdir1, tempdir2)
	}

	required := path.Join(tempdir2, "test")
	target := path.Join(required, "blam.txt")
	name, err := MakeOutputPath("test.foo", target)
	if nil != err {
		t.Errorf("Expected no error, got %v", err)
	}
	if target != name {
		t.Errorf("Expected %s, got %s", target, name)
	}

	stat, err := os.Stat(required)
	if nil != err {
		t.Errorf("Expected output dir to be created, but got stat err: %v", err)
	} else {
		if !stat.IsDir() {
			t.Errorf("Expected output dir to be a dir")
		}
	}
}
