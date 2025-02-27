package customtypes

import (
	"reflect"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

func TestRandaoMixes_Casting(t *testing.T) {
	var b [fieldparams.RandaoMixesLength][32]byte
	d := RandaoMixes(b)
	if !reflect.DeepEqual([fieldparams.RandaoMixesLength][32]byte(d), b) {
		t.Errorf("Unequal: %v = %v", d, b)
	}
}

func TestRandaoMixes_UnmarshalSSZ(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		d := RandaoMixes{}
		var b [fieldparams.RandaoMixesLength][32]byte
		b[0] = [32]byte{'f', 'o', 'o'}
		b[1] = [32]byte{'b', 'a', 'r'}
		bb := make([]byte, fieldparams.RandaoMixesLength*32)
		for i, elem32 := range b {
			for j, elem := range elem32 {
				bb[i*32+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(b, [fieldparams.RandaoMixesLength][32]byte(d)) {
			t.Errorf("Unequal: %v = %v", b, [fieldparams.RandaoMixesLength][32]byte(d))
		}
	})

	t.Run("Wrong slice length", func(t *testing.T) {
		d := RandaoMixes{}
		var b [fieldparams.RandaoMixesLength][16]byte
		b[0] = [16]byte{'f', 'o', 'o'}
		b[1] = [16]byte{'b', 'a', 'r'}
		bb := make([]byte, fieldparams.RandaoMixesLength*16)
		for i, elem16 := range b {
			for j, elem := range elem16 {
				bb[i*16+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestRandaoMixes_MarshalSSZTo(t *testing.T) {
	var d RandaoMixes
	dst := []byte("foo")
	b, err := d.MarshalSSZTo(dst)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := []byte{'f', 'o', 'o'}
	actual := []byte{b[0], b[1], b[2]}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unequal: %v = %v", expected, actual)
	}
}

func TestRandaoMixes_MarshalSSZ(t *testing.T) {
	d := RandaoMixes{}
	d[0] = [32]byte{'f', 'o', 'o'}
	d[1] = [32]byte{'b', 'a', 'r'}
	b, err := d.MarshalSSZ()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d[0][:], b[0:32]) {
		t.Errorf("Unequal: %v = %v", d[0], b[0:32])
	}
	if !reflect.DeepEqual(d[1][:], b[32:64]) {
		t.Errorf("Unequal: %v = %v", d[0], b[32:64])
	}
}

func TestRandaoMixes_SizeSSZ(t *testing.T) {
	d := RandaoMixes{}
	if d.SizeSSZ() != fieldparams.RandaoMixesLength*32 {
		t.Errorf("Wrong SSZ size. Expected %v vs actual %v", fieldparams.RandaoMixesLength*32, d.SizeSSZ())
	}
}

func TestHistoricalRoots_Casting(t *testing.T) {
	b := make([][32]byte, 4)
	d := HistoricalRoots(b)
	if !reflect.DeepEqual([][32]byte(d), b) {
		t.Errorf("Unequal: %v = %v", d, b)
	}
}

func TestHistoricalRoots_UnmarshalSSZ(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		d := HistoricalRoots(make([][32]byte, 2))
		b := make([][32]byte, 2)
		b[0] = [32]byte{'f', 'o', 'o'}
		b[1] = [32]byte{'b', 'a', 'r'}
		bb := make([]byte, 2*32)
		for i, elem32 := range b {
			for j, elem := range elem32 {
				bb[i*32+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(b, [][32]byte(d)) {
			t.Errorf("Unequal: %v = %v", b, [][32]byte(d))
		}
	})

	t.Run("Wrong slice length", func(t *testing.T) {
		d := HistoricalRoots(make([][32]byte, 2))
		b := make([][16]byte, 2)
		b[0] = [16]byte{'f', 'o', 'o'}
		b[1] = [16]byte{'b', 'a', 'r'}
		bb := make([]byte, 2*16)
		for i, elem16 := range b {
			for j, elem := range elem16 {
				bb[i*16+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestHistoricalRoots_MarshalSSZTo(t *testing.T) {
	d := HistoricalRoots(make([][32]byte, 1))
	d[0] = [32]byte{'f', 'o', 'o'}
	dst := []byte("bar")
	b, err := d.MarshalSSZTo(dst)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := []byte{'b', 'a', 'r', 'f', 'o', 'o'}
	actual := []byte{b[0], b[1], b[2], b[3], b[4], b[5]}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unequal: %v = %v", expected, actual)
	}
}

func TestHistoricalRoots_MarshalSSZ(t *testing.T) {
	d := HistoricalRoots(make([][32]byte, 2))
	d[0] = [32]byte{'f', 'o', 'o'}
	d[1] = [32]byte{'b', 'a', 'r'}
	b, err := d.MarshalSSZ()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d[0][:], b[0:32]) {
		t.Errorf("Unequal: %v = %v", d[0], b[0:32])
	}
	if !reflect.DeepEqual(d[1][:], b[32:64]) {
		t.Errorf("Unequal: %v = %v", d[0], b[32:64])
	}
}

func TestHistoricalRoots_SizeSSZ(t *testing.T) {
	d := HistoricalRoots(make([][32]byte, 2))
	if d.SizeSSZ() != 2*32 {
		t.Errorf("Wrong SSZ size. Expected %v vs actual %v", 2*32, d.SizeSSZ())
	}
}
