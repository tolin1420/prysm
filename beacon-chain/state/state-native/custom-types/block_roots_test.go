package customtypes

import (
	"reflect"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

func TestBlockRoots_Casting(t *testing.T) {
	var b [fieldparams.BlockRootsLength][32]byte
	d := BlockRoots(b)
	if !reflect.DeepEqual([fieldparams.BlockRootsLength][32]byte(d), b) {
		t.Errorf("Unequal: %v = %v", d, b)
	}
}

func TestBlockRoots_UnmarshalSSZ(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		d := BlockRoots{}
		var b [fieldparams.BlockRootsLength][32]byte
		b[0] = [32]byte{'f', 'o', 'o'}
		b[1] = [32]byte{'b', 'a', 'r'}
		bb := make([]byte, fieldparams.BlockRootsLength*32)
		for i, elem32 := range b {
			for j, elem := range elem32 {
				bb[i*32+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(b, [fieldparams.BlockRootsLength][32]byte(d)) {
			t.Errorf("Unequal: %v = %v", b, [fieldparams.BlockRootsLength][32]byte(d))
		}
	})

	t.Run("Wrong slice length", func(t *testing.T) {
		d := BlockRoots{}
		var b [fieldparams.BlockRootsLength][16]byte
		b[0] = [16]byte{'f', 'o', 'o'}
		b[1] = [16]byte{'b', 'a', 'r'}
		bb := make([]byte, fieldparams.BlockRootsLength*16)
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

func TestBlockRoots_MarshalSSZTo(t *testing.T) {
	var d BlockRoots
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

func TestBlockRoots_MarshalSSZ(t *testing.T) {
	d := BlockRoots{}
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

func TestBlockRoots_SizeSSZ(t *testing.T) {
	d := BlockRoots{}
	if d.SizeSSZ() != fieldparams.BlockRootsLength*32 {
		t.Errorf("Wrong SSZ size. Expected %v vs actual %v", fieldparams.BlockRootsLength*32, d.SizeSSZ())
	}
}
